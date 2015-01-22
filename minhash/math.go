package minhash

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

// vector represents a column of a matrix
type vector []uint32

// signature returns a base64 encoded string representation of the vector.
func (v vector) signature() string {
	buf := &bytes.Buffer{}
	for _, v := range v {
		binary.Write(buf, binary.LittleEndian, v)
	}

	return base64.URLEncoding.EncodeToString(buf.Bytes())
}

// matrix is a two-dimensional collection of values.
type matrix []vector
