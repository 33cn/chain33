package execs

//store package store the world - state data
import (
	"fmt"

	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	_ "code.aliyun.com/chain33/chain33/execs/coins"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

/*
模块主要的功能：

//批量写
1. EventStoreSet(stateHash, (k1,v1),(k2,v2),(k3,v3)) -> 返回 stateHash

//批量读
2. EventStoreGet(stateHash, k1,k2,k3)

*/

var elog = log.New("module", "execs")
var zeroHash [32]byte

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
				execute := NewExecute(datas.StateHash, q)
				var receipts []*types.Receipt
				for i := 0; i < len(datas.Txs); i++ {
					receipt := execute.Exec(datas.Txs[i])
					receipts = append(receipts, receipt)
				}
				msg.Reply(client.NewMessage("", types.EventReceipts,
					&types.Receipts{receipts}))
			}
		}
	}()
}

type Executer interface {
	SetDB(dbm.KVDB)
	Exec(tx *types.Transaction) *types.Receipt
}

var (
	drivers = make(map[string]Executer)
)

func Register(name string, driver Executer) {
	if driver == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func LoadExecute(name string) (c Executer, err error) {
	c, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}
	return c, nil
}

//执行器 -> db 环境
type Execute struct {
	cache map[string][]byte
	db    *DataBase
}

func NewExecute(stateHash []byte, q *queue.Queue) *Execute {
	return &Execute{make(map[string][]byte), NewDataBase(q, stateHash)}
}

func (e *Execute) Exec(tx *types.Transaction) *types.Receipt {
	exec, err := LoadExecute(string(tx.Execer))
	if err != nil {
		exec, err = LoadExecute("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetDB(e)
	return exec.Exec(tx)
}

func (e *Execute) Get(key []byte) (value []byte, err error) {
	if value, ok := e.cache[string(key)]; ok {
		return value, nil
	}
	return e.db.Get(key)
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
