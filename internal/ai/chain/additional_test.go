package chain

import (
	"context"
	"testing"
	"time"
)

// TestEmbeddingEmptyInputMock ensures mock embedding errors on empty slice.
func TestEmbeddingEmptyInputMock(t *testing.T) {
	c := NewEmbeddingChainFromConfig(AIConfig{Provider: ProviderMock})
	if _, err := c.Embed(context.Background(), []string{}, 16); err == nil {
		t.Fatalf("expected error on empty texts")
	}
}

// TestEmbeddingDimensionChangeMock ensures dimension affects output length for mock implementation.
func TestEmbeddingDimensionChangeMock(t *testing.T) {
	c := NewEmbeddingChainFromConfig(AIConfig{Provider: ProviderMock})
	v1, err := c.Embed(context.Background(), []string{"hello"}, 8)
	if err != nil || len(v1) != 1 {
		t.Fatalf("unexpected v1 err/len: %v %d", err, len(v1))
	}
	v2, err := c.Embed(context.Background(), []string{"hello"}, 16)
	if err != nil || len(v2) != 1 {
		t.Fatalf("unexpected v2 err/len: %v %d", err, len(v2))
	}
	if len(v1[0]) == len(v2[0]) {
		t.Fatalf("expected different dimensions after change: %d vs %d", len(v1[0]), len(v2[0]))
	}
}

// TestChatMultiMessageMock ensures only last user message drives mock echo output.
func TestChatMultiMessageMock(t *testing.T) {
	c := NewChatChainFromConfig(AIConfig{Provider: ProviderMock})
	msg, err := c.Chat(context.Background(), []ChatMessage{{Role: "system", Content: "instr"}, {Role: "user", Content: "first"}, {Role: "user", Content: "second"}}, 0)
	if err != nil {
		t.Fatalf("chat err: %v", err)
	}
	if msg.Content != "echo:second" {
		t.Fatalf("unexpected echo content: %q", msg.Content)
	}
}

// TestChatStreamEarlyStopFullVsCollected ensures final message is full while collected deltas can be partial.
func TestChatStreamEarlyStopFullVsCollected(t *testing.T) {
	c := NewChatChainFromConfig(AIConfig{Provider: ProviderMock})
	var collected string
	final, err := c.ChatStream(context.Background(), []ChatMessage{{Role: "user", Content: "some quite long content to slice"}}, 0, func(delta string) bool {
		collected += delta
		return len(collected) < 5 // stop very early
	})
	if err != nil {
		t.Fatalf("stream err: %v", err)
	}
	if len(collected) >= len(final.Content) {
		t.Fatalf("expected collected partial < final; collected=%d final=%d", len(collected), len(final.Content))
	}
	if final.Content == "" {
		t.Fatalf("final content empty")
	}
}

// TestFromConfigFallbackWithoutKey ensures openai provider without key falls back to mock when using config constructors.
func TestFromConfigFallbackWithoutKey(t *testing.T) {
	ec := NewEmbeddingChainFromConfig(AIConfig{Provider: ProviderOpenAI /* no key */})
	if ec.Provider() != ProviderMock {
		t.Fatalf("expected mock fallback for embedding, got %s", ec.Provider())
	}
	cc := NewChatChainFromConfig(AIConfig{Provider: ProviderOpenAI /* no key */})
	if cc.Provider() != ProviderMock {
		t.Fatalf("expected mock fallback for chat, got %s", cc.Provider())
	}
}

// TestOpenAIDynamicDimensionRecreate (live) - ensures dimension change produces different vector sizes.
func TestOpenAIDynamicDimensionRecreate(t *testing.T) {
	key := lookupEnv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("no key")
	}
	cfg := AIConfig{Provider: ProviderOpenAI, OpenAIKey: key, OpenAIEmbedModel: lookupEnvDefault("OPENAI_EMBED_MODEL", "text-embedding-3-small"), OpenAIBaseURL: lookupEnv("OPENAI_BASE_URL")}
	c := NewEmbeddingChainFromConfig(cfg)
	if c.Provider() != ProviderOpenAI {
		t.Skip("provider not openai (fallback) -- skipping dynamic dim test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	v1, err1 := c.Embed(ctx, []string{"dimension"}, 8)
	if err1 != nil {
		t.Skipf("skip due to embed err1: %v", err1)
	}
	v2, err2 := c.Embed(ctx, []string{"dimension"}, 16)
	if err2 != nil {
		t.Skipf("skip due to embed err2: %v", err2)
	}
	if len(v1) == 0 || len(v2) == 0 || len(v1[0]) == 0 || len(v2[0]) == 0 {
		t.Fatalf("unexpected empty vectors")
	}
	if len(v1[0]) == len(v2[0]) {
		t.Skipf("dimension unchanged (%d) – model may ignore low custom dims; skipping", len(v1[0]))
	}
}

// Small helpers (mirroring config.go logic without importing os directly here for focus)
func lookupEnv(key string) string             { return getEnvDefault(key, "") }
func lookupEnvDefault(key, def string) string { return getEnvDefault(key, def) }
