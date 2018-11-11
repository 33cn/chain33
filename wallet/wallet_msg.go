package wallet

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	wcom "github.com/33cn/chain33/wallet/common"
)

func (wallet *Wallet) ProcRecvMsg() {
	defer wallet.wg.Done()
	for msg := range wallet.client.Recv() {
		walletlog.Debug("wallet recv", "msg", types.GetEventName(int(msg.Ty)), "Id", msg.Id)
		beg := types.Now()
		reply, err := wallet.ExecWallet(&msg)
		if err != nil {
			//only for test ,del when test end
			msg.Reply(wallet.api.NewMessage("", 0, err))
		} else {
			msg.Reply(wallet.api.NewMessage("", 0, reply))
		}
		walletlog.Debug("end process", "msg.id", msg.Id, "cost", types.Since(beg))
	}
}

func (wallet *Wallet) On_WalletGetAccountList(req *types.ReqAccountList) (types.Message, error) {
	reply, err := wallet.ProcGetAccountList(req)
	if err != nil {
		walletlog.Error("onWalletGetAccountList", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_NewAccount(req *types.ReqNewAccount) (types.Message, error) {
	reply, err := wallet.ProcCreateNewAccount(req)
	if err != nil {
		walletlog.Error("onNewAccount", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletTransactionList(req *types.ReqWalletTransactionList) (types.Message, error) {
	reply, err := wallet.ProcWalletTxList(req)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletImportPrivkey(req *types.ReqWalletImportPrivkey) (types.Message, error) {
	reply, err := wallet.ProcImportPrivKey(req)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletSendToAddress(req *types.ReqWalletSendToAddress) (types.Message, error) {
	reply, err := wallet.ProcSendToAddress(req)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletSetFee(req *types.ReqWalletSetFee) (types.Message, error) {
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

func (wallet *Wallet) On_WalletSetLabel(req *types.ReqWalletSetLabel) (types.Message, error) {
	reply, err := wallet.ProcWalletSetLabel(req)
	if err != nil {
		walletlog.Error("ProcWalletSetLabel", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletMergeBalance(req *types.ReqWalletMergeBalance) (types.Message, error) {
	reply, err := wallet.ProcMergeBalance(req)
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_WalletSetPasswd(req *types.ReqWalletSetPasswd) (types.Message, error) {
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

func (wallet *Wallet) On_WalletLock(req *types.ReqNil) (types.Message, error) {
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

func (wallet *Wallet) On_WalletUnLock(req *types.WalletUnLock) (types.Message, error) {
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

func (wallet *Wallet) On_AddBlock(block *types.BlockDetail) (types.Message, error) {
	wallet.updateLastHeader(block, 1)
	wallet.ProcWalletAddBlock(block)
	return nil, nil
}

func (wallet *Wallet) On_DelBlock(block *types.BlockDetail) (types.Message, error) {
	wallet.updateLastHeader(block, -1)
	wallet.ProcWalletDelBlock(block)
	return nil, nil
}

func (wallet *Wallet) On_GenSeed(req *types.GenSeedLang) (types.Message, error) {
	reply, err := wallet.genSeed(req.Lang)
	if err != nil {
		walletlog.Error("genSeed", "err", err.Error())
	}
	return reply, err
}

func (wallet *Wallet) On_GetSeed(req *types.GetSeedByPw) (types.Message, error) {
	reply := &types.ReplySeed{}
	seed, err := wallet.getSeed(req.Passwd)
	if err != nil {
		walletlog.Error("getSeed", "err", err.Error())
	} else {
		reply.Seed = seed
	}
	return reply, err
}

func (wallet *Wallet) On_SaveSeed(req *types.SaveSeedByPw) (types.Message, error) {
	reply := &types.Reply{
		IsOk: true,
	}
	ok, err := wallet.saveSeed(req.Passwd, req.Seed)
	if !ok {
		walletlog.Error("[saveSeed]", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	return reply, nil
}

func (wallet *Wallet) On_GetWalletStatus(req *types.ReqNil) (types.Message, error) {
	reply := wallet.GetWalletStatus()
	return reply, nil
}

func (wallet *Wallet) On_DumpPrivkey(req *types.ReqString) (types.Message, error) {
	reply := &types.ReplyString{}
	privkey, err := wallet.ProcDumpPrivkey(req.Data)
	if err != nil {
		walletlog.Error("ProcDumpPrivkey", "err", err.Error())
	} else {
		reply.Data = privkey
	}
	return reply, err
}

func (wallet *Wallet) On_SignRawTx(req *types.ReqSignRawTx) (types.Message, error) {
	reply := &types.ReplySignRawTx{}
	txhex, err := wallet.ProcSignRawTx(req)
	if err != nil {
		walletlog.Error("ProcSignRawTx", "err", err.Error())
	} else {
		reply.TxHex = txhex
	}
	return reply, err
}

func (wallet *Wallet) On_ErrToFront(req *types.ReportErrEvent) (types.Message, error) {
	wallet.setFatalFailure(req)
	return nil, nil
}

// onFatalFailure 定时查询是否有致命性故障产生
func (wallet *Wallet) On_FatalFailure(req *types.ReqNil) (types.Message, error) {
	reply := &types.Int32{
		Data: wallet.getFatalFailure(),
	}
	return reply, nil
}

func (wallet *Wallet) ExecWallet(msg *queue.Message) (types.Message, error) {
	if param, ok := msg.Data.(*types.ChainExecutor); ok {
		return wallet.execWallet(param, 0)
	}
	var data []byte
	if msg.Data != nil {
		if d, ok := msg.Data.(types.Message); ok {
			data = types.Encode(d)
		} else {
			return nil, types.ErrInvalidParam
		}
	}
	param := &types.ChainExecutor{
		Driver: "wallet",
		Param:  data,
	}
	return wallet.execWallet(param, msg.Ty)
}

func (wallet *Wallet) execWallet(param *types.ChainExecutor, eventId int64) (reply types.Message, err error) {
	if param.FuncName == "" && eventId > 0 {
		param.FuncName = types.GetEventName(int(eventId))
		if len(param.FuncName) <= 5 {
			return nil, types.ErrActionNotSupport
		}
		param.FuncName = param.FuncName[5:]
	}
	var paramIn types.Message
	if param.Param == nil {
		paramIn = &types.ReqNil{}
	} else {
		paramIn, err = wcom.QueryData.Decode(param.Driver, param.FuncName, param.Param)
		if err != nil {
			return nil, err
		}
	}
	//这里不判断类型是否可以调用，直接按照名字调用，如果发生panic，用recover 恢复
	return wcom.QueryData.Call(param.Driver, param.FuncName, paramIn)
}
