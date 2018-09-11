package runtime

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/gas"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/mm"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/params"
)

type (
	// 指令执行函数，每个操作指令对应一个实现，它实现了指令的具体操作逻辑
	ExecutionFunc func(pc *uint64, env *EVM, contract *Contract, memory *mm.Memory, stack *mm.Stack) ([]byte, error)
)

// 定义指令操作的结构提
type Operation struct {
	// 指令的具体操作逻辑
	Execute ExecutionFunc

	// 计算当前指令执行所需消耗的Gas
	GasCost gas.GasFunc

	// 检查内存栈中的数据是否满足本操作执行的要求
	ValidateStack mm.StackValidationFunc

	// 计算本次操作所需要的内存大小
	MemorySize mm.MemorySizeFunc

	Halts   bool // 是否需要暂停（将会结束本合约后面操作的执行）
	Jumps   bool // 是否需要执行跳转（此种情况下PC不递增）
	Writes  bool // 是否涉及到修改状态操作（在合约委托调用的情况下，此操作非法，将会抛异常）
	Valid   bool // 是否为有效操作
	Reverts bool // 是否恢复原始状态（强制暂停，将会结束本合约后面操作的执行）
	Returns bool // 是否返回
}

var (
	// 对应EVM不同版本的指令集，从上往下，从旧版本到新版本，
	// 新版本包含旧版本的指令集（目前直接使用康士坦丁堡指令集）
	FrontierInstructionSet       = NewFrontierInstructionSet()
	HomesteadInstructionSet      = NewHomesteadInstructionSet()
	ByzantiumInstructionSet      = NewByzantiumInstructionSet()
	ConstantinopleInstructionSet = NewConstantinopleInstructionSet()
)

// 康士坦丁堡 版本支持的指令集
func NewConstantinopleInstructionSet() [256]Operation {
	instructionSet := NewByzantiumInstructionSet()
	instructionSet[SHL] = Operation{
		Execute:       opSHL,
		GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
		ValidateStack: mm.MakeStackFunc(2, 1),
		Valid:         true,
	}
	instructionSet[SHR] = Operation{
		Execute:       opSHR,
		GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
		ValidateStack: mm.MakeStackFunc(2, 1),
		Valid:         true,
	}
	instructionSet[SAR] = Operation{
		Execute:       opSAR,
		GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
		ValidateStack: mm.MakeStackFunc(2, 1),
		Valid:         true,
	}
	return instructionSet
}

// 拜占庭 版本支持的指令集
func NewByzantiumInstructionSet() [256]Operation {
	instructionSet := NewHomesteadInstructionSet()
	instructionSet[STATICCALL] = Operation{
		Execute:       opStaticCall,
		GasCost:       gas.GasStaticCall,
		ValidateStack: mm.MakeStackFunc(6, 1),
		MemorySize:    mm.MemoryStaticCall,
		Valid:         true,
		Returns:       true,
	}
	instructionSet[RETURNDATASIZE] = Operation{
		Execute:       opReturnDataSize,
		GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
		ValidateStack: mm.MakeStackFunc(0, 1),
		Valid:         true,
	}
	instructionSet[RETURNDATACOPY] = Operation{
		Execute:       opReturnDataCopy,
		GasCost:       gas.GasReturnDataCopy,
		ValidateStack: mm.MakeStackFunc(3, 0),
		MemorySize:    mm.MemoryReturnDataCopy,
		Valid:         true,
	}
	instructionSet[REVERT] = Operation{
		Execute:       opRevert,
		GasCost:       gas.GasRevert,
		ValidateStack: mm.MakeStackFunc(2, 0),
		MemorySize:    mm.MemoryRevert,
		Valid:         true,
		Reverts:       true,
		Returns:       true,
	}
	return instructionSet
}

