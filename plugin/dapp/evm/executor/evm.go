package executor

import (
	"bytes"
	"math/big"

	"os"

	"reflect"

	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/runtime"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/state"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	evmDebug = false

	// 本合约地址
	EvmAddress = address.ExecAddress(types.ExecName(evmtypes.ExecutorName))
)

var driverName = evmtypes.ExecutorName

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&EVMExecutor{}))
}

func Init(name string, sub []byte) {
	driverName = name
	drivers.Register(driverName, newEVMDriver, types.GetDappFork(driverName, "Enable"))
	EvmAddress = address.ExecAddress(types.ExecName(name))
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

func (evm *EVMExecutor) GetFuncMap() map[string]reflect.Method {
	ety := types.LoadExecutorType(driverName)
	return ety.GetExecFuncMap()
}

func (evm *EVMExecutor) GetDriverName() string {
	return evmtypes.ExecutorName
}

func (evm *EVMExecutor) Allow(tx *types.Transaction, index int) error {
	err := evm.DriverBase.Allow(tx, index)
	if err == nil {
		return nil
	}
	//增加新的规则:
	//主链: user.evm.xxx  执行 evm 合约
	//平行链: user.p.guodun.user.evm.xxx 执行 evm 合约
	exec := types.GetParaExec(tx.Execer)
	if evm.AllowIsUserDot2(exec) {
		return nil
	}
	return types.ErrNotAllow
}

func (evm *EVMExecutor) IsFriend(myexec, writekey []byte, othertx *types.Transaction) bool {
	if othertx == nil {
		return false
	}
	exec := types.GetParaExec(othertx.Execer)
	if exec == nil || len(bytes.TrimSpace(exec)) == 0 {
		return false
	}
	if bytes.HasPrefix(exec, evmtypes.UserPrefix) || bytes.Equal(exec, evmtypes.ExecerEvm) {
		if bytes.HasPrefix(writekey, []byte("mavl-evm-")) {
			return true
		}
	}
	return false
}

// 生成一个新的合约对象地址
func (evm *EVMExecutor) getNewAddr(txHash []byte) common.Address {
	return common.NewAddress(txHash)
}

func (evm *EVMExecutor) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

//获取运行状态名
func (evm *EVMExecutor) GetActionName(tx *types.Transaction) string {
	if bytes.Equal(tx.Execer, []byte(types.ExecName(evmtypes.ExecutorName))) {
		return types.ExecName(evmtypes.ExecutorName)
	}
	return tx.ActionName()
}

func (evm *EVMExecutor) GetMStateDB() *state.MemoryStateDB {
	return evm.mStateDB
}

func (evm *EVMExecutor) GetVMConfig() *runtime.Config {
	return evm.vmCfg
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
