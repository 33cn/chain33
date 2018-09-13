package runtime

import (
	"fmt"
	"math/big"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common/crypto"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/mm"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/model"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/params"
)

// 加法操作
func opAdd(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()
	common.U256(y.Add(x, y))

	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 减法操作
func opSub(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	//x, y := stack.Pop(), stack.Pop()
	//stack.Push(common.U256(x.Sub(x, y)))
	//
	//evm.Interpreter.IntPool.Put(y)
	//
	//return nil, nil
	x, y := stack.Pop(), stack.Peek()
	common.U256(y.Sub(x, y))

	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 乘法操作
func opMul(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(common.U256(x.Mul(x, y)))

	evm.Interpreter.IntPool.Put(y)

	return nil, nil
}

// 除法操作
func opDiv(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()
	if y.Sign() != 0 {
		common.U256(y.Div(x, y))
	} else {
		y.SetUint64(0)
	}
	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 除法操作（带符号）
func opSdiv(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := common.S256(stack.Pop()), common.S256(stack.Pop())
	res := evm.Interpreter.IntPool.GetZero()

	if y.Sign() == 0 || x.Sign() == 0 {
		stack.Push(res)
	} else {
		if x.Sign() != y.Sign() {
			res.Div(x.Abs(x), y.Abs(y))
			res.Neg(res)
		} else {
			res.Div(x.Abs(x), y.Abs(y))
		}
		stack.Push(common.U256(res))
	}
	evm.Interpreter.IntPool.Put(x, y)
	return nil, nil
}

// 两数取模
func opMod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	if y.Sign() == 0 {
		stack.Push(x.SetUint64(0))
	} else {
		stack.Push(common.U256(x.Mod(x, y)))
	}
	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

// 两数取模（带符号）
func opSmod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := common.S256(stack.Pop()), common.S256(stack.Pop())
	res := evm.Interpreter.IntPool.GetZero()

	if y.Sign() == 0 {
		stack.Push(res)
	} else {
		if x.Sign() < 0 {
			res.Mod(x.Abs(x), y.Abs(y))
			res.Neg(res)
		} else {
			res.Mod(x.Abs(x), y.Abs(y))
		}
		stack.Push(common.U256(res))
	}
	evm.Interpreter.IntPool.Put(x, y)
	return nil, nil
}

// 指数操作
func opExp(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	base, exponent := stack.Pop(), stack.Pop()
	stack.Push(common.Exp(base, exponent))

	evm.Interpreter.IntPool.Put(base, exponent)

	return nil, nil
}

// 带符号扩展
func opSignExtend(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	back := stack.Pop()
	if back.Cmp(big.NewInt(31)) < 0 {
		bit := uint(back.Uint64()*8 + 7)
		num := stack.Pop()
		mask := back.Lsh(common.Big1, bit)
		mask.Sub(mask, common.Big1)
		if num.Bit(int(bit)) > 0 {
			num.Or(num, mask.Not(mask))
		} else {
			num.And(num, mask)
		}

		stack.Push(common.U256(num))
	}

	evm.Interpreter.IntPool.Put(back)
	return nil, nil
}

// 取非
func opNot(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x := stack.Peek()
	common.U256(x.Not(x))
	return nil, nil
}

// 小于判断
func opLt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()
	if x.Cmp(y) < 0 {
		y.SetUint64(1)
	} else {
		y.SetUint64(0)
	}
	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 大于判断
func opGt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()
	if x.Cmp(y) > 0 {
		y.SetUint64(1)
	} else {
		y.SetUint64(0)
	}
	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 小于判断（带符号）
func opSlt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()

	xSign := x.Cmp(common.TT255)
	ySign := y.Cmp(common.TT255)

	switch {
	case xSign >= 0 && ySign < 0:
		y.SetUint64(1)

	case xSign < 0 && ySign >= 0:
		y.SetUint64(0)

	default:
		if x.Cmp(y) < 0 {
			y.SetUint64(1)
		} else {
			y.SetUint64(0)
		}
	}
	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 判断两数是否相等（带符号）
func opSgt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()

	xSign := x.Cmp(common.TT255)
	ySign := y.Cmp(common.TT255)

	switch {
	case xSign >= 0 && ySign < 0:
		y.SetUint64(0)

	case xSign < 0 && ySign >= 0:
		y.SetUint64(1)

	default:
		if x.Cmp(y) > 0 {
			y.SetUint64(1)
		} else {
			y.SetUint64(0)
		}
	}
	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 判断两数是否相等
func opEq(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()
	if x.Cmp(y) == 0 {
		y.SetUint64(1)
	} else {
		y.SetUint64(0)
	}
	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 判断指定数是否为0
func opIszero(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x := stack.Peek()
	if x.Sign() > 0 {
		x.SetUint64(0)
	} else {
		x.SetUint64(1)
	}
	return nil, nil
}

// 与操作
func opAnd(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(x.And(x, y))

	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

// 或操作
func opOr(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()
	y.Or(x, y)

	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 异或操作
func opXor(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Peek()
	y.Xor(x, y)

	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

// 获取指定数据中指定位的byte值
func opByte(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	th, val := stack.Pop(), stack.Peek()

	// 只处理不超过32位整数的值，否则直接设置0
	if th.Cmp(common.Big32) < 0 {
		b := common.Byte(val, model.WordByteSize, int(th.Int64()))
		val.SetUint64(uint64(b))
	} else {
		val.SetUint64(0)
	}
	evm.Interpreter.IntPool.Put(th)
	return nil, nil
}

// 两数相加，并和第三数求模
func opAddmod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y, z := stack.Pop(), stack.Pop(), stack.Pop()
	if z.Cmp(common.Big0) > 0 {
		x.Add(x, y)
		x.Mod(x, z)
		stack.Push(common.U256(x))
	} else {
		stack.Push(x.SetUint64(0))
	}
	evm.Interpreter.IntPool.Put(y, z)
	return nil, nil
}

// 两数相乘，并和第三数求模
func opMulmod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y, z := stack.Pop(), stack.Pop(), stack.Pop()
	if z.Cmp(common.Big0) > 0 {
		mul := x.Mul(x, y)
		mul.Mod(mul, z)
		stack.Push(common.U256(mul))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(y, z)
	return nil, nil
}

// 左移位操作：
// 操作弹出x和y，将y向左移x位，并将结果入栈
func opSHL(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// 注意，这里y值并没有弹出，后面的操作直接改变其值，不需要再次入栈
	shift, value := common.U256(stack.Pop()), common.U256(stack.Peek())

	// 将x放入整数池
	defer evm.Interpreter.IntPool.Put(shift)

	// 如果移动范围超过256位整数的长度，则直接置零
	if shift.Cmp(common.Big256) >= 0 {
		value.SetUint64(0)
		return nil, nil
	}

	// 执行移位操作
	n := uint(shift.Uint64())
	common.U256(value.Lsh(value, n))

	return nil, nil
}

// 右移位操作：
// 操作弹出x和y，将y向右移x位，并将结果入栈 （左边的空位用0填充）
func opSHR(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// 注意，这里y值并没有弹出，后面的操作直接改变其值，不需要再次入栈
	shift, value := common.U256(stack.Pop()), common.U256(stack.Peek())

	// 将x放入整数池
	defer evm.Interpreter.IntPool.Put(shift)

	// 如果移动范围超过256位整数的长度，则直接置零
	if shift.Cmp(common.Big256) >= 0 {
		value.SetUint64(0)
		return nil, nil
	}

	// 执行移位操作
	n := uint(shift.Uint64())
	common.U256(value.Rsh(value, n))

	return nil, nil
}

// 右移位操作：
// 操作弹出x和y，将y向右移x位，并将结果入栈 （左边的空位用左边第一位的符号位填充）
func opSAR(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// 注意，这里y值弹出了，后面计算出结果后需要入栈
	shift, value := common.U256(stack.Pop()), common.S256(stack.Pop())
	// 将x放入整数池
	defer evm.Interpreter.IntPool.Put(shift)

	// 如果移动范围超过256位整数的长度，正数置零，负数置-1
	if shift.Cmp(common.Big256) >= 0 {
		if value.Sign() > 0 {
			value.SetUint64(0)
		} else {
			value.SetInt64(-1)
		}
		stack.Push(common.U256(value))
		return nil, nil
	}

	// 正常移位操作（注意这里需要将结果入栈）
	n := uint(shift.Uint64())
	value.Rsh(value, n)
	stack.Push(common.U256(value))

	return nil, nil
}

//使用keccak256进行哈希运算
func opSha3(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset, size := stack.Pop(), stack.Pop()
	data := memory.Get(offset.Int64(), size.Int64())
	hash := crypto.Keccak256(data)

	if evm.VmConfig.EnablePreimageRecording {
		evm.StateDB.AddPreimage(common.BytesToHash(hash), data)
	}

	stack.Push(evm.Interpreter.IntPool.Get().SetBytes(hash))

	evm.Interpreter.IntPool.Put(offset, size)
	return nil, nil
}

// 获取合约地址
func opAddress(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(contract.Address().Big())
	return nil, nil
}

// 获取指定地址的账户余额
func opBalance(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	slot := stack.Peek()
	slot.SetUint64(evm.StateDB.GetBalance(common.BigToAddress(slot).String()))
	return nil, nil
}

// 获取合约调用者地址
func opOrigin(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Origin.Big())
	return nil, nil
}

// 获取合约的调用者（最终的外部账户地址）
func opCaller(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(contract.Caller().Big())
	return nil, nil
}

// 获取调用合约的同时，进行转账操作的额度
func opCallValue(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	amount := common.BigMax(big.NewInt(0), big.NewInt(int64(contract.value)))
	stack.Push(evm.Interpreter.IntPool.Get().Set(amount))
	return nil, nil
}

// 从调用合约的入参中获取数据
func opCallDataLoad(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetBytes(common.GetDataBig(contract.Input, stack.Pop(), big32)))
	return nil, nil
}

// 获取调用合约时的入参长度
func opCallDataSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetInt64(int64(len(contract.Input))))
	return nil, nil
}

// 将调用合约时的入参写入内存
func opCallDataCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		memOffset  = stack.Pop()
		dataOffset = stack.Pop()
		length     = stack.Pop()
	)
	if merr := memory.Set(memOffset.Uint64(), length.Uint64(), common.GetDataBig(contract.Input, dataOffset, length)); merr != nil {
		return nil, merr
	}
	evm.Interpreter.IntPool.Put(memOffset, dataOffset, length)
	return nil, nil
}

