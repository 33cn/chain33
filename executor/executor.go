package executor

//store package store the world - state data
import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/address"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"

	// register drivers
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var elog = log.New("module", "execs")
var coinsAccount = account.NewCoinsAccount()

const (
	FlagInit        = int64(0)
	FlagFromZero    = int64(1)
	FlagNotFromZero = int64(2)
)

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Executor struct {
	client         queue.Client
	qclient        client.QueueProtocolAPI
	enableStat     bool
	enableMVCC     bool
	enableStatFlag int64
	flagMVCC       int64
	alias          map[string]string
}

func execInit(sub map[string][]byte) {
	pluginmgr.InitExec(sub)
}

var runonce sync.Once

func New(cfg *types.Exec, sub map[string][]byte) *Executor {
	// init executor
	runonce.Do(func() {
		execInit(sub)
	})
	//设置区块链的MinFee，低于Mempool和Wallet设置的MinFee
	//在cfg.MinExecFee == 0 的情况下，必须 cfg.IsFree == true 才会起效果
	if cfg.MinExecFee == 0 && cfg.IsFree {
		elog.Warn("set executor to free fee")
		types.SetMinFee(0)
	}
	if cfg.MinExecFee > 0 {
		types.SetMinFee(cfg.MinExecFee)
	}
	exec := &Executor{}
	exec.enableStat = cfg.EnableStat
	exec.enableMVCC = cfg.EnableMVCC
	exec.alias = make(map[string]string)
	for _, v := range cfg.Alias {
		data := strings.Split(v, ":")
		if len(data) != 2 {
			panic("exec.alias config error: " + v)
		}
		if _, ok := exec.alias[data[0]]; ok {
			panic("exec.alias repeat name: " + v)
		}
		if pluginmgr.HasExec(data[0]) {
			panic("exec.alias repeat name with system Exec: " + v)
		}
		exec.alias[data[0]] = data[1]
	}
	return exec
}

func (exec *Executor) SetQueueClient(qcli queue.Client) {
	exec.client = qcli
	exec.client.Sub("execs")
	var err error
	exec.qclient, err = client.New(qcli, nil)
	if err != nil {
		panic(err)
	}
	//recv 消息的处理
	go func() {
		for msg := range exec.client.Recv() {
			elog.Debug("exec recv", "msg", msg)
			if msg.Ty == types.EventExecTxList {
				go exec.procExecTxList(msg)
			} else if msg.Ty == types.EventAddBlock {
				go exec.procExecAddBlock(msg)
			} else if msg.Ty == types.EventDelBlock {
				go exec.procExecDelBlock(msg)
			} else if msg.Ty == types.EventCheckTx {
				go exec.procExecCheckTx(msg)
			} else if msg.Ty == types.EventBlockChainQuery {
				go exec.procExecQuery(msg)
			}
		}
	}()
}

func (exec *Executor) procExecQuery(msg queue.Message) {
	header, err := exec.qclient.GetLastHeader()
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	data := msg.GetData().(*types.ChainExecutor)
	driver, err := drivers.LoadDriver(data.Driver, header.GetHeight())
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	if data.StateHash == nil {
		data.StateHash = header.StateHash
	}
	localdb := NewLocalDB(exec.client)
	driver.SetLocalDB(localdb)
	opt := &StateDBOption{EnableMVCC: exec.enableMVCC, FlagMVCC: exec.flagMVCC, Height: header.GetHeight()}

	db := NewStateDB(exec.client, data.StateHash, localdb, opt)
	db.(*StateDB).enableMVCC()
	driver.SetStateDB(db)

	//查询的情况下下，执行器不做严格校验，allow，尽可能的加载执行器，并且做查询

	ret, err := driver.Query(data.FuncName, data.Param)
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, ret))
}

