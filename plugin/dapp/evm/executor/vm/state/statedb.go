package state

import (
	"fmt"
	"strings"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/model"
	"gitlab.33.cn/chain33/chain33/types"
)

// 内存状态数据库，保存在区块操作时内部的数据变更操作
// 本数据库不会直接写文件，只会暂存变更记录
// 在区块打包完成后，这里的缓存变更数据会被清空（通过区块打包分别写入blockchain和statedb数据库）
// 在交易执行过程中，本数据库会暂存并变更，在交易执行结束后，会返回变更的数据集，返回给blockchain
// 执行器的Exec阶段会返回：交易收据、合约账户（包含合约地址、合约代码、合约存储信息）
// 执行器的ExecLocal阶段会返回：合约创建人和合约的关联信息
type MemoryStateDB struct {
	// 状态DB，从执行器框架传入
	StateDB db.KV

	// 本地DB，从执行器框架传入
	LocalDB db.KVDB

	// Coins账户操作对象，从执行器框架传入
	CoinsAccount *account.DB

	// 缓存账户对象
	accounts map[string]*ContractAccount

	// 合约执行过程中退回的资金
	refund uint64

	// 存储makeLogN指令对应的日志数据
	logs    map[common.Hash][]*model.ContractLog
	logSize uint

	// 版本号，用于标识数据变更版本
	snapshots  []*Snapshot
	currentVer *Snapshot
	versionId  int

	// 存储sha3指令对应的数据，仅用于debug日志
	preimages map[common.Hash][]byte

	// 当前临时交易哈希和交易序号
	txHash  common.Hash
	txIndex int

	// 当前区块高度
	blockHeight int64

	// 用户保存合约账户的状态数据或合约代码数据有没有发生变更
	stateDirty map[string]interface{}
	dataDirty  map[string]interface{}
}

// 基于执行器框架的三个DB构建内存状态机对象
// 此对象的生命周期对应一个区块，在同一个区块内的多个交易执行时共享同一个DB对象
// 开始执行下一个区块时（执行器框架调用setEnv设置的区块高度发生变更时），会重新创建此DB对象
func NewMemoryStateDB(StateDB db.KV, LocalDB db.KVDB, CoinsAccount *account.DB, blockHeight int64) *MemoryStateDB {
	mdb := &MemoryStateDB{
		StateDB:      StateDB,
		LocalDB:      LocalDB,
		CoinsAccount: CoinsAccount,
		accounts:     make(map[string]*ContractAccount),
		logs:         make(map[common.Hash][]*model.ContractLog),
		preimages:    make(map[common.Hash][]byte),
		stateDirty:   make(map[string]interface{}),
		dataDirty:    make(map[string]interface{}),
		blockHeight:  blockHeight,
		refund:       0,
		txIndex:      0,
	}
	return mdb
}

// 每一个交易执行之前调用此方法，设置此交易的上下文信息
// 目前的上下文中包含交易哈希以及交易在区块中的序号
func (self *MemoryStateDB) Prepare(txHash common.Hash, txIndex int) {
	self.txHash = txHash
	self.txIndex = txIndex
}

// 创建一个新的合约账户对象
func (self *MemoryStateDB) CreateAccount(addr, creator string, execName, alias string) {
	acc := self.GetAccount(addr)
	if acc == nil {
		// 这种情况下为新增合约账户
		acc := NewContractAccount(addr, self)
		acc.SetCreator(creator)
		acc.SetExecName(execName)
		acc.SetAliasName(alias)
		self.accounts[addr] = acc
		self.addChange(createAccountChange{baseChange: baseChange{}, account: addr})
	}
}

func (self *MemoryStateDB) addChange(entry DataChange) {
	if self.currentVer != nil {
		self.currentVer.append(entry)
	}
}

// 从外部账户地址扣钱（钱其实是打到合约账户中的）
func (self *MemoryStateDB) SubBalance(addr, caddr string, value uint64) {
	res := self.Transfer(addr, caddr, value)
	log15.Debug("transfer result", "from", addr, "to", caddr, "amount", value, "result", res)
}

