package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlloc(t *testing.T) {
	BufferReset()
	data := BufferAlloc(10)
	assert.Equal(t, 0, len(data))

	data2 := BufferAlloc(10)
	assert.Equal(t, 0, len(data2))

	for i := range data {
		data[i] = 1
	}
	for i := range data2 {
		data2[i] = 2
	}

	for i := range data {
		assert.Equal(t, 1, data[i])
	}
	for i := range data2 {
		assert.Equal(t, 2, data2[i])
	}
}

func BenchmarkAlloc(b *testing.B) {
	BufferReset()
	data := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		a := BufferAlloc(10)
		if a == nil {
			panic("alloc")
		}
		data[i] = a
	}
}

func BenchmarkAllocMake(b *testing.B) {
	data := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		a := make([]byte, 10, 10)
		if a == nil {
			panic("alloc")
		}
		data[i] = a
	}
}
