package chain

import (
	"context"
	"os"
	"testing"
)

// Test that with OPENAI_API_KEY set we return an openai provider (either eino or legacy) not mock.
func TestEinoProviderSelectionWithKey(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("no key; skip")
	}
	ch := NewEmbeddingChain(ProviderOpenAI)
	if ch.Provider() != ProviderOpenAI {
		t.Fatalf("expected provider openai got %s", ch.Provider())
	}
	// small smoke embed (skip errors network wise)
	_, _ = ch.Embed(context.Background(), []string{"ping"}, 8)
}

// Test fallback when key is absent: provider should degrade to mock.
func TestEinoProviderFallbackNoKey(t *testing.T) {
	old := os.Getenv("OPENAI_API_KEY")
	_ = os.Unsetenv("OPENAI_API_KEY")
	defer func() { _ = os.Setenv("OPENAI_API_KEY", old) }()

	ch := NewEmbeddingChain(ProviderOpenAI)
	if ch.Provider() == ProviderOpenAI {
		// This could still be openai if legacy path somehow bypassed key (should not)
		t.Fatalf("expected fallback mock when no key, got openai")
	}
	if ch.Provider() != ProviderMock {
		t.Fatalf("expected provider mock, got %s", ch.Provider())
	}
}
