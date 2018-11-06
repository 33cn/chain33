package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	privacy "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/crypto"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

func (policy *privacyPolicy) rescanAllTxAddToUpdateUTXOs() {
	accounts, err := policy.getWalletOperate().GetWalletAccounts()
	if err != nil {
		bizlog.Debug("rescanAllTxToUpdateUTXOs", "walletOperate.GetWalletAccounts error", err)
		return
	}
	bizlog.Debug("rescanAllTxToUpdateUTXOs begin!")
	for _, acc := range accounts {
		//从blockchain模块同步Account.Addr对应的所有交易详细信息
		policy.rescanwg.Add(1)
		go policy.rescanReqTxDetailByAddr(acc.Addr, policy.rescanwg)
	}
	policy.rescanwg.Wait()
	bizlog.Debug("rescanAllTxToUpdateUTXOs sucess!")
}

//从blockchain模块同步addr参与的所有交易详细信息
func (policy *privacyPolicy) rescanReqTxDetailByAddr(addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	policy.reqTxDetailByAddr(addr)
}

//从blockchain模块同步addr参与的所有交易详细信息
func (policy *privacyPolicy) reqTxDetailByAddr(addr string) {
	if len(addr) == 0 {
		bizlog.Error("reqTxDetailByAddr input addr is nil!")
		return
	}
	var txInfo types.ReplyTxInfo

	i := 0
	operater := policy.getWalletOperate()
	for {
		//首先从blockchain模块获取地址对应的所有交易hashs列表,从最新的交易开始获取
		var ReqAddr types.ReqAddr
		ReqAddr.Addr = addr
		ReqAddr.Flag = 0
		ReqAddr.Direction = 0
		ReqAddr.Count = int32(MaxTxHashsPerTime)
		if i == 0 {
			ReqAddr.Height = -1
			ReqAddr.Index = 0
		} else {
			ReqAddr.Height = txInfo.GetHeight()
			ReqAddr.Index = txInfo.GetIndex()
		}
		i++
		ReplyTxInfos, err := operater.GetAPI().GetTransactionByAddr(&ReqAddr)
		if err != nil {
			bizlog.Error("reqTxDetailByAddr", "GetTransactionByAddr error", err, "addr", addr)
			return
		}
		if ReplyTxInfos == nil {
			bizlog.Info("reqTxDetailByAddr ReplyTxInfos is nil")
			return
		}
		txcount := len(ReplyTxInfos.TxInfos)

		var ReqHashes types.ReqHashes
		ReqHashes.Hashes = make([][]byte, len(ReplyTxInfos.TxInfos))
		for index, ReplyTxInfo := range ReplyTxInfos.TxInfos {
			ReqHashes.Hashes[index] = ReplyTxInfo.GetHash()
			txInfo.Hash = ReplyTxInfo.GetHash()
			txInfo.Height = ReplyTxInfo.GetHeight()
			txInfo.Index = ReplyTxInfo.GetIndex()
		}
		operater.GetTxDetailByHashs(&ReqHashes)
		if txcount < int(MaxTxHashsPerTime) {
			return
		}
	}
}

func (policy *privacyPolicy) isRescanUtxosFlagScaning() (bool, error) {
	if privacytypes.UtxoFlagScaning == policy.GetRescanFlag() {
		return true, privacytypes.ErrRescanFlagScaning
	}
	return false, nil
}

func (policy *privacyPolicy) createUTXOs(createUTXOs *privacytypes.ReqCreateUTXOs) (*types.Reply, error) {
	ok, err := policy.getWalletOperate().CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		return nil, err
	}
	if createUTXOs == nil {
		bizlog.Error("createUTXOs input para is nil")
		return nil, types.ErrInvalidParam
	}
	if !checkAmountValid(createUTXOs.GetAmount()) {
		bizlog.Error("not allow amount number")
		return nil, types.ErrAmount
	}
	priv, err := policy.getPrivKeyByAddr(createUTXOs.GetSender())
	if err != nil {
		return nil, err
	}
	return policy.createUTXOsByPub2Priv(priv, createUTXOs)
}

func (policy *privacyPolicy) parseViewSpendPubKeyPair(in string) (viewPubKey, spendPubKey []byte, err error) {
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

//批量创建通过public2Privacy实现
func (policy *privacyPolicy) createUTXOsByPub2Priv(priv crypto.PrivKey, reqCreateUTXOs *privacytypes.ReqCreateUTXOs) (*types.Reply, error) {
	viewPubSlice, spendPubSlice, err := parseViewSpendPubKeyPair(reqCreateUTXOs.GetPubkeypair())
	if err != nil {
		bizlog.Error("createUTXOsByPub2Priv", "parseViewSpendPubKeyPair error.", err)
		return nil, err
	}
	operater := policy.getWalletOperate()
	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	//因为此时是pub2priv的交易，此时不需要构造找零的输出，同时设置fee为0，也是为了简化计算
	privacyOutput, err := generateOuts(viewPublic, spendPublic, nil, nil, reqCreateUTXOs.Amount, reqCreateUTXOs.Amount, 0)
	if err != nil {
		bizlog.Error("createUTXOsByPub2Priv", "genCustomOuts error.", err)
		return nil, err
	}

	value := &privacytypes.Public2Privacy{
		Tokenname: reqCreateUTXOs.Tokenname,
		Amount:    reqCreateUTXOs.Amount,
		Note:      reqCreateUTXOs.Note,
		Output:    privacyOutput,
	}
	action := &privacytypes.PrivacyAction{
		Ty:    privacytypes.ActionPublic2Privacy,
		Value: &privacytypes.PrivacyAction_Public2Privacy{value},
	}

	tx := &types.Transaction{
		Execer:  []byte("privacy"),
		Payload: types.Encode(action),
		Nonce:   operater.Nonce(),
		To:      address.ExecAddress(privacytypes.PrivacyX),
	}
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.GInt("MinFee")
	tx.Fee = realFee
	tx.Sign(int32(operater.GetSignType()), priv)

	reply, err := operater.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPub2PriV2", "Send err", err)
		return nil, err
	}
	return reply, nil
}

func (policy *privacyPolicy) getPrivKeyByAddr(addr string) (crypto.PrivKey, error) {
	//获取指定地址在钱包里的账户信息
	Accountstor, err := policy.store.getAccountByAddr(addr)
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
	operater := policy.getWalletOperate()
	password := []byte(operater.GetPassword())
	privkey := wcom.CBCDecrypterPrivkey(password, prikeybyte)
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignName("privacy", operater.GetSignType()))
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

