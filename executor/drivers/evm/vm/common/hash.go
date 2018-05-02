package common

import (
	"gitlab.33.cn/chain33/chain33/common"
	"math/big"
)

const (
	HashLength = 32
)

type Hash common.Hash

func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return ToHex(h[:]) }

// 设置哈希中的字节值，如果字节数组长度超过哈希长度，则被截断，只保留后面的部分
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

