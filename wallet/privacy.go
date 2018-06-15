package wallet

import (
	"bytes"
	"errors"
	"sort"
	"unsafe"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/types"
)

type realkeyInput struct {
	realInputIndex int
	onetimePrivKey []byte
}

type buildInputInfo struct {
	tokenname *string
	sender    *string
	amount    int64
	mixcount  int32
}

func checkAmountValid(amount int64) bool {
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

func (wallet *Wallet) procPublic2PrivacyV2(public2private *types.ReqPub2Pri) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if public2private == nil {
		walletlog.Error("public2private input para is nil")
		return nil, types.ErrInputPara
	}
	if len(public2private.GetTokenname()) <= 0 {
		return nil, types.ErrInvalidParams
	}
	if !checkAmountValid(public2private.GetAmount()) {
		return nil, types.ErrAmount
	}

	priv, err := wallet.getPrivKeyByAddr(public2private.GetSender())
	if err != nil {
		return nil, err
	}

	return wallet.transPub2PriV2(priv, public2private)
}

func (wallet *Wallet) procPrivacy2PrivacyV2(privacy2privacy *types.ReqPri2Pri) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if privacy2privacy == nil {
		walletlog.Error("privacy2privacy input para is nil")
		return nil, types.ErrInputPara
	}
	if !checkAmountValid(privacy2privacy.GetAmount()) {
		return nil, types.ErrAmount
	}

	privacyInfo, err := wallet.getPrivacykeyPair(privacy2privacy.GetSender())
	if err != nil {
		walletlog.Error("privacy2privacy failed to getPrivacykeyPair")
		return nil, err
	}

	return wallet.transPri2PriV2(privacyInfo, privacy2privacy)
}

func (wallet *Wallet) procPrivacy2PublicV2(privacy2Pub *types.ReqPri2Pub) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if privacy2Pub == nil {
		walletlog.Error("privacy2privacy input para is nil")
		return nil, types.ErrInputPara
	}
	if !checkAmountValid(privacy2Pub.GetAmount()) {
		return nil, types.ErrAmount
	}
	//get 'a'
	privacyInfo, err := wallet.getPrivacykeyPair(privacy2Pub.GetSender())
	if err != nil {
		return nil, err
	}

	return wallet.transPri2PubV2(privacyInfo, privacy2Pub)
}

func (wallet *Wallet) procCreateUTXOs(createUTXOs *types.ReqCreateUTXOs) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if createUTXOs == nil {
		walletlog.Error("privacy2privacy input para is nil")
		return nil, types.ErrInputPara
	}
	if !checkAmountValid(createUTXOs.GetAmount()) {
		walletlog.Error("not allow amount number")
		return nil, types.ErrAmount
	}
	priv, err := wallet.getPrivKeyByAddr(createUTXOs.GetSender())
	if err != nil {
		return nil, err
	}

	return wallet.createUTXOsByPub2Priv(priv, createUTXOs)
}

//批量创建通过public2Privacy实现
func (wallet *Wallet) createUTXOsByPub2Priv(priv crypto.PrivKey, reqCreateUTXOs *types.ReqCreateUTXOs) (*types.ReplyHash, error) {
	viewPubSlice, spendPubSlice, err := parseViewSpendPubKeyPair(reqCreateUTXOs.GetPubkeypair())
	if err != nil {
		return nil, err
	}

	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	//因为此时是pub2priv的交易，此时不需要构造找零的输出，同时设置fee为0，也是为了简化计算
	privacyOutput, err := genCustomOuts(viewPublic, spendPublic, reqCreateUTXOs.Amount, reqCreateUTXOs.Count)
	if err != nil {
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
		Nonce:   wallet.random.Int63(),
		To:      account.ExecAddress(types.PrivacyX),
	}
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.FeePerKB
	tx.Fee = realFee
	tx.Sign(int32(SignType), priv)

	msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("transPub2PriV2", "Send err", err)
		return nil, err
	}

	reply := resp.GetData().(*types.Reply)
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()
	return &hash, nil
}

func parseViewSpendPubKeyPair(in string) (viewPubKey, spendPubKey []byte, err error) {
	src, err := common.FromHex(in)
	if err != nil {
		return nil, nil, err
	}
	if 64 != len(src) {
		walletlog.Error("parseViewSpendPubKeyPair", "pair with len", len(src))
		return nil, nil, types.ErrPubKeyLen
	}
	viewPubKey = src[:32]
	spendPubKey = src[32:]
	return
}