func (policy *privacyPolicy) getPrivacykeyPair(addr string) (*privacy.Privacy, error) {
	if accPrivacy, _ := policy.store.getWalletAccountPrivacy(addr); accPrivacy != nil {
		privacyInfo := &privacy.Privacy{}
		password := []byte(policy.getWalletOperate().GetPassword())
		copy(privacyInfo.ViewPubkey[:], accPrivacy.ViewPubkey)
		decrypteredView := wcom.CBCDecrypterPrivkey(password, accPrivacy.ViewPrivKey)
		copy(privacyInfo.ViewPrivKey[:], decrypteredView)
		copy(privacyInfo.SpendPubkey[:], accPrivacy.SpendPubkey)
		decrypteredSpend := wcom.CBCDecrypterPrivkey(password, accPrivacy.SpendPrivKey)
		copy(privacyInfo.SpendPrivKey[:], decrypteredSpend)

		return privacyInfo, nil
	} else {
		_, err := policy.getPrivKeyByAddr(addr)
		if err != nil {
			return nil, err
		}
		return nil, privacytypes.ErrPrivacyNotEnabled
	}
}

func (policy *privacyPolicy) savePrivacykeyPair(addr string) (*privacy.Privacy, error) {
	priv, err := policy.getPrivKeyByAddr(addr)
	if err != nil {
		return nil, err
	}

	newPrivacy, err := privacy.NewPrivacyWithPrivKey((*[privacy.KeyLen32]byte)(unsafe.Pointer(&priv.Bytes()[0])))
	if err != nil {
		return nil, err
	}

	password := []byte(policy.getWalletOperate().GetPassword())
	encrypteredView := wcom.CBCEncrypterPrivkey(password, newPrivacy.ViewPrivKey.Bytes())
	encrypteredSpend := wcom.CBCEncrypterPrivkey(password, newPrivacy.SpendPrivKey.Bytes())
	walletPrivacy := &privacytypes.WalletAccountPrivacy{
		ViewPubkey:   newPrivacy.ViewPubkey[:],
		ViewPrivKey:  encrypteredView,
		SpendPubkey:  newPrivacy.SpendPubkey[:],
		SpendPrivKey: encrypteredSpend,
	}
	//save the privacy created to wallet db
	policy.store.setWalletAccountPrivacy(addr, walletPrivacy)
	return newPrivacy, nil
}

func (policy *privacyPolicy) enablePrivacy(req *privacytypes.ReqEnablePrivacy) (*privacytypes.RepEnablePrivacy, error) {
	var addrs []string
	if 0 == len(req.Addrs) {
		WalletAccStores, err := policy.store.getAccountByPrefix("Account")
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

	var rep privacytypes.RepEnablePrivacy
	for _, addr := range addrs {
		str := ""
		isOK := true
		_, err := policy.getPrivacykeyPair(addr)
		if err != nil {
			_, err = policy.savePrivacykeyPair(addr)
			if err != nil {
				isOK = false
				str = err.Error()
			}
		}

		priAddrResult := &privacytypes.PriAddrResult{
			Addr: addr,
			IsOK: isOK,
			Msg:  str,
		}

		rep.Results = append(rep.Results, priAddrResult)
	}
	return &rep, nil
}

func (policy *privacyPolicy) showPrivacyKeyPair(reqAddr *types.ReqString) (*privacytypes.ReplyPrivacyPkPair, error) {
	privacyInfo, err := policy.getPrivacykeyPair(reqAddr.GetData())
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "getPrivacykeyPair error ", err)
		return nil, err
	}

	pair := privacyInfo.ViewPubkey[:]
	pair = append(pair, privacyInfo.SpendPubkey[:]...)

	replyPrivacyPkPair := &privacytypes.ReplyPrivacyPkPair{
		ShowSuccessful: true,
		Pubkeypair:     makeViewSpendPubKeyPairToString(privacyInfo.ViewPubkey[:], privacyInfo.SpendPubkey[:]),
	}
	return replyPrivacyPkPair, nil
}

func (policy *privacyPolicy) getPrivacyAccountInfo(req *privacytypes.ReqPPrivacyAccount) (*privacytypes.ReplyPrivacyAccount, error) {
	addr := strings.Trim(req.GetAddr(), " ")
	token := req.GetToken()
	reply := &privacytypes.ReplyPrivacyAccount{}
	reply.Displaymode = req.Displaymode
	if len(addr) == 0 {
		return nil, errors.New("Address is empty")
	}

	// 搜索可用余额
	privacyDBStore, err := policy.store.listAvailableUTXOs(token, addr)
	utxos := make([]*privacytypes.UTXO, 0)
	for _, ele := range privacyDBStore {
		utxoBasic := &privacytypes.UTXOBasic{
			UtxoGlobalIndex: &privacytypes.UTXOGlobalIndex{
				Outindex: ele.OutIndex,
				Txhash:   ele.Txhash,
			},
			OnetimePubkey: ele.OnetimePublicKey,
		}
		utxo := &privacytypes.UTXO{
			Amount:    ele.Amount,
			UtxoBasic: utxoBasic,
		}
		utxos = append(utxos, utxo)
	}
	reply.Utxos = &privacytypes.UTXOs{Utxos: utxos}

	// 搜索冻结余额
	utxos = make([]*privacytypes.UTXO, 0)
	ftxoslice, err := policy.store.listFrozenUTXOs(token, addr)
	if err == nil && ftxoslice != nil {
		for _, ele := range ftxoslice {
			utxos = append(utxos, ele.Utxos...)
		}
	}

	reply.Ftxos = &privacytypes.UTXOs{Utxos: utxos}

	return reply, nil
}

