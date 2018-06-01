package executor

//store package store the world - state data
import (
	"bytes"
	"sync"

	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	// register drivers
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/executor/drivers/coins"
	"gitlab.33.cn/chain33/chain33/executor/drivers/hashlock"
	"gitlab.33.cn/chain33/chain33/executor/drivers/manage"
	"gitlab.33.cn/chain33/chain33/executor/drivers/none"
	"gitlab.33.cn/chain33/chain33/executor/drivers/norm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/retrieve"
	"gitlab.33.cn/chain33/chain33/executor/drivers/ticket"
	"gitlab.33.cn/chain33/chain33/executor/drivers/token"
	"gitlab.33.cn/chain33/chain33/executor/drivers/trade"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var elog = log.New("module", "execs")
var coinsAccount = account.NewCoinsAccount()
var isCounted bool

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Executor struct {
	client  queue.Client
	qclient client.QueueProtocolAPI
}

func execInit() {
	coins.Init()
	hashlock.Init()
	manage.Init()
	none.Init()
	norm.Init()
	retrieve.Init()
	ticket.Init()
	token.Init()
	trade.Init()
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
	isCounted = cfg.IsCounted

	exec := &Executor{}
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
	driver, err := LoadDriver(data.Driver, header.GetHeight())
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	driver = driver.Clone()
	driver.SetLocalDB(NewLocalDB(exec.client.Clone()))
	driver.SetStateDB(NewStateDB(exec.client.Clone(), data.StateHash))
	ret, err := driver.Query(data.FuncName, data.Param)
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, ret))
}

func (exec *Executor) procExecCheckTx(msg queue.Message) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, exec.client.Clone(), datas.Height, datas.BlockTime)
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
	execute := newExecutor(datas.StateHash, exec.client.Clone(), datas.Height, datas.BlockTime)
	var receipts []*types.Receipt
	index := 0
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		if execute.height == 0 { //genesis block 不检查手续费
			receipt, err := execute.Exec(tx, i)
			if err != nil {
				panic(err)
			}
			receipts = append(receipts, receipt)
			continue
		}
		//交易检查规则：
		//1. mempool 检查区块，尽量检查更多的错误
		//2. 打包的时候，尽量打包更多的交易，只要基本的签名，以及格式没有问题
		err := execute.checkTx(tx, index)
		if err != nil {
			receipt := types.NewErrReceipt(err)
			receipts = append(receipts, receipt)
			continue
		}
		//处理交易手续费(先把手续费收了)
		//如果收了手续费，表示receipt 至少是pack 级别
		//收不了手续费的交易才是 error 级别
		feelog := &types.Receipt{Ty: types.ExecPack}
		e, err := LoadDriver(string(tx.Execer), execute.height)
		if err != nil {
			e, err = LoadDriver("none", execute.height)
			if err != nil {
				panic(err)
			}
		}
		//公链不允许手续费为0
		if types.MinFee > 0 && (!e.IsFree() || types.IsPublicChain()) {
			feelog, err = execute.processFee(tx)
			if err != nil {
				receipt := types.NewErrReceipt(err)
				receipts = append(receipts, receipt)
				continue
			}
		}
		//只有到pack级别的，才会增加index
		receipt, err := execute.Exec(tx, index)
		index++

		if err != nil {
			elog.Error("exec tx error = ", "err", err, "exec", string(tx.Execer), "action", tx.ActionName())
			//add error log
			errlog := &types.ReceiptLog{types.TyLogErr, []byte(err.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
		} else {
			//合并两个receipt，如果执行不返回错误，那么就认为成功
			if receipt != nil {
				for _, kv := range receipt.GetKV() {
					k := kv.GetKey()
					if !isAllowExec(k, tx.GetExecer()) {
						elog.Error("err receipt key", "key", string(k), "tx.exec", string(tx.GetExecer()),
							"tx.action", tx.ActionName())
						if types.IsTestNet() {
							//如果是测试网络，直接崩溃
							panic("err receipt key")
						}
					}
				}
				feelog.KV = append(feelog.KV, receipt.KV...)
				feelog.Logs = append(feelog.Logs, receipt.Logs...)
				feelog.Ty = receipt.Ty
			}
		}
		receipts = append(receipts, feelog)
		elog.Debug("exec tx = ", "index", index, "execer", string(tx.Execer))
	}
	msg.Reply(exec.client.NewMessage("", types.EventReceipts,
		&types.Receipts{receipts}))
}

