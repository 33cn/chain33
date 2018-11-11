// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type mockConsensus struct {
}

func (m *mockConsensus) SetQueueClient(q queue.Queue) {
	go func() {
		consensusKey := "consensus"
		client := q.Client()
		client.Sub(consensusKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetTicketCount:
				msg.Reply(client.NewMessage(consensusKey, types.EventReplyGetTicketCount, &types.Int64{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockConsensus) Close() {
}
