package privacybizpolicy

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"time"
	"unsafe"

	"sync"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

func (biz *walletPrivacyBiz) rescanAllTxAddToUpdateUTXOs() {
	accounts, err := biz.walletOperate.GetWalletAccounts()
	if err != nil {
		bizlog.Error("rescanAllTxToUpdateUTXOs", "walletOperate.GetWalletAccounts error", err)
		return
	}
	bizlog.Debug("rescanAllTxToUpdateUTXOs begin!")
	for _, acc := range accounts {
		//从blockchain模块同步Account.Addr对应的所有交易详细信息
		biz.rescanwg.Add(1)
		go biz.rescanReqTxDetailByAddr(acc.Addr, biz.rescanwg)
	}
	biz.rescanwg.Wait()

	bizlog.Debug("rescanAllTxToUpdateUTXOs sucess!")
}

//从blockchain模块同步addr参与的所有交易详细信息
func (biz *walletPrivacyBiz) rescanReqTxDetailByAddr(addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	biz.reqTxDetailByAddr(addr)
}

//从blockchain模块同步addr参与的所有交易详细信息
func (biz *walletPrivacyBiz) reqTxDetailByAddr(addr string) {
	if len(addr) == 0 {
		bizlog.Error("reqTxDetailByAddr input addr is nil!")
		return
	}
	var txInfo types.ReplyTxInfo

	i := 0
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
		ReplyTxInfos, err := biz.walletOperate.GetAPI().GetTransactionByAddr(&ReqAddr)
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
		biz.walletOperate.GetTxDetailByHashs(&ReqHashes)
		if txcount < int(MaxTxHashsPerTime) {
			return
		}
	}
}

func (biz *walletPrivacyBiz) isRescanUtxosFlagScaning() (bool, error) {
	if types.UtxoFlagScaning == biz.walletOperate.GetRescanFlag() {
		return true, types.ErrRescanFlagScaning
	}
	return false, nil
}

