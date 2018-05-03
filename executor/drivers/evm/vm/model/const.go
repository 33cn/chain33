package model

const (
	// Gas计费日志
	TyLogGasFee   = 1

	// 合约代码变更日志
	TyLogContractData = 2

	// 合约状态数据变更日志
	TyLogContractState = 3
)

const (
	// 内存中存储的字，占用多少位
	WordBitSize = 256
	// 内存中存储的字，占用多少字节
	WordByteSize = WordBitSize/8
)