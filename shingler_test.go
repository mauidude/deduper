package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShingler(t *testing.T) {
	cases := []struct {
		input    string
		n        int
		expected []string
	}{
		{
			input: "this is a test of the shingler",
			n:     2,
			expected: []string{
				"this is",
				"is a",
				"a test",
				"test of",
				"of the",
				"the shingler",
			},
		},
		{
			input:    "this is a test of the shingler",
			n:        7,
			expected: []string{"this is a test of the shingler"},
		},
	}
	for _, c := range cases {
		r := strings.NewReader(c.input)
		s := NewShingler(r, c.n)

		i := 0
		for s.Scan() {
			assert.Equal(t, c.expected[i], s.Text())
			i++
		}

		assert.Equal(t, len(c.expected), i)
	}
}

func TestShingler_SameLength(t *testing.T) {
	r := strings.NewReader("this is a test of the shingler")
	s := NewShingler(r, 7)

	expected := "this is a test of the shingler"

	i := 0
	for s.Scan() {
		assert.Equal(t, expected, s.Text())
		i++
	}

	assert.Equal(t, 1, i)
}
