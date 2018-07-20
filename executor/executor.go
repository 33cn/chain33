package executor

//store package store the world - state data
import (
	"bytes"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/address"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/executor/drivers"

	// register drivers
	"gitlab.33.cn/chain33/chain33/executor/drivers/coins"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/hashlock"
	"gitlab.33.cn/chain33/chain33/executor/drivers/manage"
	"gitlab.33.cn/chain33/chain33/executor/drivers/none"
	"gitlab.33.cn/chain33/chain33/executor/drivers/norm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/relay"
	"gitlab.33.cn/chain33/chain33/executor/drivers/retrieve"
	"gitlab.33.cn/chain33/chain33/executor/drivers/ticket"
	"gitlab.33.cn/chain33/chain33/executor/drivers/token"
	"gitlab.33.cn/chain33/chain33/executor/drivers/trade"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/executor/drivers/cert"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	exectype "gitlab.33.cn/chain33/chain33/types/executor"
)

var elog = log.New("module", "execs")
var coinsAccount = account.NewCoinsAccount()
var keyMVCCFlag = []byte("FLAG:keyMVCCFlag")

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
}

func execInit() {
	exectype.Init()
	coins.Init()
	hashlock.Init()
	manage.Init()
	none.Init()
	norm.Init()
	retrieve.Init()
	ticket.Init()
	token.Init()
	trade.Init()
	evm.Init()
	relay.Init()
	cert.Init()
}

var runonce sync.Once

func New(cfg *types.Exec) *Executor {
	// init executor
	runonce.Do(func() {
		execInit()
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
	data := msg.GetData().(*types.BlockChainQuery)
	driver, err := drivers.LoadDriver(data.Driver, header.GetHeight())
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	driver.SetLocalDB(NewLocalDB(exec.client))
	driver.SetStateDB(NewStateDB(exec.client, data.StateHash, exec.enableMVCC, exec.flagMVCC))
	ret, err := driver.Query(data.FuncName, data.Param)
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, ret))
}

func (exec *Executor) procExecCheckTx(msg queue.Message) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, exec, datas.Height, datas.BlockTime, datas.Difficulty)
	execute.api = exec.qclient
	//返回一个列表表示成功还是失败
	result := &types.ReceiptCheckTxList{}
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		err := execute.execCheckTx(tx, i)
		if err != nil {
			result.Errs = append(result.Errs, err.Error())
		} else {
			result.Errs = append(result.Errs, "")
		}
	}
	msg.Reply(exec.client.NewMessage("", types.EventReceiptCheckTx, result))
}

var commonPrefix = []byte("mavl-")

func (exec *Executor) procExecTxList(msg queue.Message) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, exec, datas.Height, datas.BlockTime, datas.Difficulty)
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

func isAllowExec(key, txexecer []byte, toaddr string, height int64) bool {
	keyexecer, err := findExecer(key)
	if err != nil {
		elog.Error("find execer ", "err", err)
		return false
	}
	//其他合约可以修改自己合约内部
	if bytes.Equal(keyexecer, txexecer) {
		return true
	}
	//如果是运行运行deposit的执行器，可以修改coins 的值（只有挖矿合约运行这样做）
	for _, execer := range types.AllowDepositExec {
		if bytes.Equal(txexecer, execer) && bytes.Equal(keyexecer, types.ExecerCoins) {
			return true
		}
	}
	//每个合约中，都会开辟一个区域，这个区域是另外一个合约可以修改的区域
	//我们把数据限制在这个位置，防止合约的其他位置被另外一个合约修改
	execaddr, ok := getExecKey(key)
	if ok && execaddr == address.ExecAddress(string(txexecer)) {
		return true
	}

	// 特殊化处理一下
	// manage 的key 是 config
	// token 的部分key 是 mavl-create-token-
	if !types.IsMatchFork(height, types.ForkV13ExecKey) {
		elog.Info("mavl key", "execer", keyexecer, "keyexecer", keyexecer)
		if bytes.Equal(txexecer, types.ExecerManage) && bytes.Equal(keyexecer, types.ExecerConfig) {
			return true
		}
		if bytes.Equal(txexecer, types.ExecerToken) {
			if bytes.HasPrefix(key, []byte("mavl-create-token-")) {
				return true
			}
		}
	}

	// user.evm 的交易，使用evm执行器
	// 这部分判断逻辑不能放到前面几步，因为它修改了txexecer，会影响别的判断逻辑
	if bytes.HasPrefix(txexecer, []byte("user.evm.")) {
		txexecer = types.ExecerEvm
		if bytes.Equal(keyexecer, txexecer) {
			return true
		}
	}

	return false
}

