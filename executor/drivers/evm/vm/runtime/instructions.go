package runtime

import (
	"fmt"
	"math/big"

	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common/crypto"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/params"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/mm"
)

var (
	bigZero                  = new(big.Int)
)

func opAdd(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(common.U256(x.Add(x, y)))

	evm.Interpreter.IntPool.Put(y)

	return nil, nil
}

func opSub(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(common.U256(x.Sub(x, y)))

	evm.Interpreter.IntPool.Put(y)

	return nil, nil
}

func opMul(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(common.U256(x.Mul(x, y)))

	evm.Interpreter.IntPool.Put(y)

	return nil, nil
}

func opDiv(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	if y.Sign() != 0 {
		stack.Push(common.U256(x.Div(x, y)))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(y)

	return nil, nil
}

func opSdiv(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := common.S256(stack.Pop()), common.S256(stack.Pop())
	if y.Sign() == 0 {
		stack.Push(new(big.Int))
		return nil, nil
	} else {
		n := new(big.Int)
		if evm.Interpreter.IntPool.Get().Mul(x, y).Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Div(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.Push(common.U256(res))
	}
	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

func opMod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	if y.Sign() == 0 {
		stack.Push(new(big.Int))
	} else {
		stack.Push(common.U256(x.Mod(x, y)))
	}
	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

func opSmod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := common.S256(stack.Pop()), common.S256(stack.Pop())

	if y.Sign() == 0 {
		stack.Push(new(big.Int))
	} else {
		n := new(big.Int)
		if x.Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Mod(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.Push(common.U256(res))
	}
	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

func opExp(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	base, exponent := stack.Pop(), stack.Pop()
	stack.Push(common.Exp(base, exponent))

	evm.Interpreter.IntPool.Put(base, exponent)

	return nil, nil
}

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

func opNot(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x := stack.Pop()
	stack.Push(common.U256(x.Not(x)))
	return nil, nil
}

func opLt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	if x.Cmp(y) < 0 {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(x, y)
	return nil, nil
}

func opGt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	if x.Cmp(y) > 0 {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(x, y)
	return nil, nil
}

func opSlt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := common.S256(stack.Pop()), common.S256(stack.Pop())
	if x.Cmp(common.S256(y)) < 0 {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(x, y)
	return nil, nil
}

func opSgt(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := common.S256(stack.Pop()), common.S256(stack.Pop())
	if x.Cmp(y) > 0 {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(x, y)
	return nil, nil
}

func opEq(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	if x.Cmp(y) == 0 {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(x, y)
	return nil, nil
}

func opIszero(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x := stack.Pop()
	if x.Sign() > 0 {
		stack.Push(new(big.Int))
	} else {
		stack.Push(evm.Interpreter.IntPool.Get().SetUint64(1))
	}

	evm.Interpreter.IntPool.Put(x)
	return nil, nil
}

func opAnd(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(x.And(x, y))

	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

func opOr(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(x.Or(x, y))

	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

func opXor(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y := stack.Pop(), stack.Pop()
	stack.Push(x.Xor(x, y))

	evm.Interpreter.IntPool.Put(y)
	return nil, nil
}

func opByte(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	th, val := stack.Pop(), stack.Peek()
	if th.Cmp(common.Big32) < 0 {
		b := common.Byte(val, 32, int(th.Int64()))
		val.SetUint64(uint64(b))
	} else {
		val.SetUint64(0)
	}
	evm.Interpreter.IntPool.Put(th)
	return nil, nil
}

func opAddmod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y, z := stack.Pop(), stack.Pop(), stack.Pop()
	if z.Cmp(bigZero) > 0 {
		add := x.Add(x, y)
		add.Mod(add, z)
		stack.Push(common.U256(add))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(y, z)
	return nil, nil
}

func opMulmod(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	x, y, z := stack.Pop(), stack.Pop(), stack.Pop()
	if z.Cmp(bigZero) > 0 {
		mul := x.Mul(x, y)
		mul.Mod(mul, z)
		stack.Push(common.U256(mul))
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(y, z)
	return nil, nil
}

// opSHL implements Shift Left
// The SHL instruction (shift left) pops 2 values from the stack, first arg1 and then arg2,
// and pushes on the stack arg2 shifted to the left by arg1 number of bits.
func opSHL(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// Note, second operand is left in the stack; accumulate result into it, and no need to push it afterwards
	shift, value := common.U256(stack.Pop()), common.U256(stack.Peek())
	defer evm.Interpreter.IntPool.Put(shift) // First operand back into the pool

	if shift.Cmp(common.Big256) >= 0 {
		value.SetUint64(0)
		return nil, nil
	}
	n := uint(shift.Uint64())
	common.U256(value.Lsh(value, n))

	return nil, nil
}

// opSHR implements Logical Shift Right
// The SHR instruction (logical shift right) pops 2 values from the stack, first arg1 and then arg2,
// and pushes on the stack arg2 shifted to the right by arg1 number of bits with zero fill.
func opSHR(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// Note, second operand is left in the stack; accumulate result into it, and no need to push it afterwards
	shift, value := common.U256(stack.Pop()), common.U256(stack.Peek())
	defer evm.Interpreter.IntPool.Put(shift) // First operand back into the pool

	if shift.Cmp(common.Big256) >= 0 {
		value.SetUint64(0)
		return nil, nil
	}
	n := uint(shift.Uint64())
	common.U256(value.Rsh(value, n))

	return nil, nil
}

// opSAR implements Arithmetic Shift Right
// The SAR instruction (arithmetic shift right) pops 2 values from the stack, first arg1 and then arg2,
// and pushes on the stack arg2 shifted to the right by arg1 number of bits with sign extension.
func opSAR(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// Note, S256 Returns (potentially) a new bigint, so we're popping, not peeking this one
	shift, value := common.U256(stack.Pop()), common.S256(stack.Pop())
	defer evm.Interpreter.IntPool.Put(shift) // First operand back into the pool

	if shift.Cmp(common.Big256) >= 0 {
		if value.Sign() > 0 {
			value.SetUint64(0)
		} else {
			value.SetInt64(-1)
		}
		stack.Push(common.U256(value))
		return nil, nil
	}
	n := uint(shift.Uint64())
	value.Rsh(value, n)
	stack.Push(common.U256(value))

	return nil, nil
}

func opSha3(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset, size := stack.Pop(), stack.Pop()
	data := memory.Get(offset.Int64(), size.Int64())
	hash := crypto.Keccak256(data)

	if evm.VmConfig.EnablePreimageRecording {
		evm.StateDB.AddPreimage(common.BytesToHash(hash), data)
	}

	stack.Push(new(big.Int).SetBytes(hash))

	evm.Interpreter.IntPool.Put(offset, size)
	return nil, nil
}

func opAddress(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(contract.Address().Big())
	return nil, nil
}

func opBalance(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	addr := common.BigToAddress(stack.Pop())
	balance := evm.StateDB.GetBalance(addr)

	stack.Push(new(big.Int).Set(balance))
	return nil, nil
}

func opOrigin(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Origin.Big())
	return nil, nil
}

func opCaller(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(contract.Caller().Big())
	return nil, nil
}

func opCallValue(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().Set(contract.value))
	return nil, nil
}

func opCallDataLoad(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(new(big.Int).SetBytes(common.GetDataBig(contract.Input, stack.Pop(), big32)))
	return nil, nil
}

func opCallDataSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetInt64(int64(len(contract.Input))))
	return nil, nil
}

func opCallDataCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		memOffset  = stack.Pop()
		dataOffset = stack.Pop()
		length     = stack.Pop()
	)
	memory.Set(memOffset.Uint64(), length.Uint64(), common.GetDataBig(contract.Input, dataOffset, length))

	evm.Interpreter.IntPool.Put(memOffset, dataOffset, length)
	return nil, nil
}

func opReturnDataSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetUint64(uint64(len(evm.Interpreter.ReturnData))))
	return nil, nil
}

func opReturnDataCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		memOffset  = stack.Pop()
		dataOffset = stack.Pop()
		length     = stack.Pop()
	)
	defer evm.Interpreter.IntPool.Put(memOffset, dataOffset, length)

	end := new(big.Int).Add(dataOffset, length)
	if end.BitLen() > 64 || uint64(len(evm.Interpreter.ReturnData)) < end.Uint64() {
		return nil, model.ErrReturnDataOutOfBounds
	}
	memory.Set(memOffset.Uint64(), length.Uint64(), evm.Interpreter.ReturnData[dataOffset.Uint64():end.Uint64()])

	return nil, nil
}

func opExtCodeSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	a := stack.Pop()

	addr := common.BigToAddress(a)
	a.SetInt64(int64(evm.StateDB.GetCodeSize(addr)))
	stack.Push(a)

	return nil, nil
}

func opCodeSize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	l := evm.Interpreter.IntPool.Get().SetInt64(int64(len(contract.Code)))
	stack.Push(l)
	return nil, nil
}

func opCodeCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		memOffset  = stack.Pop()
		codeOffset = stack.Pop()
		length     = stack.Pop()
	)
	codeCopy := common.GetDataBig(contract.Code, codeOffset, length)
	memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)

	evm.Interpreter.IntPool.Put(memOffset, codeOffset, length)
	return nil, nil
}

func opExtCodeCopy(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		addr       = common.BigToAddress(stack.Pop())
		memOffset  = stack.Pop()
		codeOffset = stack.Pop()
		length     = stack.Pop()
	)
	codeCopy := common.GetDataBig(evm.StateDB.GetCode(addr), codeOffset, length)
	memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)

	evm.Interpreter.IntPool.Put(memOffset, codeOffset, length)
	return nil, nil
}

func opGasprice(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().Set(evm.GasPrice))
	return nil, nil
}

func opBlockhash(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	num := stack.Pop()

	n := evm.Interpreter.IntPool.Get().Sub(evm.BlockNumber, common.Big257)
	if num.Cmp(n) > 0 && num.Cmp(evm.BlockNumber) < 0 {
		stack.Push(evm.GetHash(num.Uint64()).Big())
	} else {
		stack.Push(new(big.Int))
	}

	evm.Interpreter.IntPool.Put(num, n)
	return nil, nil
}

