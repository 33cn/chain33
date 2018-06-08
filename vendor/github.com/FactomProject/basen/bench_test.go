package basen_test

import (
	"testing"

	"github.com/cmars/basen"
)

// These benchmarks mirror the ones in encoding/base64, and results
// should be comparable to those.

func BenchmarkBase58EncodeToString(b *testing.B) {
	data := make([]byte, 8192)
	data[0] = 0xff // without this, it's just skipping zero bytes
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		x := basen.Base58.EncodeToString(data)
		_ = x
	}
}

func BenchmarkBase58DecodeString(b *testing.B) {
	data := make([]byte, 8192)
	data[0] = 0xff // without this, it's just skipping zero bytes
	s := basen.Base58.EncodeToString(data)
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		x, err := basen.Base58.DecodeString(s)
		if err != nil {
			b.Fatal(err)
		}
		_ = x
	}
}

func BenchmarkBase62EncodeToString(b *testing.B) {
	data := make([]byte, 8192)
	data[0] = 0xff // without this, it's just skipping zero bytes
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		x := basen.Base62.EncodeToString(data)
		_ = x
	}
}

func BenchmarkBase62DecodeString(b *testing.B) {
	data := make([]byte, 8192)
	data[0] = 0xff // without this, it's just skipping zero bytes
	s := basen.Base62.EncodeToString(data)
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		x, err := basen.Base62.DecodeString(s)
		if err != nil {
			b.Fatal(err)
		}
		_ = x
	}
}
