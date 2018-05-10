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
	opmap = make(map[OpCode]string)
	opmap[STOP] = "STOP"
	opmap[ADD] = "ADD"
	opmap[MUL] = "MUL"
	opmap[SUB] = "SUB"
	opmap[DIV] = "DIV"
	opmap[SDIV] = "SDIV"
	opmap[MOD] = "MOD"
	opmap[SMOD] = "SMOD"
	opmap[ADDMOD] = "ADDMOD"
	opmap[MULMOD] = "MULMOD"
	opmap[EXP] = "EXP"
	opmap[SIGNEXTEND] = "SIGNEXTEND"
	opmap[LT] = "LT"
	opmap[GT] = "GT"
	opmap[SLT] = "SLT"
	opmap[SGT] = "SGT"
	opmap[EQ] = "EQ"
	opmap[ISZERO] = "ISZERO"
	opmap[AND] = "AND"
	opmap[OR] = "OR"
	opmap[XOR] = "XOR"
	opmap[NOT] = "NOT"
	opmap[BYTE] = "BYTE"
	opmap[SHL] = "SHL"
	opmap[SHR] = "SHR"
	opmap[SAR] = "SAR"
	opmap[SHA3] = "SHA3"
	opmap[ADDRESS] = "ADDRESS"
	opmap[BALANCE] = "BALANCE"
	opmap[ORIGIN] = "ORIGIN"
	opmap[CALLER] = "CALLER"
	opmap[CALLVALUE] = "CALLVALUE"
	opmap[CALLDATALOAD] = "CALLDATALOAD"
	opmap[CALLDATASIZE] = "CALLDATASIZE"
	opmap[CALLDATACOPY] = "CALLDATACOPY"
	opmap[CODESIZE] = "CODESIZE"
	opmap[CODECOPY] = "CODECOPY"
	opmap[GASPRICE] = "GASPRICE"
	opmap[EXTCODESIZE] = "EXTCODESIZE"
	opmap[EXTCODECOPY] = "EXTCODECOPY"
	opmap[RETURNDATASIZE] = "RETURNDATASIZE"
	opmap[RETURNDATACOPY] = "RETURNDATACOPY"
	opmap[BLOCKHASH] = "BLOCKHASH"
	opmap[COINBASE] = "COINBASE"
	opmap[TIMESTAMP] = "TIMESTAMP"
	opmap[NUMBER] = "NUMBER"
	opmap[DIFFICULTY] = "DIFFICULTY"
	opmap[GASLIMIT] = "GASLIMIT"
	opmap[POP] = "POP"
	opmap[MLOAD] = "MLOAD"
	opmap[MSTORE] = "MSTORE"
	opmap[MSTORE8] = "MSTORE8"
	opmap[SLOAD] = "SLOAD"
	opmap[SSTORE] = "SSTORE"
	opmap[JUMP] = "JUMP"
	opmap[JUMPI] = "JUMPI"
	opmap[PC] = "PC"
	opmap[MSIZE] = "MSIZE"
	opmap[GAS] = "GAS"
	opmap[JUMPDEST] = "JUMPDEST"
	opmap[PUSH1] = "PUSH1"
	opmap[PUSH2] = "PUSH2"
	opmap[PUSH3] = "PUSH3"
	opmap[PUSH4] = "PUSH4"
	opmap[PUSH5] = "PUSH5"
	opmap[PUSH6] = "PUSH6"
	opmap[PUSH7] = "PUSH7"
	opmap[PUSH8] = "PUSH8"
	opmap[PUSH9] = "PUSH9"
	opmap[PUSH10] = "PUSH10"
	opmap[PUSH11] = "PUSH11"
	opmap[PUSH12] = "PUSH12"
	opmap[PUSH13] = "PUSH13"
	opmap[PUSH14] = "PUSH14"
	opmap[PUSH15] = "PUSH15"
	opmap[PUSH16] = "PUSH16"
	opmap[PUSH17] = "PUSH17"
	opmap[PUSH18] = "PUSH18"
	opmap[PUSH19] = "PUSH19"
	opmap[PUSH20] = "PUSH20"
	opmap[PUSH21] = "PUSH21"
	opmap[PUSH22] = "PUSH22"
	opmap[PUSH23] = "PUSH23"
	opmap[PUSH24] = "PUSH24"
	opmap[PUSH25] = "PUSH25"
	opmap[PUSH26] = "PUSH26"
	opmap[PUSH27] = "PUSH27"
	opmap[PUSH28] = "PUSH28"
	opmap[PUSH29] = "PUSH29"
	opmap[PUSH30] = "PUSH30"
	opmap[PUSH31] = "PUSH31"
	opmap[PUSH32] = "PUSH32"
	opmap[DUP1] = "DUP1"
	opmap[DUP2] = "DUP2"
	opmap[DUP3] = "DUP3"
	opmap[DUP4] = "DUP4"
	opmap[DUP5] = "DUP5"
	opmap[DUP6] = "DUP6"
	opmap[DUP7] = "DUP7"
	opmap[DUP8] = "DUP8"
	opmap[DUP9] = "DUP9"
	opmap[DUP10] = "DUP10"
	opmap[DUP11] = "DUP11"
	opmap[DUP12] = "DUP12"
	opmap[DUP13] = "DUP13"
	opmap[DUP14] = "DUP14"
	opmap[DUP15] = "DUP15"
	opmap[DUP16] = "DUP16"
	opmap[SWAP1] = "SWAP1"
	opmap[SWAP2] = "SWAP2"
	opmap[SWAP3] = "SWAP3"
	opmap[SWAP4] = "SWAP4"
	opmap[SWAP5] = "SWAP5"
	opmap[SWAP6] = "SWAP6"
	opmap[SWAP7] = "SWAP7"
	opmap[SWAP8] = "SWAP8"
	opmap[SWAP9] = "SWAP9"
	opmap[SWAP10] = "SWAP10"
	opmap[SWAP11] = "SWAP11"
	opmap[SWAP12] = "SWAP12"
	opmap[SWAP13] = "SWAP13"
	opmap[SWAP14] = "SWAP14"
	opmap[SWAP15] = "SWAP15"
	opmap[SWAP16] = "SWAP16"
	opmap[LOG0] = "LOG0"
	opmap[LOG1] = "LOG1"
	opmap[LOG2] = "LOG2"
	opmap[LOG3] = "LOG3"
	opmap[LOG4] = "LOG4"
	opmap[CREATE] = "CREATE"
	opmap[CALL] = "CALL"
	opmap[CALLCODE] = "CALLCODE"
	opmap[RETURN] = "RETURN"
	opmap[DELEGATECALL] = "DELEGATECALL"
	opmap[STATICCALL] = "STATICCALL"
	opmap[REVERT ] = "REVERT "
	opmap[SELFDESTRUCT] = "SELFDESTRUCT"
}
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
