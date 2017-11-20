package execs

//store package store the world - state data
import (
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
	_ "code.aliyun.com/chain33/chain33/execs/coins"
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
	slog.SetHandler(log.DiscardHandler())
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
			if msg.Ty == types.ReplyTxList {
				datas := msg.GetData().(*types.ReplyTxList)
				execute := NewExecute()
				var receipts []*types.Receipt
				for i := 0; i < len(datas.Txs); i++ {
					receipt := execute.Exec(datas.Txs[i])
					receipts = append(receipts, receipt)
				}
				msg.Reply(client.NewMessage("", types.EventReceipts,
					&types.Receipts{receipts})
			}
		}
	}()
}

type Executer interface {
	Exec(tx *types.Transaction) (*types.Receipt)
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
	db *DataBase
}

func NewExecute() *Execute {
	return &Execute{make(map[string][]byte), NewDataBase()}
}

func (exec *Execute) Exec(tx *types.Transaction) (*types.Receipt) {
	exec, err := LoadExecute(string(tx.Execer))
	if err != nil {
		return &types.Receipt{}
	}
}