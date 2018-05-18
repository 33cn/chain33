package evm

import (
	"math/big"

	"encoding/hex"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common/crypto"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/runtime"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/state"
	"gitlab.33.cn/chain33/chain33/types"
	"os"
)

const (
	// 交易payload中，前8个字节固定存储转账信息
	BALANCE_SIZE = 8
)

var (
	//目前这里先用默认配置，后面在增加具体实现 TODO
	VMConfig = &runtime.Config{}
)

// EVM执行器结构
type EVMExecutor struct {
	drivers.DriverBase
	vmCfg    *runtime.Config
	mStateDB *state.MemoryStateDB
}

func Init() {
	evm := NewEVMExecutor()

	// TODO 注册的驱动高度需要更新为上线时的正确高度
	drivers.Register(evm.GetName(), evm, 0)
}

func NewEVMExecutor() *EVMExecutor {
	exec := &EVMExecutor{}

	exec.vmCfg = VMConfig
	exec.vmCfg.Tracer = runtime.NewJSONLogger(os.Stdout)

	// 打开EVM调试开关
	exec.vmCfg.Debug = true

	exec.SetChild(exec)
	return exec
}

func (evm *EVMExecutor) GetName() string {
	return "user.evm"
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
func (evm *EVMExecutor) getNewAddr(env *runtime.EVM) *common.Address {
	var addr *common.Address
	for count := 0; count < 10; count++ {
		addr = crypto.RandomContractAddress()
		if env.StateDB.Empty(*addr) {
			return addr
		}
	}
	return nil
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
	)

	// 为了方便计费，即使合约为新生成，也将地址的初始化放到外面操作
	if isCreate {
		// 使用随机生成的地址作为合约地址（这个可以保证每次创建的合约地址不会重复，不存在冲突的情况）
		newAddr := evm.getNewAddr(env)
		if newAddr == nil {
			return nil, model.ErrContractAddressCollision
		}
		contractAddr = *newAddr
	} else {
		contractAddr = *msg.To()
	}

	// 状态机中设置当前交易状态
	evm.mStateDB.Prepare(common.BytesToHash(tx.Hash()), index)

	if isCreate {
		ret, snapshot, leftOverGas, vmerr = env.Create(runtime.AccountRef(msg.From()), contractAddr, msg.Data(), context.GasLimit, msg.Value())
	} else {

		ret, snapshot, leftOverGas, vmerr = env.Call(runtime.AccountRef(msg.From()), *msg.To(), msg.Data(), context.GasLimit, msg.Value())
	}

	usedGas := msg.GasLimit() - leftOverGas
	logMsg := "call contract details:"
	if isCreate {
		logMsg = "create contract details:"
	}

	log.Info(logMsg, "caller address", msg.From().Str(), "contract address", contractAddr.Str(), "usedGas", usedGas, "return data", hex.EncodeToString(ret))

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
	if logs != nil {
		logs = append(logs, evm.mStateDB.GetReceiptLogs(contractAddr, isCreate)...)
	}
	receipt := &types.Receipt{Ty: types.ExecOk, KV: data, Logs: logs}
	return receipt, nil
}

func (evm *EVMExecutor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	accFrom := evm.GetCoinsAccount().LoadAccount(from)
	if accFrom.GetBalance()-tx.Fee >= 0 {
		copyfrom := *accFrom
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		receiptBalance := &types.ReceiptAccountTransfer{&copyfrom, accFrom}
		evm.GetCoinsAccount().SaveAccount(accFrom)
		return evm.cutFeeReceipt(accFrom, receiptBalance), nil
	}
	return nil, types.ErrNoBalance
}

func (evm *EVMExecutor) cutFeeReceipt(acc *types.Account, receiptBalance proto.Message) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecPack, evm.GetCoinsAccount().GetKVSet(acc), []*types.ReceiptLog{feelog}}
}

func (evm *EVMExecutor) GetMStateDB() *state.MemoryStateDB {
	return evm.mStateDB
}

func (evm *EVMExecutor) GetVMConfig() *runtime.Config {
	return evm.vmCfg
}

// 目前的交易中，如果是coins交易，金额是放在payload的，但是合约不行，需要修改Transaction结构
func (evm *EVMExecutor) GetMessage(tx *types.Transaction) (msg *common.Message, err error) {
	var action model.ContractAction
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
func CanTransfer(db state.StateDB, addr common.Address, amount uint64) bool {
	return db.CanTransfer(addr, amount)
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
