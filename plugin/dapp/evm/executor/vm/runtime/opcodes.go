package runtime

// EVM操作码定义，本质上就是一个字节，所以操作码最多只支持256个
type OpCode byte

// 是否为压栈操作
func (op OpCode) IsPush() bool {
	if op >= PUSH1 && op <= PUSH32 {
		return true
	}
	return false
}

// 是否为跳转操作
func (op OpCode) IsStaticJump() bool {
	return op == JUMP
}

var opmap map[OpCode]string

// 是否为跳转操作
func (op OpCode) String() string {
	if opmap == nil {
		initMap()
	}
	return opmap[op]
}

func initMap() {
	opmap = map[OpCode]string{
		// 0x0 range - arithmetic ops
		STOP:       "STOP",
		ADD:        "ADD",
		MUL:        "MUL",
		SUB:        "SUB",
		DIV:        "DIV",
		SDIV:       "SDIV",
		MOD:        "MOD",
		SMOD:       "SMOD",
		EXP:        "EXP",
		NOT:        "NOT",
		LT:         "LT",
		GT:         "GT",
		SLT:        "SLT",
		SGT:        "SGT",
		EQ:         "EQ",
		ISZERO:     "ISZERO",
		SIGNEXTEND: "SIGNEXTEND",

		// 0x10 range - bit ops
		AND:    "AND",
		OR:     "OR",
		XOR:    "XOR",
		BYTE:   "BYTE",
		SHL:    "SHL",
		SHR:    "SHR",
		SAR:    "SAR",
		ADDMOD: "ADDMOD",
		MULMOD: "MULMOD",

		// 0x20 range - crypto
		SHA3: "SHA3",

		// 0x30 range - closure state
		ADDRESS:        "ADDRESS",
		BALANCE:        "BALANCE",
		ORIGIN:         "ORIGIN",
		CALLER:         "CALLER",
		CALLVALUE:      "CALLVALUE",
		CALLDATALOAD:   "CALLDATALOAD",
		CALLDATASIZE:   "CALLDATASIZE",
		CALLDATACOPY:   "CALLDATACOPY",
		CODESIZE:       "CODESIZE",
		CODECOPY:       "CODECOPY",
		GASPRICE:       "GASPRICE",
		EXTCODESIZE:    "EXTCODESIZE",
		EXTCODECOPY:    "EXTCODECOPY",
		RETURNDATASIZE: "RETURNDATASIZE",
		RETURNDATACOPY: "RETURNDATACOPY",

		// 0x40 range - block operations
		BLOCKHASH:  "BLOCKHASH",
		COINBASE:   "COINBASE",
		TIMESTAMP:  "TIMESTAMP",
		NUMBER:     "NUMBER",
		DIFFICULTY: "DIFFICULTY",
		GASLIMIT:   "GASLIMIT",

		// 0x50 range - 'storage' and execution
		POP: "POP",
		//DUP:     "DUP",
		//SWAP:    "SWAP",
		MLOAD:    "MLOAD",
		MSTORE:   "MSTORE",
		MSTORE8:  "MSTORE8",
		SLOAD:    "SLOAD",
		SSTORE:   "SSTORE",
		JUMP:     "JUMP",
		JUMPI:    "JUMPI",
		PC:       "PC",
		MSIZE:    "MSIZE",
		GAS:      "GAS",
		JUMPDEST: "JUMPDEST",

		// 0x60 range - push
		PUSH1:  "PUSH1",
		PUSH2:  "PUSH2",
		PUSH3:  "PUSH3",
		PUSH4:  "PUSH4",
		PUSH5:  "PUSH5",
		PUSH6:  "PUSH6",
		PUSH7:  "PUSH7",
		PUSH8:  "PUSH8",
		PUSH9:  "PUSH9",
		PUSH10: "PUSH10",
		PUSH11: "PUSH11",
		PUSH12: "PUSH12",
		PUSH13: "PUSH13",
		PUSH14: "PUSH14",
		PUSH15: "PUSH15",
		PUSH16: "PUSH16",
		PUSH17: "PUSH17",
		PUSH18: "PUSH18",
		PUSH19: "PUSH19",
		PUSH20: "PUSH20",
		PUSH21: "PUSH21",
		PUSH22: "PUSH22",
		PUSH23: "PUSH23",
		PUSH24: "PUSH24",
		PUSH25: "PUSH25",
		PUSH26: "PUSH26",
		PUSH27: "PUSH27",
		PUSH28: "PUSH28",
		PUSH29: "PUSH29",
		PUSH30: "PUSH30",
		PUSH31: "PUSH31",
		PUSH32: "PUSH32",

		DUP1:  "DUP1",
		DUP2:  "DUP2",
		DUP3:  "DUP3",
		DUP4:  "DUP4",
		DUP5:  "DUP5",
		DUP6:  "DUP6",
		DUP7:  "DUP7",
		DUP8:  "DUP8",
		DUP9:  "DUP9",
		DUP10: "DUP10",
		DUP11: "DUP11",
		DUP12: "DUP12",
		DUP13: "DUP13",
		DUP14: "DUP14",
		DUP15: "DUP15",
		DUP16: "DUP16",

		SWAP1:  "SWAP1",
		SWAP2:  "SWAP2",
		SWAP3:  "SWAP3",
		SWAP4:  "SWAP4",
		SWAP5:  "SWAP5",
		SWAP6:  "SWAP6",
		SWAP7:  "SWAP7",
		SWAP8:  "SWAP8",
		SWAP9:  "SWAP9",
		SWAP10: "SWAP10",
		SWAP11: "SWAP11",
		SWAP12: "SWAP12",
		SWAP13: "SWAP13",
		SWAP14: "SWAP14",
		SWAP15: "SWAP15",
		SWAP16: "SWAP16",
		LOG0:   "LOG0",
		LOG1:   "LOG1",
		LOG2:   "LOG2",
		LOG3:   "LOG3",
		LOG4:   "LOG4",

		// 0xf0 range
		CREATE:       "CREATE",
		CALL:         "CALL",
		RETURN:       "RETURN",
		CALLCODE:     "CALLCODE",
		DELEGATECALL: "DELEGATECALL",
		STATICCALL:   "STATICCALL",
		REVERT:       "REVERT",
		SELFDESTRUCT: "SELFDESTRUCT",

		PUSH: "PUSH",
		DUP:  "DUP",
		SWAP: "SWAP",
	}
}

