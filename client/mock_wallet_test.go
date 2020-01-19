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
				if msg.GetData().(*types.ChainExecutor).FuncName == "WalletGetAccountList" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccounts{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "NewAccount" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccount{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletTransactionList" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletTxDetails{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletImportPrivkey" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccount{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "DumpPrivkey" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplyString{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletSendToAddress" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplyHash{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletSetFee" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletSetLabel" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.WalletAccount{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletMergeBalance" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplyHashes{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletSetPasswd" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletLock" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "WalletUnLock" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "GenSeed" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplySeed{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "SaveSeed" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "GetSeed" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplySeed{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "GetWalletStatus" {
					msg.Reply(client.NewMessage(walletKey, types.EventReplyWalletStatus, &types.WalletStatus{IsWalletLock: true, IsAutoMining: false, IsHasSeed: true, IsTicketLock: false}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "SignRawTx" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.ReplySignRawTx{}))
				} else if msg.GetData().(*types.ChainExecutor).FuncName == "FatalFailure" {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Int32{}))
				} else {
					msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
				}
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockWallet) Close() {
}
