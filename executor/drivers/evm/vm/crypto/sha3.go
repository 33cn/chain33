package crypto

import (
	"hash"
	"golang.org/x/crypto/sha3"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
)

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	d := NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// NewKeccak256 creates a new Keccak-256 hash.
// FIXME 直接使用了sha3的shake256，和以太坊的区别是dsbyte为0x06，目前还不清楚是否需要修改，待确定
func NewKeccak256() hash.Hash {
	return sha3.New256()
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h model.Hash) {
	d := NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}