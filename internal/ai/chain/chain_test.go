package chain

import (
	"context"
	"os"
	"testing"
)

func TestMockEmbeddingDeterministic(t *testing.T) {
	os.Unsetenv("AI_PROVIDER")
	c := NewEmbeddingChain(DetectProvider())
	if c.Provider() != ProviderMock {
		t.Fatalf("expected mock provider")
	}
	v1, err := c.Embed(context.Background(), []string{"hello"}, 8)
	if err != nil {
		t.Fatalf("embed err: %v", err)
	}
	v2, err := c.Embed(context.Background(), []string{"hello"}, 8)
	if err != nil {
		t.Fatalf("embed err2: %v", err)
	}
	if len(v1) != 1 || len(v2) != 1 || len(v1[0]) != len(v2[0]) {
		t.Fatalf("unexpected dims")
	}
	for i := range v1[0] {
		if v1[0][i] != v2[0][i] {
			t.Fatalf("non-deterministic component %d", i)
		}
	}
}

func TestOpenAIProviderWithoutKeyFallbackToMock(t *testing.T) {
	os.Setenv("AI_PROVIDER", "openai")
	t.Cleanup(func() { os.Unsetenv("AI_PROVIDER") })
	os.Unsetenv("OPENAI_API_KEY")
	c := NewEmbeddingChain(DetectProvider())
	// Because constructor fails, we fall back to mock
	if c.Provider() != ProviderMock {
		t.Fatalf("expected fallback to mock when no key; got %s", c.Provider())
	}
}
