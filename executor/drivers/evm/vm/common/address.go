package common

import "math/big"

// 封装string类型的地址为结构体，并提供各种常用操作封装
type Address struct {
	addr string
}

func (a Address) Str() string   { return a.addr }
func (a Address) Bytes() []byte { return []byte(a.addr) }
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a.Bytes()) }
func (a Address) Hash() Hash    { return BytesToHash(a.Bytes()) }

func BytesToAddress(b []byte) Address  { return StringToAddress(string(b)) }
func StringToAddress(s string) Address { return Address{addr: s} }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func EmptyAddress() Address            { return Address{""} }
