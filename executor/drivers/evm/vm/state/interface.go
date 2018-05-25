package state

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
)

// 状态数据库封装，面向EVM业务执行逻辑；
// 生命周期为一个区块，在同一个区块内多个交易执行时使用的是同一个StateDB实例；
// StateDB包含区块的状态和交易的状态（当前上下文），所以不支持并发操作，区块内的多个交易只能按顺序单线程执行；
// StateDB除了查询状态数据，还会保留在交易执行时对数据的变更信息，每个交易完成之后会返回变更影响的数据给执行器；
type StateDB interface {
	// 创建新的合约对象
	CreateAccount(common.Address, common.Address, string)

	// 从从指定地址扣除金额
	SubBalance(common.Address,common.Address,uint64)
	// 向指定地址增加金额
	AddBalance(common.Address,common.Address, uint64)
	// 获取指定地址的余额
	GetBalance(common.Address) uint64

	// 获取nonce值（只有合约对象有，外部对象为0）
	GetNonce(common.Address) uint64
	// 设置nonce值（只有合约对象有，外部对象为0）
	SetNonce(common.Address, uint64)

	// 获取指定地址合约的代码哈希
	GetCodeHash(common.Address) common.Hash
	// 获取指定地址合约代码
	GetCode(common.Address) []byte
	// 设置指定地址合约代码
	SetCode(common.Address, []byte)
	// 获取指定地址合约代码大小
	GetCodeSize(common.Address) int

	// 合约Gas奖励回馈
	AddRefund(uint64)
	// 获取合约Gas奖励
	GetRefund() uint64

	// 获取合约状态数据
	GetState(common.Address, common.Hash) common.Hash
	// 设置合约状态数据
	SetState(common.Address, common.Hash, common.Hash)

	// 合约自销毁
	Suicide(common.Address) bool
	// 合约是否已经销毁
	HasSuicided(common.Address) bool

	// 判断一个合约地址是否存在（已经销毁的合约地址对象依然存在）
	Exist(common.Address) bool
	// 判断一个合约地址是否为空（不包含任何代码、也没有余额的合约为空）
	Empty(common.Address) bool

	// 回滚到制定版本（从当前版本到回滚版本之间的数据变更全部撤销）
	RevertToSnapshot(int)
	// 生成一个新版本号（递增操作）
	Snapshot() int

	// 添加新的日志信息
	AddLog(*model.ContractLog)
	// 添加sha3记录
	AddPreimage(common.Hash, []byte)

	// 当前账户余额是否足够转账
	CanTransfer(sender, recipient common.Address, amount uint64) bool
	// 转账交易
	Transfer(sender, recipient common.Address, amount uint64) bool
}
