package evm

import (
	"math/big"

	"bytes"
	"fmt"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/runtime"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/state"
	"gitlab.33.cn/chain33/chain33/types"
	"os"
	"strings"
)

func Init() {
	// TODO 注册的驱动高度需要更新为上线时的正确高度
	drivers.Register(model.ExecutorName, newEVMDriver, 0)
}


func newEVMDriver() drivers.Driver {
	evm := NewEVMExecutor()
	evm.vmCfg.Debug = debugEVM()
	return evm
}

// EVM执行器结构
type EVMExecutor struct {
	drivers.DriverBase
	vmCfg    *runtime.Config
	mStateDB *state.MemoryStateDB
}

// 根据命令行启动隐藏的参数判断是否启动EVM执行指令跟踪能力
func debugEVM() bool {
	params := os.Args[1:]
	for _, v := range params {
		if model.DebugFlag == v {
			return true
		}
	}
	return false
}

func NewEVMExecutor() *EVMExecutor {
	exec := &EVMExecutor{}

	exec.vmCfg = &runtime.Config{}
	exec.vmCfg.Tracer = runtime.NewJSONLogger(os.Stdout)

	exec.SetChild(exec)
	return exec
}

func (evm *EVMExecutor) GetName() string {
	return model.ExecutorName
}

func (evm *EVMExecutor) SetEnv(height, blockTime int64, coinBase string, difficulty uint64) {
	// 需要从这里识别出当前执行的Transaction所在的区块高度
	// 因为执行器框架在调用每一个Transaction时，都会先设置StateDB，在设置区块环境
	// 因此，在这里判断当前设置的区块高度和上一次缓存的区块高度是否为同一高度，即可判断是否在同一个区块内执行的Transaction
	if height != evm.DriverBase.GetHeight() || blockTime != evm.DriverBase.GetBlockTime() || coinBase != evm.DriverBase.GetCoinBase() || difficulty != evm.DriverBase.GetDifficulty() {
		// 这时说明区块发生了变化，需要集成原来的设置逻辑，并执行自定义操作
		evm.DriverBase.SetEnv(height, blockTime, coinBase, difficulty)

		// 在生成新的区块状态DB之前，把先前区块中生成的合约日志集中打印出来
		if evm.mStateDB != nil {
			evm.mStateDB.WritePreimages(height)
		}

		// 重新初始化MemoryStateDB
		// 需要注意的时，在执行器中只执行单个Transaction，但是并没有提交区块的动作
		// 所以，这个mStateDB只用来缓存一个区块内执行的Transaction引起的状态数据变更
		evm.mStateDB = state.NewMemoryStateDB(evm.DriverBase.GetStateDB(), evm.DriverBase.GetLocalDB(), evm.DriverBase.GetCoinsAccount())
	}
	// 两者都和上次的设置相同，不需要任何操作
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
	// 先转换消息
	msg, err := evm.GetMessage(tx)
	if err != nil {
		return nil, err
	}

	// 获取当前区块的上下文信息构造EVM上下文
	context := evm.NewEVMContext(msg)

	// 创建EVM运行时对象
	env := runtime.NewEVM(context, evm.mStateDB, *evm.vmCfg)

	isCreate := msg.To() == nil

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
		if !env.StateDB.Empty(contractAddr) {
			return nil, model.ErrContractAddressCollision
		}

		// 只有新创建的合约才能生成合约名称
		execName = fmt.Sprintf("%s%s", model.EvmPrefix, common.BytesToHash(tx.Hash()).Hex())
	} else {
		contractAddr = *msg.To()
	}

	// 状态机中设置当前交易状态
	evm.mStateDB.Prepare(common.BytesToHash(tx.Hash()), index)

	if isCreate {
		ret, snapshot, leftOverGas, vmerr = env.Create(runtime.AccountRef(msg.From()), contractAddr, msg.Data(), context.GasLimit, execName)
	} else {

		ret, snapshot, leftOverGas, vmerr = env.Call(runtime.AccountRef(msg.From()), *msg.To(), msg.Data(), context.GasLimit, msg.Value())
	}

	usedGas := msg.GasLimit() - leftOverGas
	logMsg := "call contract details:"
	if isCreate {
		logMsg = "create contract details:"
	}

	log.Info(logMsg, "caller address", msg.From().String(), "contract address", contractAddr.String(), "exec name", execName, "usedGas", usedGas, "return data", common.Bytes2Hex(ret))

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
	logs = append(logs, evm.mStateDB.GetReceiptLogs(contractAddr, isCreate)...)

	receipt := &types.Receipt{Ty: types.ExecOk, KV: data, Logs: logs}
	return receipt, nil
}

