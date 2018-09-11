package mm

import (
	"math/big"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
)

// 本文件中定义各种操作下计算内存大小的逻辑

type (
	// 计算所需内存大小
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
