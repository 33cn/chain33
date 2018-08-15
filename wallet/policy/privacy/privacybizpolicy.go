package privacy

import (
	"sync"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

var (
	bizlog = log15.New("module", "wallet.privacy")

	MaxTxHashsPerTime int64 = 100
	maxTxNumPerBlock  int64 = types.MaxTxsPerBlock
)

func init() {
	wcom.RegisterPolicy(types.PrivacyX, New())
}

func New() wcom.WalletBizPolicy {
	return &privacyPolicy{}
}

type privacyPolicy struct {
	store         *privacyStore
	walletOperate wcom.WalletOperate
	rescanwg      *sync.WaitGroup
}

func (policy *privacyPolicy) initFuncMap(walletOperate wcom.WalletOperate) {
	wcom.RegisterMsgFunc(types.EventEnablePrivacy, policy.onEnablePrivacy)
	wcom.RegisterMsgFunc(types.EventShowPrivacyPK, policy.onShowPrivacyPK)
	wcom.RegisterMsgFunc(types.EventCreateUTXOs, policy.onCreateUTXOs)
	wcom.RegisterMsgFunc(types.EventPublic2privacy, policy.onPublic2Privacy)
	wcom.RegisterMsgFunc(types.EventPrivacy2privacy, policy.onPrivacy2Privacy)
	wcom.RegisterMsgFunc(types.EventPrivacy2public, policy.onPrivacy2Public)
	wcom.RegisterMsgFunc(types.EventCreateTransaction, policy.onCreateTransaction)
	wcom.RegisterMsgFunc(types.EventPrivacyAccountInfo, policy.onPrivacyAccountInfo)
	wcom.RegisterMsgFunc(types.EventShowPrivacyAccountSpend, policy.onShowPrivacyAccountSpend)
	wcom.RegisterMsgFunc(types.EventPrivacyTransactionList, policy.onPrivacyTransactionList)
	wcom.RegisterMsgFunc(types.EventRescanUtxos, policy.onRescanUtxos)
}

func (policy *privacyPolicy) Init(walletOperate wcom.WalletOperate) {
	policy.walletOperate = walletOperate
	policy.store = NewStore(walletOperate.GetDBStore())
	policy.rescanwg = &sync.WaitGroup{}

	policy.initFuncMap(walletOperate)

	version := policy.store.getVersion()
	if version < PRIVACYDBVERSION {
		policy.rescanAllTxAddToUpdateUTXOs()
		policy.store.setVersion()
	}
	// 启动定时检查超期FTXO的协程
	walletOperate.GetWaitGroup().Add(1)
	go policy.checkWalletStoreData()
}

func (policy *privacyPolicy) OnCreateNewAccount(acc *types.Account) {
	wg := policy.walletOperate.GetWaitGroup()
	wg.Add(1)
	go policy.rescanReqTxDetailByAddr(acc.Addr, wg)
}

func (policy *privacyPolicy) OnImportPrivateKey(acc *types.Account) {
	wg := policy.walletOperate.GetWaitGroup()
	wg.Add(1)
	go policy.rescanReqTxDetailByAddr(acc.Addr, wg)
}

func (policy *privacyPolicy) OnAddBlockFinish(block *types.BlockDetail) {

}

func (policy *privacyPolicy) OnDeleteBlockFinish(block *types.BlockDetail) {

}

func (policy *privacyPolicy) SignTransaction(key crypto.PrivKey, req *types.ReqSignRawTx) (needSysSign bool, signtxhex string, err error) {
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
		tx.Sign(int32(policy.walletOperate.GetSignType()), key)

	case types.ActionPrivacy2Privacy, types.ActionPrivacy2Public:
		// 隐私交易的私对私、私对公需要进行特殊签名
		if err = policy.signatureTx(tx, action.GetInput(), signParam.GetUtxobasics(), signParam.GetRealKeyInputs()); err != nil {
			return
		}
	default:
		bizlog.Error("SignTransaction", "Invalid action type ", action.Ty)
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

func (policy *privacyPolicy) OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {
	policy.addDelPrivacyTxsFromBlock(tx, index, block, dbbatch, AddTx)
}

func (policy *privacyPolicy) OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {
	policy.addDelPrivacyTxsFromBlock(tx, index, block, dbbatch, DelTx)
}

func (policy *privacyPolicy) onShowPrivacyAccountSpend(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyAccountSpend)

	req, ok := msg.Data.(*types.ReqPrivBal4AddrToken)
	if !ok {
		bizlog.Error("onShowPrivacyAccountSpend", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()
	reply, err := policy.showPrivacyAccountsSpend(req)
	if err != nil {
		bizlog.Error("showPrivacyAccountsSpend", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onShowPrivacyPK(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyPK)

	req, ok := msg.Data.(*types.ReqStr)
	if !ok {
		bizlog.Error("onShowPrivacyPK", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()
	reply, err := policy.showPrivacyKeyPair(req)
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onPublic2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPublic2privacy)

	req, ok := msg.Data.(*types.ReqPub2Pri)
	if !ok {
		bizlog.Error("onPublic2Privacy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()
	reply, err := policy.sendPublic2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPublic2PrivacyTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onPrivacy2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacy2privacy)

	req, ok := msg.Data.(*types.ReqPri2Pri)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()
	reply, err := policy.sendPrivacy2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onPrivacy2Public(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacy2public)

	req, ok := msg.Data.(*types.ReqPri2Pub)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()
	reply, err := policy.sendPrivacy2PublicTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PublicTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onCreateUTXOs(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateUTXOs)

	req, ok := msg.Data.(*types.ReqCreateUTXOs)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()
	reply, err := policy.createUTXOs(req)
	if err != nil {
		bizlog.Error("createUTXOs", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onCreateTransaction(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateTransaction)
	req, ok := msg.Data.(*types.ReqCreateTransaction)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	if req == nil {
		bizlog.Error("privacyPolicy request param is nil")
		return topic, retty, nil, types.ErrInvalidParam
	}
	ok, err := policy.walletOperate.CheckWalletStatus()
	if !ok {
		bizlog.Error("createTransaction", "CheckWalletStatus cause error.", err)
		return topic, retty, nil, err
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}
	if !checkAmountValid(req.Amount) {
		err = types.ErrAmount
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()

	reply, err := policy.createTransaction(req)
	if err != nil {
		bizlog.Error("createTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onPrivacyAccountInfo(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacyAccountInfo)
	req, ok := msg.Data.(*types.ReqPPrivacyAccount)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	if req == nil {
		bizlog.Error("privacyPolicy request param is nil")
		return topic, retty, nil, types.ErrInvalidParam
	}
	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()

	reply, err := policy.getPrivacyAccountInfo(req)
	if err != nil {
		bizlog.Error("getPrivacyAccountInfo", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onPrivacyTransactionList(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacyTransactionList)
	req, ok := msg.Data.(*types.ReqPrivacyTransactionList)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
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

	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()

	reply, err := policy.store.getWalletPrivacyTxDetails(req)
	if err != nil {
		bizlog.Error("getWalletPrivacyTxDetails", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onRescanUtxos(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyRescanUtxos)
	req, ok := msg.Data.(*types.ReqRescanUtxos)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	if req == nil {
		bizlog.Error("privacyPolicy request param is nil")
		return topic, retty, nil, types.ErrInvalidParam
	}
	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()

	reply, err := policy.rescanUTXOs(req)
	if err != nil {
		bizlog.Error("rescanUTXOs", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (policy *privacyPolicy) onEnablePrivacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyEnablePrivacy)
	req, ok := msg.Data.(*types.ReqEnablePrivacy)
	if !ok {
		bizlog.Error("privacyPolicy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	if req == nil {
		bizlog.Error("privacyPolicy request param is nil")
		return topic, retty, nil, types.ErrInvalidParam
	}
	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()

	reply, err := policy.enablePrivacy(req)
	if err != nil {
		bizlog.Error("enablePrivacy", "err", err.Error())
	}
	return topic, retty, reply, err
}