func (biz *walletPrivacyBiz) createUTXOs(createUTXOs *types.ReqCreateUTXOs) (*types.Reply, error) {
	ok, err := biz.walletOperate.CheckWalletStatus()
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
	if !checkAmountValid(createUTXOs.GetAmount()) {
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

//批量创建通过public2Privacy实现
func (biz *walletPrivacyBiz) createUTXOsByPub2Priv(priv crypto.PrivKey, reqCreateUTXOs *types.ReqCreateUTXOs) (*types.Reply, error) {
	viewPubSlice, spendPubSlice, err := parseViewSpendPubKeyPair(reqCreateUTXOs.GetPubkeypair())
	if err != nil {
		bizlog.Error("createUTXOsByPub2Priv", "parseViewSpendPubKeyPair error.", err)
		return nil, err
	}

	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	//因为此时是pub2priv的交易，此时不需要构造找零的输出，同时设置fee为0，也是为了简化计算
	privacyOutput, err := genCustomOuts(viewPublic, spendPublic, reqCreateUTXOs.Amount, reqCreateUTXOs.Count)
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
		Nonce:   biz.walletOperate.Nonce(),
		To:      address.ExecAddress(types.PrivacyX),
	}
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.FeePerKB
	tx.Fee = realFee
	tx.Sign(int32(biz.walletOperate.GetSignType()), priv)

	reply, err := biz.walletOperate.GetAPI().SendTx(tx)
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

	password := []byte(biz.walletOperate.GetPassword())
	privkey := CBCDecrypterPrivkey(password, prikeybyte)
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(biz.walletOperate.GetSignType()))
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
		password := []byte(biz.walletOperate.GetPassword())
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

	password := []byte(biz.walletOperate.GetPassword())
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
		Pubkeypair:     makeViewSpendPubKeyPairToString(privacyInfo.ViewPubkey[:], privacyInfo.SpendPubkey[:]),
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
	curBlockHeight := biz.walletOperate.GetBlockHeight()
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
		index := biz.walletOperate.GetRandom().Intn(len(confirmUTXOs))
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
	sort.Slice(selectedUtxo, func(i, j int) bool {
		return selectedUtxo[i].amount <= selectedUtxo[j].amount
	})

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
		resUTXOGlobalIndex, err = biz.walletOperate.GetAPI().BlockChainQuery(query)
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
		positions := biz.walletOperate.GetRandom().Perm(len(utxoIndex4Amount.Utxos))
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

func (biz *walletPrivacyBiz) createTransaction(req *types.ReqCreateTransaction) (*types.Transaction, error) {
	switch req.Type {
	case types.PrivacyTypePublic2Privacy:
		return biz.createPublic2PrivacyTx(req)
	case types.PrivacyTypePrivacy2Privacy:
		return biz.createPrivacy2PrivacyTx(req)
	case types.PrivacyTypePrivacy2Public:
		return biz.createPrivacy2PublicTx(req)
	}
	return nil, types.ErrInvalidParams
}

func (biz *walletPrivacyBiz) createPublic2PrivacyTx(req *types.ReqCreateTransaction) (*types.Transaction, error) {
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
		Nonce:   biz.walletOperate.Nonce(),
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
	privacyOutput, err := generateOuts(viewPublic, spendPublic, viewPub4chgPtr, spendPub4chgPtr, req.GetAmount(), selectedAmounTotal, types.PrivacyTxFee)
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
		Nonce:   biz.walletOperate.Nonce(),
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
	privacyOutput, err := generateOuts(nil, nil, viewPub4chgPtr, spendPub4chgPtr, 0, changeAmount, types.PrivacyTxFee)
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
		Nonce:   biz.walletOperate.Nonce(),
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

	if biz.walletOperate.IsWalletLocked() {
		return nil, types.ErrWalletIsLocked
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		return nil, err
	}
	_, err := biz.getPrivacyKeyPairs()
	if err != nil {
		return nil, err
	}
	biz.walletOperate.SetRescanFlag(types.UtxoFlagScaning)
	biz.walletOperate.GetWaitGroup().Add(1)
	go biz.rescanReqUtxosByAddr(req.Addrs)
	return &repRescanUtxos, nil
}

//从blockchain模块同步addr参与的所有交易详细信息
func (biz *walletPrivacyBiz) rescanReqUtxosByAddr(addrs []string) {
	defer biz.walletOperate.GetWaitGroup().Done()
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
		case <-biz.walletOperate.GetWalletDone():
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
		msg, err := biz.walletOperate.GetAPI().Query(&types.Query{
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
	biz.walletOperate.SetRescanFlag(types.UtxoFlagNoScan)
	// 删除privacyInput
	biz.deleteScanPrivacyInputUtxo()
	biz.store.saveREscanUTXOsAddresses(storeAddrs)
}

func (biz *walletPrivacyBiz) deleteScanPrivacyInputUtxo() {
	maxUTXOsPerTime := 1000
	for {
		utxoGlobalIndexs := biz.store.setScanPrivacyInputUTXO(int32(maxUTXOsPerTime))
		biz.store.updateScanInputUTXOs(utxoGlobalIndexs)
		if len(utxoGlobalIndexs) < maxUTXOsPerTime {
			break
		}
	}
}

func (biz *walletPrivacyBiz) getPrivacyTxDetailByHashs(ReqHashes *types.ReqHashes, addrs []string) {
	//通过txhashs获取对应的txdetail
	TxDetails, err := biz.walletOperate.GetAPI().GetTransactionByHash(ReqHashes)
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

func (biz *walletPrivacyBiz) sendPublic2PrivacyTransaction(public2private *types.ReqPub2Pri) (*types.Reply, error) {
	if ok, err := biz.walletOperate.CheckWalletStatus(); !ok {
		bizlog.Error("sendPublic2PrivacyTransaction", "CheckWalletStatus error", err)
		return nil, err
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("sendPublic2PrivacyTransaction", "isRescanUtxosFlagScaning error", err)
		return nil, err
	}
	if public2private == nil {
		bizlog.Error("sendPublic2PrivacyTransaction public2private is nil")
		return nil, types.ErrInputPara
	}
	if len(public2private.GetTokenname()) <= 0 {
		bizlog.Error("sendPublic2PrivacyTransaction tokenname is nil")
		return nil, types.ErrInvalidParams
	}
	if !checkAmountValid(public2private.GetAmount()) {
		bizlog.Error("sendPublic2PrivacyTransaction", "invalid amount", public2private.GetAmount())
		return nil, types.ErrAmount
	}

	priv, err := biz.getPrivKeyByAddr(public2private.GetSender())
	if err != nil {
		bizlog.Error("sendPublic2PrivacyTransaction", "getPrivKeyByAddr error", err)
		return nil, err
	}

	return biz.transPub2PriV2(priv, public2private)
}

//公开向隐私账户转账
func (biz *walletPrivacyBiz) transPub2PriV2(priv crypto.PrivKey, reqPub2Pri *types.ReqPub2Pri) (*types.Reply, error) {
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

	value := &types.Public2Privacy{
		Tokenname: reqPub2Pri.Tokenname,
		Amount:    reqPub2Pri.Amount,
		Note:      reqPub2Pri.Note,
		Output:    privacyOutput,
	}
	action := &types.PrivacyAction{
		Ty:    types.ActionPublic2Privacy,
		Value: &types.PrivacyAction_Public2Privacy{value},
	}
	tx := &types.Transaction{
		Execer:  []byte("privacy"),
		Payload: types.Encode(action),
		Nonce:   biz.walletOperate.Nonce(),
		// TODO: 采用隐私合约地址来设定目标合约接收的目标地址,让验证通过
		To: address.ExecAddress(types.PrivacyX),
	}
	tx.SetExpire(time.Duration(reqPub2Pri.GetExpire()))
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.FeePerKB
	tx.Fee = realFee
	tx.Sign(int32(biz.walletOperate.GetSignType()), priv)

	reply, err := biz.walletOperate.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPub2PriV2", "Send err", err)
		return nil, err
	}
	return reply, err
}

func (biz *walletPrivacyBiz) sendPrivacy2PrivacyTransaction(privacy2privacy *types.ReqPri2Pri) (*types.Reply, error) {
	if ok, err := biz.walletOperate.CheckWalletStatus(); !ok {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "CheckWalletStatus error", err)
		return nil, err
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "isRescanUtxosFlagScaning error", err)
		return nil, err
	}
	if privacy2privacy == nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction input para is nil")
		return nil, types.ErrInputPara
	}
	if !checkAmountValid(privacy2privacy.GetAmount()) {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "invalid amount ", privacy2privacy.GetAmount())
		return nil, types.ErrAmount
	}

	privacyInfo, err := biz.getPrivacykeyPair(privacy2privacy.GetSender())
	if err != nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "getPrivacykeyPair error ", err)
		return nil, err
	}

	return biz.transPri2PriV2(privacyInfo, privacy2privacy)
}
func (biz *walletPrivacyBiz) transPri2PriV2(privacykeyParirs *privacy.Privacy, reqPri2Pri *types.ReqPri2Pri) (*types.Reply, error) {
	buildInfo := &buildInputInfo{
		tokenname: reqPri2Pri.Tokenname,
		sender:    reqPri2Pri.Sender,
		amount:    reqPri2Pri.Amount + types.PrivacyTxFee,
		mixcount:  reqPri2Pri.Mixin,
	}

	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := biz.buildInput(privacykeyParirs, buildInfo)
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
	privacyOutput, err := generateOuts(viewPublic, spendPublic, viewPub4chgPtr, spendPub4chgPtr, reqPri2Pri.Amount, selectedAmounTotal, types.PrivacyTxFee)
	if err != nil {
		bizlog.Error("transPub2Pri", "generateOuts  ", err)
		return nil, err
	}

	value := &types.Privacy2Privacy{
		Tokenname: reqPri2Pri.Tokenname,
		Amount:    reqPri2Pri.Amount,
		Note:      reqPri2Pri.Note,
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &types.PrivacyAction{
		Ty:    types.ActionPrivacy2Privacy,
		Value: &types.PrivacyAction_Privacy2Privacy{value},
	}

	tx := &types.Transaction{
		Execer:  []byte(types.PrivacyX),
		Payload: types.Encode(action),
		Fee:     types.PrivacyTxFee,
		Nonce:   biz.walletOperate.Nonce(),
		// TODO: 采用隐私合约地址来设定目标合约接收的目标地址,让验证通过
		To: address.ExecAddress(types.PrivacyX),
	}
	tx.SetExpire(time.Duration(reqPri2Pri.GetExpire()))
	//完成了input和output的添加之后，即已经完成了交易基本内容的添加，
	//这时候就需要进行交易的签名了
	err = biz.signatureTx(tx, privacyInput, utxosInKeyInput, realkeyInputSlice)
	if err != nil {
		return nil, err
	}
	reply, err := biz.walletOperate.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPub2Pri", "SendTx  ", err)
		return nil, err
	}
	biz.saveFTXOInfo(tx, reqPri2Pri.Tokenname, reqPri2Pri.Sender, common.Bytes2Hex(tx.Hash()), selectedUtxo)
	return reply, nil
}

