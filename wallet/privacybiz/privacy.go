package privacybiz

import (
	"bytes"
	"errors"
	"sort"
	"sync/atomic"
	"unsafe"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet/walletbiz"

	"github.com/inconshreveable/log15"
)

var (
	bizlog = log15.New("module", "privacybiz")

	MaxTxHashsPerTime int64 = 100
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
	biz.funcmap.Register(types.EventShowPrivacyPK, biz.onShowPrivacyPK)
	biz.funcmap.Register(types.EventShowPrivacyAccountSpend, biz.onShowPrivacyAccountSpend)
	//biz.funcmap.Register(types.EventPublic2privacy, biz.onPublic2Privacy)
	//biz.funcmap.Register(types.EventPrivacy2privacy, biz.onPrivacy2Privacy)
	//biz.funcmap.Register(types.EventPrivacy2public, biz.onPrivacy2Public)
	biz.funcmap.Register(types.EventCreateUTXOs, biz.onCreateUTXOs)
	biz.funcmap.Register(types.EventCreateTransaction, biz.onCreateTransaction)
	biz.funcmap.Register(types.EventPrivacyAccountInfo, biz.onPrivacyAccountInfo)
	biz.funcmap.Register(types.EventPrivacyTransactionList, biz.onPrivacyTransactionList)
	biz.funcmap.Register(types.EventRescanUtxos, biz.onRescanUtxos)
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

func (biz *walletPrivacyBiz) onShowPrivacyAccountSpend(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyAccountSpend)

	req, ok := msg.Data.(*types.ReqPrivBal4AddrToken)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()
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

	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()
	reply, err := biz.showPrivacyKeyPair(req)
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "err", err.Error())
	}
	return topic, retty, reply, err
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

func (biz *walletPrivacyBiz) onCreateUTXOs(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateUTXOs)

	req, ok := msg.Data.(*types.ReqCreateUTXOs)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()
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
	ok, err := biz.walletBiz.CheckWalletStatus()
	if !ok {
		bizlog.Error("createTransaction", "CheckWalletStatus cause error.", err)
		return topic, retty, nil, err
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}
	if !biz.checkAmountValid(req.Amount) {
		err = types.ErrAmount
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}

	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()

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
	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()

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

	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()

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
	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()

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
	biz.walletBiz.GetMutex().Lock()
	defer biz.walletBiz.GetMutex().Unlock()

	reply, err := biz.enablePrivacy(req)
	if err != nil {
		bizlog.Error("enablePrivacy", "err", err.Error())
	}
	return topic, retty, reply, err
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

func (biz *walletPrivacyBiz) createUTXOs(createUTXOs *types.ReqCreateUTXOs) (*types.Reply, error) {
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
func (biz *walletPrivacyBiz) createUTXOsByPub2Priv(priv crypto.PrivKey, reqCreateUTXOs *types.ReqCreateUTXOs) (*types.Reply, error) {
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

	reply, err := biz.walletBiz.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPub2PriV2", "Send err", err)
		return nil, err
	}
	return reply, nil
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
func (biz *walletPrivacyBiz) makeViewSpendPubKeyPairToString(viewPubKey, spendPubKey []byte) string {
	pair := viewPubKey
	pair = append(pair, spendPubKey...)
	return common.Bytes2Hex(pair)
}

func (biz *walletPrivacyBiz) showPrivacyKeyPair(reqAddr *types.ReqStr) (*types.ReplyPrivacyPkPair, error) {
	privacyInfo, err := biz.getPrivacykeyPair(reqAddr.GetReqStr())
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "getPrivacykeyPair error ", err)
		return nil, err
	}

	pair := privacyInfo.ViewPubkey[:]
	pair = append(pair, privacyInfo.SpendPubkey[:]...)

	replyPrivacyPkPair := &types.ReplyPrivacyPkPair{
		ShowSuccessful: true,
		Pubkeypair:     biz.makeViewSpendPubKeyPairToString(privacyInfo.ViewPubkey[:], privacyInfo.SpendPubkey[:]),
	}
	return replyPrivacyPkPair, nil
}

