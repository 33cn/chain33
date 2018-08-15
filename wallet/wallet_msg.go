package wallet

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

func (wallet *Wallet) initFuncMap() {
	wcom.RegisterMsgFunc(types.EventWalletGetAccountList, wallet.onWalletGetAccountList)
	wcom.RegisterMsgFunc(types.EventWalletAutoMiner, wallet.onWalletAutoMiner)
	wcom.RegisterMsgFunc(types.EventNewAccount, wallet.onNewAccount)
	wcom.RegisterMsgFunc(types.EventWalletTransactionList, wallet.onWalletTransactionList)
	wcom.RegisterMsgFunc(types.EventWalletImportprivkey, wallet.onWalletImportprivkey)
	wcom.RegisterMsgFunc(types.EventWalletSendToAddress, wallet.onWalletSendToAddress)
	wcom.RegisterMsgFunc(types.EventWalletSetFee, wallet.onWalletSetFee)
	wcom.RegisterMsgFunc(types.EventWalletSetLabel, wallet.onWalletSetLabel)
	wcom.RegisterMsgFunc(types.EventWalletMergeBalance, wallet.onWalletMergeBalance)
	wcom.RegisterMsgFunc(types.EventWalletSetPasswd, wallet.ontWalletSetPasswd)
	wcom.RegisterMsgFunc(types.EventWalletLock, wallet.onWalletLock)
	wcom.RegisterMsgFunc(types.EventWalletUnLock, wallet.onWalletUnLock)
	wcom.RegisterMsgFunc(types.EventAddBlock, wallet.onAddBlock)
	wcom.RegisterMsgFunc(types.EventDelBlock, wallet.onDelBlock)
	wcom.RegisterMsgFunc(types.EventGenSeed, wallet.onGenSeed)
	wcom.RegisterMsgFunc(types.EventGetSeed, wallet.onGetSeed)
	wcom.RegisterMsgFunc(types.EventSaveSeed, wallet.onSaveSeed)
	wcom.RegisterMsgFunc(types.EventGetWalletStatus, wallet.onGetWalletStatus)
	wcom.RegisterMsgFunc(types.EventDumpPrivkey, wallet.onDumpPrivKey)
	wcom.RegisterMsgFunc(types.EventSignRawTx, wallet.onSignRawTx)
	wcom.RegisterMsgFunc(types.EventErrToFront, wallet.onErrToFront)
	wcom.RegisterMsgFunc(types.EventFatalFailure, wallet.onFatalFailure)
}

func (wallet *Wallet) ProcRecvMsg() {
	defer wallet.wg.Done()
	for msg := range wallet.client.Recv() {
		walletlog.Debug("wallet recv", "msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)

		funcExisted, topic, retty, reply, err := wcom.FuncMap.Process(&msg)
		if funcExisted {
			if err != nil {
				msg.Reply(wallet.api.NewMessage(topic, retty, err))
			} else {
				msg.Reply(wallet.api.NewMessage(topic, retty, reply))
			}
		} else {
			walletlog.Error("ProcRecvMsg", "Do not support msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)
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
	wallet.walletStore.SetAutoMinerFlag(req.Flag)
	wallet.setAutoMining(req.Flag)
	wallet.flushTicket()
	return topic, retty, &types.Reply{IsOk: true}, nil
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
