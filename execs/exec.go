package execs

//store package store the world - state data
import (
	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	_ "code.aliyun.com/chain33/chain33/execs/execdrivers/coins"
	_ "code.aliyun.com/chain33/chain33/execs/execdrivers/none"
	_ "code.aliyun.com/chain33/chain33/execs/execdrivers/ticket"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var elog = log.New("module", "execs")
var zeroHash [32]byte

const minFee int64 = 1e6

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Execs struct {
	qclient queue.IClient
}

func New() *Execs {
	exec := &Execs{}
	return exec
}

func (exec *Execs) SetQueue(q *queue.Queue) {
	exec.qclient = q.GetClient()
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
			}
		}
	}()
}

func (exec *Execs) procExecTxList(msg queue.Message, q *queue.Queue) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := NewExecute(datas.StateHash, q, datas.Height, datas.BlockTime)
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
		//正常的区块：
		err := execute.checkTx(tx, index)
		if err != nil {
			receipt := types.NewErrReceipt(err)
			receipts = append(receipts, receipt)
			continue
		}
		//处理交易手续费(先把手续费收了)
		//如果收了手续费，表示receipt 至少是pack 级别
		//收不了手续费的交易才是 error 级别
		feelog, err := execute.ProcessFee(tx)
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
	}
	msg.Reply(q.GetClient().NewMessage("", types.EventReceipts,
		&types.Receipts{receipts}))
}

func (exec *Execs) procExecAddBlock(msg queue.Message, q *queue.Queue) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := NewExecute(b.StateHash, q, b.Height, b.BlockTime)
	var kvset types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		kv, err := execute.ExecLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(q.GetClient().NewMessage("", types.EventAddBlock, err))
			return
		}
		if kv != nil && kv.KV != nil {
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	msg.Reply(q.GetClient().NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Execs) procExecDelBlock(msg queue.Message, q *queue.Queue) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := NewExecute(b.StateHash, q, b.Height, b.BlockTime)
	var kvset types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		kv, err := execute.ExecDelLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(q.GetClient().NewMessage("", types.EventAddBlock, err))
			return
		}
		if kv != nil && kv.KV != nil {
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	msg.Reply(q.GetClient().NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Execs) Close() {
	elog.Info("exec module closed")
}

//执行器 -> db 环境
type Execute struct {
	stateDB   dbm.KVDB
	localDB   dbm.KVDB
	height    int64
	blocktime int64
}

func NewExecute(stateHash []byte, q *queue.Queue, height, blocktime int64) *Execute {
	return &Execute{
		stateDB:   NewStateDB(q, stateHash),
		localDB:   NewLocalDB(q),
		height:    height,
		blocktime: blocktime,
	}
}

func (e *Execute) ProcessFee(tx *types.Transaction) (*types.Receipt, error) {
	accFrom := account.LoadAccount(e.stateDB, account.PubKeyToAddress(tx.Signature.Pubkey).String())
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

func (e *Execute) checkTx(tx *types.Transaction, index int) error {
	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.height, e.blocktime) { //如果已经过期
		return types.ErrTxExpire
	}
	if tx.Fee < minFee {
		return types.ErrFeeTooLow
	}
	return nil
}

func (e *Execute) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec, err := execdrivers.LoadExecute(string(tx.Execer))
	if err != nil {
		exec, err = execdrivers.LoadExecute("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetDB(e.stateDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.Exec(tx, index)
}

func (e *Execute) ExecLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec, err := execdrivers.LoadExecute(string(tx.Execer))
	if err != nil {
		exec, err = execdrivers.LoadExecute("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecLocal(tx, r, index)
}

func (e *Execute) ExecDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec, err := execdrivers.LoadExecute(string(tx.Execer))
	if err != nil {
		exec, err = execdrivers.LoadExecute("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecDelLocal(tx, r, index)
}

type StateDB struct {
	cache map[string][]byte
	db    *DataBase
}

func NewStateDB(q *queue.Queue, stateHash []byte) *StateDB {
	return &StateDB{make(map[string][]byte), NewDataBase(q, stateHash)}
}

func (e *StateDB) Get(key []byte) (value []byte, err error) {
	if value, ok := e.cache[string(key)]; ok {
		return value, nil
	}
	value, err = e.db.Get(key)
	if err != nil {
		return nil, err
	}
	e.cache[string(key)] = value
	return value, nil
}

func (e *StateDB) Set(key []byte, value []byte) error {
	e.cache[string(key)] = value
	return nil
}

type LocalDB struct {
	cache map[string][]byte
	db    *DataBaseLocal
}

func NewLocalDB(q *queue.Queue) *LocalDB {
	return &LocalDB{make(map[string][]byte), NewDataBaseLocal(q)}
}

func (e *LocalDB) Get(key []byte) (value []byte, err error) {
	if value, ok := e.cache[string(key)]; ok {
		return value, nil
	}
	value, err = e.db.Get(key)
	if err != nil {
		return nil, err
	}
	e.cache[string(key)] = value
	return value, nil
}

func (e *LocalDB) Set(key []byte, value []byte) error {
	e.cache[string(key)] = value
	return nil
}

type DataBase struct {
	qclient   queue.IClient
	stateHash []byte
}

func NewDataBase(q *queue.Queue, stateHash []byte) *DataBase {
	return &DataBase{q.GetClient(), stateHash}
}

func (db *DataBase) Get(key []byte) (value []byte, err error) {
	query := &types.StoreGet{db.stateHash, [][]byte{key}}
	msg := db.qclient.NewMessage("store", types.EventStoreGet, query)
	db.qclient.Send(msg, true)
	resp, err := db.qclient.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		return nil, types.ErrNotFound
	}
	return value, nil
}

type DataBaseLocal struct {
	qclient queue.IClient
}

func NewDataBaseLocal(q *queue.Queue) *DataBaseLocal {
	return &DataBaseLocal{q.GetClient()}
}

func (db *DataBaseLocal) Get(key []byte) (value []byte, err error) {
	query := &types.LocalDBGet{[][]byte{key}}
	msg := db.qclient.NewMessage("blockchain", types.EventLocalGet, query)
	db.qclient.Send(msg, true)
	resp, err := db.qclient.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.LocalReplyValue).Values[0]
	if value == nil {
		return nil, types.ErrNotFound
	}
	return value, nil
}