// 修改选择UTXO的算法
// 优先选择UTXO高度与当前高度建个12个区块以上的UTXO
// 如果选择还不够则再从老到新选择12个区块内的UTXO
// 当该地址上的可用UTXO比较多时，可以考虑改进算法，优先选择币值小的，花掉小票，然后再选择币值接近的，减少找零，最后才选择大面值的找零
func (policy *privacyPolicy) selectUTXO(token, addr string, amount int64) ([]*txOutputInfo, error) {
	if len(token) == 0 || len(addr) == 0 || amount <= 0 {
		return nil, types.ErrInvalidParam
	}
	wutxos, err := policy.store.getPrivacyTokenUTXOs(token, addr)
	if err != nil {
		return nil, types.ErrInsufficientBalance
	}
	operater := policy.getWalletOperate()
	curBlockHeight := operater.GetBlockHeight()
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
		index := operater.GetRandom().Intn(len(confirmUTXOs))
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
func (policy *privacyPolicy) buildInput(privacykeyParirs *privacy.Privacy, buildInfo *buildInputInfo) (*privacytypes.PrivacyInput, []*privacytypes.UTXOBasics, []*privacytypes.RealKeyInput, []*txOutputInfo, error) {
	operater := policy.getWalletOperate()
	//挑选满足额度的utxo
	selectedUtxo, err := policy.selectUTXO(buildInfo.tokenname, buildInfo.sender, buildInfo.amount)
	if err != nil {
		bizlog.Error("buildInput", "Failed to selectOutput for amount", buildInfo.amount,
			"Due to cause", err)
		return nil, nil, nil, nil, err
	}
	sort.Slice(selectedUtxo, func(i, j int) bool {
		return selectedUtxo[i].amount <= selectedUtxo[j].amount
	})

	reqGetGlobalIndex := privacytypes.ReqUTXOGlobalIndex{
		Tokenname: buildInfo.tokenname,
		MixCount:  0,
	}

	if buildInfo.mixcount > 0 {
		reqGetGlobalIndex.MixCount = common.MinInt32(int32(privacytypes.PrivacyMaxCount), common.MaxInt32(buildInfo.mixcount, 0))
	}
	for _, out := range selectedUtxo {
		reqGetGlobalIndex.Amount = append(reqGetGlobalIndex.Amount, out.amount)
	}
	// 混淆数大于0时候才向blockchain请求
	var resUTXOGlobalIndex *privacytypes.ResUTXOGlobalIndex
	if buildInfo.mixcount > 0 {
		query := &types.ChainExecutor{
			Driver:   "privacy",
			FuncName: "GetUTXOGlobalIndex",
			Param:    types.Encode(&reqGetGlobalIndex),
		}
		//向blockchain请求相同额度的不同utxo用于相同额度的混淆作用
		data, err := operater.GetAPI().QueryChain(query)
		if err != nil {
			bizlog.Error("buildInput BlockChainQuery", "err", err)
			return nil, nil, nil, nil, err
		}
		resUTXOGlobalIndex = data.(*privacytypes.ResUTXOGlobalIndex)
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
	privacyInput := &privacytypes.PrivacyInput{}
	utxosInKeyInput := make([]*privacytypes.UTXOBasics, len(selectedUtxo))
	realkeyInputSlice := make([]*privacytypes.RealKeyInput, len(selectedUtxo))
	for i, utxo2pay := range selectedUtxo {
		var utxoIndex4Amount *privacytypes.UTXOIndex4Amount
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
			utxoIndex4Amount = &privacytypes.UTXOIndex4Amount{}
		}
		if utxoIndex4Amount.Utxos == nil {
			utxoIndex4Amount.Utxos = make([]*privacytypes.UTXOBasic, 0)
		}
		//如果请求返回的用于混淆的utxo不包含自身且达到mix的上限，则将最后一条utxo删除，保证最后的混淆度不大于设置
		if len(utxoIndex4Amount.Utxos) > int(buildInfo.mixcount) {
			utxoIndex4Amount.Utxos = utxoIndex4Amount.Utxos[:len(utxoIndex4Amount.Utxos)-1]
		}

		utxo := &privacytypes.UTXOBasic{
			UtxoGlobalIndex: utxo2pay.utxoGlobalIndex,
			OnetimePubkey:   utxo2pay.onetimePublicKey,
		}
		//将真实的utxo添加到最后一个
		utxoIndex4Amount.Utxos = append(utxoIndex4Amount.Utxos, utxo)
		positions := operater.GetRandom().Perm(len(utxoIndex4Amount.Utxos))
		utxos := make([]*privacytypes.UTXOBasic, len(utxoIndex4Amount.Utxos))
		for k, position := range positions {
			utxos[position] = utxoIndex4Amount.Utxos[k]
		}
		utxosInKeyInput[i] = &privacytypes.UTXOBasics{Utxos: utxos}

		//x = Hs(aR) + b
		onetimePriv, err := privacy.RecoverOnetimePriKey(utxo2pay.txPublicKeyR, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(utxo2pay.utxoGlobalIndex.Outindex))
		if err != nil {
			bizlog.Error("transPri2Pri", "Failed to RecoverOnetimePriKey", err)
			return nil, nil, nil, nil, err
		}

		realkeyInput := &privacytypes.RealKeyInput{
			Realinputkey:   int32(positions[len(positions)-1]),
			Onetimeprivkey: onetimePriv.Bytes(),
		}
		realkeyInputSlice[i] = realkeyInput

		keyImage, err := privacy.GenerateKeyImage(onetimePriv, utxo2pay.onetimePublicKey)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		keyInput := &privacytypes.KeyInput{
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

func (policy *privacyPolicy) createTransaction(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	switch req.Type {
	case types.PrivacyTypePublic2Privacy:
		return policy.createPublic2PrivacyTx(req)
	case types.PrivacyTypePrivacy2Privacy:
		return policy.createPrivacy2PrivacyTx(req)
	case types.PrivacyTypePrivacy2Public:
		return policy.createPrivacy2PublicTx(req)
	}
	return nil, types.ErrInvalidParam
}

func (policy *privacyPolicy) createPublic2PrivacyTx(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	viewPubSlice, spendPubSlice, err := parseViewSpendPubKeyPair(req.GetPubkeypair())
	if err != nil {
		bizlog.Error("parse view spend public key pair failed.  err ", err)
		return nil, err
	}
	amount := req.GetAmount()
	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	privacyOutput, err := generateOuts(viewPublic, spendPublic, nil, nil, amount, amount, 0)
	if err != nil {
		bizlog.Error("generate output failed.  err ", err)
		return nil, err
	}

	value := &privacytypes.Public2Privacy{
		Tokenname: req.Tokenname,
		Amount:    amount,
		Note:      req.GetNote(),
		Output:    privacyOutput,
	}

	action := &privacytypes.PrivacyAction{
		Ty:    privacytypes.ActionPublic2Privacy,
		Value: &privacytypes.PrivacyAction_Public2Privacy{Public2Privacy: value},
	}

	tx := &types.Transaction{
		Execer:  []byte(privacytypes.PrivacyX),
		Payload: types.Encode(action),
		Nonce:   policy.getWalletOperate().Nonce(),
		To:      address.ExecAddress(privacytypes.PrivacyX),
	}
	tx.Signature = &types.Signature{
		Signature: types.Encode(&privacytypes.PrivacySignatureParam{
			ActionType: action.Ty,
		}),
	}

	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.GInt("MinFee")
	tx.Fee = realFee
	return tx, nil
}

func (policy *privacyPolicy) createPrivacy2PrivacyTx(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	buildInfo := &buildInputInfo{
		tokenname: req.GetTokenname(),
		sender:    req.GetFrom(),
		amount:    req.GetAmount() + privacytypes.PrivacyTxFee,
		mixcount:  req.GetMixcount(),
	}
	privacyInfo, err := policy.getPrivacykeyPair(req.GetFrom())
	if err != nil {
		bizlog.Error("createPrivacy2PrivacyTx", "getPrivacykeyPair error", err)
		return nil, err
	}
	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := policy.buildInput(privacyInfo, buildInfo)
	if err != nil {
		return nil, err
	}
	//step 2,generateOuts
	viewPublicSlice, spendPublicSlice, err := parseViewSpendPubKeyPair(req.GetPubkeypair())
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
	privacyOutput, err := generateOuts(viewPublic, spendPublic, viewPub4chgPtr, spendPub4chgPtr, req.GetAmount(), selectedAmounTotal, privacytypes.PrivacyTxFee)
	if err != nil {
		return nil, err
	}

	value := &privacytypes.Privacy2Privacy{
		Tokenname: req.GetTokenname(),
		Amount:    req.GetAmount(),
		Note:      req.GetNote(),
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &privacytypes.PrivacyAction{
		Ty:    privacytypes.ActionPrivacy2Privacy,
		Value: &privacytypes.PrivacyAction_Privacy2Privacy{Privacy2Privacy: value},
	}

	tx := &types.Transaction{
		Execer:  []byte(privacytypes.PrivacyX),
		Payload: types.Encode(action),
		Fee:     privacytypes.PrivacyTxFee,
		Nonce:   policy.getWalletOperate().Nonce(),
		To:      address.ExecAddress(privacytypes.PrivacyX),
	}
	// 创建交易成功，将已经使用掉的UTXO冻结
	policy.saveFTXOInfo(tx, req.GetTokenname(), req.GetFrom(), common.Bytes2Hex(tx.Hash()), selectedUtxo)
	tx.Signature = &types.Signature{
		Signature: types.Encode(&privacytypes.PrivacySignatureParam{
			ActionType:    action.Ty,
			Utxobasics:    utxosInKeyInput,
			RealKeyInputs: realkeyInputSlice,
		}),
	}
	return tx, nil
}

func (policy *privacyPolicy) createPrivacy2PublicTx(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	buildInfo := &buildInputInfo{
		tokenname: req.GetTokenname(),
		sender:    req.GetFrom(),
		amount:    req.GetAmount() + privacytypes.PrivacyTxFee,
		mixcount:  req.GetMixcount(),
	}
	privacyInfo, err := policy.getPrivacykeyPair(req.GetFrom())
	if err != nil {
		bizlog.Error("createPrivacy2PublicTx failed to getPrivacykeyPair")
		return nil, err
	}
	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := policy.buildInput(privacyInfo, buildInfo)
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
	privacyOutput, err := generateOuts(nil, nil, viewPub4chgPtr, spendPub4chgPtr, 0, changeAmount, privacytypes.PrivacyTxFee)
	if err != nil {
		return nil, err
	}

	value := &privacytypes.Privacy2Public{
		Tokenname: req.GetTokenname(),
		Amount:    req.GetAmount(),
		Note:      req.GetNote(),
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &privacytypes.PrivacyAction{
		Ty:    privacytypes.ActionPrivacy2Public,
		Value: &privacytypes.PrivacyAction_Privacy2Public{Privacy2Public: value},
	}

	tx := &types.Transaction{
		Execer:  []byte(privacytypes.PrivacyX),
		Payload: types.Encode(action),
		Fee:     privacytypes.PrivacyTxFee,
		Nonce:   policy.getWalletOperate().Nonce(),
		To:      req.GetTo(),
	}
	// 创建交易成功，将已经使用掉的UTXO冻结
	policy.saveFTXOInfo(tx, req.GetTokenname(), req.GetFrom(), common.Bytes2Hex(tx.Hash()), selectedUtxo)
	tx.Signature = &types.Signature{
		Signature: types.Encode(&privacytypes.PrivacySignatureParam{
			ActionType:    action.Ty,
			Utxobasics:    utxosInKeyInput,
			RealKeyInputs: realkeyInputSlice,
		}),
	}
	return tx, nil
}

func (policy *privacyPolicy) saveFTXOInfo(tx *types.Transaction, token, sender, txhash string, selectedUtxos []*txOutputInfo) {
	//将已经作为本次交易输入的utxo进行冻结，防止产生双花交易
	policy.store.moveUTXO2FTXO(tx, token, sender, txhash, selectedUtxos)
	//TODO:需要加入超时处理，需要将此处的txhash写入到数据库中，以免钱包瞬间奔溃后没有对该笔隐私交易的记录，
	//TODO:然后当该交易得到执行之后，没法将FTXO转化为STXO，added by hezhengjun on 2018.6.5
}

func (policy *privacyPolicy) getPrivacyKeyPairs() ([]addrAndprivacy, error) {
	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := policy.store.getAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		bizlog.Info("getPrivacyKeyPairs", "store getAccountByPrefix error", err)
		return nil, err
	}

	var infoPriRes []addrAndprivacy
	for _, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			if privacyInfo, err := policy.getPrivacykeyPair(AccStore.Addr); err == nil {
				var priInfo addrAndprivacy
				priInfo.Addr = &AccStore.Addr
				priInfo.PrivacyKeyPair = privacyInfo
				infoPriRes = append(infoPriRes, priInfo)
			}
		}
	}

	if 0 == len(infoPriRes) {
		return nil, privacytypes.ErrPrivacyNotEnabled
	}

	return infoPriRes, nil

}

func (policy *privacyPolicy) rescanUTXOs(req *privacytypes.ReqRescanUtxos) (*privacytypes.RepRescanUtxos, error) {
	if req.Flag != 0 {
		return policy.store.getRescanUtxosFlag4Addr(req)
	}
	// Rescan请求
	var repRescanUtxos privacytypes.RepRescanUtxos
	repRescanUtxos.Flag = req.Flag

	operater := policy.getWalletOperate()
	if operater.IsWalletLocked() {
		return nil, types.ErrWalletIsLocked
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		return nil, err
	}
	_, err := policy.getPrivacyKeyPairs()
	if err != nil {
		return nil, err
	}
	policy.SetRescanFlag(privacytypes.UtxoFlagScaning)
	operater.GetWaitGroup().Add(1)
	go policy.rescanReqUtxosByAddr(req.Addrs)
	return &repRescanUtxos, nil
}

//从blockchain模块同步addr参与的所有交易详细信息
func (policy *privacyPolicy) rescanReqUtxosByAddr(addrs []string) {
	defer policy.getWalletOperate().GetWaitGroup().Done()
	bizlog.Debug("RescanAllUTXO begin!")
	policy.reqUtxosByAddr(addrs)
	bizlog.Debug("RescanAllUTXO sucess!")
}

func (policy *privacyPolicy) reqUtxosByAddr(addrs []string) {
	// 更新数据库存储状态
	var storeAddrs []string
	if len(addrs) == 0 {
		WalletAccStores, err := policy.store.getAccountByPrefix("Account")
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
	policy.store.saveREscanUTXOsAddresses(storeAddrs)

	reqAddr := address.ExecAddress(privacytypes.PrivacyX)
	var txInfo types.ReplyTxInfo
	i := 0
	operater := policy.getWalletOperate()
	for {
		select {
		case <-operater.GetWalletDone():
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
			if !types.IsDappFork(ReqAddr.Height, privacytypes.PrivacyX, "ForkV21Privacy") { // 小于隐私分叉高度不做扫描
				break
			}
		}
		i++
		//请求交易信息
		msg, err := operater.GetAPI().Query(privacytypes.PrivacyX, "GetTxsByAddr", &ReqAddr)
		if err != nil {
			bizlog.Error("reqUtxosByAddr", "GetTxsByAddr error", err, "addr", reqAddr)
			break
		}
		ReplyTxInfos := msg.(*types.ReplyTxInfos)
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

		policy.getPrivacyTxDetailByHashs(&ReqHashes, addrs)
		if txcount < int(MaxTxHashsPerTime) {
			break
		}
	}
	// 扫描完毕
	policy.SetRescanFlag(privacytypes.UtxoFlagNoScan)
	// 删除privacyInput
	policy.deleteScanPrivacyInputUtxo()
	policy.store.saveREscanUTXOsAddresses(storeAddrs)
}

func (policy *privacyPolicy) deleteScanPrivacyInputUtxo() {
	maxUTXOsPerTime := 1000
	for {
		utxoGlobalIndexs := policy.store.setScanPrivacyInputUTXO(int32(maxUTXOsPerTime))
		policy.store.updateScanInputUTXOs(utxoGlobalIndexs)
		if len(utxoGlobalIndexs) < maxUTXOsPerTime {
			break
		}
	}
}

func (policy *privacyPolicy) getPrivacyTxDetailByHashs(ReqHashes *types.ReqHashes, addrs []string) {
	//通过txhashs获取对应的txdetail
	TxDetails, err := policy.getWalletOperate().GetAPI().GetTransactionByHash(ReqHashes)
	if err != nil {
		bizlog.Error("getPrivacyTxDetailByHashs", "GetTransactionByHash error", err)
		return
	}
	var privacyInfo []addrAndprivacy
	if len(addrs) > 0 {
		for _, addr := range addrs {
			if privacy, err := policy.getPrivacykeyPair(addr); err != nil {
				priInfo := &addrAndprivacy{
					Addr:           &addr,
					PrivacyKeyPair: privacy,
				}
				privacyInfo = append(privacyInfo, *priInfo)
			}

		}
	} else {
		privacyInfo, _ = policy.getPrivacyKeyPairs()
	}
	policy.store.selectPrivacyTransactionToWallet(TxDetails, privacyInfo)
}

func (policy *privacyPolicy) showPrivacyAccountsSpend(req *privacytypes.ReqPrivBal4AddrToken) (*privacytypes.UTXOHaveTxHashs, error) {
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		return nil, err
	}

	addr := req.GetAddr()
	token := req.GetToken()
	utxoHaveTxHashs, err := policy.store.listSpendUTXOs(token, addr)
	if err != nil {
		return nil, err
	}
	return utxoHaveTxHashs, nil
}

func (policy *privacyPolicy) sendPublic2PrivacyTransaction(public2private *privacytypes.ReqPub2Pri) (*types.Reply, error) {
	if ok, err := policy.getWalletOperate().CheckWalletStatus(); !ok {
		bizlog.Error("sendPublic2PrivacyTransaction", "CheckWalletStatus error", err)
		return nil, err
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("sendPublic2PrivacyTransaction", "isRescanUtxosFlagScaning error", err)
		return nil, err
	}
	if public2private == nil {
		bizlog.Error("sendPublic2PrivacyTransaction public2private is nil")
		return nil, types.ErrInvalidParam
	}
	if len(public2private.GetTokenname()) <= 0 {
		bizlog.Error("sendPublic2PrivacyTransaction tokenname is nil")
		return nil, types.ErrInvalidParam
	}
	if !checkAmountValid(public2private.GetAmount()) {
		bizlog.Error("sendPublic2PrivacyTransaction", "invalid amount", public2private.GetAmount())
		return nil, types.ErrAmount
	}

	priv, err := policy.getPrivKeyByAddr(public2private.GetSender())
	if err != nil {
		bizlog.Error("sendPublic2PrivacyTransaction", "getPrivKeyByAddr error", err)
		return nil, err
	}

	return policy.transPub2PriV2(priv, public2private)
}

//公开向隐私账户转账
func (policy *privacyPolicy) transPub2PriV2(priv crypto.PrivKey, reqPub2Pri *privacytypes.ReqPub2Pri) (*types.Reply, error) {
	viewPubSlice, spendPubSlice, err := parseViewSpendPubKeyPair(reqPub2Pri.Pubkeypair)
	if err != nil {
		bizlog.Error("transPub2Pri", "parseViewSpendPubKeyPair error", err)
		return nil, err
	}

	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	//因为此时是pub2priv的交易，此时不需要构造找零的输出，同时设置fee为0，也是为了简化计算
	privacyOutput, err := generateOuts(viewPublic, spendPublic, nil, nil, reqPub2Pri.Amount, reqPub2Pri.Amount, 0)
	if err != nil {
		bizlog.Error("transPub2Pri", "generateOuts error", err)
		return nil, err
	}

	operater := policy.getWalletOperate()
	value := &privacytypes.Public2Privacy{
		Tokenname: reqPub2Pri.Tokenname,
		Amount:    reqPub2Pri.Amount,
		Note:      reqPub2Pri.Note,
		Output:    privacyOutput,
	}
	action := &privacytypes.PrivacyAction{
		Ty:    privacytypes.ActionPublic2Privacy,
		Value: &privacytypes.PrivacyAction_Public2Privacy{value},
	}
	tx := &types.Transaction{
		Execer:  []byte("privacy"),
		Payload: types.Encode(action),
		Nonce:   operater.Nonce(),
		// TODO: 采用隐私合约地址来设定目标合约接收的目标地址,让验证通过
		To: address.ExecAddress(privacytypes.PrivacyX),
	}
	tx.SetExpire(time.Duration(reqPub2Pri.GetExpire()))
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.GInt("MinFee")
	tx.Fee = realFee
	tx.Sign(int32(operater.GetSignType()), priv)

	reply, err := operater.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPub2PriV2", "Send err", err)
		return nil, err
	}
	return reply, err
}

func (policy *privacyPolicy) sendPrivacy2PrivacyTransaction(privacy2privacy *privacytypes.ReqPri2Pri) (*types.Reply, error) {
	if ok, err := policy.getWalletOperate().CheckWalletStatus(); !ok {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "CheckWalletStatus error", err)
		return nil, err
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "isRescanUtxosFlagScaning error", err)
		return nil, err
	}
	if privacy2privacy == nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction input para is nil")
		return nil, types.ErrInvalidParam
	}
	if !checkAmountValid(privacy2privacy.GetAmount()) {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "invalid amount ", privacy2privacy.GetAmount())
		return nil, types.ErrAmount
	}

	privacyInfo, err := policy.getPrivacykeyPair(privacy2privacy.GetSender())
	if err != nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "getPrivacykeyPair error ", err)
		return nil, err
	}

	return policy.transPri2PriV2(privacyInfo, privacy2privacy)
}
func (policy *privacyPolicy) transPri2PriV2(privacykeyParirs *privacy.Privacy, reqPri2Pri *privacytypes.ReqPri2Pri) (*types.Reply, error) {
	buildInfo := &buildInputInfo{
		tokenname: reqPri2Pri.Tokenname,
		sender:    reqPri2Pri.Sender,
		amount:    reqPri2Pri.Amount + privacytypes.PrivacyTxFee,
		mixcount:  reqPri2Pri.Mixin,
	}

	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := policy.buildInput(privacykeyParirs, buildInfo)
	if err != nil {
		bizlog.Error("transPri2PriV2", "buildInput error", err)
		return nil, err
	}

	//step 2,generateOuts
	viewPublicSlice, spendPublicSlice, err := parseViewSpendPubKeyPair(reqPri2Pri.Pubkeypair)
	if err != nil {
		bizlog.Error("transPub2Pri", "parseViewSpendPubKeyPair  ", err)
		return nil, err
	}

	viewPub4change, spendPub4change := privacykeyParirs.ViewPubkey.Bytes(), privacykeyParirs.SpendPubkey.Bytes()
	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPublicSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPublicSlice[0]))
	viewPub4chgPtr := (*[32]byte)(unsafe.Pointer(&viewPub4change[0]))
	spendPub4chgPtr := (*[32]byte)(unsafe.Pointer(&spendPub4change[0]))

	selectedAmounTotal := int64(0)
	for _, input := range privacyInput.Keyinput {
		selectedAmounTotal += input.Amount
	}
	//构造输出UTXO
	privacyOutput, err := generateOuts(viewPublic, spendPublic, viewPub4chgPtr, spendPub4chgPtr, reqPri2Pri.Amount, selectedAmounTotal, privacytypes.PrivacyTxFee)
	if err != nil {
		bizlog.Error("transPub2Pri", "generateOuts  ", err)
		return nil, err
	}

	operater := policy.getWalletOperate()
	value := &privacytypes.Privacy2Privacy{
		Tokenname: reqPri2Pri.Tokenname,
		Amount:    reqPri2Pri.Amount,
		Note:      reqPri2Pri.Note,
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &privacytypes.PrivacyAction{
		Ty:    privacytypes.ActionPrivacy2Privacy,
		Value: &privacytypes.PrivacyAction_Privacy2Privacy{value},
	}

	tx := &types.Transaction{
		Execer:  []byte(privacytypes.PrivacyX),
		Payload: types.Encode(action),
		Fee:     privacytypes.PrivacyTxFee,
		Nonce:   operater.Nonce(),
		// TODO: 采用隐私合约地址来设定目标合约接收的目标地址,让验证通过
		To: address.ExecAddress(privacytypes.PrivacyX),
	}
	tx.SetExpire(time.Duration(reqPri2Pri.GetExpire()))
	//完成了input和output的添加之后，即已经完成了交易基本内容的添加，
	//这时候就需要进行交易的签名了
	err = policy.signatureTx(tx, privacyInput, utxosInKeyInput, realkeyInputSlice)
	if err != nil {
		return nil, err
	}
	reply, err := operater.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPub2Pri", "SendTx  ", err)
		return nil, err
	}
	policy.saveFTXOInfo(tx, reqPri2Pri.Tokenname, reqPri2Pri.Sender, common.Bytes2Hex(tx.Hash()), selectedUtxo)
	return reply, nil
}

