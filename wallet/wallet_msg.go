package wallet

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

func (wallet *Wallet) initFuncMap() {
	wallet.funcmap.Init()

	wallet.funcmap.Register(types.EventWalletGetAccountList, wallet.onWalletGetAccountList)
	wallet.funcmap.Register(types.EventWalletAutoMiner, wallet.onWalletAutoMiner)
	wallet.funcmap.Register(types.EventWalletGetTickets, wallet.onWalletGetTickets)
	wallet.funcmap.Register(types.EventNewAccount, wallet.onNewAccount)
	wallet.funcmap.Register(types.EventWalletTransactionList, wallet.onWalletTransactionList)
	wallet.funcmap.Register(types.EventWalletImportprivkey, wallet.onWalletImportprivkey)
	wallet.funcmap.Register(types.EventWalletSendToAddress, wallet.onWalletSendToAddress)
	wallet.funcmap.Register(types.EventWalletSetFee, wallet.onWalletSetFee)
	wallet.funcmap.Register(types.EventWalletSetLabel, wallet.onWalletSetLabel)
	wallet.funcmap.Register(types.EventWalletMergeBalance, wallet.onWalletMergeBalance)
	wallet.funcmap.Register(types.EventWalletSetPasswd, wallet.ontWalletSetPasswd)
	wallet.funcmap.Register(types.EventWalletLock, wallet.onWalletLock)
	wallet.funcmap.Register(types.EventWalletUnLock, wallet.onWalletUnLock)
	wallet.funcmap.Register(types.EventAddBlock, wallet.onAddBlock)
	wallet.funcmap.Register(types.EventDelBlock, wallet.onDelBlock)
	wallet.funcmap.Register(types.EventGenSeed, wallet.onGenSeed)
	wallet.funcmap.Register(types.EventGetSeed, wallet.onGetSeed)
	wallet.funcmap.Register(types.EventSaveSeed, wallet.onSaveSeed)
	wallet.funcmap.Register(types.EventGetWalletStatus, wallet.onGetWalletStatus)
	wallet.funcmap.Register(types.EventDumpPrivkey, wallet.onDumpPrivKey)
	wallet.funcmap.Register(types.EventCloseTickets, wallet.onCloseTickets)
	wallet.funcmap.Register(types.EventSignRawTx, wallet.onSignRawTx)
	wallet.funcmap.Register(types.EventErrToFront, wallet.onErrToFront)
	wallet.funcmap.Register(types.EventFatalFailure, wallet.onFatalFailure)
}