// 获取合约执行返回结果的大小
func opReturnDataSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetUint64(uint64(len(evm.Interpreter.ReturnData))))
	return nil, nil
}

// 将合约执行的返回结果复制到内存
func opReturnDataCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		memOffset  = stack.Pop()
		dataOffset = stack.Pop()
		length     = stack.Pop()

		end = evm.Interpreter.IntPool.Get().Add(dataOffset, length)
	)
	defer evm.Interpreter.IntPool.Put(memOffset, dataOffset, length, end)

	if end.BitLen() > 64 || uint64(len(evm.Interpreter.ReturnData)) < end.Uint64() {
		return nil, model.ErrReturnDataOutOfBounds
	}
	memory.Set(memOffset.Uint64(), length.Uint64(), evm.Interpreter.ReturnData[dataOffset.Uint64():end.Uint64()])

	return nil, nil
}

// 获取指定合约地址的合约代码大小
func opExtCodeSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	a := stack.Pop()

	addr := common.BigToAddress(a)
	a.SetInt64(int64(evm.StateDB.GetCodeSize(addr.String())))
	stack.Push(a)

	return nil, nil
}

// 获取合约代码大小
func opCodeSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	l := evm.Interpreter.IntPool.Get().SetInt64(int64(len(contract.Code)))
	stack.Push(l)
	return nil, nil
}