func (policy *privacyPolicy) signatureTx(tx *types.Transaction, privacyInput *privacytypes.PrivacyInput, utxosInKeyInput []*privacytypes.UTXOBasics, realkeyInputSlice []*privacytypes.RealKeyInput) (err error) {
	tx.Signature = nil
	data := types.Encode(tx)
	ringSign := &types.RingSignature{}
	ringSign.Items = make([]*types.RingSignatureItem, len(privacyInput.Keyinput))
	for i, input := range privacyInput.Keyinput {
		utxos := utxosInKeyInput[i]
		h := common.BytesToHash(data)
		item, err := privacy.GenerateRingSignature(h.Bytes(),
			utxos.Utxos,
			realkeyInputSlice[i].Onetimeprivkey,
			int(realkeyInputSlice[i].Realinputkey),
			input.KeyImage)
		if err != nil {
			return err
		}
		ringSign.Items[i] = item
	}

	ringSignData := types.Encode(ringSign)
	tx.Signature = &types.Signature{
		Ty:        privacytypes.RingBaseonED25519,
		Signature: ringSignData,
		// 这里填的是隐私合约的公钥，让框架保持一致
		Pubkey: address.ExecPubKey(privacytypes.PrivacyX),
	}
	return nil
}

func (policy *privacyPolicy) sendPrivacy2PublicTransaction(privacy2Pub *privacytypes.ReqPri2Pub) (*types.Reply, error) {
	if ok, err := policy.getWalletOperate().CheckWalletStatus(); !ok {
		bizlog.Error("sendPrivacy2PublicTransaction", "CheckWalletStatus error", err)
		return nil, err
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("sendPrivacy2PublicTransaction", "isRescanUtxosFlagScaning error", err)
		return nil, err
	}
	if privacy2Pub == nil {
		bizlog.Error("privacy2privacy input para is nil")
		return nil, types.ErrInvalidParam
	}
	if !checkAmountValid(privacy2Pub.GetAmount()) {
		return nil, types.ErrAmount
	}
	//get 'a'
	privacyInfo, err := policy.getPrivacykeyPair(privacy2Pub.GetSender())
	if err != nil {
		bizlog.Error("sendPrivacy2PublicTransaction", "getPrivacykeyPair error", err)
		return nil, err
	}

	return policy.transPri2PubV2(privacyInfo, privacy2Pub)
}