func opCoinbase(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Coinbase.Big())
	return nil, nil
}

func opTimestamp(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(new(big.Int).Set(evm.Time)))
	return nil, nil
}

func opNumber(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(new(big.Int).Set(evm.BlockNumber)))
	return nil, nil
}

func opDifficulty(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(new(big.Int).Set(evm.Difficulty)))
	return nil, nil
}

func opGasLimit(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(common.U256(new(big.Int).SetUint64(evm.GasLimit)))
	return nil, nil
}

func opPop(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	evm.Interpreter.IntPool.Put(stack.Pop())
	return nil, nil
}

func opMload(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset := stack.Pop()
	val := new(big.Int).SetBytes(memory.Get(offset.Int64(), 32))
	stack.Push(val)

	evm.Interpreter.IntPool.Put(offset)
	return nil, nil
}

func opMstore(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// pop value of the stack
	mStart, val := stack.Pop(), stack.Pop()
	memory.Set(mStart.Uint64(), 32, common.PaddedBigBytes(val, 32))

	evm.Interpreter.IntPool.Put(mStart, val)
	return nil, nil
}

func opMstore8(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	off, val := stack.Pop().Int64(), stack.Pop().Int64()
	memory.Store[off] = byte(val & 0xff)

	return nil, nil
}

func opSload(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	loc := common.BigToHash(stack.Pop())
	val := evm.StateDB.GetState(contract.Address(), loc).Big()
	stack.Push(val)
	return nil, nil
}

func opSstore(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	loc := common.BigToHash(stack.Pop())
	val := stack.Pop()
	evm.StateDB.SetState(contract.Address(), loc, common.BigToHash(val))

	evm.Interpreter.IntPool.Put(val)
	return nil, nil
}

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

func opJumpdest(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	return nil, nil
}

func opPc(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetUint64(*pc))
	return nil, nil
}

func opMsize(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetInt64(int64(memory.Len())))
	return nil, nil
}