// 蒋合约中指定位置的数据复制到内存中
func opCodeCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		memOffset  = stack.Pop()
		codeOffset = stack.Pop()
		length     = stack.Pop()
	)
	codeCopy := common.GetDataBig(contract.Code, codeOffset, length)
	if merr := memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy); merr != nil {
		return nil, merr
	}
	evm.Interpreter.IntPool.Put(memOffset, codeOffset, length)
	return nil, nil
}

// 蒋指定合约中指定位置的数据复制到内存中
func opExtCodeCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		addr       = common.BigToAddress(stack.Pop())
		memOffset  = stack.Pop()
		codeOffset = stack.Pop()
		length     = stack.Pop()
	)
	codeCopy := common.GetDataBig(evm.StateDB.GetCode(addr.String()), codeOffset, length)
	if merr := memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy); merr != nil {
		return nil, merr
	}
	evm.Interpreter.IntPool.Put(memOffset, codeOffset, length)
	return nil, nil
}

// 获取当前区块的GasPrice
func opGasprice(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	gasPrice := common.BigMax(big.NewInt(0), big.NewInt(int64(evm.GasPrice)))
	stack.Push(evm.Interpreter.IntPool.Get().Set(gasPrice))
	return nil, nil
}

// 获取区块哈希
func opBlockhash(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	num := stack.Pop()

	n := evm.Interpreter.IntPool.Get().Sub(evm.BlockNumber, common.Big257)
	if num.Cmp(n) > 0 && num.Cmp(evm.BlockNumber) < 0 {
		stack.Push(evm.GetHash(num.Uint64()).Big())
	} else {
		stack.Push(evm.Interpreter.IntPool.GetZero())
	}
	evm.Interpreter.IntPool.Put(num, n)
	return nil, nil
}

