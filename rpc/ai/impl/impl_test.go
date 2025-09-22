package impl

import (
	"context"
	"os"
	"testing"

	"github.com/gogogo1024/assist-fusion/internal/observability"
	aidl "github.com/gogogo1024/assist-fusion/kitex_gen/ai"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
)

// reset counters helper
func resetAICounters() {
	observability.AIEmbeddingCalls.Store(0)
	observability.AIEmbeddingSuccessMock.Store(0)
	observability.AIEmbeddingFallbackMock.Store(0)
	observability.AIEmbeddingError.Store(0)
	observability.AIChatSuccessMock.Store(0)
	observability.AIChatFallbackMock.Store(0)
	observability.AIChatError.Store(0)
}

func TestEmbeddingsFallbackMock(t *testing.T) {
	resetAICounters()
	os.Setenv("AI_CHAIN_DISABLE", "1")
	t.Cleanup(func() { os.Unsetenv("AI_CHAIN_DISABLE") })
	svc := NewAIService()
	resp, err := svc.Embeddings(context.Background(), &kcommon.EmbeddingRequest{Texts: []string{"hello"}, Dim: 16})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Vectors) != 1 || int(resp.Dim) != len(resp.Vectors[0]) {
		t.Fatalf("dimension mismatch: dim=%d vec=%v", resp.Dim, resp.Vectors)
	}
	if observability.AIEmbeddingCalls.Load() != 1 {
		t.Fatalf("expected AIEmbeddingCalls=1")
	}
	if observability.AIEmbeddingFallbackMock.Load() != 1 {
		t.Fatalf("expected fallback counter=1")
	}
}

func TestChatFallbackEcho(t *testing.T) {
	resetAICounters()
	os.Setenv("AI_CHAIN_DISABLE", "1")
	t.Cleanup(func() { os.Unsetenv("AI_CHAIN_DISABLE") })
	svc := NewAIService()
	resp, err := svc.Chat(context.Background(), &aidl.ChatRequest{Messages: []*aidl.ChatMessage{{Role: "user", Content: "ping"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Message == nil || resp.Message.Content != "echo:ping" {
		t.Fatalf("expected echo response, got %+v", resp.Message)
	}
	if observability.AIChatFallbackMock.Load() != 1 {
		t.Fatalf("expected chat fallback=1")
	}
}