func opGas(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	stack.Push(evm.Interpreter.IntPool.Get().SetUint64(contract.Gas))
	return nil, nil
}

func opCreate(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	var (
		value        = stack.Pop()
		offset, size = stack.Pop(), stack.Pop()
		input        = memory.Get(offset.Int64(), size.Int64())
		gas          = contract.Gas
	)

	contract.UseGas(gas)
	res, addr, returnGas, suberr := evm.Create(contract, input, gas, value)
	if suberr != nil && suberr != model.ErrCodeStoreOutOfGas {
		stack.Push(new(big.Int))
	} else {
		stack.Push(addr.Big())
	}
	contract.Gas += returnGas
	evm.Interpreter.IntPool.Put(value, offset, size)

	if suberr == model.ErrExecutionReverted {
		return res, nil
	}
	return nil, nil
}

func opCall(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// Pop gas. The actual gas in in evm.callGasTemp.
	evm.Interpreter.IntPool.Put(stack.Pop())
	gas := evm.CallGasTemp
	// Pop other call parameters.
	addr, value, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	value = common.U256(value)
	// Get the arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		gas += params.CallStipend
	}
	ret, returnGas, err := evm.Call(contract, toAddr, args, gas, value)
	if err != nil {
		stack.Push(new(big.Int))
	} else {
		stack.Push(big.NewInt(1))
	}
	if err == nil || err == model.ErrExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evm.Interpreter.IntPool.Put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opCallCode(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// Pop gas. The actual gas is in evm.callGasTemp.
	evm.Interpreter.IntPool.Put(stack.Pop())
	gas := evm.CallGasTemp
	// Pop other call parameters.
	addr, value, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	value = common.U256(value)
	// Get arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		gas += params.CallStipend
	}
	ret, returnGas, err := evm.CallCode(contract, toAddr, args, gas, value)
	if err != nil {
		stack.Push(new(big.Int))
	} else {
		stack.Push(big.NewInt(1))
	}
	if err == nil || err == model.ErrExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evm.Interpreter.IntPool.Put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opDelegateCall(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// Pop gas. The actual gas is in evm.callGasTemp.
	evm.Interpreter.IntPool.Put(stack.Pop())
	gas := evm.CallGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	// Get arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := evm.DelegateCall(contract, toAddr, args, gas)
	if err != nil {
		stack.Push(new(big.Int))
	} else {
		stack.Push(big.NewInt(1))
	}
	if err == nil || err == model.ErrExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evm.Interpreter.IntPool.Put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opStaticCall(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	// Pop gas. The actual gas is in evm.callGasTemp.
	evm.Interpreter.IntPool.Put(stack.Pop())
	gas := evm.CallGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.BigToAddress(addr)
	// Get arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := evm.StaticCall(contract, toAddr, args, gas)
	if err != nil {
		stack.Push(new(big.Int))
	} else {
		stack.Push(big.NewInt(1))
	}
	if err == nil || err == model.ErrExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evm.Interpreter.IntPool.Put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opReturn(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset, size := stack.Pop(), stack.Pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	evm.Interpreter.IntPool.Put(offset, size)
	return ret, nil
}

func opRevert(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	offset, size := stack.Pop(), stack.Pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	evm.Interpreter.IntPool.Put(offset, size)
	return ret, nil
}

func opStop(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	return nil, nil
}

func opSuicide(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
	balance := evm.StateDB.GetBalance(contract.Address())
	evm.StateDB.AddBalance(common.BigToAddress(stack.Pop()), balance)

	evm.StateDB.Suicide(contract.Address())
	return nil, nil
}

// following functions are used by the instruction jump  table

// make log instruction function
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

// make push instruction function
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

// make push instruction function
func makeDup(size int64) ExecutionFunc {
	return func(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
		stack.Dup(evm.Interpreter.IntPool, int(size))
		return nil, nil
	}
}

// make swap instruction function
func makeSwap(size int64) ExecutionFunc {
	// switch n + 1 otherwise n would be swapped with n
	size += 1
	return func(pc *uint64, evm *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error) {
		stack.Swap(int(size))
		return nil, nil
	}
}
