package execs

//store package store the world - state data
import (
	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	_ "code.aliyun.com/chain33/chain33/execs/execdrivers/coins"
	_ "code.aliyun.com/chain33/chain33/execs/execdrivers/none"
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
			elog.Info("exec recv", "msg", msg)
			if msg.Ty == types.EventExecTxList {
				datas := msg.GetData().(*types.ExecTxList)
				execute := NewExecute(datas.StateHash, q, datas.Height, datas.BlockTime)
				var receipts []*types.Receipt
				elog.Warn("exec block begin", "ntx", len(datas.Txs))
				for i := 0; i < len(datas.Txs); i++ {
					tx := datas.Txs[i]
					elog.Warn("exec tx", "index", i)
					if execute.height == 0 { //genesis block 不检查手续费
						receipt, err := execute.Exec(tx)
						if err != nil {
							panic(err)
						}
						receipts = append(receipts, receipt)
						continue
					}
					//正常的区块：
					if datas.Height > 0 && datas.BlockTime > 0 && tx.IsExpire(datas.Height, datas.BlockTime) { //如果已经过期
						receipt := types.NewErrReceipt(types.ErrTxExpire)
						receipts = append(receipts, receipt)
						continue
					}
					if tx.Fee < minFee {
						receipt := types.NewErrReceipt(types.ErrFeeTooLow)
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
					receipt, err := execute.Exec(tx)
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
					elog.Debug("receipt of tx", "receipt=", feelog)
					receipts = append(receipts, feelog)
				}
				msg.Reply(client.NewMessage("", types.EventReceipts,
					&types.Receipts{receipts}))
			}
		}
	}()
}

func (exec *Execs) Close() {
	elog.Info("exec module closed")
}

//执行器 -> db 环境
type Execute struct {
	cache     map[string][]byte
	db        *DataBase
	height    int64
	blocktime int64
}

func NewExecute(stateHash []byte, q *queue.Queue, height, blocktime int64) *Execute {
	return &Execute{make(map[string][]byte), NewDataBase(q, stateHash), height, blocktime}
}

func (e *Execute) ProcessFee(tx *types.Transaction) (*types.Receipt, error) {
	accFrom := account.LoadAccount(e, account.PubKeyToAddress(tx.Signature.Pubkey).String())
	if accFrom.GetBalance()-tx.Fee >= 0 {
		receiptBalance := &types.ReceiptBalance{accFrom.GetBalance(), accFrom.GetBalance() - tx.Fee, -tx.Fee}
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		account.SaveAccount(e, accFrom)
		return cutFeeReceipt(accFrom, receiptBalance), nil
	}
	return nil, types.ErrNoBalance
}

func cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptBalance) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecPack, account.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}

func (e *Execute) Exec(tx *types.Transaction) (*types.Receipt, error) {
	elog.Info("exec", "execer", string(tx.Execer))
	exec, err := execdrivers.LoadExecute(string(tx.Execer))
	if err != nil {
		exec, err = execdrivers.LoadExecute("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetDB(e)
	exec.SetEnv(e.height, e.blocktime)
	return exec.Exec(tx)
}

func (e *Execute) Get(key []byte) (value []byte, err error) {
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

func (e *Execute) Set(key []byte, value []byte) error {
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
