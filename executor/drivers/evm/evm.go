package evm

import (
	"math/big"

	"encoding/hex"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/state"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/params"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/runtime"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
)

const (
	// 交易payload中，前8个字节固定存储转账信息
	BALANCE_SIZE = 8

	// 在一个交易中，GasLimit设置为交易费用的倍数(整数)
	TX_GAS_TIMES_FEE = 2
)

var (
	GasPrice = big.NewInt(1)
)
var clog = log.New("module", "execs.evm")

type FakeEVM struct {
	drivers.DriverBase
	vmCfg *runtime.Config

	mStateDB *state.MemoryStateDB
}

func init() {
	evm := NewFakeEVM()

	// TODO 注册的驱动高度需要更新为上线时的正确高度
	drivers.Register(evm.GetName(), evm, 0)
}

func NewFakeEVM() *FakeEVM {
	fake := &FakeEVM{}
	fake.vmCfg = &runtime.Config{}
	fake.SetChild(fake)
	return fake
}

func (evm *FakeEVM) GetName() string {
	return "evm"
}

func (evm *FakeEVM) SetEnv(height, blocktime int64) {
	// 需要从这里识别出当前执行的Transaction所在的区块高度
	// 因为执行器框架在调用每一个Transaction时，都会先设置StateDB，在设置区块环境
	// 因此，在这里判断当前设置的区块高度和上一次缓存的区块高度是否为同一高度，即可判断是否在同一个区块内执行的Transaction
	if height != evm.DriverBase.GetHeight() || blocktime != evm.DriverBase.GetBlockTime() {
		// 这时说明区块发生了变化，需要集成原来的设置逻辑，并执行自定义操作
		evm.DriverBase.SetEnv(height, blocktime)

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

func (evm *FakeEVM) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	// TODO:GAS计费信息先不考虑，后续补充
	var (
	//gp      = new(vm.GasPool).AddGas(uint64(tx.Fee))
	//usedGas = uint64(0)
	)

	// 先转换消息
	msg := evm.GetMessage(tx)

	// 创建EVM上下文
	//header := wrapper.GetBlockHeader()
	config := evm.GetChainConfig()
	vmcfg := evm.GetVMConfig()

	// 获取当前区块的高度和时间
	height := evm.DriverBase.GetHeight()
	time := evm.DriverBase.GetBlockTime()

	//FIXME 需要获取coinbase，目前没有
	//FIXME 还有难度值，也需要获取  这两个信息都需要在执行区块时传进来
	coinbase := common.StringToAddress("1CNCzMdMNjYHkNUdfnEjat2i2bR9NdXrmR")
	difficulty := uint64(10000)

	context := NewEVMContext(msg, height, time, coinbase, difficulty)

	// 创建EVM运行时对象
	env := runtime.NewEVM(context, evm.mStateDB, config, *vmcfg)

	isCreate := msg.To() == nil

	var (
		ret         = []byte("")
		vmerr       error
		leftOverGas = uint64(0)
		addr        = *msg.To()
	)

	// 状态机中设置当前交易状态
	evm.mStateDB.Prepare(common.BytesToHash(tx.Hash()), index)

	// 合约执行之前，预先扣除GasLimit费用
	evm.mStateDB.SubBalance(msg.From(), big.NewInt(1).Mul(big.NewInt(int64(msg.GasLimit())), msg.GasPrice()))

	if isCreate {
		ret, addr, leftOverGas, vmerr = env.Create(runtime.AccountRef(msg.From()), msg.Data(), context.GasLimit, big.NewInt(0))
	} else {
		ret, leftOverGas, vmerr = env.Call(runtime.AccountRef(msg.From()), *msg.To(), msg.Data(), context.GasLimit, big.NewInt(0))
	}

	if vmerr != nil {
		log.Info("VM returned with error", "err", vmerr)

		// 只有在出现账户余额不足的时候，才认为是有效的的错误
		// 其它错误情况目前不需要处理
		if vmerr == model.ErrInsufficientBalance {
			return nil, vmerr
		}
	}

	// 根据真实使用的Gas，将多余的费用返还
	// 此处将合约执行过程中给予的奖励也一并返还
	refundGas := leftOverGas + evm.mStateDB.GetRefund()
	evm.mStateDB.AddBalance(msg.From(), big.NewInt(1).Mul(big.NewInt(int64(refundGas)), msg.GasPrice()))

	// 计算消耗了多少Gas
	// 注意：这里和以太坊EVM计费有几处不同：
	// 1. GasPrice 始终为1，不支持动态调整  TODO 后继版本考虑支持动态调整
	// 2. 不计算IntrinsicGas，即合约代码的存储字节计费以及合约创建或调用动作本身的计费
	// 3. 因为前面计费内容较少，不考虑单独的refund奖励（合约执行过程中给予的refund支持返还）
	usedGas := msg.GasLimit() - refundGas

	// 将消耗的费用奖励给区块作者
	evm.mStateDB.AddBalance(coinbase, big.NewInt(1).Mul(big.NewInt(int64(usedGas)), msg.GasPrice()))

	log.Info("usedGas ", usedGas)
	log.Info("return data is " + hex.EncodeToString(ret))
	log.Info("contract address is ", addr.Str())

	// 打印合约中生成的日志
	evm.mStateDB.PrintLogs()

	// 这里还需要再进行其它错误的判断（余额不足的错误，前面已经返回）
	// 因为其它情况的错误，即使合约执行失败，也消耗了资源，调用者需要为此支付费用
	if vmerr != nil {
		return nil, vmerr
	}

	// 从状态机中获取数据变更和变更日志
	data, logs := evm.mStateDB.GetChangedData(evm.mStateDB.GetLastSnapshot())
	logs = append(logs, evm.mStateDB.GetReceiptLogs(addr, isCreate)...)
	receipt := &types.Receipt{Ty: types.ExecOk, KV: data, Logs: logs}

	return receipt, nil
}


func (evm *FakeEVM) GetChainConfig() *params.ChainConfig {
	// FIXME 这里先使用测试配置，之后根据代码逻辑再修改
	return &params.ChainConfig{
		ChainId: big.NewInt(1),
		HomesteadBlock: big.NewInt(0),
		DAOForkBlock: nil,
		DAOForkSupport: false,
		EIP150Block: big.NewInt(0),
		EIP155Block: big.NewInt(0),
		EIP158Block: big.NewInt(0),
		ByzantiumBlock: big.NewInt(0),
		ConstantinopleBlock: nil,
		Ethash: new(params.EthashConfig),
		Clique: nil}
}

func (evm *FakeEVM) GetMStateDB() *state.MemoryStateDB {
	return evm.mStateDB
}

// 目前的交易中，如果是coins交易，金额是放在payload的，但是合约不行，需要修改Transaction结构
func (evm *FakeEVM) GetMessage(tx *types.Transaction) (msg common.Message) {

	// 此处暂时不考虑消息发送签名的处理，chain33在mempool中对签名做了检查
	from := getCaller(tx)
	to := getReceiver(tx)

	// 注意Transaction中的payload内容同时包含转账金额和合约代码
	// payload[:8]为转账金额，payload[8:]为合约代码
	//amount := binary.BigEndian.Uint64(tx.Payload[:8])
	// FIXME 目前不支持对EVM合约的转账逻辑，等合约其它功能稳定后再考虑加入
	amount := int64(0)

	gasLimit := uint64(tx.Fee * TX_GAS_TIMES_FEE)
	// 合约的GasLimit即为调用者为本次合约调用准备支付的手续费
	msg = common.NewMessage(from, to, uint64(tx.Nonce), big.NewInt(amount), gasLimit, GasPrice, tx.Payload[BALANCE_SIZE:], false)
	return msg
}

func (evm *FakeEVM) GetVMConfig() *runtime.Config {
	return evm.vmCfg
}

// 从交易信息中获取交易发起人地址
func getCaller(tx *types.Transaction) common.Address {
	return common.StringToAddress(account.From(tx).String())
}

// 从交易信息中获取交易目标地址，在创建合约交易中，此地址为空
func getReceiver(tx *types.Transaction) *common.Address {
	if tx.To == "" {
		return nil
	}
	addr := common.StringToAddress(tx.To)
	return &addr
}

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(msg common.Message, height int64, time int64, coinbase common.Address, difficulty uint64) runtime.Context {

	return runtime.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     getHashFn,
		Origin:      msg.From(),
		Coinbase:    coinbase,
		BlockNumber: new(big.Int).SetInt64(height),
		Time:        new(big.Int).SetInt64(time),
		Difficulty:  new(big.Int).SetUint64(difficulty),
		GasLimit:    msg.GasLimit(),
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// 检查合约调用账户是否有充足的金额进行转账交易操作
func CanTransfer(db state.StateDB, addr common.Address, amount *big.Int) bool {
	if amount.Uint64() == 0 {
		return true
	}
	return db.CanTransfer(addr, amount)
}

// 在内存数据库中执行转账操作（只修改内存中的金额）
func Transfer(db state.StateDB, sender, recipient common.Address, amount *big.Int) {
	if amount.Uint64() == 0 {
		return
	}
	db.Transfer(sender, recipient, amount)
}

// 获取制定高度区块的哈希
func getHashFn(blockHeight uint64) common.Hash {
	// TODO 此处逻辑需要补充，获取指定数字高度区块对应的哈希，可参考evm.go/GetHashFn
	return common.Hash{}
}
