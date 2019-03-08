package blockchain

import (
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (chain *BlockChain) procLocalDB(msgtype int64, msg *queue.Message, reqnum chan struct{}) bool {
	switch msgtype {
	case types.EventLocalGet:
		go chain.processMsg(msg, reqnum, chain.localGet)
	case types.EventLocalSet:
		go chain.processMsg(msg, reqnum, chain.localSet)
	case types.EventLocalBegin:
		go chain.processMsg(msg, reqnum, chain.localBegin)
	case types.EventLocalCommit:
		go chain.processMsg(msg, reqnum, chain.localCommit)
	case types.EventLocalRollback:
		go chain.processMsg(msg, reqnum, chain.localRollback)
	case types.EventLocalList:
		go chain.processMsg(msg, reqnum, chain.localList)
	case types.EventLocalPrefixCount:
		go chain.processMsg(msg, reqnum, chain.localPrefixCount)
	case types.EventLocalNew:
		go chain.processMsg(msg, reqnum, chain.localNew)
	case types.EventLocalClose:
		go chain.processMsg(msg, reqnum, chain.localClose)
	default:
		return false
	}
	return true
}

func (chain *BlockChain) localGet(msg *queue.Message) {
	keys := (msg.Data).(*types.LocalDBGet)
	if keys.Txid == 0 {
		values := chain.blockStore.Get(keys)
		msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, values))
		return
	}
	tx, err := common.GetPointer(keys.Txid)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, err))
		return
	}
	var reply types.LocalReplyValue
	for i := 0; i < len(keys.Keys); i++ {
		key := keys.Keys[i]
		value, err := tx.(db.KVDB).Get(key)
		if err != nil {
			chainlog.Debug("localGet", "i", i, "key", string(key), "err", err)
		}
		reply.Values = append(reply.Values, value)
	}
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &reply))
}

//只允许设置 通过 transaction 来 set 信息
func (chain *BlockChain) localSet(msg *queue.Message) {
	kvs := (msg.Data).(*types.LocalDBSet)
	if kvs.Txid == 0 {
		msg.Reply(chain.client.NewMessage("", types.EventLocalSet, types.ErrNotSetInTransaction))
		return
	}
	txp, err := common.GetPointer(kvs.Txid)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalSet, err))
		return
	}
	tx := txp.(db.KVDB)
	for i := 0; i < len(kvs.KV); i++ {
		err := tx.Set(kvs.KV[i].Key, kvs.KV[i].Value)
		if err != nil {
			chainlog.Error("localSet", "i", i, "key", string(kvs.KV[i].Key), "err", err)
		}
	}
	msg.Reply(chain.client.NewMessage("", types.EventLocalSet, nil))
}

//创建 localdb transaction
func (chain *BlockChain) localNew(msg *queue.Message) {
	tx := db.NewLocalDB(chain.blockStore.db)
	id := common.StorePointer(tx)
	msg.Reply(chain.client.NewMessage("", types.EventLocalNew, &types.Int64{Data: id}))
}

//关闭 localdb transaction
func (chain *BlockChain) localClose(msg *queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	_, err := common.GetPointer(id)
	common.RemovePointer(id)
	msg.Reply(chain.client.NewMessage("", types.EventLocalClose, err))
}

func (chain *BlockChain) localBegin(msg *queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalBegin, err))
		return
	}
	tx.(db.KVDB).Begin()
	msg.Reply(chain.client.NewMessage("", types.EventLocalBegin, nil))
}

func (chain *BlockChain) localCommit(msg *queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalCommit, err))
		return
	}
	err = tx.(db.KVDB).Commit()
	msg.Reply(chain.client.NewMessage("", types.EventLocalCommit, err))
}

func (chain *BlockChain) localRollback(msg *queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalRollback, err))
		return
	}
	tx.(db.KVDB).Rollback()
	msg.Reply(chain.client.NewMessage("", types.EventLocalRollback, nil))
}

func (chain *BlockChain) localList(msg *queue.Message) {
	q := (msg.Data).(*types.LocalDBList)
	var values [][]byte
	if q.Txid > 0 {
		tx, err := common.GetPointer(q.Txid)
		if err != nil {
			msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, err))
			return
		}
		values, err = tx.(db.KVDB).List(q.Prefix, q.Key, q.Count, q.Direction)
		if err != nil {
			msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, err))
			return
		}
	} else {
		values = db.NewListHelper(chain.blockStore.db).List(q.Prefix, q.Key, q.Count, q.Direction)
	}
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))
}

//获取指定前缀key的数量
func (chain *BlockChain) localPrefixCount(msg *queue.Message) {
	Prefix := (msg.Data).(*types.ReqKey)
	counts := db.NewListHelper(chain.blockStore.db).PrefixCount(Prefix.Key)
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &types.Int64{Data: counts}))
}
