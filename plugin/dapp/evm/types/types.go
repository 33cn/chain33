package types

import "gitlab.33.cn/chain33/chain33/types"

const (
	// 本执行器前缀
	EvmPrefix = types.UserEvmX
	// 本执行器名称
	ExecutorName = types.EvmX

	EvmCreateAction = 1
	EvmCallAction   = 2

	// 合约代码变更日志
	TyLogContractData = 601
	// 合约状态数据变更日志
	TyLogContractState = 602
	// 合约状态数据变更日志
	TyLogCallContract = 603
	// 合约状态数据变更项日志
	TyLogEVMStateChangeItem = 604

	// 查询方法
	FuncCheckAddrExists = "CheckAddrExists"
	FuncEstimateGas     = "EstimateGas"
	FuncEvmDebug        = "EvmDebug"

	// 最大Gas消耗上限
	MaxGasLimit = 10000000
)

var (
	JRPCName = "evm"
)
