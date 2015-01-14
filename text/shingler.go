package text

import (
	"bufio"
	"io"
	"strings"
)

type Shingler struct {
	r io.Reader
	s *bufio.Scanner
	n int
	q []string
}

func (s *Shingler) Scan() bool {
	if s.q == nil {
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

func NewShingler(r io.Reader, n int) *Shingler {
	return &Shingler{
		r: r,
		n: n,
	}
}
