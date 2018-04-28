package common

import "math/big"

type Address struct {
	addr string
}

func BytesToAddress(b []byte) Address {
	return StringToAddress(string(b))
}
func StringToAddress(s string) Address { return Address{addr: s} }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func EmptyAddress() Address {
	return Address{""}
}

// Get the string representation of the underlying address
func (a Address) Str() string   { return a.addr }
func (a Address) Bytes() []byte { return []byte(a.addr) }
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a.Bytes()) }
func (a Address) Hash() Hash    { return BytesToHash(a.Bytes()) }
