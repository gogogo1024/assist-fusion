package ai

import "testing"

func TestMockEmbeddingsDeterministic(t *testing.T) {
	texts := []string{"hello", "world"}
	vecs1 := MockEmbeddings(texts, 8)
	vecs2 := MockEmbeddings(texts, 8)
	if len(vecs1) != len(vecs2) {
		t.Fatalf("length mismatch")
	}
	for i := range vecs1 {
		if len(vecs1[i]) != 8 || len(vecs2[i]) != 8 {
			t.Fatalf("unexpected dim")
		}
		for j := range vecs1[i] {
			if vecs1[i][j] != vecs2[i][j] {
				t.Fatalf("non-deterministic at %d,%d", i, j)
			}
		}
	}
}