func (exec *Executor) procExecCheckTx(msg queue.Message) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, exec, datas.Height, datas.BlockTime, datas.Difficulty, datas.Txs, nil)
	execute.enableMVCC()
	execute.api = exec.qclient
	//返回一个列表表示成功还是失败
	result := &types.ReceiptCheckTxList{}
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		index := i
		if datas.IsMempool {
			index = -1
		}
		err := execute.execCheckTx(tx, index)
		if err != nil {
			result.Errs = append(result.Errs, err.Error())
		} else {
			result.Errs = append(result.Errs, "")
		}
	}
	msg.Reply(exec.client.NewMessage("", types.EventReceiptCheckTx, result))
}

func (exec *Executor) procExecTxList(msg queue.Message) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, exec, datas.Height, datas.BlockTime, datas.Difficulty, datas.Txs, nil)
	execute.enableMVCC()
	execute.api = exec.qclient
	var receipts []*types.Receipt
	index := 0
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		//检查groupcount
		if tx.GroupCount < 0 || tx.GroupCount == 1 || tx.GroupCount > 20 {
			receipts = append(receipts, types.NewErrReceipt(types.ErrTxGroupCount))
			continue
		}
		if tx.GroupCount == 0 {
			receipt, err := execute.execTx(tx, index)
			if err != nil {
				receipts = append(receipts, types.NewErrReceipt(err))
				continue
			}
			receipts = append(receipts, receipt)
			index++
			continue
		}
		//所有tx.GroupCount > 0 的交易都是错误的交易
		if !types.IsMatchFork(datas.Height, types.ForkV14TxGroup) {
			receipts = append(receipts, types.NewErrReceipt(types.ErrTxGroupNotSupport))
			continue
		}
		//判断GroupCount 是否会产生越界
		if i+int(tx.GroupCount) > len(datas.Txs) {
			receipts = append(receipts, types.NewErrReceipt(types.ErrTxGroupCount))
			continue
		}
		receiptlist, err := execute.execTxGroup(datas.Txs[i:i+int(tx.GroupCount)], index)
		i = i + int(tx.GroupCount) - 1
		if len(receiptlist) > 0 && len(receiptlist) != int(tx.GroupCount) {
			panic("len(receiptlist) must be equal tx.GroupCount")
		}
		if err != nil {
			for n := 0; n < int(tx.GroupCount); n++ {
				receipts = append(receipts, types.NewErrReceipt(err))
			}
			continue
		}
		receipts = append(receipts, receiptlist...)
		index += int(tx.GroupCount)
	}
	msg.Reply(exec.client.NewMessage("", types.EventReceipts,
		&types.Receipts{receipts}))
}

//allowExec key 行为判断放入 执行器
/*
权限控制规则:
1. 默认行为:
执行器只能修改执行器下面的 key
或者能修改其他执行器 exec key 下面的数据

2. friend 合约行为, 合约可以定义其他合约 可以修改的 key的内容
*/
func (execute *executor) isAllowExec(key []byte, tx *types.Transaction, index int) bool {
	realExecer := execute.getRealExecName(tx, index)
	height := execute.height
	return isAllowExec(key, realExecer, tx, height)
}

func isAllowExec(key, realExecer []byte, tx *types.Transaction, height int64) bool {
	keyExecer, err := types.FindExecer(key)
	if err != nil {
		elog.Error("find execer ", "err", err)
		return false
	}
	//平行链中 user.p.guodun.xxxx -> 实际上是 xxxx
	//TODO: 后面 GetDriverName(), 中驱动的名字可以被配置
	if types.IsPara() && bytes.Equal(keyExecer, realExecer) {
		return true
	}
	//其他合约可以修改自己合约内部(执行器只能修改执行器自己内部的数据)
	if bytes.Equal(keyExecer, tx.Execer) {
		return true
	}
	//每个合约中，都会开辟一个区域，这个区域是另外一个合约可以修改的区域
	//我们把数据限制在这个位置，防止合约的其他位置被另外一个合约修改
	//  execaddr 是加了前缀生成的地址， 而参数 realExecer 是没有前缀的执行器名字
	keyExecAddr, ok := types.GetExecKey(key)
	if ok && keyExecAddr == drivers.ExecAddress(string(tx.Execer)) {
		return true
	}
	// 历史原因做只针对对bityuan的fork特殊化处理一下
	// manage 的key 是 config
	// token 的部分key 是 mavl-create-token-
	if !types.IsMatchFork(height, types.ForkV13ExecKey) {
		if bytes.Equal(realExecer, types.ExecerManage) && bytes.Equal(keyExecer, types.ExecerConfig) {
			return true
		}
		if bytes.Equal(realExecer, types.ExecerToken) {
			if bytes.HasPrefix(key, []byte("mavl-create-token-")) {
				return true
			}
		}
	}
	//分成两种情况:
	//是执行器余额，判断 friend
	execdriver := keyExecer
	if ok && keyExecAddr == drivers.ExecAddress(string(realExecer)) {
		//判断user.p.xxx.token 是否可以写 token 合约的内容之类的
		execdriver = realExecer
	}
	d, err := drivers.LoadDriver(string(execdriver), height)
	if err != nil {
		elog.Error("load drivers error", "err", err)
		return false
	}
	//交给 -> friend 来判定
	return d.IsFriend(execdriver, key, tx)
}