func (policy *privacyPolicy) transPri2PubV2(privacykeyParirs *privacy.Privacy, reqPri2Pub *privacytypes.ReqPri2Pub) (*types.Reply, error) {
	buildInfo := &buildInputInfo{
		tokenname: reqPri2Pub.Tokenname,
		sender:    reqPri2Pub.Sender,
		amount:    reqPri2Pub.Amount + privacytypes.PrivacyTxFee,
		mixcount:  reqPri2Pub.Mixin,
	}
	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := policy.buildInput(privacykeyParirs, buildInfo)
	if err != nil {
		bizlog.Error("transPri2PubV2", "buildInput error", err)
		return nil, err
	}

	viewPub4change, spendPub4change := privacykeyParirs.ViewPubkey.Bytes(), privacykeyParirs.SpendPubkey.Bytes()
	viewPub4chgPtr := (*[32]byte)(unsafe.Pointer(&viewPub4change[0]))
	spendPub4chgPtr := (*[32]byte)(unsafe.Pointer(&spendPub4change[0]))

	selectedAmounTotal := int64(0)
	for _, input := range privacyInput.Keyinput {
		if input.Amount <= 0 {
			return nil, types.ErrAmount
		}
		selectedAmounTotal += input.Amount
	}
	changeAmount := selectedAmounTotal - reqPri2Pub.Amount
	//step 2,generateOuts
	//构造输出UTXO,只生成找零的UTXO
	privacyOutput, err := generateOuts(nil, nil, viewPub4chgPtr, spendPub4chgPtr, 0, changeAmount, privacytypes.PrivacyTxFee)
	if err != nil {
		bizlog.Error("transPri2PubV2", "generateOuts error", err)
		return nil, err
	}

	operater := policy.getWalletOperate()
	value := &privacytypes.Privacy2Public{
		Tokenname: reqPri2Pub.Tokenname,
		Amount:    reqPri2Pub.Amount,
		Note:      reqPri2Pub.Note,
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &privacytypes.PrivacyAction{
		Ty:    privacytypes.ActionPrivacy2Public,
		Value: &privacytypes.PrivacyAction_Privacy2Public{value},
	}

	tx := &types.Transaction{
		Execer:  []byte(privacytypes.PrivacyX),
		Payload: types.Encode(action),
		Fee:     privacytypes.PrivacyTxFee,
		Nonce:   operater.Nonce(),
		To:      reqPri2Pub.Receiver,
	}
	tx.SetExpire(time.Duration(reqPri2Pub.GetExpire()))
	//step 3,generate ring signature
	err = policy.signatureTx(tx, privacyInput, utxosInKeyInput, realkeyInputSlice)
	if err != nil {
		bizlog.Error("transPri2PubV2", "signatureTx error", err)
		return nil, err
	}

	reply, err := operater.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPri2PubV2", "SendTx error", err)
		return nil, err
	}
	txhashstr := common.Bytes2Hex(tx.Hash())
	policy.saveFTXOInfo(tx, reqPri2Pub.Tokenname, reqPri2Pub.Sender, txhashstr, selectedUtxo)
	bizlog.Info("transPri2PubV2", "txhash", txhashstr)
	return reply, nil
}