func (wallet *Wallet) ProcRecvMsg() {
	defer wallet.wg.Done()
	for msg := range wallet.client.Recv() {
		walletlog.Debug("wallet recv", "msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)

		// 优先处理钱包中的基础业务
		funcExisted, topic, retty, reply, err := wallet.funcmap.Process(&msg)
		if funcExisted {
			if err != nil {
				msg.Reply(wallet.api.NewMessage(topic, retty, err))
			} else {
				msg.Reply(wallet.api.NewMessage(topic, retty, reply))
			}
		}

		// 再处理业务逻辑中的业务
		existed := false
		for _, policy := range wallet.policyContainer {
			existed, err := policy.OnRecvQueueMsg(&msg)
			if existed && err != nil {
				walletlog.Error("ProcRecvMsg", "OnRecvQueueMsg error ", err, "msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)
			}
		}

		if !funcExisted && !existed {
			walletlog.Error("ProcRecvMsg", "Do not support msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)
			continue
		}
		walletlog.Debug("end process", "msg.id", msg.Id)
	}
}

func (wallet *Wallet) onWalletGetAccountList(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletAccountList)

	reply, err := wallet.ProcGetAccountList()
	if err != nil {
		walletlog.Error("onWalletGetAccountList", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletAutoMiner(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletAutoMiner)
	req, ok := msg.Data.(*types.MinerFlag)
	if !ok {
		walletlog.Error("onWalletAutoMiner", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	if req.Flag == 1 {
		wallet.walletStore.db.Set([]byte("WalletAutoMiner"), []byte("1"))
	} else {
		wallet.walletStore.db.Set([]byte("WalletAutoMiner"), []byte("0"))
	}
	wallet.setAutoMining(req.Flag)
	wallet.flushTicket()
	return topic, retty, &types.Reply{IsOk: true}, nil
}

func (wallet *Wallet) onWalletGetTickets(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletTickets)

	tickets, privs, err := wallet.GetTickets(1)
	tks := &types.ReplyWalletTickets{tickets, privs}
	return topic, retty, tks, err
}

func (wallet *Wallet) onNewAccount(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletAccount)
	req, ok := msg.Data.(*types.ReqNewAccount)
	if !ok {
		walletlog.Error("onNewAccount", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply, err := wallet.ProcCreateNewAccount(req)
	if err != nil {
		walletlog.Error("onNewAccount", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletTransactionList(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventTransactionDetails)
	req, ok := msg.Data.(*types.ReqWalletTransactionList)
	if !ok {
		walletlog.Error("onWalletTransactionList", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply, err := wallet.ProcWalletTxList(req)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletImportprivkey(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventTransactionDetails)
	req, ok := msg.Data.(*types.ReqWalletImportPrivKey)
	if !ok {
		walletlog.Error("onWalletImportprivkey", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply, err := wallet.ProcImportPrivKey(req)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "err", err.Error())
	}
	// TODO: 导入成功才需要刷新吧
	wallet.flushTicket()
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletSendToAddress(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyHashes)
	req, ok := msg.Data.(*types.ReqWalletSendToAddress)
	if !ok {
		walletlog.Error("onWalletSendToAddress", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply, err := wallet.ProcSendToAddress(req)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletSetFee(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReply)
	req, ok := msg.Data.(*types.ReqWalletSetFee)
	if !ok {
		walletlog.Error("onWalletSetFee", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletSetFee(req)
	if err != nil {
		walletlog.Error("ProcWalletSetFee", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletSetLabel(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletAccount)
	req, ok := msg.Data.(*types.ReqWalletSetLabel)
	if !ok {
		walletlog.Error("onWalletSetLabel", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply, err := wallet.ProcWalletSetLabel(req)
	if err != nil {
		walletlog.Error("ProcWalletSetLabel", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletMergeBalance(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyHashes)
	req, ok := msg.Data.(*types.ReqWalletMergeBalance)
	if !ok {
		walletlog.Error("onWalletMergeBalance", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply, err := wallet.ProcMergeBalance(req)
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) ontWalletSetPasswd(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReply)
	req, ok := msg.Data.(*types.ReqWalletSetPasswd)
	if !ok {
		walletlog.Error("ontWalletSetPasswd", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletSetPasswd(req)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return topic, retty, reply, nil
}

func (wallet *Wallet) onWalletLock(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReply)

	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletLock()
	if err != nil {
		walletlog.Error("ProcWalletLock", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onWalletUnLock(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReply)
	req, ok := msg.Data.(*types.WalletUnLock)
	if !ok {
		walletlog.Error("onWalletUnLock", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletUnLock(req)
	if err != nil {
		walletlog.Error("ProcWalletLock", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	// TODO: 这里应该是解锁成功才需要通知挖矿
	wallet.flushTicket()
	return topic, retty, reply, nil
}

// TODO: 区块增加涉及到的逻辑比较多，还需要进行重构
func (wallet *Wallet) onAddBlock(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventAddBlock)
	block, ok := msg.Data.(*types.BlockDetail)
	if !ok {
		walletlog.Error("onAddBlock", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	wallet.updateLastHeader(block, 1)
	wallet.ProcWalletAddBlock(block)
	return topic, retty, nil, nil
}

// TODO: 区块删除涉及到的逻辑比较多，还需要进行重构
func (wallet *Wallet) onDelBlock(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventDelBlock)
	block, ok := msg.Data.(*types.BlockDetail)
	if !ok {
		walletlog.Error("onAddBlock", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	wallet.updateLastHeader(block, -1)
	wallet.ProcWalletDelBlock(block)
	return topic, retty, nil, nil
}

func (wallet *Wallet) onGenSeed(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyGenSeed)
	req, ok := msg.Data.(*types.GenSeedLang)
	if !ok {
		walletlog.Error("onGenSeed", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply, err := wallet.genSeed(req.Lang)
	if err != nil {
		walletlog.Error("genSeed", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onGetSeed(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyGetSeed)
	req, ok := msg.Data.(*types.GetSeedByPw)
	if !ok {
		walletlog.Error("onGetSeed", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply := &types.ReplySeed{}
	seed, err := wallet.getSeed(req.Passwd)
	if err != nil {
		walletlog.Error("getSeed", "err", err.Error())
	} else {
		reply.Seed = seed
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onSaveSeed(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReply)
	req, ok := msg.Data.(*types.SaveSeedByPw)
	if !ok {
		walletlog.Error("onGetSeed", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply := &types.Reply{
		IsOk: true,
	}
	ok, err := wallet.saveSeed(req.Passwd, req.Seed)
	if !ok {
		walletlog.Error("saveSeed", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return topic, retty, reply, nil
}

func (wallet *Wallet) onGetWalletStatus(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReply)
	reply := wallet.GetWalletStatus()
	return topic, retty, reply, nil
}

func (wallet *Wallet) onDumpPrivKey(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivkey)
	req, ok := msg.Data.(*types.ReqStr)
	if !ok {
		walletlog.Error("onWalletMergeBalance", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply := &types.ReplyStr{}
	privkey, err := wallet.ProcDumpPrivkey(req.ReqStr)
	if err != nil {
		walletlog.Error("ProcDumpPrivkey", "err", err.Error())
	} else {
		reply.Replystr = privkey
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onCloseTickets(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyHashes)

	reply, err := wallet.forceCloseTicket(wallet.GetHeight() + 1)
	if err != nil {
		walletlog.Error("ProcDumpPrivkey", "err", err.Error())
	} else {
		go func() {
			if len(reply.Hashes) > 0 {
				wallet.waitTxs(reply.Hashes)
				wallet.flushTicket()
			}
		}()
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onSignRawTx(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplySignRawTx)
	req, ok := msg.Data.(*types.ReqSignRawTx)
	if !ok {
		walletlog.Error("onSignRawTx", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	reply := &types.ReplySignRawTx{}
	txhex, err := wallet.ProcSignRawTx(req)
	if err != nil {
		walletlog.Error("ProcSignRawTx", "err", err.Error())
	} else {
		reply.TxHex = txhex
	}
	return topic, retty, reply, err
}

func (wallet *Wallet) onErrToFront(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventErrToFront)
	req, ok := msg.Data.(*types.ReportErrEvent)
	if !ok {
		walletlog.Error("onErrToFront", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	wallet.setFatalFailure(req)
	return topic, retty, nil, nil
}

// onFatalFailure 定时查询是否有致命性故障产生
func (wallet *Wallet) onFatalFailure(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventFatalFailure)
	reply := &types.Int32{
		Data: wallet.getFatalFailure(),
	}
	return topic, retty, reply, nil
}
