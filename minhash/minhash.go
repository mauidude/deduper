package minhash

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"math"
	"strings"

	"github.com/mauidude/deduper/text"
)

const (
	p1 = uint64(4294967311)
	p2 = uint64(7562380294967317)
)

type vector []uint32

func (v vector) signature() string {
	buf := &bytes.Buffer{}
	for _, v := range v {
		binary.Write(buf, binary.LittleEndian, v)
	}

	return base64.URLEncoding.EncodeToString(buf.Bytes())
}

type matrix []vector

type hasher func(...uint32) uint32

type Match struct {
	ID         string
	Similarity float64
}

type MinHasher struct {
	ids         map[int]string
	hashers     []hasher
	bandHashers []hasher

	// matrix where rows are hash functions
	// and columns are documents. element m[i,j] is the
	// value of h[i](document[j])
	matrix matrix
	b      int
	r      int
}

func (m *MinHasher) Add(id, s string) {
	column := m.hashColumn(s)
	m.matrix = append(m.matrix, column)
	m.ids[len(m.matrix)-1] = id
}

func (m *MinHasher) FindSimilar(s string, threshold float64) []*Match {
	b := m.bandMatrix()
	col := m.hashColumn(s)
	col = m.bandColumn(col)

	similar := make([]*Match, 0)

	// for each document in the band matrix
	for i, c := range b {
		// see if they share any common bands with input
		for j := 0; j < len(col); j++ {
			if col[j] == c[j] {
				// needs deeper inspection ie jaccard similarity
				sim := jaccard(c, col)

				if sim >= threshold {
					similar = append(similar, &Match{
						ID:         m.ids[i],
						Similarity: sim,
					})
				}

				break
			}
		}
	}

	return similar
}

func (m *MinHasher) hashColumn(s string) vector {
	// the result which holds each minimum hash
	// value of h_i at the ith index of each n-gram
	column := make(vector, len(m.hashers))

	shingler := text.NewShingler(strings.NewReader(s), 2)

	// initialize to max value to find the min
	for i, _ := range m.hashers {
		column[i] = uint32(math.MaxUint32)
	}

	for shingler.Scan() {
		sh := shingler.Text()

		// convert the string to a number by
		// hashing it... similar to GetHashCode
		// in C#
		v := hashCode(sh)

		for i, h := range m.hashers {
			hash := h(v)
			if hash < column[i] {
				column[i] = hash
			}
		}
	}

	return column
}

func (m *MinHasher) bandColumn(col vector) vector {
	bcol := make(vector, 0, m.b)

	for _, hash := range m.bandHashers {
		for i := 0; i < len(col); i += m.r {
			rows := col[i : i+m.r]
			h := hash(rows...)

			bcol = append(bcol, h)
		}
	}

	return bcol
}

func (m *MinHasher) bandMatrix() matrix {
	b := make(matrix, 0, len(m.matrix))

	for _, col := range m.matrix {
		bcol := m.bandColumn(col)
		b = append(b, bcol)
	}

	return b
}

func New(b int, r int) *MinHasher {
	return &MinHasher{
		hashers:     generateHahsers(b*r, p1),
		bandHashers: generateHahsers(b, p2),
		matrix:      make(matrix, 0),
		r:           r,
		b:           b,
		ids:         make(map[int]string),
	}
}
