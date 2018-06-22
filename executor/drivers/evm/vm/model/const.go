package model

import "gitlab.33.cn/chain33/chain33/types"

const (
	// 内存中存储的字，占用多少位
	WordBitSize = 256
	// 内存中存储的字，占用多少字节
	WordByteSize = WordBitSize / 8

	// 本执行器前缀
	EvmPrefix = types.UserEvmString
	// 本执行器名称
	ExecutorName = types.ExecerEvmString

	// 最大Gas消耗上限
	MaxGasLimit = 10000000

	// EVM本执行器支持的查询方法
	CheckAddrExistsFunc = "CheckAddrExists"
	EstimateGasFunc     = "EstimateGas"
	EvmDebug            = "EvmDebug"
)
