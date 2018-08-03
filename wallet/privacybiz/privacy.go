package privacybiz

import (
	"unsafe"

	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"

	"sync/atomic"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/wallet/walletbiz"
	)

var (
	bizlog = log15.New("module", "privacybiz")
)

type walletPrivacyBiz struct {
	funcmap   queue.FuncMap
	store     *privacyStore
	walletBiz walletbiz.WalletBiz

	rescanUTXOflag int32
}

func (biz *walletPrivacyBiz) Init(wbiz walletbiz.WalletBiz) {
	biz.walletBiz = wbiz
	biz.store = &privacyStore{db: wbiz.GetDBStore()}

	biz.funcmap.Init()

	//biz.funcmap.Register(types.EventAddBlock, biz.onAddBlock)
	//biz.funcmap.Register(types.EventDelBlock, biz.onDeleteBlock)
	//
	//biz.funcmap.Register(types.EventShowPrivacyPK, biz.onShowPrivacyPK)
	//biz.funcmap.Register(types.EventPublic2privacy, biz.onPublic2Privacy)
	//biz.funcmap.Register(types.EventPrivacy2privacy, biz.onPrivacy2Privacy)
	//biz.funcmap.Register(types.EventPrivacy2public, biz.onPrivacy2Public)
	biz.funcmap.Register(types.EventCreateUTXOs, biz.onCreateUTXOs)
	//biz.funcmap.Register(types.EventCreateTransaction, biz.onCreateTransaction)
	//biz.funcmap.Register(types.EventPrivacyAccountInfo, biz.onPrivacyAccountInfo)
	//biz.funcmap.Register(types.EventPrivacyTransactionList, biz.onPrivacyTransactionList)
	//biz.funcmap.Register(types.EventRescanUtxos, biz.onRescanUtxos)
	biz.funcmap.Register(types.EventEnablePrivacy, biz.onEnablePrivacy)
}

func (biz *walletPrivacyBiz) OnRecvQueueMsg(msg *queue.Message) error {
	if msg == nil {
		return types.ErrInvalidParams
	}
	existed, topic, rettype, retdata, err := biz.funcmap.Process(msg)
	if !existed {
		return nil
	}
	if err == nil {
		msg.Reply(biz.walletBiz.GetAPI().NewMessage(topic, rettype, retdata))
	} else {
		msg.Reply(biz.walletBiz.GetAPI().NewMessage(topic, rettype, err))
	}
	return err
}