// 向外部账户地址打钱（钱其实是外部账户之前打到合约账户中的）
func (self *MemoryStateDB) AddBalance(addr, caddr string, value uint64) {
	res := self.Transfer(caddr, addr, value)
	log15.Debug("transfer result", "from", addr, "to", caddr, "amount", value, "result", res)
}

// 这里需要区分对待，如果是合约账户，则查看合约账户所有者地址在此合约下的余额；
// 如果是外部账户，则直接返回外部账户的余额
func (self *MemoryStateDB) GetBalance(addr string) uint64 {
	if self.CoinsAccount == nil {
		return 0
	}
	isExec := self.Exist(addr)
	var ac *types.Account
	if isExec {
		contract := self.GetAccount(addr)
		if contract == nil {
			return 0
		}
		creator := contract.GetCreator()
		if len(creator) == 0 {
			return 0
		}
		ac = self.CoinsAccount.LoadExecAccount(creator, addr)
	} else {
		ac = self.CoinsAccount.LoadAccount(addr)
	}
	if ac != nil {
		return uint64(ac.Balance)
	}
	return 0
}

// 目前chain33中没有保留账户的nonce信息，这里临时添加到合约账户中；
// 所以，目前只有合约对象有nonce值
func (self *MemoryStateDB) GetNonce(addr string) uint64 {
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.GetNonce()
	}
	return 0
}

func (self *MemoryStateDB) SetNonce(addr string, nonce uint64) {
	acc := self.GetAccount(addr)
	if acc != nil {
		acc.SetNonce(nonce)
	}
}

func (self *MemoryStateDB) GetCodeHash(addr string) common.Hash {
	acc := self.GetAccount(addr)
	if acc != nil {
		return common.BytesToHash(acc.Data.GetCodeHash())
	}
	return common.Hash{}
}

func (self *MemoryStateDB) GetCode(addr string) []byte {
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.Data.GetCode()
	}
	return nil
}

func (self *MemoryStateDB) SetCode(addr string, code []byte) {
	acc := self.GetAccount(addr)
	if acc != nil {
		self.dataDirty[addr] = true
		acc.SetCode(code)
	}
}

// 获取合约代码自身的大小
// 对应 EXTCODESIZE 操作码
func (self *MemoryStateDB) GetCodeSize(addr string) int {
	code := self.GetCode(addr)
	if code != nil {
		return len(code)
	}
	return 0
}

// 合约自杀或SSTORE指令时，返还Gas
func (self *MemoryStateDB) AddRefund(gas uint64) {
	self.addChange(refundChange{baseChange: baseChange{}, prev: self.refund})
	self.refund += gas
}

func (self *MemoryStateDB) GetRefund() uint64 {
	return self.refund
}

// 从缓存中获取或加载合约账户
func (self *MemoryStateDB) GetAccount(addr string) *ContractAccount {
	if acc, ok := self.accounts[addr]; ok {
		return acc
	} else {
		// 需要加载合约对象，根据是否存在合约代码来判断是否有合约对象
		contract := NewContractAccount(addr, self)
		contract.LoadContract(self.StateDB)
		if contract.Empty() {
			return nil
		}
		self.accounts[addr] = contract
		return contract
	}
}

// SLOAD 指令加载合约状态数据
func (self *MemoryStateDB) GetState(addr string, key common.Hash) common.Hash {
	// 先从合约缓存中获取
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.GetState(key)
	}
	return common.Hash{}
}

// SSTORE 指令修改合约状态数据
func (self *MemoryStateDB) SetState(addr string, key common.Hash, value common.Hash) {
	acc := self.GetAccount(addr)
	if acc != nil {
		acc.SetState(key, value)
		// 新的分叉中状态数据变更不需要单独进行标识
		if !types.IsDappFork(self.blockHeight, "evm", "ForkEVMState") {
			self.stateDirty[addr] = true
		}
	}
}

// 转换合约状态数据存储
func (self *MemoryStateDB) TransferStateData(addr string) {
	acc := self.GetAccount(addr)
	if acc != nil {
		acc.TransferState()
	}
}

// 表示合约地址的状态数据发生了变更，需要进行更新
func (self *MemoryStateDB) UpdateState(addr string) {
	self.stateDirty[addr] = true
}

