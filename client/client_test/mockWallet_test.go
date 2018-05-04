package client_test

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
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
			case types.EventWalletGetAccountList:
				msg.Reply(client.NewMessage(walletKey, types.EventWalletAccountList, &types.WalletAccounts{}))
			case types.EventNewAccount:
				msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.WalletAccount{}))
			case types.EventWalletTransactionList:
				msg.Reply(client.NewMessage(walletKey, types.EventTransactionDetails, &types.WalletTxDetails{}))
			case types.EventWalletImportprivkey:
				msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.WalletAccount{}))
			case types.EventWalletSendToAddress:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.ReplyHash{}))
			case types.EventWalletSetFee:
				msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{IsOk: true}))
			case types.EventWalletSetLabel:
				msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.WalletAccount{}))
			case types.EventWalletMergeBalance:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.ReplyHashes{}))
			case types.EventWalletSetPasswd:
				msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
			case types.EventGenSeed:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyGenSeed, &types.ReplySeed{}))
			case types.EventSaveSeed:
				msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
			case types.EventGetSeed:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyGetSeed, &types.ReplySeed{}))
			case types.EventGetWalletStatus:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyWalletStatus, &types.WalletStatus{}))
			case types.EventWalletAutoMiner:
				msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
			case types.EventDumpPrivkey:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyPrivkey, &types.ReplyStr{}))
			case types.EventCloseTickets:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.ReplyHashes{}))
			case types.EventLocalGet:
				msg.Reply(client.NewMessage(walletKey, types.EventLocalReplyValue, &types.LocalReplyValue{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockWallet) Close() {
}