// 家园 版本支持的指令集
func NewHomesteadInstructionSet() [256]Operation {
	instructionSet := NewFrontierInstructionSet()
	instructionSet[DELEGATECALL] = Operation{
		Execute:       opDelegateCall,
		GasCost:       gas.GasDelegateCall,
		ValidateStack: mm.MakeStackFunc(6, 1),
		MemorySize:    mm.MemoryDelegateCall,
		Valid:         true,
		Returns:       true,
	}
	return instructionSet
}

// 边境 版本支持的指令集
func NewFrontierInstructionSet() [256]Operation {
	return [256]Operation{
		STOP: {
			Execute:       opStop,
			GasCost:       gas.ConstGasFunc(0),
			ValidateStack: mm.MakeStackFunc(0, 0),
			Halts:         true,
			Valid:         true,
		},
		ADD: {
			Execute:       opAdd,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		MUL: {
			Execute:       opMul,
			GasCost:       gas.ConstGasFunc(gas.GasFastStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		SUB: {
			Execute:       opSub,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		DIV: {
			Execute:       opDiv,
			GasCost:       gas.ConstGasFunc(gas.GasFastStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		SDIV: {
			Execute:       opSdiv,
			GasCost:       gas.ConstGasFunc(gas.GasFastStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		MOD: {
			Execute:       opMod,
			GasCost:       gas.ConstGasFunc(gas.GasFastStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		SMOD: {
			Execute:       opSmod,
			GasCost:       gas.ConstGasFunc(gas.GasFastStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		ADDMOD: {
			Execute:       opAddmod,
			GasCost:       gas.ConstGasFunc(gas.GasMidStep),
			ValidateStack: mm.MakeStackFunc(3, 1),
			Valid:         true,
		},
		MULMOD: {
			Execute:       opMulmod,
			GasCost:       gas.ConstGasFunc(gas.GasMidStep),
			ValidateStack: mm.MakeStackFunc(3, 1),
			Valid:         true,
		},
		EXP: {
			Execute:       opExp,
			GasCost:       gas.GasExp,
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		SIGNEXTEND: {
			Execute:       opSignExtend,
			GasCost:       gas.ConstGasFunc(gas.GasFastStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		LT: {
			Execute:       opLt,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		GT: {
			Execute:       opGt,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		SLT: {
			Execute:       opSlt,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		SGT: {
			Execute:       opSgt,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		EQ: {
			Execute:       opEq,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		ISZERO: {
			Execute:       opIszero,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(1, 1),
			Valid:         true,
		},
		AND: {
			Execute:       opAnd,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		XOR: {
			Execute:       opXor,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		OR: {
			Execute:       opOr,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		NOT: {
			Execute:       opNot,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(1, 1),
			Valid:         true,
		},
		BYTE: {
			Execute:       opByte,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(2, 1),
			Valid:         true,
		},
		SHA3: {
			Execute:       opSha3,
			GasCost:       gas.GasSha3,
			ValidateStack: mm.MakeStackFunc(2, 1),
			MemorySize:    mm.MemorySha3,
			Valid:         true,
		},
		ADDRESS: {
			Execute:       opAddress,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		BALANCE: {
			Execute:       opBalance,
			GasCost:       gas.GasBalance,
			ValidateStack: mm.MakeStackFunc(1, 1),
			Valid:         true,
		},
		ORIGIN: {
			Execute:       opOrigin,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		CALLER: {
			Execute:       opCaller,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		CALLVALUE: {
			Execute:       opCallValue,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		CALLDATALOAD: {
			Execute:       opCallDataLoad,
			GasCost:       gas.ConstGasFunc(gas.GasFastestStep),
			ValidateStack: mm.MakeStackFunc(1, 1),
			Valid:         true,
		},
		CALLDATASIZE: {
			Execute:       opCallDataSize,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		CALLDATACOPY: {
			Execute:       opCallDataCopy,
			GasCost:       gas.GasCallDataCopy,
			ValidateStack: mm.MakeStackFunc(3, 0),
			MemorySize:    mm.MemoryCallDataCopy,
			Valid:         true,
		},
		CODESIZE: {
			Execute:       opCodeSize,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		CODECOPY: {
			Execute:       opCodeCopy,
			GasCost:       gas.GasCodeCopy,
			ValidateStack: mm.MakeStackFunc(3, 0),
			MemorySize:    mm.MemoryCodeCopy,
			Valid:         true,
		},
		GASPRICE: {
			Execute:       opGasprice,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		EXTCODESIZE: {
			Execute:       opExtCodeSize,
			GasCost:       gas.GasExtCodeSize,
			ValidateStack: mm.MakeStackFunc(1, 1),
			Valid:         true,
		},
		EXTCODECOPY: {
			Execute:       opExtCodeCopy,
			GasCost:       gas.GasExtCodeCopy,
			ValidateStack: mm.MakeStackFunc(4, 0),
			MemorySize:    mm.MemoryExtCodeCopy,
			Valid:         true,
		},
		BLOCKHASH: {
			Execute:       opBlockhash,
			GasCost:       gas.ConstGasFunc(gas.GasExtStep),
			ValidateStack: mm.MakeStackFunc(1, 1),
			Valid:         true,
		},
		COINBASE: {
			Execute:       opCoinbase,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		TIMESTAMP: {
			Execute:       opTimestamp,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		NUMBER: {
			Execute:       opNumber,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		DIFFICULTY: {
			Execute:       opDifficulty,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		GASLIMIT: {
			Execute:       opGasLimit,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		POP: {
			Execute:       opPop,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(1, 0),
			Valid:         true,
		},
		MLOAD: {
			Execute:       opMload,
			GasCost:       gas.GasMLoad,
			ValidateStack: mm.MakeStackFunc(1, 1),
			MemorySize:    mm.MemoryMLoad,
			Valid:         true,
		},
		MSTORE: {
			Execute:       opMstore,
			GasCost:       gas.GasMStore,
			ValidateStack: mm.MakeStackFunc(2, 0),
			MemorySize:    mm.MemoryMStore,
			Valid:         true,
		},
		MSTORE8: {
			Execute:       opMstore8,
			GasCost:       gas.GasMStore8,
			MemorySize:    mm.MemoryMStore8,
			ValidateStack: mm.MakeStackFunc(2, 0),

			Valid: true,
		},
		SLOAD: {
			Execute:       opSload,
			GasCost:       gas.GasSLoad,
			ValidateStack: mm.MakeStackFunc(1, 1),
			Valid:         true,
		},
		SSTORE: {
			Execute:       opSstore,
			GasCost:       gas.GasSStore,
			ValidateStack: mm.MakeStackFunc(2, 0),
			Valid:         true,
			Writes:        true,
		},
		JUMP: {
			Execute:       opJump,
			GasCost:       gas.ConstGasFunc(gas.GasMidStep),
			ValidateStack: mm.MakeStackFunc(1, 0),
			Jumps:         true,
			Valid:         true,
		},
		JUMPI: {
			Execute:       opJumpi,
			GasCost:       gas.ConstGasFunc(gas.GasSlowStep),
			ValidateStack: mm.MakeStackFunc(2, 0),
			Jumps:         true,
			Valid:         true,
		},
		PC: {
			Execute:       opPc,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		MSIZE: {
			Execute:       opMsize,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		GAS: {
			Execute:       opGas,
			GasCost:       gas.ConstGasFunc(gas.GasQuickStep),
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		JUMPDEST: {
			Execute:       opJumpdest,
			GasCost:       gas.ConstGasFunc(params.JumpdestGas),
			ValidateStack: mm.MakeStackFunc(0, 0),
			Valid:         true,
		},
		PUSH1: {
			Execute:       makePush(1, 1),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH2: {
			Execute:       makePush(2, 2),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH3: {
			Execute:       makePush(3, 3),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH4: {
			Execute:       makePush(4, 4),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH5: {
			Execute:       makePush(5, 5),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH6: {
			Execute:       makePush(6, 6),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH7: {
			Execute:       makePush(7, 7),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH8: {
			Execute:       makePush(8, 8),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH9: {
			Execute:       makePush(9, 9),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH10: {
			Execute:       makePush(10, 10),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH11: {
			Execute:       makePush(11, 11),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH12: {
			Execute:       makePush(12, 12),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH13: {
			Execute:       makePush(13, 13),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH14: {
			Execute:       makePush(14, 14),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH15: {
			Execute:       makePush(15, 15),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH16: {
			Execute:       makePush(16, 16),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH17: {
			Execute:       makePush(17, 17),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH18: {
			Execute:       makePush(18, 18),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH19: {
			Execute:       makePush(19, 19),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH20: {
			Execute:       makePush(20, 20),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH21: {
			Execute:       makePush(21, 21),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH22: {
			Execute:       makePush(22, 22),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH23: {
			Execute:       makePush(23, 23),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH24: {
			Execute:       makePush(24, 24),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH25: {
			Execute:       makePush(25, 25),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH26: {
			Execute:       makePush(26, 26),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH27: {
			Execute:       makePush(27, 27),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH28: {
			Execute:       makePush(28, 28),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH29: {
			Execute:       makePush(29, 29),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH30: {
			Execute:       makePush(30, 30),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH31: {
			Execute:       makePush(31, 31),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH32: {
			Execute:       makePush(32, 32),
			GasCost:       gas.GasPush,
			ValidateStack: mm.MakeStackFunc(0, 1),
			Valid:         true,
		},
		DUP1: {
			Execute:       makeDup(1),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(1),
			Valid:         true,
		},
		DUP2: {
			Execute:       makeDup(2),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(2),
			Valid:         true,
		},
		DUP3: {
			Execute:       makeDup(3),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(3),
			Valid:         true,
		},
		DUP4: {
			Execute:       makeDup(4),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(4),
			Valid:         true,
		},
		DUP5: {
			Execute:       makeDup(5),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(5),
			Valid:         true,
		},
		DUP6: {
			Execute:       makeDup(6),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(6),
			Valid:         true,
		},
		DUP7: {
			Execute:       makeDup(7),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(7),
			Valid:         true,
		},
		DUP8: {
			Execute:       makeDup(8),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(8),
			Valid:         true,
		},
		DUP9: {
			Execute:       makeDup(9),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(9),
			Valid:         true,
		},
		DUP10: {
			Execute:       makeDup(10),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(10),
			Valid:         true,
		},
		DUP11: {
			Execute:       makeDup(11),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(11),
			Valid:         true,
		},
		DUP12: {
			Execute:       makeDup(12),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(12),
			Valid:         true,
		},
		DUP13: {
			Execute:       makeDup(13),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(13),
			Valid:         true,
		},
		DUP14: {
			Execute:       makeDup(14),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(14),
			Valid:         true,
		},
		DUP15: {
			Execute:       makeDup(15),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(15),
			Valid:         true,
		},
		DUP16: {
			Execute:       makeDup(16),
			GasCost:       gas.GasDup,
			ValidateStack: mm.MakeDupStackFunc(16),
			Valid:         true,
		},
		SWAP1: {
			Execute:       makeSwap(1),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(2),
			Valid:         true,
		},
		SWAP2: {
			Execute:       makeSwap(2),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(3),
			Valid:         true,
		},
		SWAP3: {
			Execute:       makeSwap(3),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(4),
			Valid:         true,
		},
		SWAP4: {
			Execute:       makeSwap(4),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(5),
			Valid:         true,
		},
		SWAP5: {
			Execute:       makeSwap(5),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(6),
			Valid:         true,
		},
		SWAP6: {
			Execute:       makeSwap(6),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(7),
			Valid:         true,
		},
		SWAP7: {
			Execute:       makeSwap(7),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(8),
			Valid:         true,
		},
		SWAP8: {
			Execute:       makeSwap(8),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(9),
			Valid:         true,
		},
		SWAP9: {
			Execute:       makeSwap(9),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(10),
			Valid:         true,
		},
		SWAP10: {
			Execute:       makeSwap(10),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(11),
			Valid:         true,
		},
		SWAP11: {
			Execute:       makeSwap(11),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(12),
			Valid:         true,
		},
		SWAP12: {
			Execute:       makeSwap(12),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(13),
			Valid:         true,
		},
		SWAP13: {
			Execute:       makeSwap(13),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(14),
			Valid:         true,
		},
		SWAP14: {
			Execute:       makeSwap(14),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(15),
			Valid:         true,
		},
		SWAP15: {
			Execute:       makeSwap(15),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(16),
			Valid:         true,
		},
		SWAP16: {
			Execute:       makeSwap(16),
			GasCost:       gas.GasSwap,
			ValidateStack: mm.MakeSwapStackFunc(17),
			Valid:         true,
		},
		LOG0: {
			Execute:       makeLog(0),
			GasCost:       gas.MakeGasLog(0),
			ValidateStack: mm.MakeStackFunc(2, 0),
			MemorySize:    mm.MemoryLog,
			Valid:         true,
			Writes:        true,
		},
		LOG1: {
			Execute:       makeLog(1),
			GasCost:       gas.MakeGasLog(1),
			ValidateStack: mm.MakeStackFunc(3, 0),
			MemorySize:    mm.MemoryLog,
			Valid:         true,
			Writes:        true,
		},
		LOG2: {
			Execute:       makeLog(2),
			GasCost:       gas.MakeGasLog(2),
			ValidateStack: mm.MakeStackFunc(4, 0),
			MemorySize:    mm.MemoryLog,
			Valid:         true,
			Writes:        true,
		},
		LOG3: {
			Execute:       makeLog(3),
			GasCost:       gas.MakeGasLog(3),
			ValidateStack: mm.MakeStackFunc(5, 0),
			MemorySize:    mm.MemoryLog,
			Valid:         true,
			Writes:        true,
		},
		LOG4: {
			Execute:       makeLog(4),
			GasCost:       gas.MakeGasLog(4),
			ValidateStack: mm.MakeStackFunc(6, 0),
			MemorySize:    mm.MemoryLog,
			Valid:         true,
			Writes:        true,
		},
		CREATE: {
			Execute:       opCreate,
			GasCost:       gas.GasCreate,
			ValidateStack: mm.MakeStackFunc(3, 1),
			MemorySize:    mm.MemoryCreate,
			Valid:         true,
			Writes:        true,
			Returns:       true,
		},
		CALL: {
			Execute:       opCall,
			GasCost:       gas.GasCall,
			ValidateStack: mm.MakeStackFunc(7, 1),
			MemorySize:    mm.MemoryCall,
			Valid:         true,
			Returns:       true,
		},
		CALLCODE: {
			Execute:       opCallCode,
			GasCost:       gas.GasCallCode,
			ValidateStack: mm.MakeStackFunc(7, 1),
			MemorySize:    mm.MemoryCall,
			Valid:         true,
			Returns:       true,
		},
		RETURN: {
			Execute:       opReturn,
			GasCost:       gas.GasReturn,
			ValidateStack: mm.MakeStackFunc(2, 0),
			MemorySize:    mm.MemoryReturn,
			Halts:         true,
			Valid:         true,
		},
		SELFDESTRUCT: {
			Execute:       opSuicide,
			GasCost:       gas.GasSuicide,
			ValidateStack: mm.MakeStackFunc(1, 0),
			Halts:         true,
			Valid:         true,
			Writes:        true,
		},
	}
}
