package client_test

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type mockExecs struct {
}

func (m *mockExecs) SetQueueClient(q queue.Queue) {
	go func() {
		topic := "execs"
		client := q.Client()
		client.Sub(topic)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventBlockChainQuery:
				msg.Reply(client.NewMessage(topic, types.EventBlockChainQuery, &types.ResUTXOGlobalIndex{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockExecs) Close() {
}