func (policy *privacyPolicy) buildAndStoreWalletTxDetail(param *buildStoreWalletTxDetailParam) {
	blockheight := param.block.Block.Height*maxTxNumPerBlock + int64(param.index)
	heightstr := fmt.Sprintf("%018d", blockheight)
	bizlog.Debug("buildAndStoreWalletTxDetail", "heightstr", heightstr, "addDelType", param.addDelType)
	if AddTx == param.addDelType {
		var txdetail types.WalletTxDetail
		key := calcTxKey(heightstr)
		txdetail.Tx = param.tx
		txdetail.Height = param.block.Block.Height
		txdetail.Index = int64(param.index)
		txdetail.Receipt = param.block.Receipts[param.index]
		txdetail.Blocktime = param.block.Block.BlockTime

		txdetail.ActionName = txdetail.Tx.ActionName()
		txdetail.Amount, _ = param.tx.Amount()
		txdetail.Fromaddr = param.senderRecver
		//txdetail.Spendrecv = param.utxos

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			bizlog.Error("buildAndStoreWalletTxDetail err", "Height", param.block.Block.Height, "index", param.index)
			return
		}

		param.newbatch.Set(key, txdetailbyte)
		if param.isprivacy {
			//额外存储可以快速定位到接收隐私的交易
			if sendTx == param.sendRecvFlag {
				param.newbatch.Set(calcSendPrivacyTxKey(param.tokenname, param.senderRecver, heightstr), key)
			} else if recvTx == param.sendRecvFlag {
				param.newbatch.Set(calcRecvPrivacyTxKey(param.tokenname, param.senderRecver, heightstr), key)
			}
		}
	} else {
		param.newbatch.Delete(calcTxKey(heightstr))
		if param.isprivacy {
			if sendTx == param.sendRecvFlag {
				param.newbatch.Delete(calcSendPrivacyTxKey(param.tokenname, param.senderRecver, heightstr))
			} else if recvTx == param.sendRecvFlag {
				param.newbatch.Delete(calcRecvPrivacyTxKey(param.tokenname, param.senderRecver, heightstr))
			}
		}
	}
}

