package evm

import (
	"math/big"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/params"
	ctypes "gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/types"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/core"
)

var (
	GasPrice = big.NewInt(1)
)
var clog = log.New("module", "execs.evm")

type FakeEVM struct {
	drivers.DriverBase
	vmCfg *vm.Config
}

func init() {
	evm := NewFakeEVM()

	// TODO 注册的驱动高度需要更新为上线时的正确高度
	drivers.Register(evm.GetName(), evm, types.ForkV1)
}

func NewFakeEVM() *FakeEVM {
	fake := &FakeEVM{}
	fake.vmCfg = &vm.Config{}
	fake.SetChild(fake)
	return fake
}

func (evm *FakeEVM) GetName() string {
	return "evm"
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
	statedb := evm.GetStateDB()
	vmcfg := evm.GetVMConfig()

	// 获取当前区块的高度和时间
	height := evm.DriverBase.GetHeight()
	time := evm.DriverBase.GetBlockTime()

	//FIXME 需要获取coinbase，目前没有
	//FIXME 还有难度值，也需要获取  这两个信息都需要在执行区块时传进来
	coinbase := common.EmptyAddress()
	difficulty := uint64(10000)

	context := NewEVMContext(msg, height, time, coinbase, difficulty)

	// 创建EVM运行时对象
	runtime := vm.NewEVM(context, statedb, config, *vmcfg)

	isCreate := msg.To() == nil

	if isCreate {
		//ret, contractAddr, leftOverGas , err := runtime.Create(vm.AccountRef(msg.From()), tx.Payload, context.GasLimit, big.NewInt(0))
		runtime.Create(vm.AccountRef(msg.From()), tx.Payload, context.GasLimit, big.NewInt(0))
	}else{
		//ret, leftOverGas , err :=  runtime.Call(vm.AccountRef(msg.From()), *msg.To(), tx.Payload, context.GasLimit, big.NewInt(0))
		runtime.Call(vm.AccountRef(msg.From()), *msg.To(), tx.Payload, context.GasLimit, big.NewInt(0))
	}

	//if err != nil {
	//	return nil, err
	//}
	//
	//if failed {
	//	return nil, nil
	//}
	//
	//// 更新内存状态，计算root哈希
	//// Update the state with pending changes
	//statedb.Finalise(true)
	//
	//usedGas += gas
	//
	//// 组装结果
	//kvset := []*ctypes.KeyValue{
	//	&ctypes.KeyValue{[]byte("caller"), getCaller(tx).Bytes()},
	//	&ctypes.KeyValue{[]byte("constractResult"), ret},
	//	&ctypes.KeyValue{[]byte("ContractAddress"), []byte("xxx")},
	//}
	//if isCreate {
	//	addr := crypto.CreateAddress(runtime.Context.Origin, uint64(tx.Nonce))
	//	kvset = append(kvset, &ctypes.KeyValue{[]byte("ContractAddress"), addr.Bytes()})
	//} else {
	//	kvset = append(kvset, &ctypes.KeyValue{[]byte("receiver"), getReceiver(tx).Bytes()})
	//}
	//
	//receipt := &ctypes.Receipt{ctypes.ExecOk, kvset, nil}
	return nil, nil

}

func (evm *FakeEVM) GetChainConfig() *params.ChainConfig {
	// FIXME 这里先使用测试配置，之后根据代码逻辑再修改
	return params.TestChainConfig
}

func (evm *FakeEVM) GetStateDB() *core.MemoryStateDB {
	inst := core.NewMemoryStateDB()
	statedb := evm.DriverBase.GetStateDB()
	inst.StateDB = statedb
	localdb := evm.DriverBase.GetLocalDB()
	inst.LocalDB = &localdb
	return &inst
}

//FIXME 目前的交易中，如果是coins交易，金额是放在payload的，但是合约不行，需要修改Transaction结构
func (evm *FakeEVM) GetMessage(tx *types.Transaction) (msg ctypes.Message) {

	// 此处暂时不考虑消息发送这签名的处理，chain33在mempool中对签名做了检查
	from := getCaller(tx)
	to := getReceiver(tx)

	// 注意Transaction中的payload内容同时包含转账金额和合约代码
	// payload[:8]为转账金额，payload[8:]为合约代码
	//amount := binary.LittleEndian.Uint64(tx.Payload[:8])
	// FIXME 目前不支持对EVM合约的转账逻辑，等合约其它功能稳定后再考虑加入
	amount := int64(0)

	// 合约的GasLimit即为调用者为本次合约调用准备支付的手续费
	msg = ctypes.NewMessage(from, to, uint64(tx.Nonce), big.NewInt(amount), uint64(tx.Fee), GasPrice, tx.Payload[8:], false)
	return msg
}

func (evm *FakeEVM) GetVMConfig() *vm.Config {
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
func NewEVMContext(msg ctypes.Message, height int64, time int64, coinbase common.Address, difficulty uint64) vm.Context {

	return vm.Context{
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
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	if amount.Uint64() ==0 {
		return true
	}
	return db.CanTransfer(addr,amount)
}

// 在内存数据库中执行转账操作（只修改内存中的金额）
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	if amount.Uint64() ==0{
		return
	}
	db.Transfer(sender,recipient,amount)
}

// 获取制定高度区块的哈希
func getHashFn(blockHeight uint64) common.Hash {
	// TODO 此处逻辑需要补充，获取指定数字高度区块对应的哈希，可参考evm.go/GetHashFn
	return common.Hash{}
}
