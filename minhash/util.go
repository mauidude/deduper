package minhash

import (
	"hash/fnv"
	"math"
	"math/rand"

	"github.com/deckarep/golang-set"
)

// hashCode returns a hash value for a given string.
func hashCode(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// jaccard returns the Jaccard similarity of two vectors.
// The result will be between 0 and 1, inclusively, where 0 is not at all similar
// and 1 is identical.
func jaccard(a, b vector) float64 {
	// create sets of the two vectors
	setA := mapset.NewThreadUnsafeSet()
	setB := mapset.NewThreadUnsafeSet()

	for _, v := range a {
		setA.Add(v)
	}

	for _, v := range b {
		setB.Add(v)
	}

	// |A U B| / |A ^ B|
	union := setA.Union(setB).Cardinality()
	intersection := setA.Intersect(setB).Cardinality()

	return float64(intersection) / float64(union)
}

// generateHashers creates a set of n universal hashing functions
// in the form ((ax+b) % p) % m. a and b are generated uniquely
// for each hash function. p should be a large prime number. m
// is the maximum hash value + 1 which is the maximum value of a uint64
// in this case.
func generateHahsers(n int, p uint64) []hasher {
	// universal hashing
	// h(x,a,b) = ((ax+b) mod p) mod m
	// x is key you want to hash
	// a is any number you can choose between 1 to p-1 inclusive.
	// b is any number you can choose between 0 to p-1 inclusive.
	// p is a prime number that is greater than max possible value of x
	// m is a max possible value you want for hash code + 1
	// See: http://stackoverflow.com/questions/19701052/how-many-hash-functions-are-required-in-a-minhash-algorithm

	hashers := make([]hasher, 0)
	m := uint64(math.MaxUint32)

	r := rand.New(rand.NewSource(31))

	for i := 0; i < n; i++ {
		a := uint64(r.Int63n(int64(p)) + 1)
		b := uint64(r.Int63n(int64(p)))

		h := func(v ...uint32) uint32 {
			var sum uint64

			// http://stackoverflow.com/questions/539311/generate-a-hash-sum-for-several-integers
			for _, v := range v {
				sum += a*uint64(v) + b
			}

			return uint32((sum % p) % m)
		}

		hashers = append(hashers, h)
	}

	return hashers
}