// SELFDESTRUCT 合约对象自杀
// 合约自杀后，合约对象依然存在，只是无法被调用，也无法恢复
func (self *MemoryStateDB) Suicide(addr string) bool {
	acc := self.GetAccount(addr)
	if acc != nil {
		self.addChange(suicideChange{
			baseChange: baseChange{},
			account:    addr,
			prev:       acc.State.GetSuicided(),
		})
		self.stateDirty[addr] = true
		return acc.Suicide()
	}
	return false
}

// 判断此合约对象是否已经自杀
// 自杀的合约对象是不允许调用的
func (self *MemoryStateDB) HasSuicided(addr string) bool {
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.HasSuicided()
	}
	return false
}

// 判断合约对象是否存在
func (self *MemoryStateDB) Exist(addr string) bool {
	return self.GetAccount(addr) != nil
}

// 判断合约对象是否为空
func (self *MemoryStateDB) Empty(addr string) bool {
	acc := self.GetAccount(addr)

	// 如果包含合约代码，则不为空
	if acc != nil && !acc.Empty() {
		return false
	}

	// 账户有余额，也不为空
	if self.GetBalance(addr) != 0 {
		return false
	}
	return true
}

// 将数据状态回滚到指定快照版本（中间的版本数据将会被删除）
func (self *MemoryStateDB) RevertToSnapshot(version int) {
	if version >= len(self.snapshots) {
		return
	}

	ver := self.snapshots[version]

	// 如果版本号不对，回滚失败
	if ver == nil || ver.id != version {
		log15.Crit(fmt.Errorf("Snapshot id %v cannot be reverted", version).Error())
		return
	}

	// 从最近版本开始回滚
	for index := len(self.snapshots) - 1; index >= version; index-- {
		self.snapshots[index].revert()
	}

	// 只保留回滚版本之前的版本数据
	self.snapshots = self.snapshots[:version]
	self.versionId = version
	if version == 0 {
		self.currentVer = nil
	} else {
		self.currentVer = self.snapshots[version-1]
	}

}

// 对当前的数据状态打快照，并生成快照版本号，方便后面回滚数据
func (self *MemoryStateDB) Snapshot() int {
	id := self.versionId
	self.versionId++
	self.currentVer = &Snapshot{id: id, statedb: self}
	self.snapshots = append(self.snapshots, self.currentVer)
	return id
}

// 获取最后一次成功的快照版本号
func (self *MemoryStateDB) GetLastSnapshot() *Snapshot {
	if self.versionId == 0 {
		return nil
	}
	return self.snapshots[self.versionId-1]
}

// 获取合约对象的变更日志
func (self *MemoryStateDB) GetReceiptLogs(addr string) (logs []*types.ReceiptLog) {
	acc := self.GetAccount(addr)
	if acc != nil {
		if self.stateDirty[addr] != nil {
			stateLog := acc.BuildStateLog()
			if stateLog != nil {
				logs = append(logs, stateLog)
			}
		}

		if self.dataDirty[addr] != nil {
			logs = append(logs, acc.BuildDataLog())
		}
		return
	}
	return
}

// 获取本次操作所引起的状态数据变更
// 因为目前执行器每次执行都是一个新的MemoryStateDB，所以，所有的快照都是从0开始的，
// 这里获取的应该是从0到目前快照的所有变更；
// 另外，因为合约内部会调用其它合约，也会产生数据变更，所以这里返回的数据，不止是一个合约的数据。
func (self *MemoryStateDB) GetChangedData(version int) (kvSet []*types.KeyValue, logs []*types.ReceiptLog) {
	if version < 0 {
		return
	}

	for _, snapshot := range self.snapshots {
		kv, log := snapshot.getData()
		if kv != nil {
			kvSet = append(kvSet, kv...)
		}

		if log != nil {
			logs = append(logs, log...)
		}
	}
	return
}

// 借助coins执行器进行转账相关操作
func (self *MemoryStateDB) CanTransfer(sender, recipient string, amount uint64) bool {

	log15.Debug("check CanTransfer", "sender", sender, "recipient", recipient, "amount", amount)

	tType, errInfo := self.checkTransfer(sender, recipient, amount)

	if errInfo != nil {
		log15.Error("check transfer error", "sender", sender, "recipient", recipient, "amount", amount, "err info", errInfo)
		return false
	}

	value := int64(amount)
	if value < 0 {
		return false
	}

	switch tType {
	case NoNeed:
		return true
	case ToExec:
		accFrom := self.CoinsAccount.LoadExecAccount(sender, recipient)
		b := accFrom.GetBalance() - value
		if b < 0 {
			log15.Error("check transfer error", "error info", types.ErrNoBalance)
			return false
		}
		return true
	case FromExec:
		return self.checkExecAccount(sender, value)
	default:
		return false
	}
}

