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
				if req, ok := msg.GetData().(*types.ReqNewAccount); ok {
					if req.Label == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.WalletAccount{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletTransactionList:
				if req, ok := msg.GetData().(*types.ReqWalletTransactionList); ok {
					if req.Direction == 1 {
						msg.Reply(client.NewMessage(walletKey, types.EventTransactionDetails, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventTransactionDetails, &types.WalletTxDetails{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletImportprivkey:
				if req, ok := msg.GetData().(*types.ReqWalletImportPrivKey); ok {
					if req.Label == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.WalletAccount{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletSendToAddress:
				if req, ok := msg.GetData().(*types.ReqWalletSendToAddress); ok {
					if req.Note == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.ReplyHash{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletSetFee:
				if req, ok := msg.GetData().(*types.ReqWalletSetFee); ok {
					if req.Amount == 1000 {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{IsOk: true}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletSetLabel:
				if req, ok := msg.GetData().(*types.ReqWalletSetLabel); ok {
					if req.Label == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletAccount, &types.WalletAccount{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletMergeBalance:
				if req, ok := msg.GetData().(*types.ReqWalletMergeBalance); ok {
					if req.To == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.ReplyHashes{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletSetPasswd:
				if req, ok := msg.GetData().(*types.ReqWalletSetPasswd); ok {
					if req.GetOldpass() == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventWalletLock:
				msg.Reply(client.NewMessage(walletKey, types.EventWalletLock, &types.Reply{}))
			case types.EventWalletUnLock:
				if req, ok := msg.GetData().(*types.WalletUnLock); ok {
					if req.Passwd == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletUnLock, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventWalletUnLock, &types.Reply{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGenSeed:
				if req, ok := msg.GetData().(*types.GenSeedLang); ok {
					if req.Lang == 10 {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyGenSeed, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyGenSeed, &types.ReplySeed{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventSaveSeed:
				if req, ok := msg.GetData().(*types.SaveSeedByPw); ok {
					if req.Seed == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetSeed:
				if req, ok := msg.GetData().(*types.GetSeedByPw); ok {
					if req.Passwd == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyGetSeed, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyGetSeed, &types.ReplySeed{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventGetWalletStatus:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyWalletStatus, &types.WalletStatus{IsWalletLock: true, IsAutoMining: false, IsHasSeed: true, IsTicketLock: false}))
			case types.EventWalletAutoMiner:
				if req, ok := msg.GetData().(*types.MinerFlag); ok {
					if req.Flag == 10 {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReply, &types.Reply{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventDumpPrivkey:
				if req, ok := msg.GetData().(*types.ReqStr); ok {
					if req.ReqStr == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyPrivkey, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReplyPrivkey, &types.ReplyStr{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventCloseTickets:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyHashes, &types.ReplyHashes{}))
			case types.EventLocalGet:
				msg.Reply(client.NewMessage(walletKey, types.EventLocalReplyValue, &types.LocalReplyValue{}))
			case types.EventLocalList:
				msg.Reply(client.NewMessage(walletKey, types.EventLocalReplyValue, &types.LocalReplyValue{}))
			case types.EventSignRawTx:
				if req, ok := msg.GetData().(*types.ReqSignRawTx); ok {
					if req.Addr == "case1" {
						msg.Reply(client.NewMessage(walletKey, types.EventReplySignRawTx, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(walletKey, types.EventReplySignRawTx, &types.ReplySignRawTx{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			case types.EventFatalFailure:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyFatalFailure, &types.Int32{}))
			case types.EventShowPrivacyAccount:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyShowPrivacyAccount, &types.UTXOs{}))
			case types.EventShowPrivacyPK:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyShowPrivacyPK, &types.ReplyPrivacyPkPair{}))
			case types.EventShowPrivacyBalance:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyShowPrivacyBalance, &types.Account{}))
			case types.EventCreateUTXOs:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyCreateUTXOs, &types.Reply{}))
			case types.EventPublic2privacy:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyPublic2privacy, &types.Reply{}))
			case types.EventPrivacy2privacy:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyPrivacy2privacy, &types.Reply{}))
			case types.EventPrivacy2public:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyPrivacy2public, &types.Reply{}))
			case types.EventCreateTransaction:
				msg.Reply(client.NewMessage(walletKey, types.EventReplyCreateTransaction, &types.Reply{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockWallet) Close() {
}
