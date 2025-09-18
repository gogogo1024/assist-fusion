package handler

import (
	"context"

	"github.com/gogogo1024/assist-fusion/internal/ai"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
)

// AIServiceImpl implements embeddings via mock generator.
type AIServiceImpl struct{}

func NewAIService() *AIServiceImpl { return &AIServiceImpl{} }

func (s *AIServiceImpl) Embeddings(ctx context.Context, req *kcommon.EmbeddingRequest) (*kcommon.EmbeddingResponse, error) {
	if req == nil || len(req.Texts) == 0 {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: "texts required"}
	}
	vecs := ai.MockEmbeddings(req.Texts, int(req.Dim))
	observability.AIEmbeddingCalls.Add(1)
	out := make([][]float64, len(vecs))
	for i, v := range vecs {
		out[i] = v
	}
	dim := int32(0)
	if len(vecs) > 0 {
		dim = int32(len(vecs[0]))
	}
	return &kcommon.EmbeddingResponse{Vectors: out, Dim: dim}, nil
}
