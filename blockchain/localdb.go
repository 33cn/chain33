package blockchain

import (
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (chain *BlockChain) procLocalDB(msgtype int64, msg queue.Message, reqnum chan struct{}) {
	switch msgtype {
	case types.EventLocalGet:
		go chain.processMsg(msg, reqnum, chain.localGet)
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
	}
}

func (chain *BlockChain) localGet(msg queue.Message) {
	keys := (msg.Data).(*types.LocalDBGet)
	values := chain.blockStore.Get(keys)
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, values))
}

func (chain *BlockChain) localBegin(msg queue.Message) {
	tx, err := chain.blockStore.db.BeginTx()
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalBegin, err))
		return
	}
	id := common.StorePointer(tx)
	msg.Reply(chain.client.NewMessage("", types.EventLocalBegin, &types.Int64{Data: id}))
}

func (chain *BlockChain) localCommit(msg queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	common.RemovePointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalCommit, err))
		return
	}
	err = tx.(db.TxKV).Commit()
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalCommit, err))
		return
	}
	msg.Reply(chain.client.NewMessage("", types.EventLocalCommit, nil))
}

func (chain *BlockChain) localRollback(msg queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	common.RemovePointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalRollback, err))
		return
	}
	tx.(db.TxKV).Rollback()
	msg.Reply(chain.client.NewMessage("", types.EventLocalRollback, nil))
}

func (chain *BlockChain) localList(msg queue.Message) {
	q := (msg.Data).(*types.LocalDBList)
	var itdb db.IteratorDB = chain.blockStore.db
	if q.Txid > 0 {
		tx, err := common.GetPointer(q.Txid)
		if err != nil {
			msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, err))
		}
		itdb = tx.(db.IteratorDB)
	}
	values := db.NewListHelper(itdb).List(q.Prefix, q.Key, q.Count, q.Direction)
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))
}

//获取指定前缀key的数量
func (chain *BlockChain) localPrefixCount(msg queue.Message) {
	Prefix := (msg.Data).(*types.ReqKey)
	counts := db.NewListHelper(chain.blockStore.db).PrefixCount(Prefix.Key)
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &types.Int64{Data: counts}))
}