func (exec *Executor) procExecAddBlock(msg queue.Message) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, exec, b.Height, b.BlockTime, uint64(b.Difficulty), b.Txs, datas.Receipts)
	execute.api = exec.qclient
	var totalFee types.TotalFee
	var kvset types.LocalDBSet
	//打开MVCC之后中途关闭，可能会发生致命的错误
	for _, kv := range datas.KV {
		execute.stateDB.Set(kv.Key, kv.Value)
	}
	if exec.enableMVCC {
		kvs, err := exec.checkMVCCFlag(execute.localDB, datas)
		if err != nil {
			panic(err)
		}
		kvset.KV = append(kvset.KV, kvs...)
		kvs = AddMVCC(execute.localDB, datas)
		if kvs != nil {
			kvset.KV = append(kvset.KV, kvs...)
		}
		for _, kv := range kvset.KV {
			execute.localDB.Set(kv.Key, kv.Value)
		}
	}
	execute.enableMVCC()
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		totalFee.Fee += tx.Fee
		totalFee.TxCount++
		kv, err := execute.execLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		if kv != nil && kv.KV != nil {
			err := exec.checkPrefix(tx.Execer, kv.KV)
			if err != nil {
				msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	//保存手续费
	feekv, err := saveFee(execute, &totalFee, b.ParentHash, b.Hash())
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
		return
	}
	kvset.KV = append(kvset.KV, feekv)
	//定制数据统计
	if exec.enableStat {
		kvs, err := exec.stat(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		kvset.KV = append(kvset.KV, kvs...)
	}
	msg.Reply(exec.client.NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Executor) checkMVCCFlag(db dbm.KVDB, datas *types.BlockDetail) ([]*types.KeyValue, error) {
	//flag = 0 : init
	//flag = 1 : start from zero
	//flag = 2 : start from no zero //不允许flag = 2的情况
	b := datas.Block
	if atomic.LoadInt64(&exec.flagMVCC) == FlagInit {
		flag, err := loadFlag(db, types.FlagKeyMVCC)
		if err != nil {
			panic(err)
		}
		atomic.StoreInt64(&exec.flagMVCC, flag)
	}
	var kvset []*types.KeyValue
	if atomic.LoadInt64(&exec.flagMVCC) == FlagInit {
		if b.Height != 0 {
			atomic.StoreInt64(&exec.flagMVCC, FlagNotFromZero)
		} else {
			//区块为0, 写入标志
			if atomic.CompareAndSwapInt64(&exec.flagMVCC, FlagInit, FlagFromZero) {
				kvset = append(kvset, types.FlagKV(types.FlagKeyMVCC, FlagFromZero))
			}
		}
	}
	if atomic.LoadInt64(&exec.flagMVCC) != FlagFromZero {
		panic("config set enableMVCC=true, it must be synchronized from 0 height")
	}
	return kvset, nil
}

func (exec *Executor) stat(execute *executor, datas *types.BlockDetail) ([]*types.KeyValue, error) {
	// 开启数据统计，需要从0开始同步数据
	b := datas.Block
	if atomic.LoadInt64(&exec.enableStatFlag) == 0 {
		flag, err := loadFlag(execute.localDB, StatisticFlag())
		if err != nil {
			panic(err)
		}
		atomic.StoreInt64(&exec.enableStatFlag, flag)
	}
	if b.Height != 0 && atomic.LoadInt64(&exec.enableStatFlag) == 0 {
		elog.Error("chain33.toml enableStat = true, it must be synchronized from 0 height")
		panic("chain33.toml enableStat = true, it must be synchronized from 0 height")
	}
	// 初始状态置为开启状态
	var kvset []*types.KeyValue
	if atomic.CompareAndSwapInt64(&exec.enableStatFlag, 0, 1) {
		kvset = append(kvset, types.FlagKV(StatisticFlag(), 1))
	}
	kvs, err := countInfo(execute, datas)
	if err != nil {
		return nil, err
	}
	kvset = append(kvset, kvs.KV...)
	return kvset, nil
}

func (exec *Executor) procExecDelBlock(msg queue.Message) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, exec, b.Height, b.BlockTime, uint64(b.Difficulty), b.Txs, nil)
	execute.enableMVCC()
	execute.api = exec.qclient
	var kvset types.LocalDBSet
	for _, kv := range datas.KV {
		execute.stateDB.Set(kv.Key, kv.Value)
	}
	if exec.enableMVCC {
		kvs := DelMVCC(execute.localDB, datas)
		if kvs != nil {
			kvset.KV = append(kvset.KV, kvs...)
		}
	}
	for i := len(b.Txs) - 1; i >= 0; i-- {
		tx := b.Txs[i]
		kv, err := execute.execDelLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
			return
		}

		if kv != nil && kv.KV != nil {
			err := exec.checkPrefix(tx.Execer, kv.KV)
			if err != nil {
				msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}

	//删除手续费
	feekv, err := delFee(execute, b.Hash())
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
		return
	}
	kvset.KV = append(kvset.KV, feekv)
	//定制数据统计
	if exec.enableStat {
		kvs, err := delCountInfo(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
			return
		}
		kvset.KV = append(kvset.KV, kvs.KV...)
	}

	msg.Reply(exec.client.NewMessage("", types.EventDelBlock, &kvset))
}

