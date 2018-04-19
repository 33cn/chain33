package client

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type mockP2P struct {
}

func (m *mockP2P) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub(p2pKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventPeerInfo:
				msg.Reply(client.NewMessage(mempoolKey, types.EventPeerList, &types.PeerList{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockP2P) Close() {
}