func (biz *walletPrivacyBiz) signatureTx(tx *types.Transaction, privacyInput *types.PrivacyInput, utxosInKeyInput []*types.UTXOBasics, realkeyInputSlice []*types.RealKeyInput) (err error) {
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
		Ty:        types.RingBaseonED25519,
		Signature: ringSignData,
		// 这里填的是隐私合约的公钥，让框架保持一致
		Pubkey: address.ExecPubKey(types.PrivacyX),
	}
	return nil
}

func (biz *walletPrivacyBiz) sendPrivacy2PublicTransaction(privacy2Pub *types.ReqPri2Pub) (*types.Reply, error) {
	if ok, err := biz.walletOperate.CheckWalletStatus(); !ok {
		bizlog.Error("sendPrivacy2PublicTransaction", "CheckWalletStatus error", err)
		return nil, err
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("sendPrivacy2PublicTransaction", "isRescanUtxosFlagScaning error", err)
		return nil, err
	}
	if privacy2Pub == nil {
		bizlog.Error("privacy2privacy input para is nil")
		return nil, types.ErrInputPara
	}
	if !checkAmountValid(privacy2Pub.GetAmount()) {
		return nil, types.ErrAmount
	}
	//get 'a'
	privacyInfo, err := biz.getPrivacykeyPair(privacy2Pub.GetSender())
	if err != nil {
		bizlog.Error("sendPrivacy2PublicTransaction", "getPrivacykeyPair error", err)
		return nil, err
	}

	return biz.transPri2PubV2(privacyInfo, privacy2Pub)
}

