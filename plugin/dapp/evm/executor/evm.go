package executor

import (
	"bytes"
	"math/big"

	"fmt"
	"os"
	"strings"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/model"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/runtime"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/state"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	evmDebug = false

	// 本合约地址
	EvmAddress = address.ExecAddress(types.ExecName(model.ExecutorName))
)

var driverName string

func Init(name string) {
	driverName = name
	drivers.Register(driverName, newEVMDriver, types.ForkV17EVM)
	EvmAddress = address.ExecAddress(GetName())
	// 初始化硬分叉数据
	state.InitForkData()
}

func GetName() string {
	return newEVMDriver().GetName()
}

func newEVMDriver() drivers.Driver {
	evm := NewEVMExecutor()
	evm.vmCfg.Debug = evmDebug
	return evm
}

// EVM执行器结构
type EVMExecutor struct {
	drivers.DriverBase
	vmCfg    *runtime.Config
	mStateDB *state.MemoryStateDB
}

func NewEVMExecutor() *EVMExecutor {
	exec := &EVMExecutor{}

	exec.vmCfg = &runtime.Config{}
	exec.vmCfg.Tracer = runtime.NewJSONLogger(os.Stdout)

	exec.SetChild(exec)
	return exec
}

func (evm *EVMExecutor) GetDriverName() string {
	return model.ExecutorName
}

func (evm *EVMExecutor) Allow(tx *types.Transaction, index int) error {
	err := evm.DriverBase.Allow(tx, index)
	if err == nil {
		return nil
	}
	//增加新的规则:
	//主链: user.evm.xxx  执行 evm 合约
	//平行链: user.p.guodun.user.evm.xxx 执行 evm 合约
	if types.IsPara() && evm.allowPara(tx, index) {
		return nil
	}
	if !types.IsPara() && evm.allow(tx, index) {
		return nil
	}
	return types.ErrNotAllow
}

func (evm *EVMExecutor) allow(tx *types.Transaction, index int) bool {
	return evm.AllowIsUserDot2(tx.Execer)
}

func (evm *EVMExecutor) allowPara(tx *types.Transaction, index int) bool {
	return evm.AllowIsUserDot2Para(tx.Execer)
}

func (evm *EVMExecutor) CheckInit() {
	if evm.mStateDB == nil {
		evm.mStateDB = state.NewMemoryStateDB(evm.GetStateDB(), evm.GetLocalDB(), evm.GetCoinsAccount(), evm.GetHeight())
	}
}

// 生成一个新的合约对象地址
func (evm *EVMExecutor) getNewAddr(txHash []byte) common.Address {
	return common.NewAddress(txHash)
}

