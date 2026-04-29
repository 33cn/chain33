package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToHashRoundtrip(t *testing.T) {
	b := []byte("12345678901234567890123456789012") // 32 bytes
	h := BytesToHash(b)
	assert.Equal(t, b, h.Bytes())
}

func TestHashSetBytesShort(t *testing.T) {
	var h Hash
	h.SetBytes([]byte("hello"))
	assert.NotEqual(t, Hash{}, h)
}

func TestHashSetBytesLong(t *testing.T) {
	var h Hash
	long := make([]byte, 64)
	for i := range long {
		long[i] = byte(i)
	}
	h.SetBytes(long)
	// Should take last 32 bytes
	assert.Equal(t, long[32:], h.Bytes())
}

func TestAbs(t *testing.T) {
	assert.Equal(t, abs(5), abs(-5))
	assert.Equal(t, abs(0), abs(0))
	assert.Equal(t, abs(10), abs(10))
}

func TestIntToTime(t *testing.T) {
	tm := intToTime(2208988800+1000, 0)
	assert.False(t, tm.IsZero())
	assert.Greater(t, tm.Unix(), int64(0))
}