var bytesExec = []byte("exec-")

func getExecKey(key []byte) (string, bool) {
	n := 0
	start := 0
	end := 0
	for i := len(commonPrefix); i < len(key); i++ {
		if key[i] == '-' {
			n = n + 1
			if n == 2 {
				start = i + 1
			}
			if n == 3 {
				end = i
				break
			}
		}
	}
	if start > 0 && end > 0 {
		if bytes.Equal(key[start:end+1], bytesExec) {
			//find addr
			start = end + 1
			for k := end; k < len(key); k++ {
				if key[k] == ':' { //end+1
					end = k
					return string(key[start:end]), true
				}
			}
		}
	}
	return "", false
}

func findExecer(key []byte) (execer []byte, err error) {
	if !bytes.HasPrefix(key, commonPrefix) {
		return nil, types.ErrMavlKeyNotStartWithMavl
	}
	for i := len(commonPrefix); i < len(key); i++ {
		if key[i] == '-' {
			return key[len(commonPrefix):i], nil
		}
	}
	return nil, types.ErrNoExecerInMavlKey
}

func (exec *Executor) procExecAddBlock(msg queue.Message) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, exec, b.Height, b.BlockTime, uint64(b.Difficulty))
	execute.api = exec.qclient
	var totalFee types.TotalFee
	var kvset types.LocalDBSet
	//打开MVCC之后中途关闭，可能会发生致命的错误
	if exec.enableMVCC {
		kvs, err := exec.checkMVCCFlag(execute, datas)
		if err != nil {
			panic(err)
		}
		kvset.KV = append(kvset.KV, kvs...)
		kvs = execute.AddMVCC(datas)
		if kvs != nil {
			kvset.KV = append(kvset.KV, kvs...)
		}
	}
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

func flagKV(key []byte, value int64) *types.KeyValue {
	return &types.KeyValue{Key: key, Value: types.Encode(&types.Int64{Data: value})}
}