func (self *MemoryStateDB) checkExecAccount(addr string, value int64) bool {
	var err error
	defer func() {
		if err != nil {
			log15.Error("checkExecAccount error", "error info", err)
		}
	}()
	// 如果是合约地址，则需要判断创建者在本合约中的余额是否充足
	if !types.CheckAmount(value) {
		err = types.ErrAmount
		return false
	}
	contract := self.GetAccount(addr)
	if contract == nil {
		err = model.ErrAddrNotExists
		return false
	}
	creator := contract.GetCreator()
	if len(creator) == 0 {
		err = model.ErrNoCreator
		return false
	}

	accFrom := self.CoinsAccount.LoadExecAccount(contract.GetCreator(), addr)
	b := accFrom.GetBalance() - value
	if b < 0 {
		err = types.ErrNoBalance
		return false
	}
	return true
}

type TransferType int

const (
	_ TransferType = iota
	NoNeed
	ToExec
	FromExec
	Error
)

func (self *MemoryStateDB) checkTransfer(sender, recipient string, amount uint64) (tType TransferType, err error) {
	if amount == 0 {
		return NoNeed, nil
	}
	if self.CoinsAccount == nil {
		log15.Error("no coinsaccount exists", "sender", sender, "recipient", recipient, "amount", amount)
		return Error, model.NoCoinsAccount
	}

	// 首先需要检查转账双方的信息，是属于合约账户还是外部账户
	execSender := self.Exist(sender)
	execRecipient := self.Exist(recipient)

	if execRecipient && execSender {
		// 双方均为合约账户，不支持
		err = model.ErrTransferBetweenContracts
		tType = Error
	} else if execSender {
		// 从合约账户到外部账户转账 （这里调用外部账户从合约账户取钱接口）
		tType = FromExec
		err = nil
	} else if execRecipient {
		// 从外部账户到合约账户转账
		tType = ToExec
		err = nil
	} else {
		// 双方都是外部账户，不支持
		err = model.ErrTransferBetweenEOA
		tType = Error
	}

	return tType, err
}

// 借助coins执行器进行转账相关操作
// 只支持 合约账户到合约账户，其它情况不支持
func (self *MemoryStateDB) Transfer(sender, recipient string, amount uint64) bool {
	log15.Debug("transfer from contract to external(contract)", "sender", sender, "recipient", recipient, "amount", amount)

	tType, errInfo := self.checkTransfer(sender, recipient, amount)

	if errInfo != nil {
		log15.Error("transfer error", "sender", sender, "recipient", recipient, "amount", amount, "err info", errInfo)
		return false
	}

	var (
		ret *types.Receipt
		err error
	)

	value := int64(amount)
	if value < 0 {
		return false
	}

	switch tType {
	case NoNeed:
		return true
	case ToExec:
		ret, err = self.transfer2Contract(sender, recipient, value)
	case FromExec:
		ret, err = self.transfer2External(sender, recipient, value)
	default:
		return false
	}

	// 这种情况下转账失败并不进行处理，也不会从sender账户扣款，打印日志即可
	if err != nil {
		log15.Error("transfer error", "sender", sender, "recipient", recipient, "amount", amount, "err info", err)
		return false
	} else {
		if ret != nil {
			self.addChange(transferChange{
				baseChange: baseChange{},
				amount:     value,
				data:       ret.KV,
				logs:       ret.Logs,
			})
		}
		return true
	}
}

