// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"math/big"
)

// ContractRef is a reference to the contract's backing object
type ContractRef interface {
	Address() common.Address
}

// AccountRef implements ContractRef.
//
// Account references are used during EVM initialisation and
// it's primary use is to fetch addresses. Removing this object
// proves difficult because of the cached jump destinations which
// are fetched from the parent contract (i.e. the caller), which
// is a ContractRef.
type AccountRef common.Address

// Address casts AccountRef to a Address
func (ar AccountRef) Address() common.Address { return (common.Address)(ar) }

// Contract represents an ethereum contract in the state database. It contains
// the the contract code, calling arguments. Contract implements ContractRef
type Contract struct {
	// 调用者地址，应该为外部账户的地址
	// 如果是通过合约再调用合约时，会从上级合约中获取调用者地址进行赋值
	CallerAddress common.Address
	caller        ContractRef
	self          ContractRef

	// 存储跳转信息，供JUMP和JUMPI指令使用
	jumpdests destinations // result of JUMPDEST analysis.

	Code     []byte
	CodeHash common.Hash
	CodeAddr *common.Address
	Input    []byte

	Gas   uint64
	value *big.Int

	Args []byte

	DelegateCall bool
}

// 创建一个新的合约调用对象
// 不管合约是否存在，每次调用时都会新创建一个合约对象交给解释器执行，对象持有合约代码和合约地址
func NewContract(caller ContractRef, object ContractRef, value *big.Int, gas uint64) *Contract {

	c := &Contract{CallerAddress: caller.Address(), caller: caller, self: object, Args: nil}

	// 如果是合约调用合约的情况，则直接赋值父合约的jumpdests
	// 否则，创建一个新的jumpdests
	if parent, ok := caller.(*Contract); ok {
		c.jumpdests = parent.jumpdests
	} else {
		c.jumpdests = make(destinations)
	}

	// 持有gas引用，方便正确计算消耗的gas
	c.Gas = gas
	c.value = value

	return c
}

// AsDelegate sets the contract to be a delegate call and returns the current
// contract (for chaining calls)
func (c *Contract) AsDelegate() *Contract {
	c.DelegateCall = true
	// NOTE: caller must, at all times be a contract. It should never happen
	// that caller is something other than a Contract.
	parent := c.caller.(*Contract)
	c.CallerAddress = parent.CallerAddress
	c.value = parent.value

	return c
}

// GetOp returns the n'th element in the contract's byte array
func (c *Contract) GetOp(n uint64) OpCode {
	return OpCode(c.GetByte(n))
}

// GetByte returns the n'th byte in the contract's byte array
func (c *Contract) GetByte(n uint64) byte {
	if n < uint64(len(c.Code)) {
		return c.Code[n]
	}

	return 0
}

// Caller returns the caller of the contract.
//
// Caller will recursively call caller when the contract is a delegate
// call, including that of caller's caller.
func (c *Contract) Caller() common.Address {
	return c.CallerAddress
}

// 从合约的可用gas中进行gas消费
func (c *Contract) UseGas(gas uint64) (ok bool) {
	if c.Gas < gas {
		return false
	}
	c.Gas -= gas
	return true
}

// Address returns the contracts address
func (c *Contract) Address() common.Address {
	return c.self.Address()
}

// Value returns the contracts value (sent to it from it's caller)
func (c *Contract) Value() *big.Int {
	return c.value
}

// 设置合约代码内容
func (self *Contract) SetCode(hash common.Hash, code []byte) {
	self.Code = code
	self.CodeHash = hash
}


// 设置合约代码和代码哈希
func (self *Contract) SetCallCode(addr *common.Address, hash common.Hash, code []byte) {
	self.Code = code
	self.CodeHash = hash
	self.CodeAddr = addr
}
