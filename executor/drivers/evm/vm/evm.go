package vm

import (
	"math/big"
	"sync/atomic"
	"time"

	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/crypto"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/params"
)

type (
	// 检查制定账户是否有足够的金额进行转账
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool

	// 执行转账逻辑
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int)

	// 获取制定高度区块的哈希
	// 给 BLOCKHASH 指令使用
	GetHashFunc func(uint64) common.Hash
)

// 依据合约地址判断是否为预编译合约，如果不是，则全部通过解释器解释执行
func run(evm *EVM, contract *Contract, input []byte) ([]byte, error) {
	if contract.CodeAddr != nil {
		// 预编译合约以拜占庭分支为初始版本，后继如有分叉，需要在此处理
		precompiles := PrecompiledContractsByzantium
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	return evm.interpreter.Run(contract, input)
}

// EVM操作辅助上下文
// 外部在构造EVM实例时传入，EVM实例构造完成后不允许修改
type Context struct {

	// 下面这三个方法的说明，请查看方法类型的定义
	CanTransfer CanTransferFunc
	Transfer    TransferFunc
	GetHash     GetHashFunc

	// ORIGIN 指令返回数据， 合约调用者地址
	Origin common.Address
	// GASPRICE 指令返回数据
	GasPrice *big.Int

	// COINBASE 指令， 区块打包者地址
	Coinbase common.Address
	// GASLIMIT 指令，当前交易的GasLimit
	GasLimit uint64

	// NUMBER 指令，当前区块高度
	BlockNumber *big.Int
	// TIME 指令， 当前区块打包时间
	Time *big.Int
	// DIFFICULTY 指令，当前区块难度
	Difficulty *big.Int
}

// EVM 结构对象及其提供的操作方法，用于进行满足以太坊EVM黄皮书规范定义的智能合约代码的创建和执行
// 合约执行过程中会修改外部状态数据（数据操作通过外部注入）
// 在合约代码执行过程中发生的任何错误，都将会导致对状态数据的修改被回滚，并且依然消耗掉剩余的Gas
// 此对象为每个交易创建一个实例，其操作非线程安全
type EVM struct {
	// 链相关的一些辅助属性和操作方法
	Context
	// 状态数据操作入口
	StateDB StateDB
	// 当前调用深度
	depth int

	// 链的全局配置信息
	chainConfig *params.ChainConfig
	// 链的配置规则信息
	chainRules params.Rules

	// 虚拟机配置属性信息
	vmConfig Config

	// EVM指令解释器，生命周期同EVM
	interpreter *Interpreter

	// EVM执行流程结束标志
	// 不允许手工设置
	abort int32

	// 此属性用于临时存储计算出来的Gas消耗值
	// 在指令执行时，会调用指令的gasCost方法，计算出此指令需要消耗的Gas，并存放在此临时属性中
	// 然后在执行opCall时，从此属性获取消耗的Gas值
	callGasTemp uint64

	// 支持的最长合约代码大小
	maxCodeSize int
}

// 创建一个新的EVM实例对象
// 在同一个节点中，一个EVM实例对象只服务于一个交易执行的生命周期

func NewEVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *EVM {
	evm := &EVM{
		Context:     ctx,
		StateDB:     statedb,
		vmConfig:    vmConfig,
		chainConfig: chainConfig,
		chainRules:  chainConfig.Rules(ctx.BlockNumber),
		maxCodeSize: params.MaxCodeSize,
	}

	evm.interpreter = NewInterpreter(evm, vmConfig)
	return evm
}

// 调用此操作会在任意时刻取消此EVM的解释运行逻辑，支持重复调用
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// 设置合约代码的最大支持长度
func (evm *EVM) SetMaxCodeSize(maxCodeSize int) {
	if maxCodeSize < 1 || maxCodeSize > params.MaxCodeSize {
		return
	}

	evm.maxCodeSize = maxCodeSize
}

// 封装合约的各种调用逻辑中通用的预检查逻辑
func (evm *EVM) preCheck(caller ContractRef, gas uint64, value *big.Int) (pass bool, ret []byte, leftOverGas uint64, err error) {
	// 检查调用深度是否合法
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return false, nil, gas, nil
	}

	// 允许递归，但深度不合法
	if evm.depth > int(params.CallCreateDepth) {
		return false, nil, gas, ErrDepth
	}

	// 如有转账，检查余额是否充足
	if value != nil {
		if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
			return false, nil, gas, ErrInsufficientBalance
		}
	}

	// 检查通过情况下，后面三个返回参数无意义
	return true, nil, gas, nil
}

