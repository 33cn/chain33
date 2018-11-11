package client_test

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
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
				msg.Reply(client.NewMessage(topic, types.EventBlockChainQuery, &types.Reply{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockExecs) Close() {
}
