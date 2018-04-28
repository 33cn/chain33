// Copyright 2017 The go-ethereum Authors
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

package mm

import (
	"math/big"

	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
)

type (
	MemorySizeFunc func(*Stack) *big.Int
)

func MemorySha3(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

func MemoryCallDataCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func MemoryReturnDataCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func MemoryCodeCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func MemoryExtCodeCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(1), stack.Back(3))
}

func MemoryMLoad(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), big.NewInt(32))
}

func MemoryMStore8(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), big.NewInt(1))
}

func MemoryMStore(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), big.NewInt(32))
}

func MemoryCreate(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(1), stack.Back(2))
}

func MemoryCall(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(5), stack.Back(6))
	y := calcMemSize(stack.Back(3), stack.Back(4))

	return common.BigMax(x, y)
}

func MemoryCallCode(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(5), stack.Back(6))
	y := calcMemSize(stack.Back(3), stack.Back(4))

	return common.BigMax(x, y)
}
func MemoryDelegateCall(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(4), stack.Back(5))
	y := calcMemSize(stack.Back(2), stack.Back(3))

	return common.BigMax(x, y)
}

func MemoryStaticCall(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(4), stack.Back(5))
	y := calcMemSize(stack.Back(2), stack.Back(3))

	return common.BigMax(x, y)
}

func MemoryReturn(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

func MemoryRevert(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

func MemoryLog(stack *Stack) *big.Int {
	mSize, mStart := stack.Back(1), stack.Back(0)
	return calcMemSize(mStart, mSize)
}
