package minhash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashCode(t *testing.T) {
	a := hashCode("some string")
	b := hashCode("some string.")

	assert.NotEqual(t, a, b, "hashCode should generate different hashes")
}

func TestJaccard(t *testing.T) {
	cases := []struct {
		name     string
		a        vector
		b        vector
		expected float64
	}{
		{
			a:        vector{0, 1, 2, 3},
			b:        vector{0, 1, 2, 4},
			expected: 3.0 / 5.0,
		},
		{
			a:        vector{0, 1, 2, 3},
			b:        vector{0, 1, 2, 3},
			expected: 1.0,
		},
	}

	for _, c := range cases {
		result := jaccard(c.a, c.b)

		assert.Equal(t, c.expected, result)
	}
}

func TestGenerateHashers(t *testing.T) {
	hashers := generateHahsers(2, 7)

	assert.Len(t, hashers, 2)
	a := hashers[0](5)
	b := hashers[1](5)

	assert.NotEqual(t, a, b)
}
