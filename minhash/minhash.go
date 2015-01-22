package minhash

import (
	"io"
	"math"
	"sync"

	mapset "github.com/deckarep/golang-set"
	"github.com/mauidude/deduper/text"
)

const (
	// The large prime used to hash shingles
	p1 = uint64(4294967311)

	// The large prime used to hash bands
	p2 = uint64(7562380294967317)
)

type hasher func(...uint32) uint32

// Match represents a matching document.
type Match struct {
	// ID is the unique ID of the document that was
	// given when the document was added.
	ID string `json:"id"`

	// Similarity is the Jaccard similarity from 0 to 1 of this document
	// to the document it was compared against.
	Similarity float64 `json:"similarity"`
}

// New creates a new MinHasher with the given band size, number of rows, and shingle size.
func New(b int, r int, shingleSize int) *MinHasher {
	return &MinHasher{
		hashers:       generateHahsers(b*r, p1),
		bandHashers:   generateHahsers(b, p2),
		matrix:        make(matrix, 0),
		r:             r,
		b:             b,
		n:             shingleSize,
		columnMapping: make(map[int]string),
		ids:           mapset.NewSet(),
	}
}

// MinHasher provides near-similar matching capabilities on large
// strings of text.
type MinHasher struct {
	// The mapping of column indexes in the matrix to document ids.
	columnMapping map[int]string

	// The hash functions used to hash the document's shingles.
	hashers []hasher

	// The hash functions used to hash the hash function results
	// into bands.
	bandHashers []hasher

	// The matrix of documents and hash values. Each vector
	// is a list of hash values for a document's shingles, eg element m[i,j] is the
	// value of h[i](document[j]).
	matrix matrix

	// The unique list of document ids being stored.
	ids mapset.Set

	// The band matrix generated with LSH.
	bands matrix

	// Locks the bands matrix.
	bandMutex sync.RWMutex

	// Locks the matrix.
	matrixMutex sync.RWMutex

	// Number of bands.
	b int

	// Number of rows.
	r int

	// N-shingles being used.
	n int
}

// Add adds a new document with the given ID to the collection of
// documents.
func (m *MinHasher) Add(id string, r io.Reader) {
	column := m.hashColumn(r)

	m.matrixMutex.Lock()
	m.matrix = append(m.matrix, column)
	m.columnMapping[len(m.matrix)-1] = id
	m.matrixMutex.Unlock()

	m.ids.Add(id)

	m.bandMutex.Lock()
	m.bands = nil
	m.bandMutex.Unlock()
}

// FindSimilar returns a list of documents whose similarity to the given document
// is greater than or equal to the threshold provided.
func (m *MinHasher) FindSimilar(r io.Reader, threshold float64) []*Match {
	col := m.hashColumn(r)
	col = m.bandColumn(col)

	similar := make([]*Match, 0)

	m.bandMutex.RLock()
	if m.bands == nil {
		m.bandMutex.RUnlock()

		// lock as writer
		m.bandMutex.Lock()
		m.bands = m.bandMatrix()
		m.bandMutex.Unlock()

		// relock as reader
		m.bandMutex.RLock()
	}

	// for each document in the band matrix
	for i, c := range m.bands {
		// see if they share any common bands with input
		for j := 0; j < len(col); j++ {
			if col[j] == c[j] {
				// needs deeper inspection ie jaccard similarity
				sim := jaccard(c, col)

				if sim >= threshold {
					similar = append(similar, &Match{
						ID:         m.columnMapping[i],
						Similarity: sim,
					})
				}

				break
			}
		}
	}

	m.bandMutex.RUnlock()

	return similar
}

// Contains returns true if the MinHasher contains
// the document with the given id.
func (m *MinHasher) Contains(id string) bool {
	return m.ids.Contains(id)
}

func (m *MinHasher) hashColumn(r io.Reader) vector {
	// the result which holds each minimum hash
	// value of h_i at the ith index of each n-gram
	column := make(vector, len(m.hashers))

	shingler := text.NewShingler(r, m.n)

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
	m.matrixMutex.RLock()
	defer m.matrixMutex.RUnlock()

	b := make(matrix, 0, len(m.matrix))

	for _, col := range m.matrix {
		bcol := m.bandColumn(col)
		b = append(b, bcol)
	}

	return b
}
