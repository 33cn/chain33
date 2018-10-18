package common

import (
	"math/big"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

// 封装地址结构体，并提供各种常用操作封装
// 这里封装的操作主要是为了提供Address<->big.Int， Address<->[]byte 之间的互相转换
// 并且转换的核心是使用地址对象中的Hash160元素，因为在EVM中地址固定为[20]byte，超出此范围的地址无法正确解释执行
type Address struct {
	addr *address.Address
}

func (a Address) String() string { return a.addr.String() }

func (a Address) Bytes() []byte {
	return a.addr.Hash160[:]
}

func (a Address) Big() *big.Int {
	ret := new(big.Int).SetBytes(a.Bytes())
	return ret
}

// txHash生成EVM合约地址
func NewAddress(txHash []byte) Address {
	execAddr := address.GetExecAddress(types.ExecName(evmtypes.EvmPrefix) + BytesToHash(txHash).Hex())
	return Address{addr: execAddr}
}

func ExecAddress(execName string) Address {
	execAddr := address.GetExecAddress(execName)
	return Address{addr: execAddr}
}

func (a Address) Hash() Hash { return ToHash(a.Bytes()) }

func BytesToAddress(b []byte) Address {
	a := new(address.Address)
	a.Version = 0
	a.Hash160 = copyBytes(LeftPadBytes(b, 20))
	return Address{addr: a}
}

func StringToAddress(s string) *Address {
	addr, err := address.NewAddrFromString(s)
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
	a := new(address.Address)
	a.Version = 0
	a.Hash160 = bigBytes(b)
	return Address{addr: a}
}

func EmptyAddress() Address { return BytesToAddress([]byte{0}) }