// unofficial opcodes used for parsing
const (
	PUSH OpCode = 0xb0 + iota
	DUP
	SWAP
)

const (
	// 0x0 算术操作
	STOP OpCode = iota
	ADD
	MUL
	SUB
	DIV
	SDIV
	MOD
	SMOD
	ADDMOD
	MULMOD
	EXP
	SIGNEXTEND
)

const (
	// 比较、位操作
	LT OpCode = iota + 0x10
	GT
	SLT
	SGT
	EQ
	ISZERO
	AND
	OR
	XOR
	NOT
	BYTE
	SHL
	SHR
	SAR

	SHA3 = 0x20
)

const (
	// 0x30 合约数据操作
	ADDRESS OpCode = 0x30 + iota
	BALANCE
	ORIGIN
	CALLER
	CALLVALUE
	CALLDATALOAD
	CALLDATASIZE
	CALLDATACOPY
	CODESIZE
	CODECOPY
	GASPRICE
	EXTCODESIZE
	EXTCODECOPY
	RETURNDATASIZE
	RETURNDATACOPY
)

const (
	// 0x40 区块相关操作
	BLOCKHASH OpCode = 0x40 + iota
	COINBASE
	TIMESTAMP
	NUMBER
	DIFFICULTY
	GASLIMIT
)

const (
	// 0x50 存储相关操作
	POP OpCode = 0x50 + iota
	MLOAD
	MSTORE
	MSTORE8
	SLOAD
	SSTORE
	JUMP
	JUMPI
	PC
	MSIZE
	GAS
	JUMPDEST
)

const (
	// 0x60 栈操作
	PUSH1 OpCode = 0x60 + iota
	PUSH2
	PUSH3
	PUSH4
	PUSH5
	PUSH6
	PUSH7
	PUSH8
	PUSH9
	PUSH10
	PUSH11
	PUSH12
	PUSH13
	PUSH14
	PUSH15
	PUSH16
	PUSH17
	PUSH18
	PUSH19
	PUSH20
	PUSH21
	PUSH22
	PUSH23
	PUSH24
	PUSH25
	PUSH26
	PUSH27
	PUSH28
	PUSH29
	PUSH30
	PUSH31
	PUSH32
	DUP1
	DUP2
	DUP3
	DUP4
	DUP5
	DUP6
	DUP7
	DUP8
	DUP9
	DUP10
	DUP11
	DUP12
	DUP13
	DUP14
	DUP15
	DUP16
	SWAP1
	SWAP2
	SWAP3
	SWAP4
	SWAP5
	SWAP6
	SWAP7
	SWAP8
	SWAP9
	SWAP10
	SWAP11
	SWAP12
	SWAP13
	SWAP14
	SWAP15
	SWAP16
)

const (
	// 生成日志
	LOG0 OpCode = 0xa0 + iota
	LOG1
	LOG2
	LOG3
	LOG4
)

const (
	// 过程调用
	CREATE OpCode = 0xf0 + iota
	CALL
	CALLCODE
	RETURN
	DELEGATECALL
	STATICCALL = 0xfa

	REVERT       = 0xfd
	SELFDESTRUCT = 0xff
)
