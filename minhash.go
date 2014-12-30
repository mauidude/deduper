package main

import (
	"hash/fnv"
	"math"
	"math/rand"
	"reflect"
	"strings"
)

const (
	p1 = uint64(4294967311)
	p2 = uint64(7562380294967317)
)

type Vector []uint32

type Matrix []Vector

type Hasher func(...uint32) uint32

func hashCode(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func equal(a, b Vector) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

type MinHasher struct {
	hashers     []Hasher
	bandHashers []Hasher
	matrix      Matrix
	b           int
	r           int
}

func (m *MinHasher) Add(s string) {
	column := m.hashColumn(s)
	m.matrix = append(m.matrix, column)
}

func (m *MinHasher) FindSimilar(s string) int {
	b := m.bandMatrix()
	col := m.hashColumn(s)
	col = m.bandColumn(col)

	for i, c := range b {
		if reflect.DeepEqual(col, c) {
			return i
		}
	}

	return -1
}

func (m *MinHasher) hashColumn(s string) Vector {
	column := make(Vector, len(m.hashers))

	for _, h := range m.hashers {
		shingler := NewShingler(strings.NewReader(s), 5)
		min := uint32(math.MaxUint32)

		for shingler.Scan() {
			sh := shingler.Text()
			v := hashCode(sh)

			hash := h(v)
			if hash < min {
				min = hash
			}
		}

		column = append(column, min)
	}

	return column
}

func (m *MinHasher) bandColumn(col Vector) Vector {
	bcol := make(Vector, 0, m.b)

	for _, hash := range m.bandHashers {
		for i := 0; i < len(col); i += m.r {
			rows := col[i : i+m.r]
			h := hash(rows...)

			bcol = append(bcol, h)
		}
	}

	return bcol
}

func (m *MinHasher) bandMatrix() Matrix {
	matrix := make(Matrix, 0, len(m.matrix))

	for _, col := range m.matrix {
		bcol := m.bandColumn(col)
		matrix = append(matrix, bcol)
	}

	return matrix
}

func NewMinHasher(b int, r int) *MinHasher {
	return &MinHasher{
		hashers:     generateHahsers(b*r, p1),
		bandHashers: generateHahsers(b, p2),
		matrix:      make(Matrix, 0),
		r:           r,
		b:           b,
	}
}

func generateHahsers(n int, p uint64) []Hasher {
	// universal hashing
	// h(x,a,b) = ((ax+b) mod p) mod m
	// x is key you want to hash
	// a is any number you can choose between 1 to p-1 inclusive.
	// b is any number you can choose between 0 to p-1 inclusive.
	// p is a prime number that is greater than max possible value of x
	// m is a max possible value you want for hash code + 1

	hashers := make([]Hasher, 0)
	m := uint64(math.MaxUint32)

	r := rand.New(rand.NewSource(31))

	for i := 0; i < n; i++ {
		a := uint64(r.Int63n(int64(p)) + 1)
		b := uint64(r.Int63n(int64(p)))

		f := func(v ...uint32) uint32 {
			var sum uint64
			for _, v := range v {
				sum += a*uint64(v) + b
			}

			return uint32((sum % p) % m)
		}

		hashers = append(hashers, f)
	}

	return hashers
}
