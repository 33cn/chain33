package privacybizpolicy

import (
	"github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet/bizpolicy"
	"gitlab.33.cn/chain33/chain33/wallet/walletoperate"
)

var (
	bizlog = log15.New("module", "privacybiz")

	MaxTxHashsPerTime int64 = 100
)

func New() bizpolicy.WalletBizPolicy {
	return &walletPrivacyBiz{}
}

type walletPrivacyBiz struct {
	funcmap       queue.FuncMap
	store         *privacyStore
	walletOperate walletoperate.WalletOperate

	rescanUTXOflag int32
}

func (biz *walletPrivacyBiz) Init(walletOperate walletoperate.WalletOperate) {
	biz.walletOperate = walletOperate
	biz.store = &privacyStore{db: walletOperate.GetDBStore()}

	biz.funcmap.Init()

	//biz.funcmap.Register(types.EventAddBlock, biz.onAddBlock)
	//biz.funcmap.Register(types.EventDelBlock, biz.onDeleteBlock)
	//
	biz.funcmap.Register(types.EventShowPrivacyPK, biz.onShowPrivacyPK)
	biz.funcmap.Register(types.EventShowPrivacyAccountSpend, biz.onShowPrivacyAccountSpend)
	biz.funcmap.Register(types.EventPublic2privacy, biz.onPublic2Privacy)
	biz.funcmap.Register(types.EventPrivacy2privacy, biz.onPrivacy2Privacy)
	biz.funcmap.Register(types.EventPrivacy2public, biz.onPrivacy2Public)
	biz.funcmap.Register(types.EventCreateUTXOs, biz.onCreateUTXOs)
	biz.funcmap.Register(types.EventCreateTransaction, biz.onCreateTransaction)
	biz.funcmap.Register(types.EventPrivacyAccountInfo, biz.onPrivacyAccountInfo)
	biz.funcmap.Register(types.EventPrivacyTransactionList, biz.onPrivacyTransactionList)
	biz.funcmap.Register(types.EventRescanUtxos, biz.onRescanUtxos)
	biz.funcmap.Register(types.EventEnablePrivacy, biz.onEnablePrivacy)
}
func (biz *walletPrivacyBiz) OnRecvQueueMsg(msg *queue.Message) (bool, error) {
	if msg == nil {
		return false, types.ErrInvalidParams
	}
	existed, topic, rettype, retdata, err := biz.funcmap.Process(msg)
	if !existed {
		return false, nil
	}
	if err == nil {
		msg.Reply(biz.walletOperate.GetAPI().NewMessage(topic, rettype, retdata))
	} else {
		msg.Reply(biz.walletOperate.GetAPI().NewMessage(topic, rettype, err))
	}
	return true, err
}

func (biz *walletPrivacyBiz) onAddBlock(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onDeleteBlock(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onShowPrivacyAccountSpend(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyAccountSpend)

	req, ok := msg.Data.(*types.ReqPrivBal4AddrToken)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.showPrivacyAccountsSpend(req)
	if err != nil {
		bizlog.Error("showPrivacyAccountsSpend", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onShowPrivacyPK(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyPK)

	req, ok := msg.Data.(*types.ReqStr)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.showPrivacyKeyPair(req)
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPublic2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPublic2privacy)

	req, ok := msg.Data.(*types.ReqPub2Pri)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.sendPublic2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPublic2PrivacyTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacy2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacy2privacy)

	req, ok := msg.Data.(*types.ReqPri2Pri)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.sendPrivacy2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacy2Public(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacy2public)

	req, ok := msg.Data.(*types.ReqPri2Pub)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.sendPrivacy2PublicTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PublicTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onCreateUTXOs(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateUTXOs)

	req, ok := msg.Data.(*types.ReqCreateUTXOs)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.createUTXOs(req)
	if err != nil {
		bizlog.Error("createUTXOs", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onCreateTransaction(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateTransaction)
	req, ok := msg.Data.(*types.ReqCreateTransaction)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	ok, err := biz.walletOperate.CheckWalletStatus()
	if !ok {
		bizlog.Error("createTransaction", "CheckWalletStatus cause error.", err)
		return topic, retty, nil, err
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}
	if !checkAmountValid(req.Amount) {
		err = types.ErrAmount
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.createTransaction(req)
	if err != nil {
		bizlog.Error("createTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacyAccountInfo(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacyAccountInfo)
	req, ok := msg.Data.(*types.ReqPPrivacyAccount)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.getPrivacyAccountInfo(req)
	if err != nil {
		bizlog.Error("getPrivacyAccountInfo", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacyTransactionList(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacyTransactionList)
	req, ok := msg.Data.(*types.ReqPrivacyTransactionList)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	if req == nil {
		bizlog.Error("onPrivacyTransactionList param is nil")
		return topic, retty, nil, types.ErrInvalidParam
	}
	if req.Direction != 0 && req.Direction != 1 {
		bizlog.Error("getPrivacyTransactionList", "invalid direction ", req.Direction)
		return topic, retty, nil, types.ErrInvalidParam
	}
	// convert to sendTx / recvTx
	sendRecvFlag := req.SendRecvFlag + sendTx
	if sendRecvFlag != sendTx && sendRecvFlag != recvTx {
		bizlog.Error("getPrivacyTransactionList", "invalid sendrecvflag ", req.SendRecvFlag)
		return topic, retty, nil, types.ErrInvalidParam
	}
	req.SendRecvFlag = sendRecvFlag

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.getPrivacyTransactionList(req)
	if err != nil {
		bizlog.Error("getPrivacyTransactionList", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onRescanUtxos(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyRescanUtxos)
	req, ok := msg.Data.(*types.ReqRescanUtxos)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.rescanUTXOs(req)
	if err != nil {
		bizlog.Error("rescanUTXOs", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onEnablePrivacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyEnablePrivacy)
	req, ok := msg.Data.(*types.ReqEnablePrivacy)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.enablePrivacy(req)
	if err != nil {
		bizlog.Error("enablePrivacy", "err", err.Error())
	}
	return topic, retty, reply, err
}