func (biz *walletPrivacyBiz) getPrivacyAccountInfo(req *types.ReqPPrivacyAccount) (*types.ReplyPrivacyAccount, error) {
	addr := req.GetAddr()
	token := req.GetToken()
	reply := &types.ReplyPrivacyAccount{}
	reply.Displaymode = req.Displaymode
	// 搜索可用余额
	privacyDBStore, err := biz.store.listAvailableUTXOs(token, addr)
	utxos := make([]*types.UTXO, 0)
	for _, ele := range privacyDBStore {
		utxoBasic := &types.UTXOBasic{
			UtxoGlobalIndex: &types.UTXOGlobalIndex{
				Outindex: ele.OutIndex,
				Txhash:   ele.Txhash,
			},
			OnetimePubkey: ele.OnetimePublicKey,
		}
		utxo := &types.UTXO{
			Amount:    ele.Amount,
			UtxoBasic: utxoBasic,
		}
		utxos = append(utxos, utxo)
	}
	reply.Utxos = &types.UTXOs{Utxos: utxos}

	// 搜索冻结余额
	utxos = make([]*types.UTXO, 0)
	ftxoslice, err := biz.store.listFrozenUTXOs(token, addr)
	if err == nil && ftxoslice != nil {
		for _, ele := range ftxoslice {
			utxos = append(utxos, ele.Utxos...)
		}
	}

	reply.Ftxos = &types.UTXOs{Utxos: utxos}

	return reply, nil
}

func (biz *walletPrivacyBiz) getPrivacyTransactionList(req *types.ReqPrivacyTransactionList) (*types.WalletTxDetails, error) {
	return biz.store.getWalletPrivacyTxDetails(req)
}

// 62387455827 -> 455827 + 7000000 + 80000000 + 300000000 + 2000000000 + 60000000000, where 455827 <= dust_threshold
//res:[455827, 7000000, 80000000, 300000000, 2000000000, 60000000000]
func (biz *walletPrivacyBiz) decomposeAmount2digits(amount, dust_threshold int64) []int64 {
	res := make([]int64, 0)
	if 0 >= amount {
		return res
	}

	is_dust_handled := false
	var dust int64 = 0
	var order int64 = 1
	var chunk int64 = 0

	for 0 != amount {
		chunk = (amount % 10) * order
		amount /= 10
		order *= 10
		if dust+chunk < dust_threshold {
			dust += chunk //累加小数，直到超过dust_threshold为止
		} else {
			if !is_dust_handled && 0 != dust {
				//1st 正常情况下，先把dust保存下来
				res = append(res, dust)
				is_dust_handled = true
			}
			if 0 != chunk {
				//2nd 然后依次将大的整数额度进行保存
				goodAmount := biz.decomAmount2Nature(chunk, order/10)
				res = append(res, goodAmount...)
			}
		}
	}

	//如果需要被拆分的额度 < dust_threshold，则直接将其进行保存
	if !is_dust_handled && 0 != dust {
		res = append(res, dust)
	}

	return res
}

//将amount切分为1,2,5的组合，这样在进行amount混淆的时候就能够方便获取相同额度的utxo
func (biz *walletPrivacyBiz) decomAmount2Nature(amount int64, order int64) []int64 {
	res := make([]int64, 0)
	if order == 0 {
		return res
	}
	mul := amount / order
	switch mul {
	case 3:
		res = append(res, order)
		res = append(res, 2*order)
	case 4:
		res = append(res, 2*order)
		res = append(res, 2*order)
	case 6:
		res = append(res, 5*order)
		res = append(res, order)
	case 7:
		res = append(res, 5*order)
		res = append(res, 2*order)
	case 8:
		res = append(res, 5*order)
		res = append(res, 2*order)
		res = append(res, 1*order)
	case 9:
		res = append(res, 5*order)
		res = append(res, 2*order)
		res = append(res, 2*order)
	default:
		res = append(res, mul*order)
		return res
	}
	return res
}

// 修改选择UTXO的算法
// 优先选择UTXO高度与当前高度建个12个区块以上的UTXO
// 如果选择还不够则再从老到新选择12个区块内的UTXO
// 当该地址上的可用UTXO比较多时，可以考虑改进算法，优先选择币值小的，花掉小票，然后再选择币值接近的，减少找零，最后才选择大面值的找零
func (biz *walletPrivacyBiz) selectUTXO(token, addr string, amount int64) ([]*txOutputInfo, error) {
	if len(token) == 0 || len(addr) == 0 || amount <= 0 {
		return nil, types.ErrInvalidParams
	}
	wutxos, err := biz.store.getPrivacyTokenUTXOs(token, addr)
	if err != nil {
		return nil, types.ErrInsufficientBalance
	}
	curBlockHeight := biz.walletBiz.GetBlockHeight()
	var confirmUTXOs, unconfirmUTXOs []*walletUTXO
	var balance int64
	for _, wutxo := range wutxos.utxos {
		if curBlockHeight < wutxo.height {
			continue
		}
		if curBlockHeight-wutxo.height > types.PrivacyMaturityDegree {
			balance += wutxo.outinfo.amount
			confirmUTXOs = append(confirmUTXOs, wutxo)
		} else {
			unconfirmUTXOs = append(unconfirmUTXOs, wutxo)
		}
	}
	if balance < amount && len(unconfirmUTXOs) > 0 {
		// 已经确认的UTXO还不够支付，则需要按照从老到新的顺序，从可能回退的队列中获取
		// 高度从低到高获取
		sort.Slice(unconfirmUTXOs, func(i, j int) bool {
			return unconfirmUTXOs[i].height < unconfirmUTXOs[j].height
		})
		for _, wutxo := range unconfirmUTXOs {
			confirmUTXOs = append(confirmUTXOs, wutxo)
			balance += wutxo.outinfo.amount
			if balance >= amount {
				break
			}
		}
	}
	if balance < amount {
		return nil, types.ErrInsufficientBalance
	}
	balance = 0
	var selectedOuts []*txOutputInfo
	for balance < amount {
		index := biz.walletBiz.GetRandom().Intn(len(confirmUTXOs))
		selectedOuts = append(selectedOuts, confirmUTXOs[index].outinfo)
		balance += confirmUTXOs[index].outinfo.amount
		// remove selected utxo
		confirmUTXOs = append(confirmUTXOs[:index], confirmUTXOs[index+1:]...)
	}
	return selectedOuts, nil
}

