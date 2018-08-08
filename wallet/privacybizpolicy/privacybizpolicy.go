package privacybizpolicy

import (
	"github.com/inconshreveable/log15"

	"sync"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet/bizpolicy"
	"gitlab.33.cn/chain33/chain33/wallet/walletoperate"
)

var (
	bizlog = log15.New("module", "wallet.privacy")

	MaxTxHashsPerTime int64 = 100
	maxTxNumPerBlock  int64 = types.MaxTxsPerBlock
)

func New() bizpolicy.WalletBizPolicy {
	return &walletPrivacyBiz{}
}

type walletPrivacyBiz struct {
	funcmap       queue.FuncMap
	store         *privacyStore
	walletOperate walletoperate.WalletOperate
	rescanwg      *sync.WaitGroup
}

func (biz *walletPrivacyBiz) initFuncMap() {
	biz.funcmap.Init()
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

func (biz *walletPrivacyBiz) Init(walletOperate walletoperate.WalletOperate) {
	biz.walletOperate = walletOperate
	biz.store = &privacyStore{db: walletOperate.GetDBStore()}
	biz.rescanwg = &sync.WaitGroup{}

	biz.initFuncMap()

	version := biz.store.getVersion()
	if version < PRIVACYDBVERSION {
		biz.rescanAllTxAddToUpdateUTXOs()
		biz.store.setVersion()
	}
	// 启动定时检查超期FTXO的协程
	walletOperate.AddWaitGroup(1)
	go biz.checkWalletStoreData()
}

func (biz *walletPrivacyBiz) OnCreateNewAccount(addr string) {
	biz.walletOperate.AddWaitGroup(1)
	go biz.rescanReqTxDetailByAddr(addr)
}

func (biz *walletPrivacyBiz) OnImportPrivateKey(addr string) {
	biz.walletOperate.AddWaitGroup(1)
	go biz.rescanReqTxDetailByAddr(addr)
}

func (biz *walletPrivacyBiz) OnAddBlockFinish() {

}

func (biz *walletPrivacyBiz) OnDeleteBlockFinish() {

}

func (biz *walletPrivacyBiz) SignTransaction(key crypto.PrivKey, req *types.ReqSignRawTx) (needSysSign bool, signtxhex string, err error) {
	needSysSign = false
	bytes, err := common.FromHex(req.GetTxHex())
	if err != nil {
		bizlog.Error("SignTransaction", "common.FromHex error", err)
		return
	}
	tx := new(types.Transaction)
	if err = types.Decode(bytes, tx); err != nil {
		bizlog.Error("SignTransaction", "Decode Transaction error", err)
		return
	}
	signParam := &types.PrivacySignatureParam{}
	if err = types.Decode(tx.Signature.Signature, signParam); err != nil {
		bizlog.Error("SignTransaction", "Decode PrivacySignatureParam error", err)
		return
	}
	action := new(types.PrivacyAction)
	if err = types.Decode(tx.Payload, action); err != nil {
		bizlog.Error("SignTransaction", "Decode PrivacyAction error", err)
		return
	}
	if action.Ty != signParam.ActionType {
		bizlog.Error("SignTransaction", "action type ", action.Ty, "signature action type ", signParam.ActionType)
		return
	}
	switch action.Ty {
	case types.ActionPublic2Privacy:
		// 隐私交易的公对私动作，不存在交易组的操作
		tx.Sign(int32(biz.walletOperate.GetSignType()), key)

	case types.ActionPrivacy2Privacy, types.ActionPrivacy2Public:
		// 隐私交易的私对私、私对公需要进行特殊签名
		if err = biz.signatureTx(tx, action.GetInput(), signParam.GetUtxobasics(), signParam.GetRealKeyInputs()); err != nil {
			return
		}
	default:
		bizlog.Error("PrivacyTrading signTxWithPrivacy", "Invalid action type ", action.Ty)
		err = types.ErrInvalidParams
	}
	signtxhex = common.ToHex(types.Encode(tx))
	return
}

type buildStoreWalletTxDetailParam struct {
	tokenname    string
	block        *types.BlockDetail
	tx           *types.Transaction
	index        int
	newbatch     db.Batch
	senderRecver string
	isprivacy    bool
	addDelType   int32
	sendRecvFlag int32
	utxos        []*types.UTXO
}

func (biz *walletPrivacyBiz) OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {
	biz.addDelPrivacyTxsFromBlock(tx, index, block, dbbatch, AddTx)
}

func (biz *walletPrivacyBiz) OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {
	biz.addDelPrivacyTxsFromBlock(tx, index, block, dbbatch, DelTx)
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

func (biz *walletPrivacyBiz) onShowPrivacyAccountSpend(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyAccountSpend)

	req, ok := msg.Data.(*types.ReqPrivBal4AddrToken)
	if !ok {
		bizlog.Error("onShowPrivacyAccountSpend", "Invalid data type.", ok)
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
		bizlog.Error("onShowPrivacyPK", "Invalid data type.", ok)
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
		bizlog.Error("onPublic2Privacy", "Invalid data type.", ok)
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

	reply, err := biz.store.getWalletPrivacyTxDetails(req)
	if err != nil {
		bizlog.Error("getWalletPrivacyTxDetails", "err", err.Error())
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
