package client_test

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type mockMempool struct {
}

func (m *mockMempool) SetQueueClient(q queue.Queue) {
	go func() {
		mempoolKey := "mempool"
		client := q.Client()
		client.Sub(mempoolKey)
		for msg := range client.Recv() {
			log.Debug("receive ok", "msg", msg)
			switch msg.Ty {
			case types.EventTx:
				msg.Reply(client.NewMessage(mempoolKey, types.EventReply, &types.Reply{IsOk: true, Msg: []byte("word")}))
			case types.EventTxList:
				msg.Reply(client.NewMessage(mempoolKey, types.EventReplyTxList, &types.ReplyTxList{}))
			case types.EventGetMempool:
				msg.Reply(client.NewMessage(mempoolKey, types.EventReplyTxList, &types.ReplyTxList{}))
			case types.EventGetLastMempool:
				msg.Reply(client.NewMessage(mempoolKey, types.EventReplyTxList, &types.ReplyTxList{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockMempool) Close() {
}