// 在区块上的执行操作，同一个区块内的多个交易会循环调用此方法进行处理；
// 返回的结果types.Receipt数据，将会被统一写入到本地状态数据库中；
// 本操作返回的ReceiptLog数据，在后面调用ExecLocal时会被传入，同样ExecLocal返回的数据将会被写入blockchain.db；
// FIXME 目前evm执行器暂时没有ExecLocal，执行默认逻辑，后面根据需要再考虑增加；
func (evm *EVMExecutor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	evm.CheckInit()
	// 先转换消息
	msg, err := evm.GetMessage(tx)
	if err != nil {
		return nil, err
	}

	// 获取当前区块的上下文信息构造EVM上下文
	context := evm.NewEVMContext(msg)

	// 创建EVM运行时对象
	env := runtime.NewEVM(context, evm.mStateDB, *evm.vmCfg)

	// 目标地址为空，或者为Evm合约的固定地址时，认为新增合约

	isCreate := strings.Compare(msg.To().String(), EvmAddress) == 0
	var (
		ret          = []byte("")
		vmerr        error
		leftOverGas  = uint64(0)
		contractAddr common.Address
		snapshot     = -1
		execName     = ""
	)

	// 为了方便计费，即使合约为新生成，也将地址的初始化放到外面操作
	if isCreate {
		// 使用随机生成的地址作为合约地址（这个可以保证每次创建的合约地址不会重复，不存在冲突的情况）
		contractAddr = evm.getNewAddr(tx.Hash())
		if !env.StateDB.Empty(contractAddr.String()) {
			return nil, model.ErrContractAddressCollision
		}
		// 只有新创建的合约才能生成合约名称
		execName = fmt.Sprintf("%s%s", types.ExecName(model.EvmPrefix), common.BytesToHash(tx.Hash()).Hex())
	} else {
		contractAddr = *msg.To()
	}

	// 状态机中设置当前交易状态
	evm.mStateDB.Prepare(common.BytesToHash(tx.Hash()), index)

	if isCreate {
		ret, snapshot, leftOverGas, vmerr = env.Create(runtime.AccountRef(msg.From()), contractAddr, msg.Data(), context.GasLimit, execName, msg.Alias())
	} else {
		ret, snapshot, leftOverGas, vmerr = env.Call(runtime.AccountRef(msg.From()), *msg.To(), msg.Data(), context.GasLimit, msg.Value())
	}

	log.Debug("call(create) contract ", "input", common.Bytes2Hex(msg.Data()))
	usedGas := msg.GasLimit() - leftOverGas
	logMsg := "call contract details:"
	if isCreate {
		logMsg = "create contract details:"
	}
	log.Debug(logMsg, "caller address", msg.From().String(), "contract address", contractAddr.String(), "exec name", execName, "alias name", msg.Alias(), "usedGas", usedGas, "return data", common.Bytes2Hex(ret))

	curVer := evm.mStateDB.GetLastSnapshot()

	if vmerr != nil {
		log.Error("evm contract exec error", "error info", vmerr)
		return nil, vmerr
	} else {
		// 计算消耗了多少费用（实际消耗的费用）
		usedFee, overflow := common.SafeMul(usedGas, uint64(msg.GasPrice()))
		// 费用消耗溢出，执行失败
		if overflow || usedFee > uint64(tx.Fee) {
			// 如果操作没有回滚，则在这里处理
			if curVer != nil && snapshot >= curVer.GetId() && curVer.GetId() > -1 {
				evm.mStateDB.RevertToSnapshot(snapshot)
			}
			return nil, model.ErrOutOfGas
		}
	}

	// 打印合约中生成的日志
	evm.mStateDB.PrintLogs()

	if curVer == nil {
		return nil, nil
	}
	// 从状态机中获取数据变更和变更日志
	data, logs := evm.mStateDB.GetChangedData(curVer.GetId())

	contractReceipt := &types.ReceiptEVMContract{msg.From().String(), execName, contractAddr.String(), usedGas, ret}

	logs = append(logs, &types.ReceiptLog{types.TyLogCallContract, types.Encode(contractReceipt)})
	logs = append(logs, evm.mStateDB.GetReceiptLogs(contractAddr.String())...)

	receipt := &types.Receipt{Ty: types.ExecOk, KV: data, Logs: logs}

	// 返回之前，把本次交易在区块中生成的合约日志集中打印出来
	if evm.mStateDB != nil {
		evm.mStateDB.WritePreimages(evm.GetHeight())
	}

	// 替换导致分叉的执行数据信息
	state.ProcessFork(evm.GetHeight(), tx.Hash(), receipt)

	evm.collectEvmTxLog(tx, contractReceipt, receipt)
	return receipt, nil
}

func (evm *EVMExecutor) collectEvmTxLog(tx *types.Transaction, cr *types.ReceiptEVMContract, receipt *types.Receipt) {
	log.Debug("evm collect begin")
	log.Debug("Tx info", "txHash", common.Bytes2Hex(tx.Hash()), "height", evm.GetHeight())
	log.Debug("ReceiptEVMContract", "data", fmt.Sprintf("caller=%v, name=%v, addr=%v, usedGas=%v, ret=%v", cr.Caller, cr.ContractName, cr.ContractAddr, cr.UsedGas, common.Bytes2Hex(cr.Ret)))
	log.Debug("receipt data", "type", receipt.Ty)
	for _, kv := range receipt.KV {
		log.Debug("KeyValue", "key", common.Bytes2Hex(kv.Key), "value", common.Bytes2Hex(kv.Value))
	}
	for _, kv := range receipt.Logs {
		log.Debug("ReceiptLog", "Type", kv.Ty, "log", common.Bytes2Hex(kv.Log))
	}
	log.Debug("evm collect end")
}

//获取运行状态名
func (evm *EVMExecutor) GetActionName(tx *types.Transaction) string {
	if bytes.Equal(tx.Execer, []byte(types.ExecName(model.ExecutorName))) {
		return types.ExecName(model.ExecutorName)
	}
	return tx.ActionName()
}

