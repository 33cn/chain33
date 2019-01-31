package blockchain

import (
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (chain *BlockChain) procLocalDB(msgtype int64, msg queue.Message, reqnum chan struct{}) {
	switch msgtype {
	case types.EventLocalGet:
		go chain.processMsg(msg, reqnum, chain.localGet)
	case types.EventLocalList:
		go chain.processMsg(msg, reqnum, chain.localList)
	case types.EventLocalPrefixCount:
		go chain.processMsg(msg, reqnum, chain.localPrefixCount)
	}
}

func (chain *BlockChain) localGet(msg queue.Message) {
	keys := (msg.Data).(*types.LocalDBGet)
	values := chain.blockStore.Get(keys)
	msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, values))
}

func (chain *BlockChain) localList(msg queue.Message) {
	q := (msg.Data).(*types.LocalDBList)
	values := db.NewListHelper(chain.blockStore.db).List(q.Prefix, q.Key, q.Count, q.Direction)
	msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))
}

//获取指定前缀key的数量
func (chain *BlockChain) localPrefixCount(msg queue.Message) {
	Prefix := (msg.Data).(*types.ReqKey)
	counts := db.NewListHelper(chain.blockStore.db).PrefixCount(Prefix.Key)
	msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, &types.Int64{Data: counts}))
}
