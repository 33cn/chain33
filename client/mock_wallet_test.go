// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type mockWallet struct {
}

func (m *mockWallet) SetQueueClient(q queue.Queue) {
	go func() {
		walletKey := "wallet"
		client := q.Client()
		client.Sub(walletKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventCloseTickets:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.ReplyHashes{}))
			case types.EventLocalGet:
				msg.Reply(client.NewMessage(walletKey, types.EventLocalReplyValue, &types.LocalReplyValue{}))
			case types.EventLocalList:
				msg.Reply(client.NewMessage(walletKey, types.EventLocalReplyValue, &types.LocalReplyValue{}))
			case types.EventWalletExecutor:
				if req, ok := msg.GetData().(*types.ChainExecutor); ok {
					switch req.FuncName {
					case "WalletGetAccountList":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccounts{}))
					case "NewAccount":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccount{}))
					case "WalletTransactionList":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletTxDetails{}))
					case "WalletImportPrivkey":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccount{}))
					case "DumpPrivkey":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplyString{}))
					case "WalletSendToAddress":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplyHash{}))
					case "WalletSetLabel":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccount{}))
					case "WalletMergeBalance":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplyHashes{}))
					case "GenSeed":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplySeed{}))
					case "GetSeed":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplySeed{}))
					case "GetWalletStatus":
						msg.Reply(client.NewMessage(walletKey, types.EventReplyWalletStatus, &types.WalletStatus{IsWalletLock: true, IsAutoMining: false, IsHasSeed: true, IsTicketLock: false}))
					case "SignRawTx":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplySignRawTx{}))
					case "FatalFailure":
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Int32{}))
					default:
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
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

func (m *mockWallet) Close() {
}
