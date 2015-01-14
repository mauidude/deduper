package minhash

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
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
