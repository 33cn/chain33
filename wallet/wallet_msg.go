package wallet

import (
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

func (wallet *Wallet) ProcRecvMsg() {
	defer wallet.wg.Done()
	for msg := range wallet.client.Recv() {
		walletlog.Debug("wallet recv", "msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)
		msgtype := msg.Ty
		switch msgtype {

		case types.EventWalletGetAccountList:
			WalletAccounts, err := wallet.ProcGetAccountList()
			if err != nil {
				walletlog.Error("ProcGetAccountList", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccountList, err))
			} else {
				walletlog.Debug("process WalletAccounts OK")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccountList, WalletAccounts))
			}

		case types.EventWalletAutoMiner:
			flag := msg.GetData().(*types.MinerFlag).Flag
			if flag == 1 {
				wallet.walletStore.db.Set([]byte("WalletAutoMiner"), []byte("1"))
			} else {
				wallet.walletStore.db.Set([]byte("WalletAutoMiner"), []byte("0"))
			}
			wallet.setAutoMining(flag)
			wallet.flushTicket()
			msg.ReplyErr("WalletSetAutoMiner", nil)

		case types.EventWalletGetTickets:
			tickets, privs, err := wallet.GetTickets(1)
			if err != nil {
				walletlog.Error("GetTickets", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("consensus", types.EventWalletTickets, err))
			} else {
				tks := &types.ReplyWalletTickets{tickets, privs}
				walletlog.Debug("process GetTickets OK")
				msg.Reply(wallet.client.NewMessage("consensus", types.EventWalletTickets, tks))
			}

		case types.EventNewAccount:
			NewAccount := msg.Data.(*types.ReqNewAccount)
			WalletAccount, err := wallet.ProcCreateNewAccount(NewAccount)
			if err != nil {
				walletlog.Error("ProcCreateNewAccount", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}

		case types.EventWalletTransactionList:
			WalletTxList := msg.Data.(*types.ReqWalletTransactionList)
			TransactionDetails, err := wallet.ProcWalletTxList(WalletTxList)
			if err != nil {
				walletlog.Error("ProcWalletTxList", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventTransactionDetails, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventTransactionDetails, TransactionDetails))
			}

		case types.EventWalletImportprivkey:
			ImportPrivKey := msg.Data.(*types.ReqWalletImportPrivKey)
			WalletAccount, err := wallet.ProcImportPrivKey(ImportPrivKey)
			if err != nil {
				walletlog.Error("ProcImportPrivKey", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}
			wallet.flushTicket()

		case types.EventWalletSendToAddress:
			SendToAddress := msg.Data.(*types.ReqWalletSendToAddress)
			ReplyHash, err := wallet.ProcSendToAddress(SendToAddress)
			if err != nil {
				walletlog.Error("ProcSendToAddress", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, ReplyHash))
			}

		case types.EventWalletSetFee:
			WalletSetFee := msg.Data.(*types.ReqWalletSetFee)

			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletSetFee(WalletSetFee)
			if err != nil {
				walletlog.Error("ProcWalletSetFee", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletSetLabel:
			WalletSetLabel := msg.Data.(*types.ReqWalletSetLabel)
			WalletAccount, err := wallet.ProcWalletSetLabel(WalletSetLabel)

			if err != nil {
				walletlog.Error("ProcWalletSetLabel", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}

		case types.EventWalletMergeBalance:
			MergeBalance := msg.Data.(*types.ReqWalletMergeBalance)
			ReplyHashes, err := wallet.ProcMergeBalance(MergeBalance)
			if err != nil {
				walletlog.Error("ProcMergeBalance", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, ReplyHashes))
			}

		case types.EventWalletSetPasswd:
			SetPasswd := msg.Data.(*types.ReqWalletSetPasswd)

			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletSetPasswd(SetPasswd)
			if err != nil {
				walletlog.Error("ProcWalletSetPasswd", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletLock:
			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletLock()
			if err != nil {
				walletlog.Error("ProcWalletLock", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletUnLock:
			WalletUnLock := msg.Data.(*types.WalletUnLock)
			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletUnLock(WalletUnLock)
			if err != nil {
				walletlog.Error("ProcWalletUnLock", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))
			wallet.flushTicket()

		case types.EventAddBlock:
			block := msg.Data.(*types.BlockDetail)
			wallet.ProcWalletAddBlock(block)
			walletlog.Debug("wallet add block --->", "height", block.Block.GetHeight())

		case types.EventDelBlock:
			block := msg.Data.(*types.BlockDetail)
			wallet.ProcWalletDelBlock(block)
			walletlog.Debug("wallet del block --->", "height", block.Block.GetHeight())

		//seed
		case types.EventGenSeed:
			genSeedLang := msg.Data.(*types.GenSeedLang)
			replySeed, err := wallet.genSeed(genSeedLang.Lang)
			if err != nil {
				walletlog.Error("genSeed", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGenSeed, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGenSeed, replySeed))
			}

		case types.EventGetSeed:
			Pw := msg.Data.(*types.GetSeedByPw)
			seed, err := wallet.getSeed(Pw.Passwd)
			if err != nil {
				walletlog.Error("getSeed", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGetSeed, err))
			} else {
				var replySeed types.ReplySeed
				replySeed.Seed = seed
				//walletlog.Error("EventGetSeed", "seed", seed)
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGetSeed, &replySeed))
			}

		case types.EventSaveSeed:
			saveseed := msg.Data.(*types.SaveSeedByPw)
			var reply types.Reply
			reply.IsOk = true
			ok, err := wallet.saveSeed(saveseed.Passwd, saveseed.Seed)
			if !ok {
				walletlog.Error("saveSeed", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventGetWalletStatus:
			s := wallet.GetWalletStatus()
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyWalletStatus, s))

		case types.EventDumpPrivkey:
			addr := msg.Data.(*types.ReqStr)
			privkey, err := wallet.ProcDumpPrivkey(addr.ReqStr)
			if err != nil {
				walletlog.Error("ProcDumpPrivkey", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivkey, err))
			} else {
				var replyStr types.ReplyStr
				replyStr.Replystr = privkey
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivkey, &replyStr))
			}

		case types.EventCloseTickets:
			hashes, err := wallet.forceCloseTicket(wallet.GetHeight() + 1)
			if err != nil {
				walletlog.Error("closeTicket", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, hashes))
				go func() {
					if len(hashes.Hashes) > 0 {
						wallet.waitTxs(hashes.Hashes)
						wallet.flushTicket()
					}
				}()
			}

		case types.EventSignRawTx:
			unsigned := msg.GetData().(*types.ReqSignRawTx)
			txHex, err := wallet.ProcSignRawTx(unsigned)
			if err != nil {
				walletlog.Error("EventSignRawTx", "err", err)
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySignRawTx, err))
			} else {
				walletlog.Info("Reply EventSignRawTx", "msg", msg)
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySignRawTx, &types.ReplySignRawTx{TxHex: txHex}))
			}
		case types.EventErrToFront: //收到系统发生致命性错误事件
			reportErrEvent := msg.Data.(*types.ReportErrEvent)
			wallet.setFatalFailure(reportErrEvent)
			walletlog.Debug("EventErrToFront")

		case types.EventFatalFailure: //定时查询是否有致命性故障产生
			fatalFailure := wallet.getFatalFailure()
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyFatalFailure, &types.Int32{Data: fatalFailure}))
		case types.EventShowPrivacyAccountSpend:
			reqPrivBal4AddrToken := msg.Data.(*types.ReqPrivBal4AddrToken)
			UTXOs, err := wallet.showPrivacyAccountsSpend(reqPrivBal4AddrToken)
			if err != nil {
				walletlog.Error("showPrivacyAccountSpend", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccountSpend, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccountSpend, UTXOs))
			}
		case types.EventShowPrivacyPK:
			reqAddr := msg.Data.(*types.ReqStr)
			replyPrivacyPair, err := wallet.showPrivacyPkPair(reqAddr)
			if err != nil {
				walletlog.Error("showPrivacyPkPair", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyPK, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyPK, replyPrivacyPair))
			}
		case types.EventPublic2privacy:
			reqPub2Pri := msg.Data.(*types.ReqPub2Pri)
			replyHash, err := wallet.procPublic2PrivacyV2(reqPub2Pri)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procPublic2Privacy", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPublic2privacy, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procPublic2Privacy", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPublic2privacy, &reply))
			}

		case types.EventPrivacy2privacy:
			reqPri2Pri := msg.Data.(*types.ReqPri2Pri)
			replyHash, err := wallet.procPrivacy2PrivacyV2(reqPri2Pri)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procPrivacy2Privacy", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2privacy, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procPrivacy2Privacy", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2privacy, &reply))
			}
		case types.EventPrivacy2public:
			reqPri2Pub := msg.Data.(*types.ReqPri2Pub)
			replyHash, err := wallet.procPrivacy2PublicV2(reqPri2Pub)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procPrivacy2Public", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2public, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procPrivacy2Public", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2public, &reply))
			}
		case types.EventCreateUTXOs:
			reqCreateUTXOs := msg.Data.(*types.ReqCreateUTXOs)
			replyHash, err := wallet.procCreateUTXOs(reqCreateUTXOs)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procCreateUTXOs", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyCreateUTXOs, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procCreateUTXOs", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyCreateUTXOs, &reply))
			}
		case types.EventCreateTransaction:
			req := msg.Data.(*types.ReqCreateTransaction)
			replyHash, err := wallet.procCreateTransaction(req)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procCreateTransaction", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyCreateTransaction, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procCreateTransaction", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyCreateTransaction, &reply))
			}
		case types.EventSendTxHashToWallet:
			req := msg.Data.(*types.ReqCreateCacheTxKey)
			replyHash, err := wallet.procSendTxHashToWallet(req)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procSendTxHashToWallet", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySendTxHashToWallet, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procSendTxHashToWallet", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySendTxHashToWallet, &reply))
			}
		case types.EventQueryCacheTransaction:
			req := msg.Data.(*types.ReqCacheTxList)
			reply, err := wallet.procReqCacheTxList(req)
			if err != nil {
				walletlog.Error("procReqCacheTxList", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyQueryCacheTransaction, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyQueryCacheTransaction, reply))
			}
		case types.EventDeleteCacheTransaction:
			req := msg.Data.(*types.ReqCreateCacheTxKey)
			replyHash, err := wallet.procDeleteCacheTransaction(req)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procDeleteCacheTransaction", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyDeleteCacheTransaction, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procDeleteCacheTransaction", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyDeleteCacheTransaction, &reply))
			}
		case types.EventPrivacyAccountInfo:
			req := msg.Data.(*types.ReqPPrivacyAccount)
			reply, err := wallet.procPrivacyAccountInfo(req)
			if err != nil {
				walletlog.Error("procPrivacyAccountInfo", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacyAccountInfo, err))
			} else {
				walletlog.Info("procPrivacyAccountInfo", "req", req)
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacyAccountInfo, reply))
			}

		default:
			walletlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
		walletlog.Debug("end process", "msg.id", msg.Id)
	}
}