/*
buildInput 构建隐私交易的输入信息
操作步骤
	1.从当前钱包中选择可用并且足够支付金额的UTXO列表
	2.如果需要混淆(mixcout>0)，则根据UTXO的金额从数据库中获取足够数量的UTXO，与当前UTXO进行混淆
	3.通过公式 x=Hs(aR)+b，计算出一个整数，因为 xG = Hs(ar)G+bG = Hs(aR)G+B，所以可以继续使用这笔交易
*/
func (biz *walletPrivacyBiz) buildInput(privacykeyParirs *privacy.Privacy, buildInfo *buildInputInfo) (*types.PrivacyInput, []*types.UTXOBasics, []*types.RealKeyInput, []*txOutputInfo, error) {
	//挑选满足额度的utxo
	selectedUtxo, err := biz.selectUTXO(buildInfo.tokenname, buildInfo.sender, buildInfo.amount)
	if err != nil {
		bizlog.Error("buildInput", "Failed to selectOutput for amount", buildInfo.amount,
			"Due to cause", err)
		return nil, nil, nil, nil, err
	}

	bizlog.Debug("buildInput", "Before sort selectedUtxo", selectedUtxo)
	sort.Slice(selectedUtxo, func(i, j int) bool {
		return selectedUtxo[i].amount <= selectedUtxo[j].amount
	})
	bizlog.Debug("buildInput", "After sort selectedUtxo", selectedUtxo)

	reqGetGlobalIndex := types.ReqUTXOGlobalIndex{
		Tokenname: buildInfo.tokenname,
		MixCount:  0,
	}

	if buildInfo.mixcount > 0 {
		reqGetGlobalIndex.MixCount = common.MinInt32(int32(types.PrivacyMaxCount), common.MaxInt32(buildInfo.mixcount, 0))
	}
	for _, out := range selectedUtxo {
		reqGetGlobalIndex.Amount = append(reqGetGlobalIndex.Amount, out.amount)
	}
	// 混淆数大于0时候才向blockchain请求
	var resUTXOGlobalIndex *types.ResUTXOGlobalIndex
	if buildInfo.mixcount > 0 {
		query := &types.BlockChainQuery{
			Driver:   "privacy",
			FuncName: "GetUTXOGlobalIndex",
			Param:    types.Encode(&reqGetGlobalIndex),
		}
		//向blockchain请求相同额度的不同utxo用于相同额度的混淆作用
		resUTXOGlobalIndex, err = biz.walletBiz.GetAPI().BlockChainQuery(query)
		if err != nil {
			bizlog.Error("buildInput BlockChainQuery", "err", err)
			return nil, nil, nil, nil, err
		}
		if resUTXOGlobalIndex == nil {
			bizlog.Info("buildInput EventBlockChainQuery is nil")
			return nil, nil, nil, nil, err
		}

		sort.Slice(resUTXOGlobalIndex.UtxoIndex4Amount, func(i, j int) bool {
			return resUTXOGlobalIndex.UtxoIndex4Amount[i].Amount <= resUTXOGlobalIndex.UtxoIndex4Amount[j].Amount
		})

		if len(selectedUtxo) != len(resUTXOGlobalIndex.UtxoIndex4Amount) {
			bizlog.Error("buildInput EventBlockChainQuery get not the same count for mix",
				"len(selectedUtxo)", len(selectedUtxo),
				"len(resUTXOGlobalIndex.UtxoIndex4Amount)", len(resUTXOGlobalIndex.UtxoIndex4Amount))
		}
	}

	//构造输入PrivacyInput
	privacyInput := &types.PrivacyInput{}
	utxosInKeyInput := make([]*types.UTXOBasics, len(selectedUtxo))
	realkeyInputSlice := make([]*types.RealKeyInput, len(selectedUtxo))
	for i, utxo2pay := range selectedUtxo {
		var utxoIndex4Amount *types.UTXOIndex4Amount
		if nil != resUTXOGlobalIndex && i < len(resUTXOGlobalIndex.UtxoIndex4Amount) && utxo2pay.amount == resUTXOGlobalIndex.UtxoIndex4Amount[i].Amount {
			utxoIndex4Amount = resUTXOGlobalIndex.UtxoIndex4Amount[i]
			for j, utxo := range utxoIndex4Amount.Utxos {
				//查找自身这条UTXO是否存在，如果存在则将其从其中删除
				if bytes.Equal(utxo.OnetimePubkey, utxo2pay.onetimePublicKey) {
					utxoIndex4Amount.Utxos = append(utxoIndex4Amount.Utxos[:j], utxoIndex4Amount.Utxos[j+1:]...)
					break
				}
			}
		}

		if utxoIndex4Amount == nil {
			utxoIndex4Amount = &types.UTXOIndex4Amount{}
		}
		if utxoIndex4Amount.Utxos == nil {
			utxoIndex4Amount.Utxos = make([]*types.UTXOBasic, 0)
		}
		//如果请求返回的用于混淆的utxo不包含自身且达到mix的上限，则将最后一条utxo删除，保证最后的混淆度不大于设置
		if len(utxoIndex4Amount.Utxos) > int(buildInfo.mixcount) {
			utxoIndex4Amount.Utxos = utxoIndex4Amount.Utxos[:len(utxoIndex4Amount.Utxos)-1]
		}

		utxo := &types.UTXOBasic{
			UtxoGlobalIndex: utxo2pay.utxoGlobalIndex,
			OnetimePubkey:   utxo2pay.onetimePublicKey,
		}
		//将真实的utxo添加到最后一个
		utxoIndex4Amount.Utxos = append(utxoIndex4Amount.Utxos, utxo)
		positions := biz.walletBiz.GetRandom().Perm(len(utxoIndex4Amount.Utxos))
		utxos := make([]*types.UTXOBasic, len(utxoIndex4Amount.Utxos))
		for k, position := range positions {
			utxos[position] = utxoIndex4Amount.Utxos[k]
		}
		utxosInKeyInput[i] = &types.UTXOBasics{Utxos: utxos}

		//x = Hs(aR) + b
		onetimePriv, err := privacy.RecoverOnetimePriKey(utxo2pay.txPublicKeyR, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(utxo2pay.utxoGlobalIndex.Outindex))
		if err != nil {
			bizlog.Error("transPri2Pri", "Failed to RecoverOnetimePriKey", err)
			return nil, nil, nil, nil, err
		}

		realkeyInput := &types.RealKeyInput{
			Realinputkey:   int32(positions[len(positions)-1]),
			Onetimeprivkey: onetimePriv.Bytes(),
		}
		realkeyInputSlice[i] = realkeyInput

		keyImage, err := privacy.GenerateKeyImage(onetimePriv, utxo2pay.onetimePublicKey)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		keyInput := &types.KeyInput{
			Amount:   utxo2pay.amount,
			KeyImage: keyImage[:],
		}

		for _, utxo := range utxos {
			keyInput.UtxoGlobalIndex = append(keyInput.UtxoGlobalIndex, utxo.UtxoGlobalIndex)
		}
		//完成一个input的构造，包括基于其环签名的生成，keyImage的生成，
		//必须要注意的是，此处要添加用于混淆的其他utxo添加到最终keyinput的顺序必须和生成环签名时提供pubkey的顺序一致
		//否则会导致环签名验证的失败
		privacyInput.Keyinput = append(privacyInput.Keyinput, keyInput)
	}

	return privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, nil
}

