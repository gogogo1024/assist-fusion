package chain

import (
	"context"
	"os"
	"testing"
	"time"
)

// These tests hit real OpenAI endpoints when OPENAI_API_KEY is present.
// They are skipped automatically if the key is absent or a network/API error occurs.

func requireOpenAIKey(t *testing.T) string {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set; skipping live OpenAI tests")
	}
	return key
}

func TestOpenAILiveEmbedding(t *testing.T) {
	requireOpenAIKey(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	chain := NewEmbeddingChain(ProviderOpenAI)
	vecs, err := chain.Embed(ctx, []string{"hello world"}, 64)
	if err != nil {
		t.Skipf("skip due to live embedding error: %v", err)
	}
	if len(vecs) != 1 || len(vecs[0]) == 0 {
		t.Fatalf("unexpected embedding shape: %#v", vecs)
	}
}

func TestOpenAILiveChat(t *testing.T) {
	requireOpenAIKey(t)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	chain := NewChatChain(ProviderOpenAI)
	msg, err := chain.Chat(ctx, []ChatMessage{{Role: "user", Content: "Say 'pong'"}}, 32)
	if err != nil {
		t.Skipf("skip due to live chat error: %v", err)
	}
	if msg.Content == "" {
		t.Fatalf("empty chat content")
	}
}

func TestOpenAILiveChatStream(t *testing.T) {
	requireOpenAIKey(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chain := NewChatChain(ProviderOpenAI)
	var collected string
	_, err := chain.ChatStream(ctx, []ChatMessage{{Role: "user", Content: "Respond with a short haiku about Go"}}, 64, func(delta string) bool {
		collected += delta
		// Stop early if we already have reasonable output to reduce token usage
		return len(collected) < 160
	})
	if err != nil {
		t.Skipf("skip due to live chat stream error: %v", err)
	}
	if len(collected) == 0 {
		t.Fatalf("did not collect any streamed content")
	}
}
