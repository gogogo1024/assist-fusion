package impl

import (
	"context"
	"os"
	"sync"

	"github.com/gogogo1024/assist-fusion/internal/ai"
	aichain "github.com/gogogo1024/assist-fusion/internal/ai/chain"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	aidl "github.com/gogogo1024/assist-fusion/kitex_gen/ai"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
)

// AIServiceImpl implements the AI service using internal embedding/chat chains.
type AIServiceImpl struct {
	embedOnce sync.Once
	chatOnce  sync.Once
	embed     aichain.EmbeddingChain
	chat      aichain.ChatChain
}

func NewAIService() *AIServiceImpl { return &AIServiceImpl{} }

func (s *AIServiceImpl) initEmbed() {
	s.embedOnce.Do(func() {
		if os.Getenv("AI_CHAIN_DISABLE") == "1" { // forced mock fallback
			s.embed = nil
			return
		}
		provider := aichain.DetectProvider()
		s.embed = aichain.NewEmbeddingChain(provider)
	})
}

func (s *AIServiceImpl) initChat() {
	s.chatOnce.Do(func() {
		if os.Getenv("AI_CHAIN_DISABLE") == "1" { // forced mock fallback
			s.chat = nil
			return
		}
		provider := aichain.DetectProvider()
		s.chat = aichain.NewChatChain(provider)
	})
}

// Embeddings implements the AI Embeddings RPC (adapter wrapper kept for chain package relocation earlier)
// NOTE: This function already defined further below after refactor; duplicate removed.

// Embeddings returns vector embeddings with fallback + metrics instrumentation.
func (s *AIServiceImpl) Embeddings(ctx context.Context, req *kcommon.EmbeddingRequest) (*kcommon.EmbeddingResponse, error) {
	if req == nil || len(req.Texts) == 0 {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: "texts required"}
	}
	s.initEmbed()

	var vecs [][]float64
	var err error
	provider := "mock"
	if s.embed == nil { // forced mock fallback
		vecs = ai.MockEmbeddings(req.Texts, int(req.Dim))
		observability.AIEmbeddingFallbackMock.Add(1)
	} else {
		provider = s.embed.Provider()
		vecs, err = s.embed.Embed(ctx, req.Texts, int(req.Dim))
		if err != nil || len(vecs) == 0 {
			// provider error path
			switch provider {
			case aichain.ProviderOpenAI:
				observability.AIEmbeddingErrorOpenAI.Add(1)
				observability.AIEmbeddingFallbackOpenAI.Add(1)
			default:
				observability.AIEmbeddingError.Add(1)
				observability.AIEmbeddingFallbackMock.Add(1)
			}
			vecs = ai.MockEmbeddings(req.Texts, int(req.Dim))
		} else {
			switch provider {
			case aichain.ProviderOpenAI:
				observability.AIEmbeddingSuccessOpenAI.Add(1)
			default:
				observability.AIEmbeddingSuccessMock.Add(1)
			}
		}
	}
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

// Chat provides simple chat with fallback echo + metrics instrumentation.
func (s *AIServiceImpl) Chat(ctx context.Context, req *aidl.ChatRequest) (*aidl.ChatResponse, error) {
	if req == nil || len(req.Messages) == 0 {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: "messages required"}
	}
	s.initChat()

	// Convert messages
	msgs := make([]aichain.ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, aichain.ChatMessage{Role: m.Role, Content: m.Content})
	}
	if s.chat == nil { // fallback
		last := msgs[len(msgs)-1]
		observability.AIChatFallbackMock.Add(1)
		return &aidl.ChatResponse{Message: &aidl.ChatMessage{Role: "assistant", Content: "echo:" + last.Content}}, nil
	}
	maxTokens := 0
	if req.MaxTokens != nil {
		maxTokens = int(*req.MaxTokens)
	}
	resp, err := s.chat.Chat(ctx, msgs, maxTokens)
	provider := s.chat.Provider()
	if err != nil || resp.Content == "" {
		switch provider {
		case aichain.ProviderOpenAI:
			observability.AIChatErrorOpenAI.Add(1)
			observability.AIChatFallbackOpenAI.Add(1)
		default:
			observability.AIChatError.Add(1)
			observability.AIChatFallbackMock.Add(1)
		}
		last := msgs[len(msgs)-1]
		return &aidl.ChatResponse{Message: &aidl.ChatMessage{Role: "assistant", Content: "echo:" + last.Content}}, nil
	}
	switch provider {
	case aichain.ProviderOpenAI:
		observability.AIChatSuccessOpenAI.Add(1)
	default:
		observability.AIChatSuccessMock.Add(1)
	}
	return &aidl.ChatResponse{Message: &aidl.ChatMessage{Role: resp.Role, Content: resp.Content}}, nil
}