func isAllowExec(key, txexecer []byte) bool {
	//coins 和 token 可以修改所有的其他合约的值
	keyexecer, err := findExecer(key)
	if err != nil {
		elog.Error("find execer ", "err", err)
		return false
	}
	if bytes.Equal(txexecer, types.ExecerCoins) || bytes.Equal(txexecer, types.ExecerToken) {
		return true
	}
	//其他合约可以修改自己合约内部
	if bytes.Equal(keyexecer, txexecer) {
		return true
	}
	//如果是运行运行deposit的执行器，可以修改coins 的值
	for _, execer := range types.AllowDepositExec {
		if bytes.Equal(txexecer, execer) && bytes.Equal(keyexecer, types.ExecerCoins) {
			return true
		}
	}
	//如果keyexecer 是 coins 和  token，那么只能修改mavl-coins-symbol-exec 下面的字段
	if bytes.Equal(keyexecer, types.ExecerCoins) || bytes.Equal(keyexecer, types.ExecerToken) {
		if isExecKey(key) {
			return true
		}
	}
	//manage 的key 是 config
	if bytes.Equal(txexecer, types.ExecerManage) && bytes.Equal(keyexecer, types.ExecerConfig) {
		return true
	}
	return false
}

var bytesExec = []byte("exec")

func isExecKey(key []byte) bool {
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
		if bytes.Equal(key[start:end], bytesExec) {
			return true
		}
	}
	return false
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
	execute := newExecutor(b.StateHash, exec.client.Clone(), b.Height, b.BlockTime)
	var totalFee types.TotalFee
	var kvset types.LocalDBSet
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

	//定制数据统计
	if isCounted {
		kvs, err := countInfo(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		kvset.KV = append(kvset.KV, kvs.KV...)
	}

	msg.Reply(exec.client.NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Executor) procExecDelBlock(msg queue.Message) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, exec.client.Clone(), b.Height, b.BlockTime)
	var kvset types.LocalDBSet
	for i := len(b.Txs) - 1; i >= 0; i-- {
		tx := b.Txs[i]
		kv, err := execute.execDelLocal(tx, datas.Receipts[i], i)
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
				msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}

	//定制数据统计
	if isCounted {
		kvs, err := delCountInfo(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		kvset.KV = append(kvset.KV, kvs.KV...)
	}

	msg.Reply(exec.client.NewMessage("", types.EventAddBlock, &kvset))
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
	execDriver   *drivers.ExecDrivers
	height       int64
	blocktime    int64
}

func newExecutor(stateHash []byte, client queue.Client, height, blocktime int64) *executor {
	e := &executor{
		stateDB:      NewStateDB(client.Clone(), stateHash),
		localDB:      NewLocalDB(client.Clone()),
		coinsAccount: account.NewCoinsAccount(),
		execDriver:   drivers.CreateDrivers4CurrentHeight(height),
		height:       height,
		blocktime:    blocktime,
	}
	e.coinsAccount.SetDB(e.stateDB)
	return e
}

func (e *executor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
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

func (e *executor) execCheckTx(tx *types.Transaction, index int) error {
	//基本检查
	err := e.checkTx(tx, index)
	if err != nil {
		return err
	}
	//checkInExec
	exec := e.loadDriverForExec(string(tx.Execer))
	//手续费检查
	if !exec.IsFree() && types.MinFee > 0 {
		from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
		accFrom := e.coinsAccount.LoadAccount(from)
		if accFrom.GetBalance() < types.MinBalanceTransfer {
			return types.ErrBalanceLessThanTenTimesFee
		}
	}

	exec.SetStateDB(e.stateDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.CheckTx(tx, index)
}

func (e *executor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec := e.loadDriverForExec(string(tx.Execer))
	exec.SetStateDB(e.stateDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.Exec(tx, index)
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriverForExec(string(tx.Execer))
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriverForExec(string(tx.Execer))
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecDelLocal(tx, r, index)
}

func (e *executor) loadDriverForExec(exector string) (c drivers.Driver) {
	exec, err := e.execDriver.LoadDriver(exector)
	if err != nil {
		exec, err = e.execDriver.LoadDriver("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetExecDriver(e.execDriver)

	return exec
}

func LoadDriver(name string, runningHeight int64) (c drivers.Driver, err error) {
	execDrivers := drivers.CreateDrivers4CurrentHeight(runningHeight)
	return execDrivers.LoadDriver(name)
}