// 获取区块打包者地址
func opCoinbase(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// 需要注意coinbase可能为空的情况，这时将返回合约的创建者作为coinbase
	if evm.Coinbase == nil {
		stack.Push(contract.CallerAddress.Big())
	} else {
		stack.Push(evm.Coinbase.Big())
	}
	return nil, nil
}

// 获取区块打包时间
func opTimestamp(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(evm.Interpreter.IntPool.Get().Set(evm.Time)))
	return nil, nil
}

// 获取区块高度
func opNumber(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(evm.Interpreter.IntPool.Get().Set(evm.BlockNumber)))
	return nil, nil
}

// 获取区块难度
func opDifficulty(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(evm.Interpreter.IntPool.Get().Set(evm.Difficulty)))
	return nil, nil
}

// 获取GasLimit
func opGasLimit(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(evm.Interpreter.IntPool.Get().SetUint64(evm.GasLimit)))
	return nil, nil
}

// 弹出栈顶数据
func opPop(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	evm.Interpreter.IntPool.Put(stack.Pop())
	return nil, nil
}

// 加载内存中的数据
func opMload(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset := stack.Pop()
	val := evm.Interpreter.IntPool.Get().SetBytes(memory.Get(offset.Int64(), 32))
	stack.Push(val)

	evm.Interpreter.IntPool.Put(offset)
	return nil, nil
}

// 写内存操作，每次写一个字长（32个字节）
func opMstore(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// 从栈中分别弹出内存偏移量和要设置的值
	mStart, val := stack.Pop(), stack.Pop()
	if merr := memory.Set32(mStart.Uint64(), val); merr != nil {
		return nil, merr
	}

	evm.Interpreter.IntPool.Put(mStart, val)
	return nil, nil
}

// 写内存操作，每次写一个字节
func opMstore8(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	off, val := stack.Pop().Int64(), stack.Pop().Int64()
	memory.Store[off] = byte(val & 0xff)

	return nil, nil
}

// 加载合约状态数据
func opSload(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	loc := stack.Peek()
	val := evm.StateDB.GetState(contract.Address().String(), common.BigToHash(loc))
	loc.SetBytes(val.Bytes())

	return nil, nil
}

// 写合约状态数据
func opSstore(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	loc := common.BigToHash(stack.Pop())
	val := stack.Pop()
	evm.StateDB.SetState(contract.Address().String(), loc, common.BigToHash(val))

	evm.Interpreter.IntPool.Put(val)
	return nil, nil
}

// 无条件跳转操作
func opJump(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	pos := stack.Pop()
	if !contract.Jumpdests.Has(contract.CodeHash, contract.Code, pos) {
		nop := contract.GetOp(pos.Uint64())
		return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
	}
	*pc = pos.Uint64()

	evm.Interpreter.IntPool.Put(pos)
	return nil, nil
}

