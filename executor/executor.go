package executor

//store package store the world - state data
import (
	"bytes"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/executor/drivers"
	_ "code.aliyun.com/chain33/chain33/executor/drivers/coins"
	_ "code.aliyun.com/chain33/chain33/executor/drivers/hashlock"
	_ "code.aliyun.com/chain33/chain33/executor/drivers/none"
	_ "code.aliyun.com/chain33/chain33/executor/drivers/retrieve"
	_ "code.aliyun.com/chain33/chain33/executor/drivers/ticket"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var elog = log.New("module", "execs")

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Execs struct {
	qclient queue.Client
}

func New() *Execs {
	exec := &Execs{}
	return exec
}

func (exec *Execs) SetQueue(q *queue.Queue) {
	exec.qclient = q.NewClient()
	client := exec.qclient
	client.Sub("execs")

	//recv 消息的处理
	go func() {
		for msg := range client.Recv() {
			elog.Debug("exec recv", "msg", msg)
			if msg.Ty == types.EventExecTxList {
				exec.procExecTxList(msg, q)
			} else if msg.Ty == types.EventAddBlock {
				exec.procExecAddBlock(msg, q)
			} else if msg.Ty == types.EventDelBlock {
				exec.procExecDelBlock(msg, q)
			} else if msg.Ty == types.EventCheckTx {
				exec.procExecCheckTx(msg, q)
			}
		}
	}()
}

func (exec *Execs) procExecCheckTx(msg queue.Message, q *queue.Queue) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, q, datas.Height, datas.BlockTime)
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
	msg.Reply(q.NewClient().NewMessage("", types.EventReceiptCheckTx, result))
}

func (exec *Execs) procExecTxList(msg queue.Message, q *queue.Queue) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, q, datas.Height, datas.BlockTime)
	var receipts []*types.Receipt
	index := 0
	for i := 0; i < len(datas.Txs); i++ {
		beg := time.Now()
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
		feelog, err := execute.processFee(tx)
		if err != nil {
			receipt := types.NewErrReceipt(err)
			receipts = append(receipts, receipt)
			continue
		}
		//只有到pack级别的，才会增加index
		receipt, err := execute.Exec(tx, index)
		index++
		if err != nil {
			elog.Error("exec tx error = ", "err", err, "tx", tx)
			//add error log
			errlog := &types.ReceiptLog{types.TyLogErr, []byte(err.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
		} else {
			//合并两个receipt，如果执行不返回错误，那么就认为成功
			feelog.KV = append(feelog.KV, receipt.KV...)
			feelog.Logs = append(feelog.Logs, receipt.Logs...)
			feelog.Ty = receipt.Ty
		}
		receipts = append(receipts, feelog)
		elog.Debug("exec tx = ", "index", index, "execer", string(tx.Execer), "cost:", time.Since(beg))
	}
	msg.Reply(q.NewClient().NewMessage("", types.EventReceipts,
		&types.Receipts{receipts}))
}

func (exec *Execs) procExecAddBlock(msg queue.Message, q *queue.Queue) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, q, b.Height, b.BlockTime)
	var kvset types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		kv, err := execute.execLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(q.NewClient().NewMessage("", types.EventAddBlock, err))
			return
		}
		if kv != nil && kv.KV != nil {
			err := exec.checkPrefix(tx.Execer, kv.KV)
			if err != nil {
				msg.Reply(q.NewClient().NewMessage("", types.EventAddBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	msg.Reply(q.NewClient().NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Execs) procExecDelBlock(msg queue.Message, q *queue.Queue) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, q, b.Height, b.BlockTime)
	var kvset types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		kv, err := execute.execDelLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(q.NewClient().NewMessage("", types.EventAddBlock, err))
			return
		}

		if kv != nil && kv.KV != nil {
			err := exec.checkPrefix(tx.Execer, kv.KV)
			if err != nil {
				msg.Reply(q.NewClient().NewMessage("", types.EventDelBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	msg.Reply(q.NewClient().NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Execs) checkPrefix(execer []byte, kvs []*types.KeyValue) error {
	if kvs == nil {
		return nil
	}
	if bytes.HasPrefix(execer, []byte("user.")) {
		for j := 0; j < len(kvs); j++ {
			if !bytes.HasPrefix(kvs[j].Key, execer) {
				return types.ErrLocalDBPerfix
			}
		}
	}
	return nil
}

func (exec *Execs) Close() {
	elog.Info("exec module closed")
}

//执行器 -> db 环境
type executor struct {
	stateDB   dbm.KVDB
	localDB   dbm.KVDB
	height    int64
	blocktime int64
}

func newExecutor(stateHash []byte, q *queue.Queue, height, blocktime int64) *executor {
	return &executor{
		stateDB:   NewStateDB(q, stateHash),
		localDB:   NewLocalDB(q),
		height:    height,
		blocktime: blocktime,
	}
}

func (e *executor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	accFrom := account.LoadAccount(e.stateDB, from)
	if accFrom.GetBalance()-tx.Fee >= 0 {
		receiptBalance := &types.ReceiptBalance{accFrom.GetBalance(), accFrom.GetBalance() - tx.Fee, -tx.Fee}
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		account.SaveAccount(e.stateDB, accFrom)
		return cutFeeReceipt(accFrom, receiptBalance), nil
	}
	return nil, types.ErrNoBalance
}

func cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptBalance) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecPack, account.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}

func (e *executor) checkTx(tx *types.Transaction, index int) error {
	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.height, e.blocktime) { //如果已经过期
		return types.ErrTxExpire
	}
	if err := tx.Check(); err != nil {
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

	//手续费检查
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	accFrom := account.LoadAccount(e.stateDB, from)
	if accFrom.GetBalance() < types.MinBalanceTransfer {
		return types.ErrBalanceLessThanTenTimesFee
	}
	//checkInExec
	exec, err := drivers.LoadDriver(string(tx.Execer))
	if err != nil {
		exec, err = drivers.LoadDriver("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetDB(e.stateDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.CheckTx(tx, index)
}

func (e *executor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec, err := drivers.LoadDriver(string(tx.Execer))
	if err != nil {
		exec, err = drivers.LoadDriver("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetDB(e.stateDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.Exec(tx, index)
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec, err := drivers.LoadDriver(string(tx.Execer))
	if err != nil {
		exec, err = drivers.LoadDriver("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec, err := drivers.LoadDriver(string(tx.Execer))
	if err != nil {
		exec, err = drivers.LoadDriver("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecDelLocal(tx, r, index)
}

func LoadDriver(name string) (c drivers.Driver, err error) {
	return drivers.LoadDriver(name)
}
