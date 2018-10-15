package wallet

import (
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

func (wallet *Wallet) ProcRecvMsg() {
	defer wallet.wg.Done()
	for msg := range wallet.client.Recv() {
		walletlog.Debug("wallet recv", "msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)
		beg := types.Now()
		funcExisted, topic, retty, reply, err := wcom.ProcessFuncMap(&msg)
		if funcExisted {
			if err != nil {
				msg.Reply(wallet.api.NewMessage(topic, retty, err))
			} else {
				msg.Reply(wallet.api.NewMessage(topic, retty, reply))
			}
		} else {
			walletlog.Error("ProcRecvMsg", "Do not support msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)
		}
		walletlog.Debug("end process", "msg.id", msg.Id, "cost", types.Since(beg))
	}
}

func (wallet *Wallet) On_WalletGetAccountList(req *types.ReqAccountList) (interface{}, error) {
	reply, err := wallet.ProcGetAccountList(req)
	if err != nil {
		walletlog.Error("onWalletGetAccountList", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_NewAccount(req *types.ReqNewAccount) (interface{}, error) {
	reply, err := wallet.ProcCreateNewAccount(req)
	if err != nil {
		walletlog.Error("onNewAccount", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletTransactionList(req *types.ReqWalletTransactionList) (interface{}, error) {
	reply, err := wallet.ProcWalletTxList(req)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletImportprivkey(req *types.ReqWalletImportPrivKey) (interface{}, error) {
	reply, err := wallet.ProcImportPrivKey(req)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletSendToAddress(req *types.ReqWalletSendToAddress) (interface{}, error) {
	reply, err := wallet.ProcSendToAddress(req)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletSetFee(req *types.ReqWalletSetFee) (interface{}, error) {
	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletSetFee(req)
	if err != nil {
		walletlog.Error("ProcWalletSetFee", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletSetLabel(req *types.ReqWalletSetLabel) (interface{}, error) {
	reply, err := wallet.ProcWalletSetLabel(req)
	if err != nil {
		walletlog.Error("ProcWalletSetLabel", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletMergeBalance(req *types.ReqWalletMergeBalance) (interface{}, error) {
	reply, err := wallet.ProcMergeBalance(req)
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletSetPasswd(req *types.ReqWalletSetPasswd) (interface{}, error) {
	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletSetPasswd(req)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return reply, nil
}

func (wallet *Wallet) On_WalletLock(req *types.ReqNil) (interface{}, error) {
	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletLock()
	if err != nil {
		walletlog.Error("ProcWalletLock", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletUnLock(req *types.WalletUnLock) (interface{}, error) {
	reply := &types.Reply{
		IsOk: true,
	}
	err := wallet.ProcWalletUnLock(req)
	if err != nil {
		walletlog.Error("ProcWalletLock", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return reply, nil
}

func (wallet *Wallet) On_AddBlock(block *types.BlockDetail) (interface{}, error) {
	wallet.updateLastHeader(block, 1)
	wallet.ProcWalletAddBlock(block)
	return nil, nil
}

func (wallet *Wallet) On_DelBlock(block *types.BlockDetail) (interface{}, error) {
	wallet.updateLastHeader(block, -1)
	wallet.ProcWalletDelBlock(block)
	return nil, nil
}

func (wallet *Wallet) On_GenSeed(req *types.GenSeedLang) (interface{}, error) {
	reply, err := wallet.genSeed(req.Lang)
	if err != nil {
		walletlog.Error("genSeed", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_GetSeed(req *types.GetSeedByPw) (interface{}, error) {
	reply := &types.ReplySeed{}
	seed, err := wallet.getSeed(req.Passwd)
	if err != nil {
		walletlog.Error("getSeed", "err", err.Error())
	} else {
		reply.Seed = seed
	}
	return reply, err
}

func (wallet *Wallet) On_SaveSeed(req *types.SaveSeedByPw) (interface{}, error) {
	reply := &types.Reply{
		IsOk: true,
	}
	ok, err := wallet.saveSeed(req.Passwd, req.Seed)
	if !ok {
		walletlog.Error("saveSeed", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return reply, nil
}

func (wallet *Wallet) On_GetWalletStatus(req *types.ReqNil) (interface{}, error) {
	reply := wallet.GetWalletStatus()
	return reply, nil
}

func (wallet *Wallet) On_DumpPrivKey(req *types.ReqString) (interface{}, error) {
	reply := &types.ReplyString{}
	privkey, err := wallet.ProcDumpPrivkey(req.ReqStr)
	if err != nil {
		walletlog.Error("ProcDumpPrivkey", "err", err.Error())
	} else {
		reply.Data = privkey
	}
	return reply, err
}

func (wallet *Wallet) On_SignRawTx(req *types.ReqSignRawTx) (interface{}, error) {
	reply := &types.ReplySignRawTx{}
	txhex, err := wallet.ProcSignRawTx(req)
	if err != nil {
		walletlog.Error("ProcSignRawTx", "err", err.Error())
	} else {
		reply.TxHex = txhex
	}
	return reply, err
}

func (wallet *Wallet) On_ErrToFront(req *types.ReportErrEvent) (interface{}, error) {
	wallet.setFatalFailure(req)
	return nil, nil
}

// onFatalFailure 定时查询是否有致命性故障产生
func (wallet *Wallet) On_FatalFailure(req *types.ReqNil) (interface{}, error) {
	reply := &types.Int32{
		Data: wallet.getFatalFailure(),
	}
	return reply, nil
}

func (wallet *Wallet) On_ExecWallet(req *types.EventWalletExecutor) (interface{}, error) {
	return nil, nil
}