func (exec *Executor) checkPrefix(execer []byte, kvs []*types.KeyValue) error {
	return nil
}

func (exec *Executor) Close() {
	elog.Info("exec module closed")
}

//执行器 -> db 环境
type executor struct {
	stateDB      dbm.KV
	localDB      dbm.KVDB
	coinsAccount *account.DB
	height       int64
	blocktime    int64

	// 增加区块的难度值，后面的执行器逻辑需要这些属性
	difficulty uint64
	txs        []*types.Transaction
	api        client.QueueProtocolAPI
	receipts   []*types.ReceiptData
}

func newExecutor(stateHash []byte, exec *Executor, height, blocktime int64, difficulty uint64,
	txs []*types.Transaction, receipts []*types.ReceiptData) *executor {
	client := exec.client
	enableMVCC := exec.enableMVCC
	flagMVCC := exec.flagMVCC
	opt := &StateDBOption{EnableMVCC: enableMVCC, FlagMVCC: flagMVCC, Height: height}
	localdb := NewLocalDB(client)
	e := &executor{
		stateDB:      NewStateDB(client, stateHash, localdb, opt),
		localDB:      localdb,
		coinsAccount: account.NewCoinsAccount(),
		height:       height,
		blocktime:    blocktime,
		difficulty:   difficulty,
		txs:          txs,
		receipts:     receipts,
	}
	e.coinsAccount.SetDB(e.stateDB)
	return e
}

func (e *executor) enableMVCC() {
	e.stateDB.(*StateDB).enableMVCC()
}

func AddMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	kvs := detail.KV
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	//检查版本号是否是连续的
	kvlist, err := mvcc.AddMVCC(kvs, hash, detail.PrevStatusHash, detail.Block.Height)
	if err != nil {
		panic(err)
	}
	return kvlist
}

func DelMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	kvlist, err := mvcc.DelMVCC(hash, detail.Block.Height, true)
	if err != nil {
		panic(err)
	}
	return kvlist
}

//隐私交易费扣除规则：
//1.公对私交易：直接从coin合约中扣除
//2.私对私交易或者私对公交易：交易费的扣除从隐私合约账户在coin合约中的账户中扣除
func (e *executor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := tx.From()
	accFrom := e.coinsAccount.LoadAccount(from)
	if accFrom.GetBalance()-tx.Fee >= 0 {
		copyfrom := *accFrom
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		receiptBalance := &types.ReceiptAccountTransfer{&copyfrom, accFrom}
		e.coinsAccount.SaveAccount(accFrom)
		return e.cutFeeReceipt(accFrom, receiptBalance), nil
	}
	return nil, types.ErrNoBalance
}

func (e *executor) cutFeeReceipt(acc *types.Account, receiptBalance proto.Message) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecPack, e.coinsAccount.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}

func (e *executor) getRealExecName(tx *types.Transaction, index int) []byte {
	exec := e.loadDriver(tx, index)
	realexec := exec.GetDriverName()
	var execer []byte
	if realexec != "none" {
		execer = []byte(realexec)
	} else {
		execer = tx.Execer
	}
	return execer
}

func (e *executor) checkTx(tx *types.Transaction, index int) error {
	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := tx.Check(e.height, types.MinFee); err != nil {
		return err
	}
	//允许重写的情况
	//看重写的名字 name, 是否被允许执行
	if !types.IsAllowExecName(e.getRealExecName(tx, index), tx.Execer) {
		return types.ErrExecNameNotAllow
	}
	return nil
}

func (e *executor) setEnv(exec drivers.Driver) {
	exec.SetStateDB(e.stateDB)
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime, e.difficulty)
	exec.SetApi(e.api)
	exec.SetTxs(e.txs)
	exec.SetReceipt(e.receipts)
}