//最后构造完成的utxo依次是2种类型，不构造交易费utxo，使其直接燃烧消失
//1.进行实际转账utxo
//2.进行找零转账utxo
func (biz *walletPrivacyBiz) generateOuts(viewpubTo, spendpubto, viewpubChangeto, spendpubChangeto *[32]byte, transAmount, selectedAmount, fee int64) (*types.PrivacyOutput, error) {
	decomDigit := biz.decomposeAmount2digits(transAmount, types.BTYDustThreshold)
	//计算找零
	changeAmount := selectedAmount - transAmount - fee
	var decomChange []int64
	if 0 < changeAmount {
		decomChange = biz.decomposeAmount2digits(changeAmount, types.BTYDustThreshold)
	}
	bizlog.Info("generateOuts", "decompose digit for amount", selectedAmount-fee, "decomDigit", decomDigit)

	pk := &privacy.PubKeyPrivacy{}
	sk := &privacy.PrivKeyPrivacy{}
	privacy.GenerateKeyPair(sk, pk)
	RtxPublicKey := pk.Bytes()

	sktx := (*[32]byte)(unsafe.Pointer(&sk[0]))
	var privacyOutput types.PrivacyOutput
	privacyOutput.RpubKeytx = RtxPublicKey
	privacyOutput.Keyoutput = make([]*types.KeyOutput, len(decomDigit)+len(decomChange))

	//添加本次转账的目的接收信息（UTXO），包括一次性公钥和额度
	for index, digit := range decomDigit {
		pubkeyOnetime, err := privacy.GenerateOneTimeAddr(viewpubTo, spendpubto, sktx, int64(index))
		if err != nil {
			bizlog.Error("generateOuts", "Fail to GenerateOneTimeAddr due to cause", err)
			return nil, err
		}
		keyOutput := &types.KeyOutput{
			Amount:        digit,
			Onetimepubkey: pubkeyOnetime[:],
		}
		privacyOutput.Keyoutput[index] = keyOutput
	}
	//添加本次转账选择的UTXO后的找零后的UTXO
	for index, digit := range decomChange {
		pubkeyOnetime, err := privacy.GenerateOneTimeAddr(viewpubChangeto, spendpubChangeto, sktx, int64(index+len(decomDigit)))
		if err != nil {
			bizlog.Error("generateOuts", "Fail to GenerateOneTimeAddr for change due to cause", err)
			return nil, err
		}
		keyOutput := &types.KeyOutput{
			Amount:        digit,
			Onetimepubkey: pubkeyOnetime[:],
		}
		privacyOutput.Keyoutput[index+len(decomDigit)] = keyOutput
	}
	//交易费不产生额外的utxo，方便执行器执行的时候直接燃烧殆尽
	if 0 != fee {
		//viewPub, _ := common.Hex2Bytes(types.ViewPubFee)
		//spendPub, _ := common.Hex2Bytes(types.SpendPubFee)
		//viewPublic := (*[32]byte)(unsafe.Pointer(&viewPub[0]))
		//spendPublic := (*[32]byte)(unsafe.Pointer(&spendPub[0]))
		//
		//pubkeyOnetime, err := privacy.GenerateOneTimeAddr(viewPublic, spendPublic, sktx, int64(len(privacyOutput.Keyoutput)))
		//if err != nil {
		//	bizlog.Error("transPub2PriV2", "Fail to GenerateOneTimeAddr for fee due to cause", err)
		//	return nil, nil, err
		//}
		//keyOutput := &types.KeyOutput{
		//	Amount:        fee,
		//	Ometimepubkey: pubkeyOnetime[:],
		//}
		//privacyOutput.Keyoutput = append(privacyOutput.Keyoutput, keyOutput)
	}

	return &privacyOutput, nil
}