func (biz *walletPrivacyBiz) transPri2PubV2(privacykeyParirs *privacy.Privacy, reqPri2Pub *types.ReqPri2Pub) (*types.Reply, error) {
	buildInfo := &buildInputInfo{
		tokenname: reqPri2Pub.Tokenname,
		sender:    reqPri2Pub.Sender,
		amount:    reqPri2Pub.Amount + types.PrivacyTxFee,
		mixcount:  reqPri2Pub.Mixin,
	}
	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := biz.buildInput(privacykeyParirs, buildInfo)
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
	privacyOutput, err := generateOuts(nil, nil, viewPub4chgPtr, spendPub4chgPtr, 0, changeAmount, types.PrivacyTxFee)
	if err != nil {
		bizlog.Error("transPri2PubV2", "generateOuts error", err)
		return nil, err
	}

	value := &types.Privacy2Public{
		Tokenname: reqPri2Pub.Tokenname,
		Amount:    reqPri2Pub.Amount,
		Note:      reqPri2Pub.Note,
		Input:     privacyInput,
		Output:    privacyOutput,
	}
	action := &types.PrivacyAction{
		Ty:    types.ActionPrivacy2Public,
		Value: &types.PrivacyAction_Privacy2Public{value},
	}

	tx := &types.Transaction{
		Execer:  []byte(types.PrivacyX),
		Payload: types.Encode(action),
		Fee:     types.PrivacyTxFee,
		Nonce:   biz.walletOperate.Nonce(),
		To:      reqPri2Pub.Receiver,
	}
	tx.SetExpire(time.Duration(reqPri2Pub.GetExpire()))
	//step 3,generate ring signature
	err = biz.signatureTx(tx, privacyInput, utxosInKeyInput, realkeyInputSlice)
	if err != nil {
		bizlog.Error("transPri2PubV2", "signatureTx error", err)
		return nil, err
	}

	reply, err := biz.walletOperate.GetAPI().SendTx(tx)
	if err != nil {
		bizlog.Error("transPri2PubV2", "SendTx error", err)
		return nil, err
	}
	txhashstr := common.Bytes2Hex(tx.Hash())
	biz.saveFTXOInfo(tx, reqPri2Pub.Tokenname, reqPri2Pub.Sender, txhashstr, selectedUtxo)
	bizlog.Info("transPri2PubV2", "txhash", txhashstr)
	return reply, nil
}

