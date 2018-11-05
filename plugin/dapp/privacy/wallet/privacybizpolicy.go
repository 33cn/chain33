package wallet

import (
	"sync"
	"sync/atomic"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

var (
	bizlog                  = log15.New("module", "wallet.privacy")
	MaxTxHashsPerTime int64 = 100
	maxTxNumPerBlock  int64 = types.MaxTxsPerBlock
)

func init() {
	wcom.RegisterPolicy(privacytypes.PrivacyX, New())
}

func New() wcom.WalletBizPolicy {
	return &privacyPolicy{
		mtx:            &sync.Mutex{},
		rescanwg:       &sync.WaitGroup{},
		rescanUTXOflag: privacytypes.UtxoFlagNoScan,
	}
}

type privacyPolicy struct {
	mtx            *sync.Mutex
	store          *privacyStore
	walletOperate  wcom.WalletOperate
	rescanwg       *sync.WaitGroup
	rescanUTXOflag int32
}

func (policy *privacyPolicy) setWalletOperate(walletBiz wcom.WalletOperate) {
	policy.mtx.Lock()
	defer policy.mtx.Unlock()
	policy.walletOperate = walletBiz
}

func (policy *privacyPolicy) getWalletOperate() wcom.WalletOperate {
	policy.mtx.Lock()
	defer policy.mtx.Unlock()
	return policy.walletOperate
}

func (policy *privacyPolicy) Init(walletOperate wcom.WalletOperate, sub []byte) {
	policy.setWalletOperate(walletOperate)
	policy.store = NewStore(walletOperate.GetDBStore())
	// 启动定时检查超期FTXO的协程
	walletOperate.GetWaitGroup().Add(1)
	go policy.checkWalletStoreData()
}

func (policy *privacyPolicy) OnCreateNewAccount(acc *types.Account) {
	wg := policy.getWalletOperate().GetWaitGroup()
	wg.Add(1)
	go policy.rescanReqTxDetailByAddr(acc.Addr, wg)
}

func (policy *privacyPolicy) OnImportPrivateKey(acc *types.Account) {
	wg := policy.getWalletOperate().GetWaitGroup()
	wg.Add(1)
	go policy.rescanReqTxDetailByAddr(acc.Addr, wg)
}

func (policy *privacyPolicy) OnAddBlockFinish(block *types.BlockDetail) {

}

func (policy *privacyPolicy) OnDeleteBlockFinish(block *types.BlockDetail) {

}

func (policy *privacyPolicy) OnClose() {

}

func (this *privacyPolicy) OnSetQueueClient() {
	version := this.store.getVersion()
	if version < PRIVACYDBVERSION {
		this.rescanAllTxAddToUpdateUTXOs()
		this.store.setVersion()
	}
}

func (policy *privacyPolicy) OnWalletLocked() {
}

func (policy *privacyPolicy) OnWalletUnlocked(WalletUnLock *types.WalletUnLock) {
}

func (this *privacyPolicy) Call(funName string, in types.Message) (ret types.Message, err error) {
	switch funName {
	case "GetUTXOScaningFlag":
		isok := this.GetRescanFlag() == privacytypes.UtxoFlagScaning
		ret = &types.Reply{IsOk: isok}
	default:
		err = types.ErrNotSupport
	}
	return
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
	signParam := &privacytypes.PrivacySignatureParam{}
	if err = types.Decode(tx.Signature.Signature, signParam); err != nil {
		bizlog.Error("SignTransaction", "Decode PrivacySignatureParam error", err)
		return
	}
	action := new(privacytypes.PrivacyAction)
	if err = types.Decode(tx.Payload, action); err != nil {
		bizlog.Error("SignTransaction", "Decode PrivacyAction error", err)
		return
	}
	if action.Ty != signParam.ActionType {
		bizlog.Error("SignTransaction", "action type ", action.Ty, "signature action type ", signParam.ActionType)
		return
	}
	switch action.Ty {
	case privacytypes.ActionPublic2Privacy:
		// 隐私交易的公对私动作，不存在交易组的操作
		tx.Sign(int32(policy.getWalletOperate().GetSignType()), key)

	case privacytypes.ActionPrivacy2Privacy, privacytypes.ActionPrivacy2Public:
		// 隐私交易的私对私、私对公需要进行特殊签名
		if err = policy.signatureTx(tx, action.GetInput(), signParam.GetUtxobasics(), signParam.GetRealKeyInputs()); err != nil {
			return
		}
	default:
		bizlog.Error("SignTransaction", "Invalid action type ", action.Ty)
		err = types.ErrInvalidParam
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
	utxos        []*privacytypes.UTXO
}

func (policy *privacyPolicy) OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail {
	policy.addDelPrivacyTxsFromBlock(tx, index, block, dbbatch, AddTx)
	// 自己处理掉所有事务，部需要外部处理了
	return nil
}

func (policy *privacyPolicy) OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail {
	policy.addDelPrivacyTxsFromBlock(tx, index, block, dbbatch, DelTx)
	// 自己处理掉所有事务，部需要外部处理了
	return nil
}

func (this *privacyPolicy) GetRescanFlag() int32 {
	return atomic.LoadInt32(&this.rescanUTXOflag)
}

func (this *privacyPolicy) SetRescanFlag(flag int32) {
	atomic.StoreInt32(&this.rescanUTXOflag, flag)
}