func (biz *walletPrivacyBiz) createTransaction(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	switch req.Type {
	case 1:
		return biz.createPublic2PrivacyTx(req)
	case 2:
		return biz.createPrivacy2PrivacyTx(req)
	case 3:
		return biz.createPrivacy2PublicTx(req)
	}
	return nil, types.ErrInvalidParams
}

func (biz *walletPrivacyBiz) createPublic2PrivacyTx(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	viewPubSlice, spendPubSlice, err := biz.parseViewSpendPubKeyPair(req.GetPubkeypair())
	if err != nil {
		bizlog.Error("parse view spend public key pair failed.  err ", err)
		return nil, err
	}
	amount := req.GetAmount()
	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	privacyOutput, err := biz.generateOuts(viewPublic, spendPublic, nil, nil, amount, amount, 0)
	if err != nil {
		bizlog.Error("generate output failed.  err ", err)
		return nil, err
	}

	value := &types.Public2Privacy{
		Tokenname: req.Tokenname,
		Amount:    amount,
		Note:      req.GetNote(),
		Output:    privacyOutput,
	}

	action := &types.PrivacyAction{
		Ty:    types.ActionPublic2Privacy,
		Value: &types.PrivacyAction_Public2Privacy{Public2Privacy: value},
	}

	tx := &types.Transaction{
		Execer:  types.ExecerPrivacy,
		Payload: types.Encode(action),
		Nonce:   biz.walletBiz.Nonce(),
		To:      address.ExecAddress(types.PrivacyX),
	}
	tx.Signature = &types.Signature{
		Signature: types.Encode(&types.PrivacySignatureParam{
			ActionType: action.Ty,
		}),
	}

	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.FeePerKB
	tx.Fee = realFee
	return tx, nil
}

