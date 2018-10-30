package wallet

import (
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (policy *privacyPolicy) On_ShowPrivacyAccountSpend(req *privacytypes.ReqPrivBal4AddrToken) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.showPrivacyAccountsSpend(req)
	if err != nil {
		bizlog.Error("showPrivacyAccountsSpend", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_ShowPrivacyKey(req *types.ReqString) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.showPrivacyKeyPair(req)
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_Public2Privacy(req *privacytypes.ReqPub2Pri) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.sendPublic2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPublic2PrivacyTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_Privacy2Privacy(req *privacytypes.ReqPri2Pri) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.sendPrivacy2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_Privacy2Public(req *privacytypes.ReqPri2Pub) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.sendPrivacy2PublicTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PublicTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_CreateUTXOs(req *privacytypes.ReqCreateUTXOs) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.createUTXOs(req)
	if err != nil {
		bizlog.Error("createUTXOs", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_CreateTransaction(req *types.ReqCreateTransaction) (types.Message, error) {
	ok, err := policy.getWalletOperate().CheckWalletStatus()
	if !ok {
		bizlog.Error("createTransaction", "CheckWalletStatus cause error.", err)
		return nil, err
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return nil, err
	}
	if !checkAmountValid(req.Amount) {
		err = types.ErrAmount
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return nil, err
	}
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()

	reply, err := policy.createTransaction(req)
	if err != nil {
		bizlog.Error("createTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_ShowPrivacyAccountInfo(req *privacytypes.ReqPPrivacyAccount) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.getPrivacyAccountInfo(req)
	if err != nil {
		bizlog.Error("getPrivacyAccountInfo", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_PrivacyTransactionList(req *privacytypes.ReqPrivacyTransactionList) (types.Message, error) {
	if req.Direction != 0 && req.Direction != 1 {
		bizlog.Error("getPrivacyTransactionList", "invalid direction ", req.Direction)
		return nil, types.ErrInvalidParam
	}
	// convert to sendTx / recvTx
	sendRecvFlag := req.SendRecvFlag + sendTx
	if sendRecvFlag != sendTx && sendRecvFlag != recvTx {
		bizlog.Error("getPrivacyTransactionList", "invalid sendrecvflag ", req.SendRecvFlag)
		return nil, types.ErrInvalidParam
	}
	req.SendRecvFlag = sendRecvFlag

	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()

	reply, err := policy.store.getWalletPrivacyTxDetails(req)
	if err != nil {
		bizlog.Error("getWalletPrivacyTxDetails", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_RescanUtxos(req *privacytypes.ReqRescanUtxos) (types.Message, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.rescanUTXOs(req)
	if err != nil {
		bizlog.Error("rescanUTXOs", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_EnablePrivacy(req *privacytypes.ReqEnablePrivacy) (types.Message, error) {
	operater := policy.getWalletOperate()
	operater.GetMutex().Lock()
	defer operater.GetMutex().Unlock()
	reply, err := policy.enablePrivacy(req)
	if err != nil {
		bizlog.Error("enablePrivacy", "err", err.Error())
	}
	return reply, err
}