func (biz *walletPrivacyBiz) onAddBlock(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onDeleteBlock(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onShowPrivacyPK(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onPublic2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onPrivacy2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onPrivacy2Public(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) isRescanUtxosFlagScaning() (bool, error) {
	if types.UtxoFlagScaning == atomic.LoadInt32(&biz.rescanUTXOflag) {
		return true, types.ErrRescanFlagScaning
	}
	return false, nil
}

func (biz *walletPrivacyBiz) checkAmountValid(amount int64) bool {
	if amount <= 0 {
		return false
	}
	// 隐私交易中，交易金额必须是types.Coin的整数倍
	// 后续调整了隐私交易中手续费计算以后需要修改
	if (int64(float64(amount)/float64(types.Coin)) * types.Coin) != amount {
		return false
	}
	return true
}

func (biz *walletPrivacyBiz) createUTXOs(createUTXOs *types.ReqCreateUTXOs) (*types.ReplyHash, error) {
	ok, err := biz.walletBiz.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		return nil, err
	}
	if createUTXOs == nil {
		bizlog.Error("createUTXOs input para is nil")
		return nil, types.ErrInputPara
	}
	if !biz.checkAmountValid(createUTXOs.GetAmount()) {
		bizlog.Error("not allow amount number")
		return nil, types.ErrAmount
	}
	priv, err := biz.getPrivKeyByAddr(createUTXOs.GetSender())
	if err != nil {
		return nil, err
	}
	return biz.createUTXOsByPub2Priv(priv, createUTXOs)
}

func (biz *walletPrivacyBiz) parseViewSpendPubKeyPair(in string) (viewPubKey, spendPubKey []byte, err error) {
	src, err := common.FromHex(in)
	if err != nil {
		return nil, nil, err
	}
	if 64 != len(src) {
		bizlog.Error("parseViewSpendPubKeyPair", "pair with len", len(src))
		return nil, nil, types.ErrPubKeyLen
	}
	viewPubKey = src[:32]
	spendPubKey = src[32:]
	return
}

// genCustomOuts 构建一个交易的输出
// 构建方式是，P=Hs(rA)G+B
func (biz *walletPrivacyBiz) genCustomOuts(viewpubTo, spendpubto *[32]byte, transAmount int64, count int32) (*types.PrivacyOutput, error) {
	decomDigit := make([]int64, count)
	for i := range decomDigit {
		decomDigit[i] = transAmount
	}

	pk := &privacy.PubKeyPrivacy{}
	sk := &privacy.PrivKeyPrivacy{}
	privacy.GenerateKeyPair(sk, pk)
	RtxPublicKey := pk.Bytes()

	sktx := (*[32]byte)(unsafe.Pointer(&sk[0]))
	var privacyOutput types.PrivacyOutput
	privacyOutput.RpubKeytx = RtxPublicKey
	privacyOutput.Keyoutput = make([]*types.KeyOutput, len(decomDigit))

	//添加本次转账的目的接收信息（UTXO），包括一次性公钥和额度
	for index, digit := range decomDigit {
		pubkeyOnetime, err := privacy.GenerateOneTimeAddr(viewpubTo, spendpubto, sktx, int64(index))
		if err != nil {
			bizlog.Error("genCustomOuts", "Fail to GenerateOneTimeAddr due to cause", err)
			return nil, err
		}
		keyOutput := &types.KeyOutput{
			Amount:        digit,
			Onetimepubkey: pubkeyOnetime[:],
		}
		privacyOutput.Keyoutput[index] = keyOutput
	}

	return &privacyOutput, nil
}

//批量创建通过public2Privacy实现
func (biz *walletPrivacyBiz) createUTXOsByPub2Priv(priv crypto.PrivKey, reqCreateUTXOs *types.ReqCreateUTXOs) (*types.ReplyHash, error) {
	viewPubSlice, spendPubSlice, err := biz.parseViewSpendPubKeyPair(reqCreateUTXOs.GetPubkeypair())
	if err != nil {
		bizlog.Error("createUTXOsByPub2Priv", "parseViewSpendPubKeyPair error.", err)
		return nil, err
	}

	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	//因为此时是pub2priv的交易，此时不需要构造找零的输出，同时设置fee为0，也是为了简化计算
	privacyOutput, err := biz.genCustomOuts(viewPublic, spendPublic, reqCreateUTXOs.Amount, reqCreateUTXOs.Count)
	if err != nil {
		bizlog.Error("createUTXOsByPub2Priv", "genCustomOuts error.", err)
		return nil, err
	}

	value := &types.Public2Privacy{
		Tokenname: reqCreateUTXOs.Tokenname,
		Amount:    reqCreateUTXOs.Amount * int64(reqCreateUTXOs.Count),
		Note:      reqCreateUTXOs.Note,
		Output:    privacyOutput,
	}
	action := &types.PrivacyAction{
		Ty:    types.ActionPublic2Privacy,
		Value: &types.PrivacyAction_Public2Privacy{value},
	}

	tx := &types.Transaction{
		Execer:  []byte("privacy"),
		Payload: types.Encode(action),
		Nonce:   biz.walletBiz.Nonce(),
		To:      address.ExecAddress(types.PrivacyX),
	}
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.FeePerKB
	tx.Fee = realFee
	tx.Sign(int32(biz.walletBiz.GetSignType()), priv)

	_, err = biz.walletBiz.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPub2PriV2", "Send err", err)
		return nil, err
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()
	return &hash, nil
}

func (biz *walletPrivacyBiz) onCreateUTXOs(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateUTXOs)

	req, ok := msg.Data.(*types.ReqCreateUTXOs)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletBiz.GetMutex().Lock()
	defer  biz.walletBiz.GetMutex().Unlock()
	reply, err := biz.createUTXOs(req)
	if err!=nil {
		bizlog.Error("createUTXOs", "err", err.Error())
	}
	return topic, retty, reply, nil
}

func (biz *walletPrivacyBiz) onCreateTransaction(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onPrivacyAccountInfo(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onPrivacyTransactionList(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) onRescanUtxos(msg *queue.Message) (string, int64, interface{}, error) {
	return "rpc", 0, nil, nil
}

func (biz *walletPrivacyBiz) getPrivKeyByAddr(addr string) (crypto.PrivKey, error) {
	//获取指定地址在钱包里的账户信息
	Accountstor, err := biz.store.getAccountByAddr(addr)
	if err != nil {
		bizlog.Error("ProcSendToAddress", "GetAccountByAddr err:", err)
		return nil, err
	}

	//通过password解密存储的私钥
	prikeybyte, err := common.FromHex(Accountstor.GetPrivkey())
	if err != nil || len(prikeybyte) == 0 {
		bizlog.Error("ProcSendToAddress", "FromHex err", err)
		return nil, err
	}

	password := []byte(biz.walletBiz.GetPassword())
	privkey := CBCDecrypterPrivkey(password, prikeybyte)
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(biz.walletBiz.GetSignType()))
	if err != nil {
		bizlog.Error("ProcSendToAddress", "err", err)
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(privkey)
	if err != nil {
		bizlog.Error("ProcSendToAddress", "PrivKeyFromBytes err", err)
		return nil, err
	}
	return priv, nil
}

func (biz *walletPrivacyBiz) getPrivacykeyPair(addr string) (*privacy.Privacy, error) {
	if accPrivacy, _ := biz.store.getWalletAccountPrivacy(addr); accPrivacy != nil {
		privacyInfo := &privacy.Privacy{}
		password := []byte(biz.walletBiz.GetPassword())
		copy(privacyInfo.ViewPubkey[:], accPrivacy.ViewPubkey)
		decrypteredView := CBCDecrypterPrivkey(password, accPrivacy.ViewPrivKey)
		copy(privacyInfo.ViewPrivKey[:], decrypteredView)
		copy(privacyInfo.SpendPubkey[:], accPrivacy.SpendPubkey)
		decrypteredSpend := CBCDecrypterPrivkey(password, accPrivacy.SpendPrivKey)
		copy(privacyInfo.SpendPrivKey[:], decrypteredSpend)

		return privacyInfo, nil
	} else {
		_, err := biz.getPrivKeyByAddr(addr)
		if err != nil {
			return nil, err
		}
		return nil, types.ErrPrivacyNotEnabled
	}
}

func (biz *walletPrivacyBiz) savePrivacykeyPair(addr string) (*privacy.Privacy, error) {
	priv, err := biz.getPrivKeyByAddr(addr)
	if err != nil {
		return nil, err
	}

	newPrivacy, err := privacy.NewPrivacyWithPrivKey((*[privacy.KeyLen32]byte)(unsafe.Pointer(&priv.Bytes()[0])))
	if err != nil {
		return nil, err
	}

	password := []byte(biz.walletBiz.GetPassword())
	encrypteredView := CBCEncrypterPrivkey(password, newPrivacy.ViewPrivKey.Bytes())
	encrypteredSpend := CBCEncrypterPrivkey(password, newPrivacy.SpendPrivKey.Bytes())
	walletPrivacy := &types.WalletAccountPrivacy{
		ViewPubkey:   newPrivacy.ViewPubkey[:],
		ViewPrivKey:  encrypteredView,
		SpendPubkey:  newPrivacy.SpendPubkey[:],
		SpendPrivKey: encrypteredSpend,
	}
	//save the privacy created to wallet db
	biz.store.setWalletAccountPrivacy(addr, walletPrivacy)
	return newPrivacy, nil
}

func (biz *walletPrivacyBiz) enablePrivacy(req *types.ReqEnablePrivacy) (*types.RepEnablePrivacy, error) {
	var addrs []string
	if 0 == len(req.Addrs) {
		WalletAccStores, err := biz.store.getAccountByPrefix("Account")
		if err != nil || len(WalletAccStores) == 0 {
			bizlog.Info("enablePrivacy", "GetAccountByPrefix:err", err)
			return nil, types.ErrNotFound
		}
		for _, WalletAccStore := range WalletAccStores {
			addrs = append(addrs, WalletAccStore.Addr)
		}
	} else {
		addrs = append(addrs, req.Addrs...)
	}

	var rep types.RepEnablePrivacy
	for _, addr := range addrs {
		str := ""
		isOK := true
		_, err := biz.getPrivacykeyPair(addr)
		if err != nil {
			_, err = biz.savePrivacykeyPair(addr)
			if err != nil {
				isOK = false
				str = err.Error()
			}
		}

		priAddrResult := &types.PriAddrResult{
			Addr: addr,
			IsOK: isOK,
			Msg:  str,
		}

		rep.Results = append(rep.Results, priAddrResult)
	}
	return &rep, nil
}

func (biz *walletPrivacyBiz) onEnablePrivacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyEnablePrivacy)
	req, ok := msg.Data.(*types.ReqEnablePrivacy)

	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()

	reply, err := biz.enablePrivacy(req)
	if err!=nil {
		bizlog.Error("enablePrivacy", "err", err.Error())
	}
	return topic, retty, reply, nil
}
