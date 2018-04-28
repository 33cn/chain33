package common

import (
	"encoding/hex"
	"gitlab.33.cn/chain33/chain33/common"
	"math/big"
)

const (
	HashLength = 32
)

type Hash common.Hash

// Get the string representation of the underlying hash
func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return Encode(h[:]) }

// Sets the hash to the value of b. If b is larger than len(h), 'b' will be cropped (from the left).
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

func BigToHash(b *big.Int) Hash {
	return Hash(common.BigToHash(b))
}

func BytesToHash(b []byte) Hash {
	return Hash(common.BytesToHash(b))
}

func EmptyHash(h Hash) bool {
	return h == Hash{}
}

// Encode encodes b as a hex string with 0x prefix.
func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

