package store

//store package store the world - state data
import (
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var slog = log.New("module", "localstore")

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	slog.SetHandler(log.DiscardHandler())
}

type LocalStore struct {
	db      dbm.DB
	qclient queue.Client
	done    chan struct{}
}

//driver
//dbpath
func New(cfg *types.LocalStore) *LocalStore {
	db := dbm.NewDB("localstore", cfg.Driver, cfg.DbPath)
	store := &LocalStore{db: db}
	store.done = make(chan struct{}, 1)
	return store
}

func (store *LocalStore) Close() {
	store.qclient.Close()
	<-store.done
	store.db.Close()
	slog.Info("store module closed")
}

func (store *LocalStore) SetQueue(q *queue.Queue) {
	store.qclient = q.NewClient()
	client := store.qclient
	client.Sub("localstore")
	//recv 消息的处理
	go func() {
		for msg := range client.Recv() {
			slog.Info("stroe recv", "msg", msg)
			store.processMessage(msg)
		}
		store.done <- struct{}{}
	}()
}

func (store *LocalStore) processMessage(msg queue.Message) {
	client := store.qclient
	if msg.Ty == types.EventLocalSet {
		datas := msg.GetData().(*types.LocalDBSet)
		batch := store.db.NewBatch(true)
		for i := 0; i < len(datas.KV); i++ {
			kv := datas.KV[i]
			if kv == nil {
				continue
			}
			if kv.Value == nil {
				batch.Delete(kv.Key)
			} else {
				batch.Set(kv.Key, kv.Value)
			}
		}
		batch.Write()
		msg.ReplyErr("EventLocalSet", nil)
	} else if msg.Ty == types.EventLocalGet {
		datas := msg.GetData().(*types.LocalDBGet)
		values := &types.LocalReplyValue{}
		for i := 0; i < len(datas.Keys); i++ {
			value := store.db.Get(datas.Keys[i])
			values.Values = append(values.Values, value)
		}
		msg.Reply(client.NewMessage("", types.EventLocalGet, values))
	} else if msg.Ty == types.EventLocalList { //只是在内存中set 一下，并不改变状态
		values := &types.LocalReplyValue{}
		datas := msg.GetData().(*types.LocalDBList)
		values.Values = store.db.List(datas.Prefix, datas.Key, datas.Direction, datas.Count)
		msg.Reply(client.NewMessage("", types.EventLocalList, values))
	}
}
