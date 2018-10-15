package executor

import (
	"fmt"
	"math/big"
	"strings"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/model"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/runtime"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/state"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

// 检查合约地址是否存在，此操作不会改变任何状态，所以可以直接从statedb查询
func (evm *EVMExecutor) Query_CheckAddrExists(in *evmtypes.CheckEVMAddrReq) (types.Message, error) {
	evm.CheckInit()
	addrStr := in.Addr
	if len(addrStr) == 0 {
		return nil, model.ErrAddrNotExists
	}

	var addr common.Address
	// 合约名称
	if strings.HasPrefix(addrStr, types.ExecName(evmtypes.EvmPrefix)) {
		addr = common.ExecAddress(addrStr)
	} else {
		// 合约地址
		nAddr := common.StringToAddress(addrStr)
		if nAddr == nil {
			return nil, model.ErrAddrNotExists
		}
		addr = *nAddr
	}

	exists := evm.GetMStateDB().Exist(addr.String())
	ret := &evmtypes.CheckEVMAddrResp{Contract: exists}
	if exists {
		account := evm.GetMStateDB().GetAccount(addr.String())
		if account != nil {
			ret.ContractAddr = account.Addr
			ret.ContractName = account.GetExecName()
			ret.AliasName = account.GetAliasName()
		}
	}
	return ret, nil
}

// 此方法用来估算合约消耗的Gas，不能修改原有执行器的状态数据
func (evm *EVMExecutor) Query_EstimateGas(in *evmtypes.EstimateEVMGasReq) (types.Message, error) {
	evm.CheckInit()
	var (
		caller common.Address
		to     *common.Address
	)

	// 如果未指定调用地址，则直接使用一个虚拟的地址发起调用
	if len(in.Caller) > 0 {
		callAddr := common.StringToAddress(in.Caller)
		if callAddr != nil {
			caller = *callAddr
		}
	} else {
		caller = common.ExecAddress(types.ExecName(evmtypes.ExecutorName))
	}

	isCreate := strings.EqualFold(in.To, EvmAddress)

	msg := common.NewMessage(caller, nil, 0, in.Amount, evmtypes.MaxGasLimit, 1, in.Code, "estimateGas")
	context := evm.NewEVMContext(msg)
	// 创建EVM运行时对象
	evm.mStateDB = state.NewMemoryStateDB(evm.GetStateDB(), evm.GetLocalDB(), evm.GetCoinsAccount(), evm.GetHeight())
	env := runtime.NewEVM(context, evm.mStateDB, *evm.vmCfg)
	evm.mStateDB.Prepare(common.BigToHash(big.NewInt(evmtypes.MaxGasLimit)), 0)

	var (
		vmerr        error
		leftOverGas  = uint64(0)
		contractAddr common.Address
		execName     = "estimateGas"
	)

	if isCreate {
		txHash := common.BigToHash(big.NewInt(evmtypes.MaxGasLimit)).Bytes()
		contractAddr = evm.getNewAddr(txHash)
		execName = fmt.Sprintf("%s%s", types.ExecName(evmtypes.EvmPrefix), common.BytesToHash(txHash).Hex())
		_, _, leftOverGas, vmerr = env.Create(runtime.AccountRef(msg.From()), contractAddr, msg.Data(), context.GasLimit, execName, "estimateGas")
	} else {
		to = common.StringToAddress(in.To)
		_, _, leftOverGas, vmerr = env.Call(runtime.AccountRef(msg.From()), *to, msg.Data(), context.GasLimit, msg.Value())
	}

	result := &evmtypes.EstimateEVMGasResp{}
	result.Gas = evmtypes.MaxGasLimit - leftOverGas
	return result, vmerr
}

// 此方法用来估算合约消耗的Gas，不能修改原有执行器的状态数据
func (evm *EVMExecutor) Query_EvmDebug(in *evmtypes.EvmDebugReq) (types.Message, error) {
	evm.CheckInit()
	optype := in.Optype

	if optype < 0 {
		evmDebug = false
	} else if optype > 0 {
		evmDebug = true
	}
	ret := &evmtypes.EvmDebugResp{DebugStatus: fmt.Sprintf("%v", evmDebug)}
	return ret, nil
}