func (evm *EVMExecutor) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := evm.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	if types.IsMatchFork(evm.GetHeight(), types.ForkV20EVMState) {
		// 需要将Exec中生成的合约状态变更信息写入localdb
		for _, logItem := range receipt.Logs {
			if types.TyLogEVMStateChangeItem == logItem.Ty {
				data := logItem.Log
				var changeItem types.EVMStateChangeItem
				err = types.Decode(data, &changeItem)
				if err != nil {
					return set, err
				}
				set.KV = append(set.KV, &types.KeyValue{Key: []byte(changeItem.Key), Value: changeItem.CurrentValue})
			}
		}
	}

	return set, err
}

func (evm *EVMExecutor) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := evm.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	if types.IsMatchFork(evm.GetHeight(), types.ForkV20EVMState) {
		// 需要将Exec中生成的合约状态变更信息从localdb中恢复
		for _, logItem := range receipt.Logs {
			if types.TyLogEVMStateChangeItem == logItem.Ty {
				data := logItem.Log
				var changeItem types.EVMStateChangeItem
				err = types.Decode(data, &changeItem)
				if err != nil {
					return set, err
				}
				set.KV = append(set.KV, &types.KeyValue{Key: []byte(changeItem.Key), Value: changeItem.PreValue})
			}
		}
	}

	return set, err
}

// 支持命令行查询功能。
// 目前支持两个命令：
// CheckAddrExists: 判断制定的地址是否为有效的EVM合约
// EstimateGas: 估计某一合约调用消耗的Gas数量
func (evm *EVMExecutor) Query(funcName string, params []byte) (types.Message, error) {
	evm.CheckInit()

	if strings.EqualFold(model.CheckAddrExistsFunc, funcName) {
		var in types.CheckEVMAddrReq
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return evm.CheckAddrExists(&in)
	} else if strings.EqualFold(model.EstimateGasFunc, funcName) {
		var in types.EstimateEVMGasReq
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return evm.EstimateGas(&in)
	} else if strings.EqualFold(model.EvmDebug, funcName) {
		var in types.EvmDebugReq
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return EvmDebug(&in)
	}

	log.Error("invalid query funcName", "funcName", funcName)
	return nil, types.ErrActionNotSupport
}

