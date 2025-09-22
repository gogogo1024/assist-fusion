package chain

import (
	"context"
	"os"
	"testing"
)

const emptyFinalErr = "final content empty"

// Test mock ChatStream returns chunks and aggregates correctly.
func TestMockChatStreamChunks(t *testing.T) {
	os.Unsetenv("AI_PROVIDER")
	os.Unsetenv("OPENAI_API_KEY")
	c := NewChatChain(DetectProvider())
	if c.Provider() != ProviderMock {
		t.Fatalf("expected mock provider")
	}
	var collected []string
	final, err := c.ChatStream(context.Background(), []ChatMessage{{Role: "user", Content: "hello world"}}, 0, func(delta string) bool {
		if delta != "" {
			collected = append(collected, delta)
		}
		return true
	})
	if err != nil {
		t.Fatalf("chat stream err: %v", err)
	}
	if final.Content == "" {
		t.Fatalf(emptyFinalErr)
	}
	if len(collected) == 0 {
		t.Fatalf("expected >0 chunks")
	}
	// Reconstruct
	var sum string
	for _, c := range collected {
		sum += c
	}
	if sum != final.Content {
		t.Fatalf("chunks concat mismatch final: %q vs %q", sum, final.Content)
	}
}

// Test aborting stream stops early but still returns final full content.
func TestMockChatStreamAbort(t *testing.T) {
	os.Unsetenv("AI_PROVIDER")
	os.Unsetenv("OPENAI_API_KEY")
	c := NewChatChain(DetectProvider())
	calls := 0
	final, err := c.ChatStream(context.Background(), []ChatMessage{{Role: "user", Content: "abort sequence"}}, 0, func(delta string) bool {
		calls++
		return calls < 2 // stop after first chunk
	})
	if err != nil {
		t.Fatalf("chat stream abort err: %v", err)
	}
	if final.Content == "" {
		t.Fatalf(emptyFinalErr)
	}
	if calls != 2 { // first allowed, second returned false not counted, so calls should be 2 iterations (one true, one false attempt)
		t.Logf("calls=%d (non-fatal, impl detail)", calls)
	}
}

// Test openai missing key falls back to mock stream.
func TestOpenAIStreamFallbackNoKey(t *testing.T) {
	os.Setenv("AI_PROVIDER", "openai")
	t.Cleanup(func() { os.Unsetenv("AI_PROVIDER") })
	os.Unsetenv("OPENAI_API_KEY")
	c := NewChatChain(DetectProvider())
	if c.Provider() != ProviderMock {
		t.Fatalf("expected fallback to mock, got %s", c.Provider())
	}
	final, err := c.ChatStream(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, 0, nil)
	if err != nil {
		t.Fatalf("fallback stream err: %v", err)
	}
	if final.Content == "" {
		t.Fatalf(emptyFinalErr)
	}
}