//公开向隐私账户转账
func (wallet *Wallet) transPub2PriV2(priv crypto.PrivKey, reqPub2Pri *types.ReqPub2Pri) (*types.ReplyHash, error) {
	viewPubSlice, spendPubSlice, err := parseViewSpendPubKeyPair(reqPub2Pri.Pubkeypair)
	if err != nil {
		walletlog.Error("transPub2Pri", "parseViewSpendPubKeyPair  ", err)
		return nil, err
	}

	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	//因为此时是pub2priv的交易，此时不需要构造找零的输出，同时设置fee为0，也是为了简化计算
	privacyOutput, err := generateOuts(viewPublic, spendPublic, nil, nil, reqPub2Pri.Amount, reqPub2Pri.Amount, 0)
	if err != nil {
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
		Nonce:   wallet.random.Int63(),
		// TODO: 采用隐私合约地址来设定目标合约接收的目标地址,让验证通过
		To: account.ExecAddress(types.PrivacyX),
	}
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.FeePerKB
	tx.Fee = realFee
	tx.Sign(int32(SignType), priv)

	msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("transPub2PriV2", "Send err", err)
		return nil, err
	}

	reply := resp.GetData().(*types.Reply)
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()
	return &hash, nil
}

func genCustomOuts(viewpubTo, spendpubto *[32]byte, transAmount int64, count int32) (*types.PrivacyOutput, error) {
	decomDigit := make([]int64, count)
	for i, _ := range decomDigit {
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
			walletlog.Error("genCustomOuts", "Fail to GenerateOneTimeAddr due to cause", err)
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

//最后构造完成的utxo依次是2种类型，不构造交易费utxo，使其直接燃烧消失
//1.进行实际转账utxo
//2.进行找零转账utxo
func generateOuts(viewpubTo, spendpubto, viewpubChangeto, spendpubChangeto *[32]byte, transAmount, selectedAmount, fee int64) (*types.PrivacyOutput, error) {
	decomDigit := decomposeAmount2digits(transAmount, types.BTYDustThreshold)
	//计算找零
	changeAmount := selectedAmount - transAmount - fee
	var decomChange []int64
	if 0 < changeAmount {
		decomChange = decomposeAmount2digits(changeAmount, types.BTYDustThreshold)
	}
	walletlog.Info("generateOuts", "decompose digit for amount", selectedAmount-fee, "decomDigit", decomDigit)

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
			walletlog.Error("generateOuts", "Fail to GenerateOneTimeAddr due to cause", err)
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
			walletlog.Error("generateOuts", "Fail to GenerateOneTimeAddr for change due to cause", err)
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
		//	walletlog.Error("transPub2PriV2", "Fail to GenerateOneTimeAddr for fee due to cause", err)
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

func (w *Wallet) signatureTx(tx *types.Transaction, privacyInput *types.PrivacyInput, utxosInKeyInput [][]*types.UTXOBasic, realkeyInputSlice []*realkeyInput) (err error) {
	tx.Signature = nil
	data := types.Encode(tx)
	ringSign := &types.RingSignature{}
	ringSign.Items = make([]*types.RingSignatureItem, len(privacyInput.Keyinput))
	for i, input := range privacyInput.Keyinput {
		utxos := utxosInKeyInput[i]
		h := common.BytesToHash(data)
		item, err := privacy.GenerateRingSignature(h.Bytes(),
			utxos,
			realkeyInputSlice[i].onetimePrivKey,
			realkeyInputSlice[i].realInputIndex,
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
		Pubkey: account.ExecPubKey(types.PrivacyX),
	}
	return nil
}

func (wallet *Wallet) transPri2PriV2(privacykeyParirs *privacy.Privacy, reqPri2Pri *types.ReqPri2Pri) (*types.ReplyHash, error) {
	buildInfo := &buildInputInfo{
		tokenname: &reqPri2Pri.Tokenname,
		sender:    &reqPri2Pri.Sender,
		// TODO: 这里存在手续费不足的情况,需要考虑扣除手续费以后的拆分问题,所以这里先简单的放大,让调试通过
		amount:   reqPri2Pri.Amount + types.PrivacyTxFee,
		mixcount: reqPri2Pri.Mixin,
	}

	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := wallet.buildInput(privacykeyParirs, buildInfo)
	if err != nil {
		return nil, err
	}

	//step 2,generateOuts
	viewPublicSlice, spendPublicSlice, err := parseViewSpendPubKeyPair(reqPri2Pri.Pubkeypair)
	if err != nil {
		walletlog.Error("transPub2Pri", "parseViewSpendPubKeyPair  ", err)
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
		Nonce:   wallet.random.Int63(),
		// TODO: 采用隐私合约地址来设定目标合约接收的目标地址,让验证通过
		To: account.ExecAddress(types.PrivacyX),
	}

	//完成了input和output的添加之后，即已经完成了交易基本内容的添加，
	//这时候就需要进行交易的签名了
	err = wallet.signatureTx(tx, privacyInput, utxosInKeyInput, realkeyInputSlice)
	if err != nil {
		return nil, err
	}

	msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("transPri2Pri", "Send err", err)
		return nil, err
	}

	reply := resp.GetData().(*types.Reply)
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()
	wallet.saveFTXOInfo(reqPri2Pri.Tokenname, reqPri2Pri.Sender, common.Bytes2Hex(hash.Hash), selectedUtxo)
	return &hash, nil
}

func (wallet *Wallet) transPri2PubV2(privacykeyParirs *privacy.Privacy, reqPri2Pub *types.ReqPri2Pub) (*types.ReplyHash, error) {
	buildInfo := &buildInputInfo{
		tokenname: &reqPri2Pub.Tokenname,
		sender:    &reqPri2Pub.Sender,
		amount:    reqPri2Pub.Amount + types.PrivacyTxFee,
		mixcount:  reqPri2Pub.Mixin,
	}
	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := wallet.buildInput(privacykeyParirs, buildInfo)
	if err != nil {
		return nil, err
	}

	viewPub4change, spendPub4change := privacykeyParirs.ViewPubkey.Bytes(), privacykeyParirs.SpendPubkey.Bytes()
	viewPub4chgPtr := (*[32]byte)(unsafe.Pointer(&viewPub4change[0]))
	spendPub4chgPtr := (*[32]byte)(unsafe.Pointer(&spendPub4change[0]))

	selectedAmounTotal := int64(0)
	for _, input := range privacyInput.Keyinput {
		if input.Amount <= 0 {
			return nil, errors.New("")
		}
		selectedAmounTotal += input.Amount
	}
	changeAmount := selectedAmounTotal - reqPri2Pub.Amount
	//step 2,generateOuts
	//构造输出UTXO,只生成找零的UTXO
	privacyOutput, err := generateOuts(nil, nil, viewPub4chgPtr, spendPub4chgPtr, 0, changeAmount, types.PrivacyTxFee)
	if err != nil {
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
		Nonce:   wallet.random.Int63(),
		To:      reqPri2Pub.Receiver,
	}
	//step 3,generate ring signature
	err = wallet.signatureTx(tx, privacyInput, utxosInKeyInput, realkeyInputSlice)
	if err != nil {
		return nil, err
	}

	msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("transPri2PubV2", "Send err", err)
		return nil, err
	}

	reply := resp.GetData().(*types.Reply)
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()

	wallet.saveFTXOInfo(reqPri2Pub.Tokenname, reqPri2Pub.Sender, common.Bytes2Hex(hash.Hash), selectedUtxo)
	return &hash, nil
}

func (wallet *Wallet) saveFTXOInfo(token, sender, txhash string, selectedUtxos []*txOutputInfo) {
	//将已经作为本次交易输入的utxo进行冻结，防止产生双花交易
	wallet.walletStore.moveUTXO2FTXO(token, sender, txhash, selectedUtxos)
	//TODO:需要加入超时处理，需要将此处的txhash写入到数据库中，以免钱包瞬间奔溃后没有对该笔隐私交易的记录，
	//TODO:然后当该交易得到执行之后，没法将FTXO转化为STXO，added by hezhengjun on 2018.6.5
}

func (wallet *Wallet) buildInput(privacykeyParirs *privacy.Privacy, buildInfo *buildInputInfo) (*types.PrivacyInput, [][]*types.UTXOBasic, []*realkeyInput, []*txOutputInfo, error) {
	//挑选满足额度的utxo
	selectedUtxo, err := wallet.selectUTXO(*buildInfo.tokenname, *buildInfo.sender, buildInfo.amount)
	if err != nil {
		walletlog.Error("buildInput", "Failed to selectOutput for amount", buildInfo.amount,
			"Due to cause", err)
		return nil, nil, nil, nil, err
	}

	walletlog.Debug("buildInput", "Before sort selectedUtxo", selectedUtxo)
	sort.Slice(selectedUtxo, func(i, j int) bool {
		return selectedUtxo[i].amount <= selectedUtxo[j].amount
	})
	walletlog.Debug("buildInput", "After sort selectedUtxo", selectedUtxo)

	reqGetGlobalIndex := types.ReqUTXOGlobalIndex{
		Tokenname: *buildInfo.tokenname,
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
		//向blockchain请求相同额度的不同utxo用于相同额度的混淆作用
		msg := wallet.client.NewMessage("blockchain", types.EventGetGlobalIndex, &reqGetGlobalIndex)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
		if err != nil {
			walletlog.Error("buildInput EventGetGlobalIndex", "err", err)
			return nil, nil, nil, nil, err
		}
		resUTXOGlobalIndex = resp.GetData().(*types.ResUTXOGlobalIndex)
		if resUTXOGlobalIndex == nil {
			walletlog.Info("buildInput EventGetGlobalIndex is nil")
			return nil, nil, nil, nil, err
		}

		sort.Slice(resUTXOGlobalIndex.UtxoIndex4Amount, func(i, j int) bool {
			return resUTXOGlobalIndex.UtxoIndex4Amount[i].Amount <= resUTXOGlobalIndex.UtxoIndex4Amount[j].Amount
		})

		if len(selectedUtxo) != len(resUTXOGlobalIndex.UtxoIndex4Amount) {
			walletlog.Error("buildInput EventGetGlobalIndex get not the same count for mix",
				"len(selectedUtxo)", len(selectedUtxo),
				"len(resUTXOGlobalIndex.UtxoIndex4Amount)", len(resUTXOGlobalIndex.UtxoIndex4Amount))
		}
	}

	//构造输入PrivacyInput
	privacyInput := &types.PrivacyInput{}
	utxosInKeyInput := make([][]*types.UTXOBasic, len(selectedUtxo))
	realkeyInputSlice := make([]*realkeyInput, len(selectedUtxo))
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
		positions := wallet.random.Perm(len(utxoIndex4Amount.Utxos))
		utxos := make([]*types.UTXOBasic, len(utxoIndex4Amount.Utxos))
		for k, position := range positions {
			utxos[position] = utxoIndex4Amount.Utxos[k]
		}
		utxosInKeyInput[i] = utxos

		//x = Hs(aR) + b
		onetimePriv, err := privacy.RecoverOnetimePriKey(utxo2pay.txPublicKeyR, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(utxo2pay.utxoGlobalIndex.Outindex))
		if err != nil {
			walletlog.Error("transPri2Pri", "Failed to RecoverOnetimePriKey", err)
			return nil, nil, nil, nil, err
		}

		realkeyInput := &realkeyInput{
			realInputIndex: positions[len(positions)-1],
			onetimePrivKey: onetimePriv.Bytes(),
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

func (wallet *Wallet) selectUTXO(token, addr string, amount int64) ([]*txOutputInfo, error) {
	walletOuts4Addr := wallet.walletStore.getPrivacyTokenUTXOs(token, addr)
	if walletOuts4Addr != nil {
		balanceLeft := int64(0)
		for _, txOutputInfo := range walletOuts4Addr.outs {
			balanceLeft += txOutputInfo.amount
			if balanceLeft > amount {
				// 余额足够支付，可以直接跳出循环
				break
			}
		}
		//在挑选具体的输出前，先确认余额是否满足转账额度
		if balanceLeft < amount {
			return nil, types.ErrInsufficientBalance
		}
		balanceFound := int64(0)
		var selectedOuts []*txOutputInfo
		for balanceFound < amount {
			//随机选择其中一个utxo
			index := wallet.random.Intn(len(walletOuts4Addr.outs))
			selectedOuts = append(selectedOuts, walletOuts4Addr.outs[index])
			balanceFound += walletOuts4Addr.outs[index].amount
			//remove from the origin slice
			walletOuts4Addr.outs = append(walletOuts4Addr.outs[:index], walletOuts4Addr.outs[index+1:]...)
		}
		return selectedOuts, nil
	} else {
		return nil, types.ErrInsufficientBalance
	}
	return nil, types.ErrInsufficientBalance
}

// 62387455827 -> 455827 + 7000000 + 80000000 + 300000000 + 2000000000 + 60000000000, where 455827 <= dust_threshold
//res:[455827, 7000000, 80000000, 300000000, 2000000000, 60000000000]
func decomposeAmount2digits(amount, dust_threshold int64) []int64 {
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
				goodAmount := decomAmount2Nature(chunk, order/10)
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
func decomAmount2Nature(amount int64, order int64) []int64 {
	res := make([]int64, 0)
	if order == 0 {
		return nil
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