// 有条件跳转操作
func opJumpi(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	pos, cond := stack.Pop(), stack.Pop()
	if cond.Sign() != 0 {
		if !contract.Jumpdests.Has(contract.CodeHash, contract.Code, pos) {
			nop := contract.GetOp(pos.Uint64())
			return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
		}
		*pc = pos.Uint64()
	} else {
		*pc++
	}

	evm.Interpreter.IntPool.Put(pos, cond)
	return nil, nil
}

// 跳转目标位置
func opJumpdest(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	return nil, nil
}

// 将指令计数器指针当前的值写入整数池
func opPc(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetUint64(*pc))
	return nil, nil
}

// 获取内存大小
func opMsize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetInt64(int64(memory.Len())))
	return nil, nil
}

// 获取可用Gas
func opGas(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetUint64(contract.Gas))
	return nil, nil
}

// 合约创建操作
func opCreate(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// 从栈和内存中分别获取操作所需的参数
	var (
		value        = stack.Pop()
		offset, size = stack.Pop(), stack.Pop()
		inPut        = memory.Get(offset.Int64(), size.Int64())
		gas          = contract.Gas
	)

	contract.UseGas(gas)

	// 调用合约创建逻辑
	addr := crypto.RandomContractAddress()
	res, _, returnGas, suberr := evm.Create(contract, *addr, inPut, gas, "innerContract", "")

	// 出错时压栈0，否则压栈创建出来的合约对象的地址
	if suberr != nil && suberr != model.ErrCodeStoreOutOfGas {
		log15.Error("evm contract opCreate instruction error", suberr)
		stack.Push(evm.Interpreter.IntPool.GetZero())
	} else {
		stack.Push(addr.Big())
	}

	// 剩余的Gas再返还给合约对象
	contract.Gas += returnGas

	// 其它参数写入整数池
	evm.Interpreter.IntPool.Put(value, offset, size)

	if suberr == model.ErrExecutionReverted {
		return res, nil
	}
	return nil, nil
}

// 合约调用操作
func opCall(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// 弹出可用的gas，并放入整数池
	evm.Interpreter.IntPool.Put(stack.Pop())
	// CallGasTemp中存储的是当前操作需要消耗的Gas
	gas := evm.CallGasTemp

	// 从栈中一次弹出其它参数
	addr, value, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	value = common.U256(value)

	// 从内存中读取调用合约时需要的输入参数，
	// 注意，这里使用栈中弹出的内存偏移量和存储长度从内存中获取参数，
	// 之所以这样通过间接的方式获取，主要是因为参数的大小可能比较大，放在栈中不方便组织，
	// 所以通过store操作提前写入内存，在这里再读出来使用
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		// 如果有转账操作，这里给予一定的Gas赠送
		gas += params.CallStipend
	}

	// 调用合约执行逻辑
	ret, _, returnGas, err := evm.Call(contract, toAddr, args, gas, value.Uint64())

	// 调用结果压栈，
	// 注意，这里的处理比较特殊，出错情况下0压栈，正确情况下1压栈
	if err != nil {
		stack.Push(evm.Interpreter.IntPool.GetZero())
		log15.Error("evm contract opCall instruction error", "error", err)
	} else {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	}

	// 如果正确执行，将结果写内存
	if err == nil || err == model.ErrExecutionReverted {
		if merr := memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); merr != nil {
			return ret, merr
		}
	}

	// 剩余的Gas再返还给合约对象
	contract.Gas += returnGas

	// 其它参数写入整数池
	evm.Interpreter.IntPool.Put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

// 合约调用操作，
// 逻辑同opCall大致相同，仅调用evm的方法不同
func opCallCode(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	evm.Interpreter.IntPool.Put(stack.Pop())
	gas := evm.CallGasTemp
	addr, value, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	value = common.U256(value)
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		gas += params.CallStipend
	}

	ret, returnGas, err := evm.CallCode(contract, toAddr, args, gas, value.Uint64())
	if err != nil {
		stack.Push(evm.Interpreter.IntPool.GetZero())
		log15.Error("evm contract opCallCode instruction error", err)
	} else {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	}
	if err == nil || err == model.ErrExecutionReverted {
		if merr := memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); merr != nil {
			return ret, merr
		}
	}
	contract.Gas += returnGas

	evm.Interpreter.IntPool.Put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

