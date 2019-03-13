// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"bytes"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type mockMempool struct {
}

func (m *mockMempool) SetQueueClient(q queue.Queue) {
	go func() {
		mempoolKey := "mempool"
		client := q.Client()
		client.Sub(mempoolKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventTx:
				if req, ok := msg.GetData().(*types.Transaction); ok {
					if bytes.Equal([]byte("case1"), req.Execer) {
						msg.Reply(client.NewMessage(mempoolKey, types.EventReply, &types.Reply{IsOk: false, Msg: []byte("do not support abc")}))
					} else if bytes.Equal([]byte("case2"), req.Execer) {
						msg.Reply(client.NewMessage(mempoolKey, types.EventReply, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(mempoolKey, types.EventReply, &types.Reply{IsOk: true, Msg: []byte("word")}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventTxList:
				if req, ok := msg.GetData().(*types.TxHashList); ok {
					if req.Count == 1 {
						msg.Reply(client.NewMessage(mempoolKey, types.EventReplyTxList, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(mempoolKey, types.EventReplyTxList, &types.ReplyTxList{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetMempool:
				msg.Reply(client.NewMessage(mempoolKey, types.EventReplyTxList, &types.ReplyTxList{}))
			case types.EventGetLastMempool:
				msg.Reply(client.NewMessage(mempoolKey, types.EventReplyTxList, &types.ReplyTxList{}))
			case types.EventGetProperFee:
				msg.Reply(client.NewMessage(mempoolKey, types.EventReplyProperFee, &types.ReplyProperFee{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockMempool) Close() {
}
