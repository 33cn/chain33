// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"bytes"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
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
					if bytes.Equal(req.Hash, []byte("case1")) {
						msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetail, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventTransactionDetail, &types.TransactionDetail{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetTransactionByHash:
				if req, ok := msg.GetData().(*types.ReqHashes); ok {
					if len(req.GetHashes()) > 0 && bytes.Equal(req.Hashes[0], []byte("case1")) {
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
					if bytes.Equal(req.Hash, []byte("case1")) {
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

			case types.EventGetSeqByHash:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.Int64{Data: 1}))
			case types.EventGetBlockBySeq:
				if req, ok := msg.GetData().(*types.Int64); ok {
					// just for cover
					if req.Data == 10 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.Reply{IsOk: false, Msg: []byte("not support")}))
					}
					msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.BlockSeq{Num: 1}))
				}
			case types.EventIsSync:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyIsSync, &types.IsCaughtUp{}))
			case types.EventIsNtpClockSync:
				msg.Reply(client.NewMessage(blockchainKey, types.EventReplyIsNtpClockSync, &types.IsNtpClockSync{}))
			case types.EventGetLastHeader:
				msg.Reply(client.NewMessage(blockchainKey, types.EventHeader, &types.Header{}))
			case types.EventLocalGet:
				if req, ok := msg.GetData().(*types.LocalDBGet); ok {
					if len(req.Keys) > 0 && bytes.Equal(req.Keys[0], []byte("TotalFeeKey:case1")) {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.LocalReplyValue{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventLocalList:
				if req, ok := msg.GetData().(*types.LocalDBList); ok {
					if len(req.Key) > 0 && bytes.Equal(req.Key, []byte("Statistics:TicketInfoOrder:Addr:case1")) {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.TicketMinerInfo{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventReplyQuery, &types.LocalReplyValue{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventLocalNew:
				msg.Reply(client.NewMessage(blockchainKey, types.EventLocalNew, &types.Int64{Data: 9999}))
			case types.EventLocalClose:
				msg.Reply(client.NewMessage(blockchainKey, types.EventLocalClose, nil))
			case types.EventLocalBegin:
				if req, ok := msg.GetData().(*types.Int64); ok && req.Data == 9999 {
					msg.Reply(client.NewMessage(blockchainKey, types.EventLocalBegin, nil))
				} else {
					msg.ReplyErr("transaction id must 9999", types.ErrInvalidParam)
				}
			case types.EventLocalCommit:
				if req, ok := msg.GetData().(*types.Int64); ok && req.Data == 9999 {
					msg.Reply(client.NewMessage(blockchainKey, types.EventLocalCommit, nil))
				} else {
					msg.ReplyErr("transaction id must 9999", types.ErrInvalidParam)
				}
			case types.EventLocalRollback:
				if req, ok := msg.GetData().(*types.Int64); ok && req.Data == 9999 {
					msg.Reply(client.NewMessage(blockchainKey, types.EventLocalRollback, nil))
				} else {
					msg.ReplyErr("transaction id must 9999", types.ErrInvalidParam)
				}
			case types.EventLocalSet:
				if req, ok := msg.GetData().(*types.LocalDBSet); ok && req.Txid == 9999 {
					msg.Reply(client.NewMessage(blockchainKey, types.EventLocalSet, nil))
				} else {
					msg.ReplyErr("transaction id must 9999", types.ErrInvalidParam)
				}
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockBlockChain) Close() {
}
