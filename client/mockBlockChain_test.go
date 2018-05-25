package client_test

import (
	"bytes"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type mockBlockChain struct {
}

func (m *mockBlockChain) SetQueueClient(q queue.Queue) {
	go func() {
		blockchainKey := "blockchain"
		client := q.Client()
		client.Sub(blockchainKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetBlocks:
				if req, ok := msg.GetData().(*types.ReqBlocks); ok {
					if req.Start == 1 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.BlockDetails{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetTransactionByAddr:
				if req, ok := msg.GetData().(*types.ReqAddr); ok {
					if req.Flag == 1 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyTxInfo, &types.ReplyTxInfos{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventQueryTx:
				if req, ok := msg.GetData().(*types.ReqHash); ok {
					if bytes.Compare(req.Hash, []byte("case1")) == 0 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetail, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetail, &types.TransactionDetail{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetTransactionByHash:
				if req, ok := msg.GetData().(*types.ReqHashes); ok {
					if len(req.GetHashes()) > 0 && bytes.Compare(req.Hashes[0], []byte("case1")) == 0 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetails, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetails, &types.TransactionDetails{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetHeaders:
				if req, ok := msg.GetData().(*types.ReqBlocks); ok {
					if req.Start == 10 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventHeaders, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventHeaders, &types.Headers{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetBlockOverview:
				if req, ok := msg.GetData().(*types.ReqHash); ok {
					if bytes.Compare(req.Hash, []byte("case1")) == 0 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyBlockOverview, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyBlockOverview, &types.BlockOverview{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetAddrOverview:
				if req, ok := msg.GetData().(*types.ReqAddr); ok {
					if req.Addr == "case1" {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyAddrOverview, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyAddrOverview, &types.AddrOverview{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetBlockHash:
				if req, ok := msg.GetData().(*types.ReqInt); ok {
					if req.Height == 10 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlockHash, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlockHash, &types.ReplyHash{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventIsSync:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyIsSync, &types.IsCaughtUp{}))
			case types.EventIsNtpClockSync:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyIsNtpClockSync, &types.IsNtpClockSync{}))
			case types.EventGetLastHeader:
				msg.Reply(client.NewMessage(blockchainKey, types.EventHeader, &types.Header{}))
			case types.EventQuery:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.Header{Version: 10, Height: 10}))
			case types.EventLocalGet:
				if req, ok := msg.GetData().(*types.LocalDBGet); ok {
					if len(req.Keys) > 0 && bytes.Compare(req.Keys[0], []byte("TotalFeeKey:case1")) == 0 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.LocalReplyValue{}))
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

func (m *mockBlockChain) Close() {
}