// 此方法提供合约外部调用入口
// 根据合约地址调用已经存在的合约，input为合约调用参数
// 合约调用逻辑支持在合约调用的同时进行向合约转账的操作（FIXME 待完善）
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	pass := false
	pass, ret, leftOverGas, err = evm.preCheck(caller, gas, value)
	if !pass {
		return ret, leftOverGas, err
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)

	if !evm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsByzantium

		// 合约地址在自定义合约和预编译合约中都不存在，说明为无效调用
		if precompiles[addr] == nil {
			return nil, gas, nil
		}

		// 否则，为预编译合约，创建一个新的账号
		evm.StateDB.CreateAccount(addr)
	}

	// 向合约地址转账
	evm.Transfer(evm.StateDB, caller.Address(), to.Address(), value)

	// 创建新的合约对象，包含双方地址以及合约代码，可用Gas信息
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	start := time.Now()

	// 调试模式下启用跟踪
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)

		defer func() {
			evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}

	ret, err = run(evm, contract, input)

	// 当合约调用出错时，操作将会回滚（对数据的变更操作会被恢复），并且会消耗掉所有的gas
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// 合约内部调用合约的入口
// 执行逻辑同Call方法，但是有以下几点不同：
// 在创建合约对象时，合约对象的上下文地址（合约对象的self属性）被设置为caller的地址
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	pass := false
	pass, ret, leftOverGas, err = evm.preCheck(caller, gas, value)
	if !pass {
		return ret, leftOverGas, err
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// 创建合约对象时，讲调用者和被调用者地址均设置为外部账户地址
	contract := NewContract(caller, to, value, gas)
	// 正常从合约地址加载合约代码
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// 合约内部调用合约的入口
// 不支持向合约转账
// 和CallCode不同的是，它会吧合约的外部调用地址设置成caller的caller
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	pass := false
	pass, ret, leftOverGas, err = evm.preCheck(caller, gas, nil)
	if !pass {
		return ret, leftOverGas, err
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// 同外部合约的创建和修改逻辑，在每次调用时，需要创建并初始化一个新的合约内存对象
	// 需要注意，这里不同的是，需要设置合约的委托调用模式（会进行一些属性设置）
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// 其它逻辑同StaticCall
	ret, err = run(evm, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// 合约内部调用合约的入口
// 不支持向合约转账
// 在合约逻辑中，可以指定其它的合约地址以及输入参数进行合约调用，但是，这种情况下禁止修改MemoryStateDB中的任何数据，否则执行会出错
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	pass := false
	pass, ret, leftOverGas, err = evm.preCheck(caller, gas, nil)
	if !pass {
		return ret, leftOverGas, err
	}

	// 如果指令解释器没有设置成只读，需要在这里强制设置，并在本操作结束后恢复
	// 在解释器执行涉及到数据修改指令时，会检查此属性，从而控制不允许修改数据
	if !evm.interpreter.readOnly {
		evm.interpreter.readOnly = true
		defer func() { evm.interpreter.readOnly = false }()
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)

	// 同外部合约的创建和修改逻辑，在每次调用时，需要创建并初始化一个新的合约内存对象
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// 执行合约指令时如果出错，需要进行回滚，并且扣除剩余的Gas
	ret, err = run(evm, contract, input)
	if err != nil {
		// 合约执行出错时进行回滚
		// 注意，虽然内部调用合约不允许变更数据，但是可以进行生成日志等其它操作，这种情况下也是需要回滚的
		evm.StateDB.RevertToSnapshot(snapshot)

		// 如果操作消耗了资源，即使失败，也扣除剩余的Gas
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// 此方法提供合约外部创建入口
// 使用传入的部署代码创建新的合约
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {

	// 检查合约代码递归调用深度，防止无限递归的情况出现，即使Gas充足，也不允许
	if evm.depth > int(params.CallCreateDepth) {
		return nil, common.EmptyAddress(), gas, ErrDepth
	}

	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, common.EmptyAddress(), gas, ErrInsufficientBalance
	}

	// 使用随机生成的地址作为合约地址（这个可以保证每次创建的合约地址不会重复，不存在冲突的情况）
	contractAddr = *crypto.RandomAddress()

	if !evm.StateDB.Empty(contractAddr) {
		return nil, common.EmptyAddress(), 0, ErrContractAddressCollision
	}

	// 使用生成的地址创建一个新的账户对象（会同时创建coins账户和合约账户）
	snapshot := evm.StateDB.Snapshot()
	evm.StateDB.CreateAccount(contractAddr)

	// 向从创建者向合约账户转账
	evm.Transfer(evm.StateDB, caller.Address(), contractAddr, value)

	// 创建新的合约对象，包含双方地址以及合约代码，可用Gas信息
	contract := NewContract(caller, AccountRef(contractAddr), value, gas)
	contract.SetCallCode(&contractAddr, common.BytesToHash(code), code)

	// 如果EVM配置中不允许递归调用，但是深度不为0（说明是通过递归调进来的），出错
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, contractAddr, gas, nil
	}

	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.Address(), contractAddr, true, code, gas, value)
	}
	start := time.Now()

	// 通过预编译指令和解释器执行合约
	ret, err = run(evm, contract, nil)

	// 检查部署后的合约代码大小是否超限
	maxCodeSizeExceeded := len(ret) > evm.maxCodeSize

	// 如果执行成功，计算存储合约代码需要花费的Gas
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			evm.StateDB.SetCode(contractAddr, ret)
		} else {
			// 如果Gas不足，返回这个错误，让外部程序处理
			err = ErrCodeStoreOutOfGas
		}
	}

	// 如果合约代码超大，或者出现除Gas不足外的其它错误情况
	// 则回滚本次合约创建操作
	if maxCodeSizeExceeded || (err != nil && err != ErrCodeStoreOutOfGas) {
		evm.StateDB.RevertToSnapshot(snapshot)

		// 如果之前步骤出错，且没有回滚过，则扣除Gas
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}

	// 如果前面的步骤都没有问题，单纯只是合约大小超大，则设置错误为合约代码超限，让外部程序处理
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}

	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}

	return ret, contractAddr, contract.Gas, err
}

// TODO 返回区块链的整体配置，供合约执行逻辑使用（后续需要精简属性）
func (evm *EVM) ChainConfig() *params.ChainConfig { return evm.chainConfig }