func (e *executor) checkTxGroup(txgroup *types.Transactions, index int) error {
	if e.height > 0 && e.blocktime > 0 && txgroup.IsExpire(e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := txgroup.Check(e.height, types.MinFee); err != nil {
		return err
	}
	return nil
}

func (e *executor) execCheckTx(tx *types.Transaction, index int) error {
	//基本检查
	err := e.checkTx(tx, index)
	if err != nil {
		return err
	}
	//检查地址的有效性
	if err := address.CheckAddress(tx.To); err != nil {
		return err
	}
	//checkInExec
	exec := e.loadDriver(tx, index)
	//手续费检查
	if !exec.IsFree() && types.MinFee > 0 {
		from := tx.From()
		accFrom := e.coinsAccount.LoadAccount(from)
		if accFrom.GetBalance() < types.MinBalanceTransfer {
			elog.Error("execCheckTx", "ispara", types.IsPara(), "exec", string(tx.Execer), "nonce", tx.Nonce)
			return types.ErrBalanceLessThanTenTimesFee
		}
	}
	e.setEnv(exec)
	return exec.CheckTx(tx, index)
}

func (e *executor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec := e.loadDriver(tx, index)
	//to 必须是一个地址
	if err := drivers.CheckAddress(tx.GetRealToAddr(), e.height); err != nil {
		return nil, err
	}
	//第一步先检查 CheckTx
	if err := exec.CheckTx(tx, index); err != nil {
		return nil, err
	}
	return exec.Exec(tx, index)
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecDelLocal(tx, r, index)
}

func (e *executor) loadDriver(tx *types.Transaction, index int) (c drivers.Driver) {
	exec := drivers.LoadDriverAllow(tx, index, e.height)
	e.setEnv(exec)
	return exec
}

func (execute *executor) execTxGroup(txs []*types.Transaction, index int) ([]*types.Receipt, error) {
	txgroup := &types.Transactions{Txs: txs}
	err := execute.checkTxGroup(txgroup, index)
	if err != nil {
		return nil, err
	}
	feelog, err := execute.execFee(txs[0], index)
	if err != nil {
		return nil, err
	}
	//开启内存事务处理，假设系统只有一个thread 执行
	//如果系统执行失败，回滚到这个状态
	rollbackLog := copyReceipt(feelog)
	execute.stateDB.Begin()
	receipts := make([]*types.Receipt, len(txs))
	for i := 1; i < len(txs); i++ {
		receipts[i] = &types.Receipt{Ty: types.ExecPack}
	}
	receipts[0], err = execute.execTxOne(feelog, txs[0], index)
	if err != nil {
		//状态数据库回滚
		if types.IsMatchFork(execute.height, types.ForkV22ExecRollback) {
			execute.stateDB.Rollback()
		}
		return receipts, nil
	}
	for i := 1; i < len(txs); i++ {
		//如果有一笔执行失败了，那么全部回滚
		receipts[i], err = execute.execTxOne(receipts[i], txs[i], index+i)
		if err != nil {
			//reset other exec , and break!
			for k := 1; k < i; k++ {
				receipts[k] = &types.Receipt{Ty: types.ExecPack}
			}
			//撤销txs[0]的交易
			if types.IsMatchFork(execute.height, types.ForkV15ResetTx0) {
				receipts[0] = rollbackLog
			}
			//撤销所有的数据库更新
			execute.stateDB.Rollback()
			return receipts, nil
		}
	}
	execute.stateDB.Commit()
	return receipts, nil
}

func (execute *executor) loadFlag(key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := execute.localDB.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound {
		return 0, nil
	}
	return 0, err
}

func (execute *executor) execFee(tx *types.Transaction, index int) (*types.Receipt, error) {
	feelog := &types.Receipt{Ty: types.ExecPack}
	execer := string(tx.Execer)
	e := execute.loadDriver(tx, index)
	execute.setEnv(e)
	//执行器名称 和  pubkey 相同，费用从内置的执行器中扣除,但是checkTx 中要过
	//默认checkTx 中对这样的交易会返回
	if bytes.Equal(address.ExecPubkey(execer), tx.GetSignature().GetPubkey()) {
		err := e.CheckTx(tx, index)
		if err != nil {
			return nil, err
		}
	}
	var err error
	//公链不允许手续费为0
	if !types.IsPara() && types.MinFee > 0 && (!e.IsFree() || types.IsPublicChain()) {
		feelog, err = execute.processFee(tx)
		if err != nil {
			return nil, err
		}
	}
	return feelog, nil
}

func copyReceipt(feelog *types.Receipt) *types.Receipt {
	receipt := types.Receipt{}
	receipt = *feelog
	receipt.KV = make([]*types.KeyValue, len(feelog.KV))
	copy(receipt.KV, feelog.KV)
	receipt.Logs = make([]*types.ReceiptLog, len(feelog.Logs))
	copy(receipt.Logs, feelog.Logs)
	return &receipt
}

func (execute *executor) execTxOne(feelog *types.Receipt, tx *types.Transaction, index int) (*types.Receipt, error) {
	//只有到pack级别的，才会增加index
	execute.stateDB.(*StateDB).StartTx()
	receipt, err := execute.Exec(tx, index)
	if err != nil {
		elog.Error("exec tx error = ", "err", err, "exec", string(tx.Execer), "action", tx.ActionName())
		//add error log
		errlog := &types.ReceiptLog{types.TyLogErr, []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	//合并两个receipt，如果执行不返回错误，那么就认为成功
	//需要检查两个东西:
	//1. statedb 中 Set的 key 必须是 在 receipt.GetKV() 这个集合中
	//2. receipt.GetKV() 中的 key, 必须符合权限控制要求
	memkvset := execute.stateDB.(*StateDB).GetSetKeys()
	feelog, err = execute.checkKV(feelog, memkvset, receipt.GetKV())
	if err != nil {
		return feelog, err
	}
	feelog, err = execute.checkKeyAllow(feelog, tx, index, receipt.GetKV())
	if err != nil {
		return feelog, err
	}
	if receipt != nil {
		feelog.KV = append(feelog.KV, receipt.KV...)
		feelog.Logs = append(feelog.Logs, receipt.Logs...)
		feelog.Ty = receipt.Ty
	}
	return feelog, nil
}

func (execute *executor) checkKV(feelog *types.Receipt, memset []string, kvs []*types.KeyValue) (*types.Receipt, error) {
	keys := make(map[string]bool)
	for _, kv := range kvs {
		k := kv.GetKey()
		keys[string(k)] = true
	}
	for _, key := range memset {
		if _, ok := keys[key]; !ok {
			elog.Error("err memset key", "key", key)
			//非法的receipt，交易执行失败
			errlog := &types.ReceiptLog{types.TyLogErr, []byte(types.ErrNotAllowMemSetKey.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
			return feelog, types.ErrNotAllowMemSetKey
		}
	}
	return feelog, nil
}

func (execute *executor) checkKeyAllow(feelog *types.Receipt, tx *types.Transaction, index int, kvs []*types.KeyValue) (*types.Receipt, error) {
	for _, kv := range kvs {
		k := kv.GetKey()
		if !execute.isAllowExec(k, tx, index) {
			elog.Error("err receipt key", "key", string(k), "tx.exec", string(tx.GetExecer()),
				"tx.action", tx.ActionName())
			//非法的receipt，交易执行失败
			errlog := &types.ReceiptLog{types.TyLogErr, []byte(types.ErrNotAllowKey.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
			return feelog, types.ErrNotAllowKey
		}
	}
	return feelog, nil
}

func (execute *executor) execTx(tx *types.Transaction, index int) (*types.Receipt, error) {
	if execute.height == 0 { //genesis block 不检查手续费
		receipt, err := execute.Exec(tx, index)
		if err != nil {
			panic(err)
		}
		if err == nil && receipt == nil {
			panic("genesis block: executor not exist")
		}
		return receipt, nil
	}
	//交易检查规则：
	//1. mempool 检查区块，尽量检查更多的错误
	//2. 打包的时候，尽量打包更多的交易，只要基本的签名，以及格式没有问题
	err := execute.checkTx(tx, index)
	if err != nil {
		return nil, err
	}
	//处理交易手续费(先把手续费收了)
	//如果收了手续费，表示receipt 至少是pack 级别
	//收不了手续费的交易才是 error 级别
	feelog, err := execute.execFee(tx, index)
	if err != nil {
		return nil, err
	}
	//ignore err
	matchfork := types.IsMatchFork(execute.height, types.ForkV22ExecRollback)
	if matchfork {
		execute.stateDB.Begin()
	}
	feelog, err = execute.execTxOne(feelog, tx, index)
	if err != nil {
		if matchfork {
			execute.stateDB.Rollback()
		}
	} else {
		if matchfork {
			execute.stateDB.Commit()
		}
	}
	elog.Debug("exec tx = ", "index", index, "execer", string(tx.Execer), "err", err)
	return feelog, nil
}

func loadFlag(localDB dbm.KVDB, key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := localDB.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound {
		return 0, nil
	}
	return 0, err
}

func totalFeeKey(hash []byte) []byte {
	key := []byte("TotalFeeKey:")
	return append(key, hash...)
}

func saveFee(ex *executor, fee *types.TotalFee, parentHash, hash []byte) (*types.KeyValue, error) {
	totalFee := &types.TotalFee{}
	totalFeeBytes, err := ex.localDB.Get(totalFeeKey(parentHash))
	if err == nil {
		err = types.Decode(totalFeeBytes, totalFee)
		if err != nil {
			return nil, err
		}
	} else if err != types.ErrNotFound {
		return nil, err
	}

	totalFee.Fee += fee.Fee
	totalFee.TxCount += fee.TxCount
	return &types.KeyValue{totalFeeKey(hash), types.Encode(totalFee)}, nil
}

func delFee(ex *executor, hash []byte) (*types.KeyValue, error) {
	return &types.KeyValue{totalFeeKey(hash), types.Encode(&types.TotalFee{})}, nil
}