//获取运行状态名
func (evm *EVMExecutor) GetActionName(tx *types.Transaction) string {
	if bytes.Compare(tx.Execer, []byte(model.ExecutorName)) == 0 {
		return model.ExecutorName
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
	return set, err
}

// 支持命令行查询功能。
// 目前支持两个命令：
// CheckAddrExists: 判断制定的地址是否为有效的EVM合约
// EstimateGas: 估计某一合约调用消耗的Gas数量
func (evm *EVMExecutor) Query(funcName string, params []byte) (types.Message, error) {
	if evm.mStateDB == nil {
		// 之所以要在这里初始化，是因为基类在clone中没有生成数据库信息，是clone之后，调用query之前设置的
		evm.mStateDB = state.NewMemoryStateDB(evm.DriverBase.GetStateDB(), evm.DriverBase.GetLocalDB(), evm.DriverBase.GetCoinsAccount())
	}
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
	if strings.HasPrefix(addrStr, model.EvmPrefix) {
		addr = common.ExecAddress(addrStr)
	} else {
		// 合约地址
		nAddr := common.StringToAddress(addrStr)
		if nAddr == nil {
			return nil, model.ErrAddrNotExists
		}
		addr = *nAddr
	}

	exists := evm.GetMStateDB().Exist(addr)
	ret := &types.CheckEVMAddrResp{Contract: exists}
	if exists {
		account := evm.GetMStateDB().GetAccount(addr)
		if account != nil {
			ret.ContractAddr = account.Addr
			ret.ContractName = account.GetExecName()
		}
	}
	return ret, nil
}

// 此方法用来估算合约消耗的Gas，不能修改原有执行器的状态数据
func (evm *EVMExecutor) EstimateGas(req *types.EstimateEVMGasReq) (types.Message, error) {
	var (
		caller   common.Address
		to       *common.Address
		isCreate bool
	)

	// 估算Gas时直接使用一个虚拟的地址发起调用
	caller = common.ExecAddress(model.ExecutorName)

	isCreate = true
	if len(req.To) > 0 {
		to = common.StringToAddress(req.To)
		return nil, model.ErrAddrNotExists
	}

	msg := common.NewMessage(caller, to, 0, 0, model.MaxGasLimit, 1, req.Code)
	context := evm.NewEVMContext(msg)
	// 创建EVM运行时对象
	evm.mStateDB = state.NewMemoryStateDB(evm.DriverBase.GetStateDB(), evm.DriverBase.GetLocalDB(), evm.DriverBase.GetCoinsAccount())
	env := runtime.NewEVM(context, evm.mStateDB, *evm.vmCfg)
	evm.mStateDB.Prepare(common.BigToHash(big.NewInt(model.MaxGasLimit)), 0)

	var (
		vmerr        error
		leftOverGas  = uint64(0)
		contractAddr common.Address
		execName     = ""
	)

	if isCreate {
		txHash := common.BigToHash(big.NewInt(model.MaxGasLimit)).Bytes()
		contractAddr = evm.getNewAddr(txHash)
		execName = fmt.Sprintf("%s%s", model.EvmPrefix, common.BytesToHash(txHash).Hex())
		_, _, leftOverGas, vmerr = env.Create(runtime.AccountRef(msg.From()), contractAddr, msg.Data(), context.GasLimit, execName)
	} else {
		_, _, leftOverGas, vmerr = env.Call(runtime.AccountRef(msg.From()), *msg.To(), msg.Data(), context.GasLimit, msg.Value())
	}

	result := &types.EstimateEVMGasResp{}
	result.Gas = model.MaxGasLimit - leftOverGas
	return result, vmerr
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
	msg = common.NewMessage(from, to, tx.Nonce, amount, gasLimit, gasPrice, code)
	return msg, nil
}

// 构造一个新的EVM上下文对象
func (evm *EVMExecutor) NewEVMContext(msg *common.Message) runtime.Context {
	return runtime.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(evm.GetApi()),
		Origin:      msg.From(),
		Coinbase:    common.StringToAddress(evm.GetCoinBase()),
		BlockNumber: new(big.Int).SetInt64(evm.GetHeight()),
		Time:        new(big.Int).SetInt64(evm.GetBlockTime()),
		Difficulty:  new(big.Int).SetUint64(evm.GetDifficulty()),
		GasLimit:    msg.GasLimit(),
		GasPrice:    msg.GasPrice(),
	}
}

// 从交易信息中获取交易发起人地址
func getCaller(tx *types.Transaction) common.Address {
	return *common.StringToAddress(account.From(tx).String())
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
	return db.CanTransfer(sender, recipient, amount)
}

// 在内存数据库中执行转账操作（只修改内存中的金额）
// 从外部账户地址到合约账户地址
func Transfer(db state.StateDB, sender, recipient common.Address, amount uint64) bool {
	return db.Transfer(sender, recipient, amount)
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