func (policy *privacyPolicy) checkExpireFTXOOnTimer() {
	operater := policy.getWalletOperate()
	operater.GetMutex().Lock()
	defer operater.GetMutex().Unlock()

	header := operater.GetLastHeader()
	if header == nil {
		bizlog.Error("checkExpireFTXOOnTimer Can not get last header.")
		return
	}
	policy.store.moveFTXO2UTXOWhenFTXOExpire(header.Height, header.BlockTime)
}

func (policy *privacyPolicy) checkWalletStoreData() {
	operater := policy.getWalletOperate()
	defer operater.GetWaitGroup().Done()
	timecount := 10
	checkTicker := time.NewTicker(time.Duration(timecount) * time.Second)
	for {
		select {
		case <-checkTicker.C:
			policy.checkExpireFTXOOnTimer()

			//newbatch := wallet.walletStore.NewBatch(true)
			//err := wallet.procInvalidTxOnTimer(newbatch)
			//if err != nil && err != dbm.ErrNotFoundInDb {
			//	walletlog.Error("checkWalletStoreData", "procInvalidTxOnTimer error ", err)
			//	return
			//}
			//newbatch.Write()
		case <-operater.GetWalletDone():
			return
		}
	}
}

func (policy *privacyPolicy) addDelPrivacyTxsFromBlock(tx *types.Transaction, index int32, block *types.BlockDetail, newbatch db.Batch, addDelType int32) {
	txhash := tx.Hash()
	txhashstr := common.Bytes2Hex(txhash)
	_, err := tx.Amount()
	if err != nil {
		bizlog.Error("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "index", index, "tx.Amount() error", err)
		return
	}

	txExecRes := block.Receipts[index].Ty
	var privateAction privacytypes.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		bizlog.Error("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "index", index, "Decode tx.GetPayload() error", err)
		return
	}
	bizlog.Info("addDelPrivacyTxsFromBlock", "Enter addDelPrivacyTxsFromBlock txhash", txhashstr, "index", index, "addDelType", addDelType)

	privacyOutput := privateAction.GetOutput()
	if privacyOutput == nil {
		bizlog.Error("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "index", index, "privacyOutput is", privacyOutput)
		return
	}
	tokenname := privateAction.GetTokenName()
	RpubKey := privacyOutput.GetRpubKeytx()

	totalUtxosLeft := len(privacyOutput.Keyoutput)
	//处理output
	if privacyInfo, err := policy.getPrivacyKeyPairs(); err == nil {
		matchedCount := 0
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			privacykeyParirs := info.PrivacyKeyPair
			matched4addr := false
			var utxos []*privacytypes.UTXO
			for indexoutput, output := range privacyOutput.Keyoutput {
				if utxoProcessed[indexoutput] {
					continue
				}
				priv, err := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
				if err == nil {
					recoverPub := priv.PubKey().Bytes()[:]
					if bytes.Equal(recoverPub, output.Onetimepubkey) {
						//为了避免匹配成功之后不必要的验证计算，需要统计匹配次数
						//因为目前只会往一个隐私账户转账，
						//1.一般情况下，只会匹配一次，如果是往其他钱包账户转账，
						//2.但是如果是往本钱包的其他地址转账，因为可能存在的change，会匹配2次
						matched4addr = true
						totalUtxosLeft--
						utxoProcessed[indexoutput] = true
						//只有当该交易执行成功才进行相应的UTXO的处理
						if types.ExecOk == txExecRes {
							if AddTx == addDelType {
								info2store := &privacytypes.PrivacyDBStore{
									Txhash:           txhash,
									Tokenname:        tokenname,
									Amount:           output.Amount,
									OutIndex:         int32(indexoutput),
									TxPublicKeyR:     RpubKey,
									OnetimePublicKey: output.Onetimepubkey,
									Owner:            *info.Addr,
									Height:           block.Block.Height,
									Txindex:          index,
									Blockhash:        block.Block.Hash(),
								}

								utxoGlobalIndex := &privacytypes.UTXOGlobalIndex{
									Outindex: int32(indexoutput),
									Txhash:   txhash,
								}

								utxoCreated := &privacytypes.UTXO{
									Amount: output.Amount,
									UtxoBasic: &privacytypes.UTXOBasic{
										UtxoGlobalIndex: utxoGlobalIndex,
										OnetimePubkey:   output.Onetimepubkey,
									},
								}

								utxos = append(utxos, utxoCreated)
								policy.store.setUTXO(info.Addr, &txhashstr, indexoutput, info2store, newbatch)
								bizlog.Info("addDelPrivacyTxsFromBlock", "add tx txhash", txhashstr, "setUTXO addr ", *info.Addr, "indexoutput", indexoutput)
							} else {
								policy.store.unsetUTXO(info.Addr, &txhashstr, indexoutput, tokenname, newbatch)
								bizlog.Info("addDelPrivacyTxsFromBlock", "delete tx txhash", txhashstr, "unsetUTXO addr ", *info.Addr, "indexoutput", indexoutput)
							}
						} else {
							//对于执行失败的交易，只需要将该交易记录在钱包就行
							bizlog.Error("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "txExecRes", txExecRes)
							break
						}
					}
				} else {
					bizlog.Error("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "RecoverOnetimePriKey error", err)
				}
			}
			if matched4addr {
				matchedCount++
				//匹配次数达到2次，不再对本钱包中的其他地址进行匹配尝试
				bizlog.Debug("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "address", *info.Addr, "totalUtxosLeft", totalUtxosLeft, "matchedCount", matchedCount)
				param := &buildStoreWalletTxDetailParam{
					tokenname:    tokenname,
					block:        block,
					tx:           tx,
					index:        int(index),
					newbatch:     newbatch,
					senderRecver: *info.Addr,
					isprivacy:    true,
					addDelType:   addDelType,
					sendRecvFlag: recvTx,
					utxos:        utxos,
				}
				policy.buildAndStoreWalletTxDetail(param)
				if 2 == matchedCount || 0 == totalUtxosLeft || types.ExecOk != txExecRes {
					bizlog.Info("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "Get matched privacy transfer for address address", *info.Addr, "totalUtxosLeft", totalUtxosLeft, "matchedCount", matchedCount)
					break
				}
			}
		}
	}

	//处理input,对于公对私的交易类型，只会出现在output类型处理中
	//如果该隐私交易是本钱包中的地址发送出去的，则需要对相应的utxo进行处理
	if AddTx == addDelType {
		ftxos, keys := policy.store.getFTXOlist()
		for i, ftxo := range ftxos {
			//查询确认该交易是否为记录的支付交易
			if ftxo.Txhash != txhashstr {
				continue
			}
			if types.ExecOk == txExecRes && privacytypes.ActionPublic2Privacy != privateAction.Ty {
				bizlog.Info("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveFTXO2STXO, key", string(keys[i]), "txExecRes", txExecRes)
				policy.store.moveFTXO2STXO(keys[i], txhashstr, newbatch)
			} else if types.ExecOk != txExecRes && privacytypes.ActionPublic2Privacy != privateAction.Ty {
				//如果执行失败
				bizlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveFTXO2UTXO, key", string(keys[i]), "txExecRes", txExecRes)
				policy.store.moveFTXO2UTXO(keys[i], newbatch)
			}
			//该交易正常执行完毕，删除对其的关注
			param := &buildStoreWalletTxDetailParam{
				tokenname:    tokenname,
				block:        block,
				tx:           tx,
				index:        int(index),
				newbatch:     newbatch,
				senderRecver: ftxo.Sender,
				isprivacy:    true,
				addDelType:   addDelType,
				sendRecvFlag: sendTx,
				utxos:        nil,
			}
			policy.buildAndStoreWalletTxDetail(param)
		}
	} else {
		//当发生交易回撤时，从记录的STXO中查找相关的交易，并将其重置为FTXO，因为该交易大概率会在其他区块中再次执行
		stxosInOneTx, _, _ := policy.store.getWalletFtxoStxo(STXOs4Tx)
		for _, ftxo := range stxosInOneTx {
			if ftxo.Txhash == txhashstr {
				param := &buildStoreWalletTxDetailParam{
					tokenname:    tokenname,
					block:        block,
					tx:           tx,
					index:        int(index),
					newbatch:     newbatch,
					senderRecver: "",
					isprivacy:    true,
					addDelType:   addDelType,
					sendRecvFlag: sendTx,
					utxos:        nil,
				}

				if types.ExecOk == txExecRes && privacytypes.ActionPublic2Privacy != privateAction.Ty {
					bizlog.Info("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveSTXO2FTXO txExecRes", txExecRes)
					policy.store.moveSTXO2FTXO(tx, txhashstr, newbatch)
					policy.buildAndStoreWalletTxDetail(param)
				} else if types.ExecOk != txExecRes && privacytypes.ActionPublic2Privacy != privateAction.Ty {
					bizlog.Info("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType)
					policy.buildAndStoreWalletTxDetail(param)
				}
			}
		}
	}
}
