package router

import (
	"context"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	aidl "github.com/gogogo1024/assist-fusion/kitex_gen/ai"
	aicli "github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	commonidl "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

// VectorIndex stores normalized embeddings for docs (gateway-side, best-effort cache).
type VectorIndex struct {
	mu   sync.RWMutex
	dim  int
	docs map[string]*vectorDoc
}

type vectorDoc struct {
	id      string
	title   string
	content string
	vec     []float32 // normalized
}

func newVectorIndex(dim int) *VectorIndex {
	return &VectorIndex{dim: dim, docs: map[string]*vectorDoc{}}
}

func (vi *VectorIndex) upsert(id, title, content string, vec []float32) {
	if id == "" || len(vec) == 0 {
		return
	}
	if vi.dim == 0 {
		vi.dim = len(vec)
	}
	if len(vec) != vi.dim {
		return
	} // skip inconsistent dims
	nvec := normalize(vec)
	vi.mu.Lock()
	vi.docs[id] = &vectorDoc{id: id, title: title, content: content, vec: nvec}
	vi.mu.Unlock()
}

func (vi *VectorIndex) delete(id string) {
	vi.mu.Lock()
	delete(vi.docs, id)
	vi.mu.Unlock()
}

type scored struct {
	id      string
	title   string
	content string
	score   float64
}

func (vi *VectorIndex) search(qv []float32, limit int) []scored {
	if vi == nil || limit <= 0 {
		return nil
	}
	if len(qv) != vi.dim {
		return nil
	}
	vi.mu.RLock()
	out := make([]scored, 0, len(vi.docs))
	for _, d := range vi.docs {
		if len(d.vec) != vi.dim {
			continue
		}
		s := dot(qv, d.vec) // both normalized => cosine
		out = append(out, scored{id: d.id, title: d.title, content: d.content, score: float64(s)})
	}
	vi.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func dot(a, b []float32) float32 {
	var s float32
	for i := 0; i < len(a) && i < len(b); i++ {
		s += a[i] * b[i]
	}
	return s
}
func normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x * x)
	}
	if sum == 0 {
		return v
	}
	inv := 1.0 / math.Sqrt(sum)
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) * inv)
	}
	return out
}

var (
	vecOnce     sync.Once
	vecIdx      *VectorIndex
	aiEmbClient aicli.Client
)

// EnableVectorSearch initializes the in-memory vector index and registers HTTP route.
func EnableVectorSearch(h *server.Hertz, ai aicli.Client) {
	if ai == nil {
		return
	}
	aiEmbClient = ai
	vecOnce.Do(func() {
		dim := 128
		if v := os.Getenv("KB_VECTOR_DIM"); v != "" { // optional override
			// ignore conv error silently
		}
		vecIdx = newVectorIndex(dim)
		registerVectorRoute(h)
	})
}

func registerVectorRoute(h *server.Hertz) {
	h.GET(PathVectorSearch, func(c context.Context, ctx *app.RequestContext) {
		q := strings.TrimSpace(string(ctx.Query("q")))
		if q == "" {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, "bad_request", gwerrors.MsgBadRequest)
			return
		}
		limit := 10
		if v := ctx.Query("limit"); len(v) > 0 {
			if n := atoiSafe(string(v)); n > 0 && n <= 50 {
				limit = n
			}
		}
		// embed query
		resp, err := aiEmbClient.Embeddings(c, &commonidl.EmbeddingRequest{Texts: []string{q}, Dim: int32(vecIdx.dim)})
		if err != nil || resp == nil || len(resp.Vectors) == 0 {
			gwerrors.HTTPError(ctx, http.StatusInternalServerError, "ai_error", gwerrors.MsgInternal)
			return
		}
		qv := toFloat32(resp.Vectors[0])
		// already normalized at insert; also normalize qv
		qv = normalize(qv)
		results := vecIdx.search(qv, limit)
		items := make([]map[string]any, 0, len(results))
		for _, r := range results {
			snippet := snippet(r.content, 120)
			items = append(items, map[string]any{"id": r.id, "title": r.title, "score": r.score, "snippet": snippet})
		}
		ctx.JSON(http.StatusOK, map[string]any{"items": items, "returned": len(items), "total": len(vecIdx.docs)})
	})
}

// UpsertDocEmbedding is called by KB doc handlers after a successful mutation.
func UpsertDocEmbedding(ctx context.Context, id, title, content string) {
	if aiEmbClient == nil || vecIdx == nil || id == "" {
		return
	}
	text := strings.TrimSpace(title + "\n" + content)
	if text == "" {
		return
	}
	resp, err := aiEmbClient.Embeddings(ctx, &commonidl.EmbeddingRequest{Texts: []string{text}, Dim: int32(vecIdx.dim)})
	if err != nil || resp == nil || len(resp.Vectors) == 0 {
		return
	}
	vecIdx.upsert(id, title, content, toFloat32(resp.Vectors[0]))
}

// DeleteDocEmbedding removes a doc from vector index.
func DeleteDocEmbedding(id string) {
	if vecIdx != nil {
		vecIdx.delete(id)
	}
}

func toFloat32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(x)
	}
	return out
}
func snippet(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// naive atoi
func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// expose for other files (kb handlers)
type AIEmbeddingClient interface {
	Embeddings(ctx context.Context, req *commonidl.EmbeddingRequest) (*commonidl.EmbeddingResponse, error)
}

var _ = aidl.ChatMessage{} // silence unused import in case