// 合约委托调用操作，
// 逻辑同opCall大致相同，仅调用evm的方法不同
func opDelegateCall(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	evm.Interpreter.IntPool.Put(stack.Pop())
	gas := evm.CallGasTemp
	addr, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := evm.DelegateCall(contract, toAddr, args, gas)
	if err != nil {
		log15.Error("evm contract opDelegateCall instruction error", err)
		stack.Push(evm.Interpreter.IntPool.GetZero())
	} else {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	}
	if err == nil || err == model.ErrExecutionReverted {
		if merr := memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); merr != nil {
			return ret, merr
		}
	}
	contract.Gas += returnGas

	evm.Interpreter.IntPool.Put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

// 合约调用操作，
// 逻辑同opCall大致相同，仅调用evm的方法不同
func opStaticCall(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	evm.Interpreter.IntPool.Put(stack.Pop())
	gas := evm.CallGasTemp
	addr, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := evm.StaticCall(contract, toAddr, args, gas)
	if err != nil {
		log15.Error("evm contract opDelegateCall instruction error", err)
		stack.Push(evm.Interpreter.IntPool.GetZero())
	} else {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	}
	if err == nil || err == model.ErrExecutionReverted {
		if merr := memory.Set(retOffset.Uint64(), retSize.Uint64(), ret); merr != nil {
			return ret, merr
		}

	}
	contract.Gas += returnGas

	evm.Interpreter.IntPool.Put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

// 返回操作
func opReturn(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset, size := stack.Pop(), stack.Pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	evm.Interpreter.IntPool.Put(offset, size)
	return ret, nil
}

// 恢复操作
func opRevert(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset, size := stack.Pop(), stack.Pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	evm.Interpreter.IntPool.Put(offset, size)
	return ret, nil
}

// 停止操作
func opStop(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	return nil, nil
}

// 自毁操作
func opSuicide(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	balance := evm.StateDB.GetBalance(contract.Address().String())
	// 合约自毁后，将剩余金额返还给创建者
	evm.StateDB.AddBalance(common.BigToAddress(stack.Pop()).String(), (*contract.CodeAddr).String(), balance)

	evm.StateDB.Suicide(contract.Address().String())
	return nil, nil
}

// 生成创建日志的操作的方法：
// 生成的方法允许在合约执行时，创建N条日志写入StateDB；
// 在合约执行完成后，这些日志会打印出来（目前不会存储）
func makeLog(size int) ExecutionFunc {
	return func(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
		topics := make([]common.Hash, size)
		mStart, mSize := stack.Pop(), stack.Pop()
		for i := 0; i < size; i++ {
			topics[i] = common.BigToHash(stack.Pop())
		}

		d := memory.Get(mStart.Int64(), mSize.Int64())
		evm.StateDB.AddLog(&model.ContractLog{
			Address: contract.Address(),
			Topics:  topics,
			Data:    d,
		})

		evm.Interpreter.IntPool.Put(mStart, mSize)
		return nil, nil
	}
}

// 生成PUSHN操作的方法，支持PUSH1-PUSH32
// 此操作可以讲合约中的数据进行压栈
func makePush(size uint64, pushByteSize int) ExecutionFunc {
	return func(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
		codeLen := len(contract.Code)

		startMin := codeLen
		if int(*pc+1) < startMin {
			startMin = int(*pc + 1)
		}

		endMin := codeLen
		if startMin+pushByteSize < endMin {
			endMin = startMin + pushByteSize
		}

		integer := evm.Interpreter.IntPool.Get()
		stack.Push(integer.SetBytes(common.RightPadBytes(contract.Code[startMin:endMin], pushByteSize)))

		*pc += size
		return nil, nil
	}
}

// 生成DUP操作，可以讲栈中指定位置数据复制并压栈
func makeDup(size int64) ExecutionFunc {
	return func(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
		stack.Dup(evm.Interpreter.IntPool, int(size))
		return nil, nil
	}
}

// 生成SWAP操作，将指定位置的数据和栈顶数据互换位置
func makeSwap(size int64) ExecutionFunc {
	size++
	return func(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
		stack.Swap(int(size))
		return nil, nil
	}
}
