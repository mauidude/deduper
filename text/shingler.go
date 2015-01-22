package text

import (
	"bufio"
	"io"
	"strings"
)

// NewShingler creates a new shingler for the given reader
// and produce shingles of size n.
func NewShingler(r io.Reader, n int) *Shingler {
	return &Shingler{
		r: r,
		n: n,
	}
}

// Shingler creates a sliding window of n-grams of each token (word).
type Shingler struct {
	// the reader we are shingling
	r io.Reader

	// scanner used to tokenize the reader
	s *bufio.Scanner

	// the size of the shingles
	n int

	// the queue of tokens in the current window
	q []string
}

// Scan will return true and advance to the next n-gram
// until an EOF has been reached on the reader
// at which time it will return false.
func (s *Shingler) Scan() bool {
	if s.q == nil {
		// initialize everything
		s.q = make([]string, s.n)
		s.s = bufio.NewScanner(s.r)
		s.s.Split(bufio.ScanWords)

		for i := 0; i < s.n; i++ {
			next, ok := s.next()
			if !ok {
				s.q = s.q[:0]
				return false
			}

			s.q = append(s.q[1:], next)
		}

		return true
	}

	next, ok := s.next()
	if !ok {
		return false
	}

	s.q = append(s.q[1:], next)
	return true
}

// Text returns the shingles in the current window.
func (s *Shingler) Text() string {
	return strings.Join(s.q, " ")
}

func (s *Shingler) next() (string, bool) {
	ok := s.s.Scan()
	if ok {
		return s.s.Text(), true
	}

	return "", false
}
