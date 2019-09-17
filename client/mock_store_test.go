// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type mockStore struct {
}

func (m *mockStore) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("store")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventStoreSet:
				msg.Reply(client.NewMessage("store", types.EventStoreSetReply, &types.ReplyHash{}))
			case types.EventStoreGet:
				msg.Reply(client.NewMessage("store", types.EventStoreGetReply, &types.StoreReplyValue{}))
			case types.EventStoreMemSet:
				msg.Reply(client.NewMessage("store", types.EventStoreSetReply, &types.ReplyHash{}))
			case types.EventStoreCommit:
				msg.Reply(client.NewMessage("store", types.EventStoreCommit, &types.ReplyHash{}))
			case types.EventStoreRollback:
				msg.Reply(client.NewMessage("store", types.EventStoreRollback, &types.ReplyHash{}))
			case types.EventStoreDel:
				msg.Reply(client.NewMessage("store", types.EventStoreDel, &types.ReplyHash{}))
			case types.EventStoreGetTotalCoins:
				if req, ok := msg.GetData().(*types.IterateRangeByStateHash); ok {
					if req.Count == 10 {
						msg.Reply(client.NewMessage("store", types.EventStoreGetReply, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage("store", types.EventStoreGetReply, &types.ReplyGetTotalCoins{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockStore) Close() {
}