func (biz *walletPrivacyBiz) createPrivacy2PrivacyTx(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	buildInfo := &buildInputInfo{
		tokenname: req.GetTokenname(),
		sender:    req.GetFrom(),
		amount:    req.GetAmount() + types.PrivacyTxFee,
		mixcount:  req.GetMixcount(),
	}

	privacyInfo, err := biz.getPrivacykeyPair(req.GetFrom())
	if err != nil {
		bizlog.Error("createPrivacy2PrivacyTx", "getPrivacykeyPair error", err)
		return nil, err
	}

	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := biz.buildInput(privacyInfo, buildInfo)
	if err != nil {
		return nil, err
	}

	//step 2,generateOuts
	viewPublicSlice, spendPublicSlice, err := biz.parseViewSpendPubKeyPair(req.GetPubkeypair())
	if err != nil {
		bizlog.Error("createPrivacy2PrivacyTx", "parseViewSpendPubKeyPair  ", err)
		return nil, err
	}

	viewPub4change, spendPub4change := privacyInfo.ViewPubkey.Bytes(), privacyInfo.SpendPubkey.Bytes()
	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPublicSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPublicSlice[0]))
	viewPub4chgPtr := (*[32]byte)(unsafe.Pointer(&viewPub4change[0]))
	spendPub4chgPtr := (*[32]byte)(unsafe.Pointer(&spendPub4change[0]))

	selectedAmounTotal := int64(0)
	for _, input := range privacyInput.Keyinput {
		selectedAmounTotal += input.Amount
	}
	//构造输出UTXO
	privacyOutput, err := biz.generateOuts(viewPublic, spendPublic, viewPub4chgPtr, spendPub4chgPtr, req.GetAmount(), selectedAmounTotal, types.PrivacyTxFee)
	if err != nil {
		return nil, err
	}

	value := &types.Privacy2Privacy{
		Tokenname: req.GetTokenname(),
		Amount:    req.GetAmount(),
		Note:      req.GetNote(),
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &types.PrivacyAction{
		Ty:    types.ActionPrivacy2Privacy,
		Value: &types.PrivacyAction_Privacy2Privacy{Privacy2Privacy: value},
	}

	tx := &types.Transaction{
		Execer:  types.ExecerPrivacy,
		Payload: types.Encode(action),
		Fee:     types.PrivacyTxFee,
		Nonce:   biz.walletBiz.Nonce(),
		To:      address.ExecAddress(types.PrivacyX),
	}
	// 创建交易成功，将已经使用掉的UTXO冻结
	biz.saveFTXOInfo(tx, req.GetTokenname(), req.GetFrom(), common.Bytes2Hex(tx.Hash()), selectedUtxo)
	tx.Signature = &types.Signature{
		Signature: types.Encode(&types.PrivacySignatureParam{
			ActionType:    action.Ty,
			Utxobasics:    utxosInKeyInput,
			RealKeyInputs: realkeyInputSlice,
		}),
	}
	return tx, nil
}