// 因为chain33的限制，在执行器中转账只能在以下几个方向进行：
// A账户的X合约 <-> B账户的X合约；
// 其它情况不支持，所以要想实现EVM合约与账户之间的转账需要经过中转处理，比如A要向B创建的X合约转账，则执行以下流程：
// A -> A:X -> B:X；  (其中第一步需要外部手工执行)
// 本方法封装第二步转账逻辑;
func (self *MemoryStateDB) transfer2Contract(sender, recipient string, amount int64) (ret *types.Receipt, err error) {
	// 首先获取合约的创建者信息
	contract := self.GetAccount(recipient)
	if contract == nil {
		return nil, model.ErrAddrNotExists
	}
	creator := contract.GetCreator()
	if len(creator) == 0 {
		return nil, model.ErrNoCreator
	}
	execAddr := recipient

	// 从自己的合约账户到创建者的合约账户
	// 有可能是外部账户调用自己创建的合约，这种情况下这一步可以省略
	ret = &types.Receipt{}
	if strings.Compare(sender, creator) != 0 {
		rs, err := self.CoinsAccount.ExecTransfer(sender, creator, execAddr, amount)
		if err != nil {
			return nil, err
		}

		ret.KV = append(ret.KV, rs.KV...)
		ret.Logs = append(ret.Logs, rs.Logs...)
	}

	return ret, nil
}

// chain33转账限制请参考方法 Transfer2Contract ；
// 本方法封装从合约账户到外部账户的转账逻辑；
func (self *MemoryStateDB) transfer2External(sender, recipient string, amount int64) (ret *types.Receipt, err error) {
	// 首先获取合约的创建者信息
	contract := self.GetAccount(sender)
	if contract == nil {
		return nil, model.ErrAddrNotExists
	}
	creator := contract.GetCreator()
	if len(creator) == 0 {
		return nil, model.ErrNoCreator
	}

	execAddr := sender

	// 第一步先从创建者的合约账户到接受者的合约账户
	// 如果是自己调用自己创建的合约，这一步也可以省略
	if strings.Compare(creator, recipient) != 0 {
		ret, err = self.CoinsAccount.ExecTransfer(creator, recipient, execAddr, amount)
		if err != nil {
			return nil, err
		}
	}

	// 第二步再从接收者的合约账户取款到接受者账户
	// 本操作不允许，需要外部操作coins账户
	//rs, err := self.CoinsAccount.TransferWithdraw(recipient.String(), sender.String(), amount)
	//if err != nil {
	//	return nil, err
	//}
	//
	//ret = self.mergeResult(ret, rs)

	return ret, nil
}

func (self *MemoryStateDB) mergeResult(one, two *types.Receipt) (ret *types.Receipt) {
	ret = one
	if ret == nil {
		ret = two
	} else if two != nil {
		ret.KV = append(ret.KV, two.KV...)
		ret.Logs = append(ret.Logs, two.Logs...)
	}
	return
}

// LOG0-4 指令对应的具体操作
// 生成对应的日志信息，目前这些生成的日志信息会在合约执行后打印到日志文件中
func (self *MemoryStateDB) AddLog(log *model.ContractLog) {
	self.addChange(addLogChange{txhash: self.txHash})
	log.TxHash = self.txHash
	log.Index = int(self.logSize)
	self.logs[self.txHash] = append(self.logs[self.txHash], log)
	self.logSize++
}

// 存储sha3指令对应的数据
func (self *MemoryStateDB) AddPreimage(hash common.Hash, data []byte) {
	// 目前只用于打印日志
	if _, ok := self.preimages[hash]; !ok {
		self.addChange(addPreimageChange{hash: hash})
		pi := make([]byte, len(data))
		copy(pi, data)
		self.preimages[hash] = pi
	}
}

// 本合约执行完毕之后打印合约生成的日志（如果有）
// 这里不保证当前区块可以打包成功，只是在执行区块中的交易时，如果交易执行成功，就会打印合约日志
func (self *MemoryStateDB) PrintLogs() {
	items := self.logs[self.txHash]
	for _, item := range items {
		item.PrintLog()
	}
}

// 打印本区块内生成的preimages日志
func (self *MemoryStateDB) WritePreimages(number int64) {
	for k, v := range self.preimages {
		log15.Debug("Contract preimages ", "key:", k.Str(), "value:", common.Bytes2Hex(v), "block height:", number)
	}
}

// 测试用，清空版本数据
func (self *MemoryStateDB) ResetDatas() {
	self.currentVer = nil
	self.snapshots = self.snapshots[:0]
}