func (biz *walletPrivacyBiz) buildAndStoreWalletTxDetail(param *buildStoreWalletTxDetailParam) {
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
		txdetail.Spendrecv = param.utxos

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

func (biz *walletPrivacyBiz) checkExpireFTXOOnTimer() {
	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	header := biz.walletOperate.GetLastHeader()
	if header == nil {
		bizlog.Error("checkExpireFTXOOnTimer Can not get last header.")
		return
	}
	biz.store.moveFTXO2UTXOWhenFTXOExpire(header.Height, header.BlockTime)
}

func (biz *walletPrivacyBiz) checkWalletStoreData() {
	defer biz.walletOperate.GetWaitGroup().Done()
	timecount := 10
	checkTicker := time.NewTicker(time.Duration(timecount) * time.Second)
	for {
		select {
		case <-checkTicker.C:
			biz.checkExpireFTXOOnTimer()

			//newbatch := wallet.walletStore.NewBatch(true)
			//err := wallet.procInvalidTxOnTimer(newbatch)
			//if err != nil && err != dbm.ErrNotFoundInDb {
			//	walletlog.Error("checkWalletStoreData", "procInvalidTxOnTimer error ", err)
			//	return
			//}
			//newbatch.Write()
		case <-biz.walletOperate.GetWalletDone():
			return
		}
	}
}

func (biz *walletPrivacyBiz) addDelPrivacyTxsFromBlock(tx *types.Transaction, index int32, block *types.BlockDetail, newbatch db.Batch, addDelType int32) {
	txhash := tx.Hash()
	txhashstr := common.Bytes2Hex(txhash)
	_, err := tx.Amount()
	if err != nil {
		bizlog.Error("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "index", index, "tx.Amount() error", err)
		return
	}

	txExecRes := block.Receipts[index].Ty
	var privateAction types.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		bizlog.Error("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "index", index, "Decode tx.GetPayload() error", err)
		return
	}
	bizlog.Info("addDelPrivacyTxsFromBlock", "Enter addDelPrivacyTxsFromBlock txhash", txhashstr, "index", index, "addDelType", addDelType)

	privacyOutput := privateAction.GetOutput()
	tokenname := privateAction.GetTokenName()
	RpubKey := privacyOutput.GetRpubKeytx()

	totalUtxosLeft := len(privacyOutput.Keyoutput)
	//处理output
	if privacyInfo, err := biz.getPrivacyKeyPairs(); err == nil {
		matchedCount := 0
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			privacykeyParirs := info.PrivacyKeyPair
			matched4addr := false
			var utxos []*types.UTXO
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
								info2store := &types.PrivacyDBStore{
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

								utxoGlobalIndex := &types.UTXOGlobalIndex{
									Outindex: int32(indexoutput),
									Txhash:   txhash,
								}

								utxoCreated := &types.UTXO{
									Amount: output.Amount,
									UtxoBasic: &types.UTXOBasic{
										UtxoGlobalIndex: utxoGlobalIndex,
										OnetimePubkey:   output.Onetimepubkey,
									},
								}

								utxos = append(utxos, utxoCreated)
								biz.store.setUTXO(info.Addr, &txhashstr, indexoutput, info2store, newbatch)
								bizlog.Info("addDelPrivacyTxsFromBlock", "add tx txhash", txhashstr, "setUTXO addr ", *info.Addr, "indexoutput", indexoutput)
							} else {
								biz.store.unsetUTXO(info.Addr, &txhashstr, indexoutput, tokenname, newbatch)
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
				biz.buildAndStoreWalletTxDetail(param)
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
		ftxos, keys := biz.store.getFTXOlist()
		for i, ftxo := range ftxos {
			//查询确认该交易是否为记录的支付交易
			if ftxo.Txhash != txhashstr {
				continue
			}
			if types.ExecOk == txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				bizlog.Info("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveFTXO2STXO, key", string(keys[i]), "txExecRes", txExecRes)
				biz.store.moveFTXO2STXO(keys[i], txhashstr, newbatch)
			} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				//如果执行失败
				bizlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveFTXO2UTXO, key", string(keys[i]), "txExecRes", txExecRes)
				biz.store.moveFTXO2UTXO(keys[i], newbatch)
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
			biz.buildAndStoreWalletTxDetail(param)
		}
	} else {
		//当发生交易回撤时，从记录的STXO中查找相关的交易，并将其重置为FTXO，因为该交易大概率会在其他区块中再次执行
		stxosInOneTx, _, _ := biz.store.getWalletFtxoStxo(STXOs4Tx)
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

				if types.ExecOk == txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
					bizlog.Info("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveSTXO2FTXO txExecRes", txExecRes)
					biz.store.moveSTXO2FTXO(tx, txhashstr, newbatch)
					biz.buildAndStoreWalletTxDetail(param)
				} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
					bizlog.Info("addDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType)
					biz.buildAndStoreWalletTxDetail(param)
				}
			}
		}
	}
}