func (biz *walletPrivacyBiz) createPrivacy2PublicTx(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	buildInfo := &buildInputInfo{
		tokenname: req.GetTokenname(),
		sender:    req.GetFrom(),
		amount:    req.GetAmount() + types.PrivacyTxFee,
		mixcount:  req.GetMixcount(),
	}
	privacyInfo, err := biz.getPrivacykeyPair(req.GetFrom())
	if err != nil {
		bizlog.Error("createPrivacy2PublicTx failed to getPrivacykeyPair")
		return nil, err
	}
	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := biz.buildInput(privacyInfo, buildInfo)
	if err != nil {
		bizlog.Error("createPrivacy2PublicTx failed to buildInput")
		return nil, err
	}

	viewPub4change, spendPub4change := privacyInfo.ViewPubkey.Bytes(), privacyInfo.SpendPubkey.Bytes()
	viewPub4chgPtr := (*[32]byte)(unsafe.Pointer(&viewPub4change[0]))
	spendPub4chgPtr := (*[32]byte)(unsafe.Pointer(&spendPub4change[0]))

	selectedAmounTotal := int64(0)
	for _, input := range privacyInput.Keyinput {
		if input.Amount <= 0 {
			return nil, errors.New("")
		}
		selectedAmounTotal += input.Amount
	}
	changeAmount := selectedAmounTotal - req.GetAmount()
	//step 2,generateOuts
	//构造输出UTXO,只生成找零的UTXO
	privacyOutput, err := biz.generateOuts(nil, nil, viewPub4chgPtr, spendPub4chgPtr, 0, changeAmount, types.PrivacyTxFee)
	if err != nil {
		return nil, err
	}

	value := &types.Privacy2Public{
		Tokenname: req.GetTokenname(),
		Amount:    req.GetAmount(),
		Note:      req.GetNote(),
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &types.PrivacyAction{
		Ty:    types.ActionPrivacy2Public,
		Value: &types.PrivacyAction_Privacy2Public{Privacy2Public: value},
	}

	tx := &types.Transaction{
		Execer:  []byte(types.PrivacyX),
		Payload: types.Encode(action),
		Fee:     types.PrivacyTxFee,
		Nonce:   biz.walletBiz.Nonce(),
		To:      req.GetTo(),
	}
	// 创建交易成功，将已经使用掉的UTXO冻结
	biz.saveFTXOInfo(tx, req.GetTokenname(), req.GetFrom(), common.Bytes2Hex(tx.Hash()), selectedUtxo)
	tx.Signature = &types.Signature{
		Signature: types.Encode(&types.PrivacySignatureParam{
			ActionType:    action.Ty,
			Utxobasics:    utxosInKeyInput,
			RealKeyInputs: realkeyInputSlice,
		}),
	}
	return tx, nil
}

func (biz *walletPrivacyBiz) saveFTXOInfo(tx *types.Transaction, token, sender, txhash string, selectedUtxos []*txOutputInfo) {
	//将已经作为本次交易输入的utxo进行冻结，防止产生双花交易
	biz.store.moveUTXO2FTXO(tx, token, sender, txhash, selectedUtxos)
	//TODO:需要加入超时处理，需要将此处的txhash写入到数据库中，以免钱包瞬间奔溃后没有对该笔隐私交易的记录，
	//TODO:然后当该交易得到执行之后，没法将FTXO转化为STXO，added by hezhengjun on 2018.6.5
}

func (biz *walletPrivacyBiz) getPrivacyKeyPairs() ([]addrAndprivacy, error) {
	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := biz.store.getAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		bizlog.Info("getPrivacyKeyPairs", "store getAccountByPrefix error", err)
		return nil, err
	}

	var infoPriRes []addrAndprivacy
	for _, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			if privacyInfo, err := biz.getPrivacykeyPair(AccStore.Addr); err == nil {
				var priInfo addrAndprivacy
				priInfo.Addr = &AccStore.Addr
				priInfo.PrivacyKeyPair = privacyInfo
				infoPriRes = append(infoPriRes, priInfo)
			}
		}
	}

	if 0 == len(infoPriRes) {
		return nil, types.ErrPrivacyNotEnabled
	}

	return infoPriRes, nil

}

func (biz *walletPrivacyBiz) rescanUTXOs(req *types.ReqRescanUtxos) (*types.RepRescanUtxos, error) {
	if req.Flag != 0 {
		return biz.store.getRescanUtxosFlag4Addr(req)
	}
	// Rescan请求
	var repRescanUtxos types.RepRescanUtxos
	repRescanUtxos.Flag = req.Flag

	if biz.walletBiz.IsWalletLocked() {
		return nil, types.ErrWalletIsLocked
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		return nil, err
	}
	_, err := biz.getPrivacyKeyPairs()
	if err != nil {
		return nil, err
	}
	atomic.StoreInt32(&biz.rescanUTXOflag, types.UtxoFlagScaning)
	biz.walletBiz.AddWaitGroup(1)
	go biz.rescanReqUtxosByAddr(req.Addrs)
	return &repRescanUtxos, nil
}

//从blockchain模块同步addr参与的所有交易详细信息
func (biz *walletPrivacyBiz) rescanReqUtxosByAddr(addrs []string) {
	defer biz.walletBiz.WaitGroupDone()
	bizlog.Debug("RescanAllUTXO begin!")
	biz.reqUtxosByAddr(addrs)
	bizlog.Debug("RescanAllUTXO sucess!")
}

