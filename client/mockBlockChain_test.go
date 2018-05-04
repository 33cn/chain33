package client

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type mockBlockChain struct {
}

func (m *mockBlockChain) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub(blockchainKey)
		for msg := range client.Recv() {
			log.Debug("receive ok", "msg", msg)
			switch msg.Ty {
			case types.EventGetBlocks:
				msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.BlockDetails{}))
			case types.EventGetTransactionByAddr:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyTxInfo, &types.ReplyTxInfos{}))
			case types.EventQueryTx:
				msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetail, &types.TransactionDetail{}))
			case types.EventGetTransactionByHash:
				msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetails, &types.TransactionDetails{}))
			case types.EventGetHeaders:
				msg.Reply(client.NewMessage(blockchainKey, types.EventHeaders, &types.Headers{}))
			case types.EventGetBlockOverview:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyBlockOverview, &types.BlockOverview{}))
			case types.EventGetAddrOverview:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyAddrOverview, &types.AddrOverview{}))
			case types.EventGetBlockHash:
				msg.Reply(client.NewMessage(blockchainKey, types.EventBlockHash, &types.ReplyHash{}))
			case types.EventIsSync:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyIsSync, &types.IsCaughtUp{}))
			case types.EventIsNtpClockSync:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyIsNtpClockSync, &types.IsNtpClockSync{}))
			case types.EventGetLastHeader:
				msg.Reply(client.NewMessage(blockchainKey, types.EventHeader, &types.Header{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockBlockChain) Close() {
}
