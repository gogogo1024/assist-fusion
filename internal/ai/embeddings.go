package ai

import (
	"crypto/sha256"
	"encoding/binary"
)

// MockEmbeddings returns deterministic vectors for given texts using hashing, for reproducibility.
func MockEmbeddings(texts []string, dim int) [][]float64 {
	if dim <= 0 {
		dim = 32
	}
	out := make([][]float64, len(texts))
	for i, t := range texts {
		h := sha256.Sum256([]byte(t))
		vec := make([]float64, dim)
		for d := 0; d < dim; d++ {
			// use hash chunks to seed values in [-1,1]
			idx := (d * 2) % len(h)
			u := binary.BigEndian.Uint16(h[idx : idx+2])
			vec[d] = (float64(u%2000) - 1000.0) / 1000.0
		}
		out[i] = vec
	}
	return out
}