func (biz *walletPrivacyBiz) reqUtxosByAddr(addrs []string) {
	// 更新数据库存储状态
	var storeAddrs []string
	if len(addrs) == 0 {
		WalletAccStores, err := biz.store.getAccountByPrefix("Account")
		if err != nil || len(WalletAccStores) == 0 {
			bizlog.Info("reqUtxosByAddr", "getAccountByPrefix error", err)
			return
		}
		for _, WalletAccStore := range WalletAccStores {
			storeAddrs = append(storeAddrs, WalletAccStore.Addr)
		}
	} else {
		storeAddrs = append(storeAddrs, addrs...)
	}
	biz.store.saveREscanUTXOsAddresses(storeAddrs)

	reqAddr := address.ExecAddress(types.PrivacyX)
	var txInfo types.ReplyTxInfo
	i := 0
	for {
		select {
		case <-biz.walletBiz.GetWalletDone():
			return
		default:
		}

		//首先从execs模块获取地址对应的所有UTXOs,
		// 1 先获取隐私合约地址相关交易
		var ReqAddr types.ReqAddr
		ReqAddr.Addr = reqAddr
		ReqAddr.Flag = 0
		ReqAddr.Direction = 0
		ReqAddr.Count = int32(MaxTxHashsPerTime)
		if i == 0 {
			ReqAddr.Height = -1
			ReqAddr.Index = 0
		} else {
			ReqAddr.Height = txInfo.GetHeight()
			ReqAddr.Index = txInfo.GetIndex()
			if types.ForkV21Privacy > ReqAddr.Height { // 小于隐私分叉高度不做扫描
				break
			}
		}
		i++

		//请求交易信息
		msg, err := biz.walletBiz.GetAPI().Query(&types.Query{
			Execer:   types.ExecerPrivacy,
			FuncName: "GetTxsByAddr",
			Payload:  types.Encode(&ReqAddr),
		})
		if err != nil {
			bizlog.Error("reqUtxosByAddr", "GetTxsByAddr error", err, "addr", reqAddr)
			break
		}
		ReplyTxInfos := (*msg).(*types.ReplyTxInfos)
		if ReplyTxInfos == nil {
			bizlog.Info("privacy ReqTxInfosByAddr ReplyTxInfos is nil")
			break
		}
		txcount := len(ReplyTxInfos.TxInfos)

		var ReqHashes types.ReqHashes
		ReqHashes.Hashes = make([][]byte, len(ReplyTxInfos.TxInfos))
		for index, ReplyTxInfo := range ReplyTxInfos.TxInfos {
			ReqHashes.Hashes[index] = ReplyTxInfo.GetHash()
		}

		if txcount > 0 {
			txInfo.Hash = ReplyTxInfos.TxInfos[txcount-1].GetHash()
			txInfo.Height = ReplyTxInfos.TxInfos[txcount-1].GetHeight()
			txInfo.Index = ReplyTxInfos.TxInfos[txcount-1].GetIndex()
		}

		biz.getPrivacyTxDetailByHashs(&ReqHashes, addrs)
		if txcount < int(MaxTxHashsPerTime) {
			break
		}
	}
	// 扫描完毕
	atomic.StoreInt32(&biz.rescanUTXOflag, types.UtxoFlagNoScan)
	// 删除privacyInput
	biz.deleteScanPrivacyInputUtxo()
	biz.store.saveREscanUTXOsAddresses(storeAddrs)
}

func (biz *walletPrivacyBiz) deleteScanPrivacyInputUtxo() {
	MaxUtxosPerTime := 1000
	for {
		utxoGlobalIndexs := biz.store.setScanPrivacyInputUTXO(int32(MaxUtxosPerTime))
		biz.store.updateScanInputUTXOs(utxoGlobalIndexs)
		if len(utxoGlobalIndexs) < MaxUtxosPerTime {
			break
		}
	}
}

func (biz *walletPrivacyBiz) getPrivacyTxDetailByHashs(ReqHashes *types.ReqHashes, addrs []string) {
	//通过txhashs获取对应的txdetail
	TxDetails, err := biz.walletBiz.GetAPI().GetTransactionByHash(ReqHashes)
	if err != nil {
		bizlog.Error("getPrivacyTxDetailByHashs", "GetTransactionByHash error", err)
		return
	}
	var privacyInfo []addrAndprivacy
	if len(addrs) > 0 {
		for _, addr := range addrs {
			if privacy, err := biz.getPrivacykeyPair(addr); err != nil {
				priInfo := &addrAndprivacy{
					Addr:           &addr,
					PrivacyKeyPair: privacy,
				}
				privacyInfo = append(privacyInfo, *priInfo)
			}

		}
	} else {
		privacyInfo, _ = biz.getPrivacyKeyPairs()
	}
	biz.store.selectPrivacyTransactionToWallet(TxDetails, privacyInfo)
}

func (biz *walletPrivacyBiz) showPrivacyAccountsSpend(req *types.ReqPrivBal4AddrToken) (*types.UTXOHaveTxHashs, error) {
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		return nil, err
	}

	addr := req.GetAddr()
	token := req.GetToken()
	utxoHaveTxHashs, err := biz.store.listSpendUTXOs(token, addr)
	if err != nil {
		return nil, err
	}

	return utxoHaveTxHashs, nil
}
