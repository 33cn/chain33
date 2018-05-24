package common

import (
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"math/big"
	"fmt"
	"strings"
	"encoding/binary"
)

var (
	ExecPrefix = "exec-evm:"
)
// 封装地址结构体，并提供各种常用操作封装
// 这里封装的操作主要是为了提供Address<->big.Int， Address<->[]byte 之间的互相转换
// 并且转换的核心是使用地址对象中的Hash160元素，因为在EVM中地址固定为[20]byte，超出此范围的地址无法正确解释执行
type Address struct {
	addr *account.Address
}

// 返回EVM合约格式的地址
func (a Address) Str() string { return fmt.Sprintf("%s%s",ExecPrefix,a.addr.String()) }
func (a Address) String() string { return a.Str() }

// 返回外部账户的正常地址格式
func (a Address) NormalString() string { return a.addr.String() }

func (a Address) Bytes() []byte {
	return a.addr.Hash160[:]
}

func (a Address) Big() *big.Int {
	ret := new(big.Int).SetBytes(a.Bytes())
	return ret
}

// 从一个已有地址和nonce值，生成一个新地址，此方法可重入
func NewAddress(addr Address, nonce int64) Address {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(nonce))
	var data = make([]byte, 20)
	copy(data, addr.addr.Hash160[:])
	copy(data[20-8:], buf)
	return BytesToAddress(data)
}

func (a Address) Hash() Hash { return ToHash(a.Bytes()) }

func BytesToAddress(b []byte) Address {
	a := new(account.Address)
	a.Version = 0
	a.Hash160 = copyBytes(LeftPadBytes(b, 20))
	return Address{addr: a}
}

func StringToAddress(s string) *Address {
	addrStr := s
	if strings.HasPrefix(s, ExecPrefix){
		addrStr = strings.TrimPrefix(s, ExecPrefix)
	}
	addr, err := account.NewAddrFromString(addrStr)
	if err != nil {
		log15.Error("create address form string error", "string:", s)
		return nil
	}
	return &Address{addr: addr}
}

func copyBytes(data []byte) (out [20]byte) {
	copy(out[:], data)
	return
}

func bigBytes(b *big.Int) (out [20]byte) {
	copy(out[:], b.Bytes())
	return
}

func BigToAddress(b *big.Int) Address {
	a := new(account.Address)
	a.Version = 0
	a.Hash160 = bigBytes(b)
	return Address{addr: a}
}

func EmptyAddress() Address { return BytesToAddress([]byte{0}) }
