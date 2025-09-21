package impl

import (
	"context"

	"github.com/gogogo1024/assist-fusion/internal/ai"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	aidl "github.com/gogogo1024/assist-fusion/kitex_gen/ai"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
)

// AIServiceImpl implements the AI service using internal mock embedding generator.
type AIServiceImpl struct{}

func NewAIService() *AIServiceImpl { return &AIServiceImpl{} }

func (s *AIServiceImpl) Embeddings(ctx context.Context, req *kcommon.EmbeddingRequest) (*kcommon.EmbeddingResponse, error) {
	if req == nil || len(req.Texts) == 0 {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: "texts required"}
	}
	vecs := ai.MockEmbeddings(req.Texts, int(req.Dim))
	observability.AIEmbeddingCalls.Add(1)
	out := make([][]float64, len(vecs))
	for i := range vecs {
		out[i] = make([]float64, len(vecs[i]))
		copy(out[i], vecs[i])
	}
	dim := int32(0)
	if len(vecs) > 0 {
		dim = int32(len(vecs[0]))
	}
	return &kcommon.EmbeddingResponse{Vectors: out, Dim: dim}, nil
}

// Chat currently a stub: returns not_implemented error for now.
func (s *AIServiceImpl) Chat(ctx context.Context, req *aidl.ChatRequest) (*aidl.ChatResponse, error) {
	return nil, &kcommon.ServiceError{Code: "not_implemented", Message: "chat not implemented"}
}