// 检查合约地址是否存在，此操作不会改变任何状态，所以可以直接从statedb查询
func (evm *EVMExecutor) CheckAddrExists(req *types.CheckEVMAddrReq) (types.Message, error) {
	addrStr := req.Addr
	if len(addrStr) == 0 {
		return nil, model.ErrAddrNotExists
	}

	var addr common.Address
	// 合约名称
	if strings.HasPrefix(addrStr, types.ExecName(model.EvmPrefix)) {
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
	ret := &types.CheckEVMAddrResp{Contract: exists}
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
func (evm *EVMExecutor) EstimateGas(req *types.EstimateEVMGasReq) (types.Message, error) {
	var (
		caller common.Address
		to     *common.Address
	)

	// 如果未指定调用地址，则直接使用一个虚拟的地址发起调用
	if len(req.Caller) > 0 {
		callAddr := common.StringToAddress(req.Caller)
		if callAddr != nil {
			caller = *callAddr
		}
	} else {
		caller = common.ExecAddress(types.ExecName(model.ExecutorName))
	}

	isCreate := strings.EqualFold(req.To, EvmAddress)

	msg := common.NewMessage(caller, nil, 0, req.Amount, model.MaxGasLimit, 1, req.Code, "estimateGas")
	context := evm.NewEVMContext(msg)
	// 创建EVM运行时对象
	evm.mStateDB = state.NewMemoryStateDB(evm.GetStateDB(), evm.GetLocalDB(), evm.GetCoinsAccount(), evm.GetHeight())
	env := runtime.NewEVM(context, evm.mStateDB, *evm.vmCfg)
	evm.mStateDB.Prepare(common.BigToHash(big.NewInt(model.MaxGasLimit)), 0)

	var (
		vmerr        error
		leftOverGas  = uint64(0)
		contractAddr common.Address
		execName     = "estimateGas"
	)

	if isCreate {
		txHash := common.BigToHash(big.NewInt(model.MaxGasLimit)).Bytes()
		contractAddr = evm.getNewAddr(txHash)
		execName = fmt.Sprintf("%s%s", types.ExecName(model.EvmPrefix), common.BytesToHash(txHash).Hex())
		_, _, leftOverGas, vmerr = env.Create(runtime.AccountRef(msg.From()), contractAddr, msg.Data(), context.GasLimit, execName, "estimateGas")
	} else {
		to = common.StringToAddress(req.To)
		_, _, leftOverGas, vmerr = env.Call(runtime.AccountRef(msg.From()), *to, msg.Data(), context.GasLimit, msg.Value())
	}

	result := &types.EstimateEVMGasResp{}
	result.Gas = model.MaxGasLimit - leftOverGas
	return result, vmerr
}

// 此方法用来估算合约消耗的Gas，不能修改原有执行器的状态数据
func EvmDebug(req *types.EvmDebugReq) (types.Message, error) {
	optype := req.Optype

	if optype < 0 {
		evmDebug = false
	} else if optype > 0 {
		evmDebug = true
	}
	ret := &types.EvmDebugResp{DebugStatus: fmt.Sprintf("%v", evmDebug)}
	return ret, nil
}

func (evm *EVMExecutor) GetMStateDB() *state.MemoryStateDB {
	return evm.mStateDB
}

func (evm *EVMExecutor) GetVMConfig() *runtime.Config {
	return evm.vmCfg
}

// 目前的交易中，如果是coins交易，金额是放在payload的，但是合约不行，需要修改Transaction结构
func (evm *EVMExecutor) GetMessage(tx *types.Transaction) (msg *common.Message, err error) {
	var action types.EVMContractAction
	err = types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	// 此处暂时不考虑消息发送签名的处理，chain33在mempool中对签名做了检查
	from := getCaller(tx)
	to := getReceiver(tx)
	if to == nil {
		return nil, types.ErrInvalidAddress
	}

	// 注意Transaction中的payload内容同时包含转账金额和合约代码
	// payload[:8]为转账金额，payload[8:]为合约代码
	amount := action.Amount
	gasLimit := action.GasLimit
	gasPrice := action.GasPrice
	code := action.Code

	if gasLimit == 0 {
		gasLimit = uint64(tx.Fee)
	}
	if gasPrice == 0 {
		gasPrice = uint32(1)
	}

	// 合约的GasLimit即为调用者为本次合约调用准备支付的手续费
	msg = common.NewMessage(from, to, tx.Nonce, amount, gasLimit, gasPrice, code, action.GetAlias())
	return msg, nil
}

// 构造一个新的EVM上下文对象
func (evm *EVMExecutor) NewEVMContext(msg *common.Message) runtime.Context {
	return runtime.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(evm.GetApi()),
		Origin:      msg.From(),
		Coinbase:    nil,
		BlockNumber: new(big.Int).SetInt64(evm.GetHeight()),
		Time:        new(big.Int).SetInt64(evm.GetBlockTime()),
		Difficulty:  new(big.Int).SetUint64(evm.GetDifficulty()),
		GasLimit:    msg.GasLimit(),
		GasPrice:    msg.GasPrice(),
	}
}

// 从交易信息中获取交易发起人地址
func getCaller(tx *types.Transaction) common.Address {
	return *common.StringToAddress(tx.From())
}

// 从交易信息中获取交易目标地址，在创建合约交易中，此地址为空
func getReceiver(tx *types.Transaction) *common.Address {
	if tx.To == "" {
		return nil
	}
	addr := common.StringToAddress(tx.To)
	return addr
}

// 检查合约调用账户是否有充足的金额进行转账交易操作
func CanTransfer(db state.StateDB, sender, recipient common.Address, amount uint64) bool {
	return db.CanTransfer(sender.String(), recipient.String(), amount)
}

// 在内存数据库中执行转账操作（只修改内存中的金额）
// 从外部账户地址到合约账户地址
func Transfer(db state.StateDB, sender, recipient common.Address, amount uint64) bool {
	return db.Transfer(sender.String(), recipient.String(), amount)
}

// 获取制定高度区块的哈希
func GetHashFn(api client.QueueProtocolAPI) func(blockHeight uint64) common.Hash {
	return func(blockHeight uint64) common.Hash {
		if api != nil {
			reply, err := api.GetBlockHash(&types.ReqInt{int64(blockHeight)})
			if nil != err {
				log.Error("Call GetBlockHash Failed.", err)
			}
			return common.BytesToHash(reply.Hash)
		}
		return common.Hash{}
	}
}