func (exec *Executor) checkMVCCFlag(execute *executor, datas *types.BlockDetail) ([]*types.KeyValue, error) {
	//flag = 0 : init
	//flag = 1 : start from zero
	//flag = 2 : start from no zero
	b := datas.Block
	if atomic.LoadInt64(&exec.flagMVCC) == FlagInit {
		flag, err := execute.loadFlag(keyMVCCFlag)
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
				kvset = append(kvset, flagKV(keyMVCCFlag, FlagFromZero))
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
		flag, err := execute.loadFlag(StatisticFlag())
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
		kvset = append(kvset, flagKV(StatisticFlag(), 1))
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
	execute := newExecutor(b.StateHash, exec, b.Height, b.BlockTime, uint64(b.Difficulty))
	execute.api = exec.qclient
	var kvset types.LocalDBSet
	if exec.enableMVCC {
		kvs := execute.DelMVCC(datas)
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

	api client.QueueProtocolAPI
}

func newExecutor(stateHash []byte, exec *Executor, height, blocktime int64, difficulty uint64) *executor {
	client := exec.client
	enableMVCC := exec.enableMVCC
	flagMVCC := exec.flagMVCC
	e := &executor{
		stateDB:      NewStateDB(client, stateHash, enableMVCC, flagMVCC),
		localDB:      NewLocalDB(client),
		coinsAccount: account.NewCoinsAccount(),
		height:       height,
		blocktime:    blocktime,
		difficulty:   difficulty,
	}
	e.coinsAccount.SetDB(e.stateDB)
	return e
}

func (e *executor) AddMVCC(detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	kvs := detail.KV
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(e.localDB)
	//检查版本号是否是连续的
	kvlist, err := mvcc.AddMVCC(kvs, hash, detail.PrevStatusHash, detail.Block.Height)
	if err != nil {
		panic(err)
	}
	return kvlist
}

func (e *executor) DelMVCC(detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	kvs := detail.KV
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(e.localDB)
	kvlist, err := mvcc.DelMVCC(kvs, hash, detail.Block.Height)
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

func (e *executor) checkTx(tx *types.Transaction, index int) error {
	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := tx.Check(types.MinFee); err != nil {
		return err
	}

	return nil
}

func (e *executor) setEnv(exec drivers.Driver) {
	exec.SetStateDB(e.stateDB)
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime, e.difficulty)
	exec.SetApi(e.api)
}

func (e *executor) checkTxGroup(txgroup *types.Transactions, index int) error {
	if e.height > 0 && e.blocktime > 0 && txgroup.IsExpire(e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := txgroup.Check(types.MinFee); err != nil {
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
	exec := e.loadDriverForExec(string(tx.Execer), e.height)
	//手续费检查
	if !exec.IsFree() && types.MinFee > 0 {
		from := tx.From()
		accFrom := e.coinsAccount.LoadAccount(from)
		if accFrom.GetBalance() < types.MinBalanceTransfer {
			return types.ErrBalanceLessThanTenTimesFee
		}
	}

	e.setEnv(exec)
	return exec.CheckTx(tx, index)
}

func (e *executor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec := e.loadDriverForExec(string(tx.Execer), e.height)
	e.setEnv(exec)
	return exec.Exec(tx, index)
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriverForExec(string(tx.Execer), e.height)
	e.setEnv(exec)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriverForExec(string(tx.Execer), e.height)
	e.setEnv(exec)
	return exec.ExecDelLocal(tx, r, index)
}

func (e *executor) loadDriverForExec(exector string, height int64) (c drivers.Driver) {
	exec, err := drivers.LoadDriver(exector, height)
	if err != nil {
		exec, err = drivers.LoadDriver(types.ExecName("none"), height)
		if err != nil {
			panic(err)
		}
	}
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
	e := execute.loadDriverForExec(execer, execute.height)
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
	if types.MinFee > 0 && (!e.IsFree() || types.IsPublicChain()) || !types.IsPara() {
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
	receipt, err := execute.Exec(tx, index)
	if err != nil {
		elog.Error("exec tx error = ", "err", err, "exec", string(tx.Execer), "action", tx.ActionName())
		//add error log
		errlog := &types.ReceiptLog{types.TyLogErr, []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	//合并两个receipt，如果执行不返回错误，那么就认为成功
	if receipt != nil {
		for _, kv := range receipt.GetKV() {
			k := kv.GetKey()
			if !isAllowExec(k, tx.GetExecer(), tx.To, execute.height) {
				elog.Error("err receipt key", "key", string(k), "tx.exec", string(tx.GetExecer()),
					"tx.action", tx.ActionName())
				//非法的receipt，交易执行失败
				errlog := &types.ReceiptLog{types.TyLogErr, []byte(types.ErrNotAllowKey.Error())}
				feelog.Logs = append(feelog.Logs, errlog)
				return feelog, types.ErrNotAllowKey
			}
		}
		feelog.KV = append(feelog.KV, receipt.KV...)
		feelog.Logs = append(feelog.Logs, receipt.Logs...)
		feelog.Ty = receipt.Ty
	}
	return feelog, nil
}

func (execute *executor) execTx(tx *types.Transaction, index int) (*types.Receipt, error) {
	if execute.height == 0 { //genesis block 不检查手续费
		receipt, err := execute.Exec(tx, index)
		if err != nil {
			panic(err)
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
	feelog, _ = execute.execTxOne(feelog, tx, index)
	elog.Debug("exec tx = ", "index", index, "execer", string(tx.Execer))
	return feelog, nil
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
