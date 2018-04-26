package common

import (
	"gitlab.33.cn/chain33/chain33/common"
	"math/big"
)

const (
	HashLength    = 32
	AddressLength = 20
)

type Hash common.Hash
type Address struct {
	addr string
}

func BytesToAddress(b []byte) Address {
	return StringToAddress(string(b))
}
func StringToAddress(s string) Address { return Address{addr:s} }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func HexToAddress(s string) Address    { return BytesToAddress(FromHex(s)) }
func EmptyAddress() Address {
	return Address{""}
}


// Get the string representation of the underlying address
func (a Address) Str() string   { return a.addr }
func (a Address) Bytes() []byte { return []byte(a.addr) }
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a.Bytes()) }
func (a Address) Hash() Hash    { return BytesToHash(a.Bytes()) }


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

func BigToHash(b *big.Int) Hash  { return Hash(common.BigToHash(b)) }

func EmptyHash(h Hash) bool {
	return common.EmptyHash(common.Hash(h))
}

func BytesToHash(b []byte) Hash {
	return Hash(common.BytesToHash(b))
}