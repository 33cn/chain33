package wallet

import (
	"bytes"
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/wallet/bipwallet"
)

var (
	testPrivateKeys = []string{
		"0x8dea7332c7bb3e3b0ce542db41161fd021e3cfda9d7dabacf24f98f2dfd69558",
		"0x920976ffe83b5a98f603b999681a0bc790d97e22ffc4e578a707c2234d55cc8a",
		"0xb59f2b02781678356c231ad565f73699753a28fd3226f1082b513ebf6756c15c",
	}
	testAddrs = []string{
		"1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t",
		"13cS5G1BDN2YfGudsxRxr7X25yu6ZdgxMU",
		"1JSRSwp16NvXiTjYBYK9iUQ9wqp3sCxz2p",
	}
	testPubkeyPairs = []string{
		"92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea85886b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389",
		"6326126c968a93a546d8f67d623ad9729da0e3e4b47c328a273dfea6930ffdc87bcc365822b80b90c72d30e955e7870a7a9725e9a946b9e89aec6db9455557eb",
		"44bf54abcbae297baf3dec4dd998b313eafb01166760f0c3a4b36509b33d3b50239de0a5f2f47c2fc98a98a382dcd95a2c5bf1f4910467418a3c2595b853338e",
	}
)

type walletTestData struct {
	mockMempool    bool
	mockBlockChain bool
	wallet         *Wallet

	blockChainHeight int64

	password string
	api      client.QueueProtocolAPI
}

func (wtd *walletTestData) init() {
	wtd.iniParams()
	wtd.initAccount()
}

func (wtd *walletTestData) initAccount() {
	wallet := wtd.wallet
	replySeed, _ := wtd.wallet.genSeed(1)
	wallet.saveSeed(wtd.password, replySeed.Seed)
	wallet.ProcWalletUnLock(&types.WalletUnLock{
		Passwd: wtd.password,
	})

	for index, key := range testPrivateKeys {
		privKey := &types.ReqWalletImportPrivKey{
			Label:   fmt.Sprintf("Label%d", index+1),
			Privkey: key,
		}
		wtd.importPrivateKey(privKey)
	}
	accCoin := account.NewCoinsAccount()
	accCoin.SetDB(wallet.walletStore.db)
	accounts, _ := accountdb.LoadAccounts(wallet.api, testAddrs)
	for _, account := range accounts {
		account.Balance = 1000 * types.Coin
		accCoin.SaveAccount(account)
	}
}

func (wtd *walletTestData) setBlockChainHeight(height int64) {
	wtd.blockChainHeight = height
}

func (wtd *walletTestData) importPrivateKey(PrivKey *types.ReqWalletImportPrivKey) {
	wallet := wtd.wallet
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return
	}

	if PrivKey == nil || len(PrivKey.GetLabel()) == 0 || len(PrivKey.GetPrivkey()) == 0 {
		return
	}

	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(PrivKey.GetLabel())
	if Account != nil {
		return
	}

	var cointype uint32
	if SignType == 1 {
		cointype = bipwallet.TypeBty
	} else if SignType == 2 {
		cointype = bipwallet.TypeYcc
	} else {
		cointype = bipwallet.TypeBty
	}

	privkeybyte, err := common.FromHex(PrivKey.Privkey)
	if err != nil || len(privkeybyte) == 0 {
		return
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, privkeybyte)
	if err != nil {
		return
	}
	addr, err := bipwallet.PubToAddress(cointype, pub)
	if err != nil {
		return
	}

	//对私钥加密
	Encryptered := CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//校验PrivKey对应的addr是否已经存在钱包中
	Account, err = wallet.walletStore.GetAccountByAddr(addr)
	if Account != nil {
		if Account.Privkey == Encrypteredstr {
			return
		} else {
			return
		}
	}

	var walletaccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	WalletAccStore.Privkey = Encrypteredstr //存储加密后的私钥
	WalletAccStore.Label = PrivKey.GetLabel()
	WalletAccStore.Addr = addr
	//存储Addr:label+privkey+addr到数据库
	err = wallet.walletStore.SetWalletAccount(false, addr, &WalletAccStore)
	if err != nil {
		return
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil {
		return
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr
	}
	walletaccount.Acc = accounts[0]
	walletaccount.Label = PrivKey.Label
}

func (wtd *walletTestData) lockWallet() {
	wtd.wallet.ProcWalletLock()
}

func (wtd *walletTestData) unlockWallet() {
	wtd.wallet.ProcWalletUnLock(&types.WalletUnLock{
		Passwd: wtd.password,
	})
}

func (wtd *walletTestData) mockBlockChainProc(q queue.Queue) {
	// blockchain
	go func() {
		topic := "blockchain"
		client := q.Client()
		client.Sub(topic)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetBlockHeight:
				msg.Reply(client.NewMessage(topic, types.EventReplyBlockHeight, &types.ReplyBlockHeight{Height: wtd.blockChainHeight}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (wtd *walletTestData) mockMempoolProc(q queue.Queue) {
	// mempool
	go func() {
		topic := "mempool"
		client := q.Client()
		client.Sub(topic)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventTx:
				msg.Reply(client.NewMessage(topic, types.EventReply, &types.Reply{IsOk: true, Msg: []byte("word")}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}
func (wtd *walletTestData) cleanDBData() {
	wallet := wtd.wallet
	storeDB := wallet.walletStore.db
	dbbatch := wallet.walletStore.NewBatch(true)
	prefixs := []string{
		PrivacyUTXO, PrivacySTXO, FTXOs4Tx + "-", STXOs4Tx + "-", RevertSendtx + "-", RecvPrivacyTx + "-", SendPrivacyTx + "-",
	}

	for _, prefix := range prefixs {
		iter := storeDB.Iterator([]byte(prefix), false)
		defer iter.Close()
		for iter.Rewind(); iter.Valid(); iter.Next() {
			dbbatch.Delete(iter.Key())
		}
	}
	dbbatch.Write()
}

func (wtd *walletTestData) createUTXOs(sender string, pubkeypair string, amount int64, height int64, count int) {
	wallet := wtd.wallet
	tokenname := types.BTY
	dbbatch := wallet.walletStore.NewBatch(true)
	for n := 0; n < count; n++ {
		res := wtd.createPublic2PrivacyTx(&types.ReqCreateTransaction{
			Tokenname:  tokenname,
			Type:       1,
			Amount:     amount,
			From:       sender,
			Pubkeypair: pubkeypair,
		})
		if res == nil {
			return
		}

		txhashInbytes := res.tx.Hash()
		txhash := common.Bytes2Hex(txhashInbytes)
		var privateAction types.PrivacyAction
		if err := types.Decode(res.tx.GetPayload(), &privateAction); err != nil {
			return
		}
		privacyOutput := privateAction.GetOutput()
		RpubKey := privacyOutput.GetRpubKeytx()
		privacyInfo, _ := wallet.getPrivacyKeyPairsOfWallet()
		totalUtxosLeft := len(privacyOutput.Keyoutput)
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			privacykeyParirs := info.PrivacyKeyPair

			var utxos []*types.UTXO
			for indexoutput, output := range privacyOutput.Keyoutput {
				if utxoProcessed[indexoutput] {
					continue
				}
				priv, _ := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
				recoverPub := priv.PubKey().Bytes()[:]
				if bytes.Equal(recoverPub, output.Onetimepubkey) {
					totalUtxosLeft--
					utxoProcessed[indexoutput] = true
					info2store := &types.PrivacyDBStore{
						Txhash:           txhashInbytes,
						Tokenname:        tokenname,
						Amount:           output.Amount,
						OutIndex:         int32(indexoutput),
						TxPublicKeyR:     RpubKey,
						OnetimePublicKey: output.Onetimepubkey,
						Owner:            *info.Addr,
						Height:           height,
						Txindex:          0,
						Blockhash:        []byte("5e3dafda"),
					}

					utxoGlobalIndex := &types.UTXOGlobalIndex{
						Outindex: int32(indexoutput),
						Txhash:   txhashInbytes,
					}

					utxoCreated := &types.UTXO{
						Amount: output.Amount,
						UtxoBasic: &types.UTXOBasic{
							UtxoGlobalIndex: utxoGlobalIndex,
							OnetimePubkey:   output.Onetimepubkey,
						},
					}

					utxos = append(utxos, utxoCreated)
					wallet.walletStore.setUTXO(info.Addr, &txhash, indexoutput, info2store, dbbatch)
				}
			}
		}
	}
	dbbatch.Write()
}

type walletTestDataReturnTx struct {
	tx           *types.Transaction
	privacyInput *types.PrivacyInput
	UTXOs        []*types.UTXOBasics
	realKeyInput []*types.RealKeyInput
	outputInfo   []*txOutputInfo
}

func (wtd *walletTestData) createPublic2PrivacyTx(req *types.ReqCreateTransaction) *walletTestDataReturnTx {
	wallet := wtd.wallet

	viewPubSlice, spendPubSlice, err := parseViewSpendPubKeyPair(req.GetPubkeypair())
	if err != nil {
		return nil
	}
	amount := req.GetAmount()
	viewPublic := (*[32]byte)(unsafe.Pointer(&viewPubSlice[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&spendPubSlice[0]))
	privacyOutput, err := generateOuts(viewPublic, spendPublic, nil, nil, amount, amount, 0)
	if err != nil {
		return nil
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
		Nonce:   wallet.random.Int63(),
		To:      address.ExecAddress(types.PrivacyX),
	}
	txSize := types.Size(tx) + types.SignatureSize
	realFee := int64((txSize+1023)>>types.Size_1K_shiftlen) * types.FeePerKB
	tx.Fee = realFee
	tx.SetExpire(time.Hour)

	return &walletTestDataReturnTx{
		tx: tx,
	}
}

func (wtd *walletTestData) createPrivate2PrivacyTx(req *types.ReqCreateTransaction) *walletTestDataReturnTx {
	wallet := wtd.wallet

	buildInfo := &buildInputInfo{
		tokenname: req.GetTokenname(),
		sender:    req.GetFrom(),
		amount:    req.GetAmount() + types.PrivacyTxFee,
		mixcount:  req.GetMixcount(),
	}

	privacyInfo, err := wallet.getPrivacykeyPair(req.GetFrom())
	if err != nil {
		walletlog.Error("createPrivacy2PrivacyTx failed to getPrivacykeyPair")
		return nil
	}

	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := wallet.buildInput(privacyInfo, buildInfo)
	if err != nil {
		return nil
	}

	//step 2,generateOuts
	viewPublicSlice, spendPublicSlice, err := parseViewSpendPubKeyPair(req.GetPubkeypair())
	if err != nil {
		return nil
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
		return nil
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
		Nonce:   wallet.random.Int63(),
		To:      address.ExecAddress(types.PrivacyX),
	}
	tx.SetExpire(time.Hour)

	return &walletTestDataReturnTx{
		tx:           tx,
		privacyInput: privacyInput,
		UTXOs:        utxosInKeyInput,
		realKeyInput: realkeyInputSlice,
		outputInfo:   selectedUtxo,
	}
}

func (wtd *walletTestData) createPrivate2PublicTx(req *types.ReqCreateTransaction) *walletTestDataReturnTx {
	wallet := wtd.wallet
	buildInfo := &buildInputInfo{
		tokenname: req.GetTokenname(),
		sender:    req.GetFrom(),
		amount:    req.GetAmount() + types.PrivacyTxFee,
		mixcount:  req.GetMixcount(),
	}
	privacyInfo, err := wallet.getPrivacykeyPair(req.GetFrom())
	if err != nil {
		return nil
	}
	//step 1,buildInput
	privacyInput, utxosInKeyInput, realkeyInputSlice, selectedUtxo, err := wallet.buildInput(privacyInfo, buildInfo)
	if err != nil {
		return nil
	}

	viewPub4change, spendPub4change := privacyInfo.ViewPubkey.Bytes(), privacyInfo.SpendPubkey.Bytes()
	viewPub4chgPtr := (*[32]byte)(unsafe.Pointer(&viewPub4change[0]))
	spendPub4chgPtr := (*[32]byte)(unsafe.Pointer(&spendPub4change[0]))

	selectedAmounTotal := int64(0)
	for _, input := range privacyInput.Keyinput {
		if input.Amount <= 0 {
			return nil
		}
		selectedAmounTotal += input.Amount
	}
	changeAmount := selectedAmounTotal - req.GetAmount()
	//step 2,generateOuts
	//构造输出UTXO,只生成找零的UTXO
	privacyOutput, err := generateOuts(nil, nil, viewPub4chgPtr, spendPub4chgPtr, 0, changeAmount, types.PrivacyTxFee)
	if err != nil {
		return nil
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
		Nonce:   wallet.random.Int63(),
		To:      req.GetTo(),
	}
	tx.SetExpire(time.Hour)

	return &walletTestDataReturnTx{
		tx:           tx,
		privacyInput: privacyInput,
		UTXOs:        utxosInKeyInput,
		realKeyInput: realkeyInputSlice,
		outputInfo:   selectedUtxo,
	}
}

func (wtd *walletTestData) iniParams() {
	var q = queue.New("channel")
	cfg := config.InitCfg("../cmd/chain33/chain33.test.toml")

	wallet := New(cfg.Wallet)
	wallet.SetQueueClient(q.Client())

	store := store.New(cfg.Store)
	store.SetQueueClient(q.Client())

	if wtd.mockBlockChain {
		wtd.mockBlockChainProc(q)
	} else {
		chain := blockchain.New(cfg.BlockChain)
		chain.SetQueueClient(q.Client())
	}

	if wtd.mockMempool {
		wtd.mockMempoolProc(q)
	} else {
		mempool := mempool.New(cfg.MemPool)
		mempool.SetQueueClient(q.Client())
		mempool.SetMinFee(1e5)
	}

	wtd.wallet = wallet
	wtd.api = wallet.api
	wtd.password = "123456"
}

func Test_checkAmountValid(t *testing.T) {
	testCases := []struct {
		amount int64
		actual bool
	}{
		{amount: 0, actual: false},
		{amount: -1, actual: false},
		{amount: 1*types.Coin + 100, actual: false},
		{amount: 5 * types.Coin, actual: true},
	}
	for _, test := range testCases {
		ret := checkAmountValid(test.amount)
		require.Equal(t, ret, test.actual)
	}
}

func Test_decomAmount2Nature(t *testing.T) {
	testCase := []struct {
		amount int64
		order  int64
		actual []int64
	}{
		{
			amount: 0,
			order:  0,
			actual: []int64{},
		},
		{
			amount: -1,
			order:  0,
			actual: []int64{},
		},
		{
			amount: 2,
			order:  0,
			actual: []int64{},
		},
		{
			amount: 2,
			order:  1,
			actual: []int64{2},
		},
		{
			amount: 3,
			order:  1,
			actual: []int64{1, 2},
		},
		{
			amount: 4,
			order:  1,
			actual: []int64{2, 2},
		},
		{
			amount: 5,
			order:  1,
			actual: []int64{5},
		},
		{
			amount: 6,
			order:  1,
			actual: []int64{5, 1},
		},
		{
			amount: 7,
			order:  1,
			actual: []int64{5, 2},
		},
		{
			amount: 8,
			order:  1,
			actual: []int64{5, 2, 1},
		},
		{
			amount: 9,
			order:  1,
			actual: []int64{5, 2, 2},
		},
	}
	for index, test := range testCase {
		res := decomAmount2Nature(test.amount, test.order)
		require.Equalf(t, res, test.actual, "testcase index %d", index)
	}
}

func Test_decomposeAmount2digits(t *testing.T) {
	testCase := []struct {
		amount         int64
		dust_threshold int64
		actual         []int64
	}{
		{
			amount:         0,
			dust_threshold: 0,
			actual:         []int64{},
		},
		{
			amount:         -1,
			dust_threshold: 0,
			actual:         []int64{},
		},
		{
			amount:         2,
			dust_threshold: 1,
			actual:         []int64{2},
		},
		{
			amount:         62387455827,
			dust_threshold: types.BTYDustThreshold,
			actual:         []int64{87455827, 1e8, 2e8, 2e9, 5e10, 1e10},
		},
	}
	for _, test := range testCase {
		res := decomposeAmount2digits(test.amount, test.dust_threshold)
		require.Equal(t, res, test.actual)
	}
}

func Test_getPrivKeyByAddr(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()

	wallet := wtd.wallet

	prikeybyte, _ := common.FromHex(testPrivateKeys[0])
	cr, _ := crypto.New(types.GetSignatureTypeName(SignType))
	priv, _ := cr.PrivKeyFromBytes(prikeybyte)

	testCase := []struct {
		req       string
		actualErr error
		actual    crypto.PrivKey
	}{
		{
			actualErr: types.ErrInputPara,
		},
		{
			req:       "12UiGcbsccfxW3zuvHXZBJfznziph5miAo",
			actualErr: types.ErrAddrNotExist,
		},
		{
			req:    testAddrs[0],
			actual: priv,
		},
	}

	for _, test := range testCase {
		res, err := wallet.getPrivKeyByAddr(test.req)
		require.Equal(t, err, test.actualErr)
		require.Equal(t, res, test.actual)
	}

}

func Test_generateOuts(t *testing.T) {
	var viewPubTo, spendpubto, viewpubChangeto, spendpubChangeto [32]byte
	tmp, _ := common.FromHex("0x92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea858")
	copy(viewPubTo[:], tmp)
	tmp, _ = common.FromHex("0x86b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389")
	copy(spendpubto[:], tmp)
	tmp, _ = common.FromHex("0x92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea858")
	copy(viewpubChangeto[:], tmp)
	tmp, _ = common.FromHex("0x86b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389")
	copy(spendpubChangeto[:], tmp)

	testCase := []struct {
		viewPubTo        *[32]byte
		spendpubto       *[32]byte
		viewpubChangeto  *[32]byte
		spendpubChangeto *[32]byte
		transAmount      int64
		selectedAmount   int64
		fee              int64
		actualErr        error
	}{
		{
			viewPubTo:        &viewPubTo,
			spendpubto:       &spendpubto,
			viewpubChangeto:  &viewpubChangeto,
			spendpubChangeto: &spendpubChangeto,
			transAmount:      10 * types.Coin,
			selectedAmount:   100 * types.Coin,
		},
		{
			viewPubTo:        &viewPubTo,
			spendpubto:       &spendpubto,
			viewpubChangeto:  &viewpubChangeto,
			spendpubChangeto: &spendpubChangeto,
			transAmount:      10 * types.Coin,
			selectedAmount:   100 * types.Coin,
			fee:              1 * types.Coin,
		},
		{
			viewPubTo:        &viewPubTo,
			spendpubto:       &spendpubto,
			viewpubChangeto:  &viewpubChangeto,
			spendpubChangeto: &spendpubChangeto,
			transAmount:      3e8,
		},
	}

	for _, test := range testCase {
		_, err := generateOuts(test.viewPubTo, test.spendpubto, test.viewpubChangeto, test.spendpubChangeto, test.transAmount, test.selectedAmount, test.fee)
		require.Equal(t, err, test.actualErr)
	}
}

func Test_genCustomOuts(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet
	privacyKeyPair, _ := wallet.getPrivacykeyPair(testAddrs[1])
	var viewpubTo, spendpubto [32]byte
	copy(viewpubTo[:], privacyKeyPair.ViewPubkey[:])
	copy(spendpubto[:], privacyKeyPair.SpendPubkey[:])

	testCase := []struct {
		viewpubTo   *[32]byte
		spendpubto  *[32]byte
		transAmount int64
		count       int32
		actualErr   error
	}{
		{
			viewpubTo:   &viewpubTo,
			spendpubto:  &spendpubto,
			transAmount: 20 * types.Coin,
			count:       5,
			actualErr:   nil,
		},
	}
	for index, test := range testCase {
		_, err := genCustomOuts(test.viewpubTo, test.spendpubto, test.transAmount, test.count)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
	}
}

func Test_parseViewSpendPubKeyPair(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()

	actualPubkeyPair, _ := common.FromHex(testPubkeyPairs[0])
	actualViewPubkey := actualPubkeyPair[:32]
	actualSpendPubKey := actualPubkeyPair[32:]

	testCase := []struct {
		req         string
		actualErr   error
		viewPubKey  []byte
		spendPubKey []byte
	}{
		{
			actualErr: types.ErrPubKeyLen,
		},
		{
			req:         testPubkeyPairs[0],
			viewPubKey:  actualViewPubkey,
			spendPubKey: actualSpendPubKey,
		},
	}

	for _, test := range testCase {
		viewPubKey, spendPubKey, err := parseViewSpendPubKeyPair(test.req)
		require.Equal(t, err, test.actualErr)
		require.Equal(t, viewPubKey, test.viewPubKey)
		require.Equal(t, spendPubKey, test.spendPubKey)
	}
}

func Test_showPrivacyPkPair(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet

	testCase := []struct {
		req       *types.ReqStr
		actualErr error
		actual    *types.ReplyPrivacyPkPair
	}{
		{
			actualErr: types.ErrInputPara,
		},
		{
			req: &types.ReqStr{
				ReqStr: "11112234567890908765876765",
			},
			actualErr: types.ErrAddrNotExist,
		},
		{
			req: &types.ReqStr{
				ReqStr: "1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t",
			},
			actual: &types.ReplyPrivacyPkPair{
				ShowSuccessful: true,
				Pubkeypair:     "92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea85886b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389",
			},
		},
	}

	for _, test := range testCase {
		res, err := wallet.showPrivacyPkPair(test.req)
		require.Equal(t, err, test.actualErr)
		require.Equal(t, res, test.actual)
	}
}

func testSelectUTXO(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet

	testCase := []struct {
		token     string
		addr      string
		amount    int64
		actual    []*txOutputInfo
		actualErr error
	}{
		{
			amount:    -1,
			actualErr: types.ErrInvalidParams,
		},
		{
			token:     types.BTY,
			addr:      testAddrs[0],
			amount:    30 * types.Coin,
			actualErr: types.ErrInsufficientBalance,
		},
		{
			token:     types.BTY,
			addr:      testAddrs[0],
			amount:    3 * types.Coin,
			actualErr: types.ErrInsufficientBalance,
		},
	}
	for index, test := range testCase {
		res, err := wallet.selectUTXO(test.token, test.addr, test.amount)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
		if err == nil {
			require.Equal(t, len(res), int(test.amount/types.Coin))
		}
	}
}

func testSelectUTXOFlow(t *testing.T) {
	wtd := &walletTestData{
		mockBlockChain: true,
		mockMempool:    true,
	}
	wtd.init()
	wallet := wtd.wallet

	addr := testAddrs[0]
	pubkeyPair := testPubkeyPairs[0]

	blockBaseHeight := int64(10000)
	for i := 1; i < 30; i++ {
		wtd.createUTXOs(addr, pubkeyPair, int64(i)*types.Coin, blockBaseHeight+int64(i), 2)
	}

	testCase := []struct {
		token       string
		addr        string
		amount      int64
		blockheight int64
		actual      []*txOutputInfo
		actualErr   error
	}{
		{
			token:       types.BTY,
			addr:        testAddrs[0],
			amount:      3 * types.Coin,
			blockheight: blockBaseHeight,
			actualErr:   types.ErrInsufficientBalance,
		},
		{
			token:       types.BTY,
			addr:        testAddrs[0],
			amount:      3000 * types.Coin,
			blockheight: blockBaseHeight + 12,
			actualErr:   types.ErrInsufficientBalance,
		},
		{
			token:       types.BTY,
			addr:        testAddrs[0],
			amount:      30 * types.Coin,
			blockheight: blockBaseHeight + 12,
		},
	}

	for index, test := range testCase {
		wtd.setBlockChainHeight(test.blockheight)
		res, err := wallet.selectUTXO(test.token, test.addr, test.amount)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
		if err == nil {
			require.Equal(t, len(res) > 0, true)
			amount := int64(0)
			for _, r := range res {
				amount += r.amount
			}
			require.Equal(t, amount >= test.amount, true)
		}
	}
}

func Test_selectUTXO(t *testing.T) {
	testSelectUTXO(t)
	testSelectUTXOFlow(t)
}

func Test_createUTXOsByPub2Priv(t *testing.T) {
	wtd := &walletTestData{mockMempool: true}
	wtd.init()
	wallet := wtd.wallet
	privateKey, _ := wallet.getPrivKeyByAddr(testAddrs[0])

	testCase := []struct {
		req       *types.ReqCreateUTXOs
		actualErr error
	}{
		{
			actualErr: types.ErrPubKeyLen,
		},
		{
			req: &types.ReqCreateUTXOs{
				Tokenname:  types.BTY,
				Amount:     10 * types.Coin,
				Note:       "say something",
				Count:      16,
				Sender:     testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}

	for _, test := range testCase {
		_, err := wallet.createUTXOsByPub2Priv(privateKey, test.req)
		require.Equal(t, err, test.actualErr)
	}
}

func Test_procCreateUTXOs(t *testing.T) {
	wtd := &walletTestData{mockMempool: true}
	wtd.init()
	wallet := wtd.wallet

	testCase := []struct {
		lockWallet bool
		req        *types.ReqCreateUTXOs
		actualErr  error
	}{
		{
			lockWallet: true,
			actualErr:  types.ErrWalletIsLocked,
		},
		{
			actualErr: types.ErrInputPara,
		},
		{
			req: &types.ReqCreateUTXOs{
				Tokenname:  types.BTY,
				Amount:     10 * types.Coin,
				Note:       "say something",
				Count:      16,
				Sender:     testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}

	for index, test := range testCase {
		if test.lockWallet {
			wtd.lockWallet()
		}
		_, err := wallet.procCreateUTXOs(test.req)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
		if test.lockWallet {
			wtd.unlockWallet()
		}
	}
}

func Test_createPublic2PrivacyTx(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet

	testCase := []struct {
		req       *types.ReqCreateTransaction
		actual    *types.ReplyHash
		actualErr error
	}{
		{
			actualErr: types.ErrPubKeyLen,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname: types.BTY,
				Type:      0,
			},
			actualErr: types.ErrPubKeyLen,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       1,
				Amount:     10 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}

	for _, test := range testCase {
		_, err := wallet.createPublic2PrivacyTx(test.req)
		require.Equal(t, err, test.actualErr)
	}
}

func Test_createPrivacy2PrivacyTx(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet

	// TODO: 需要在这里插入测试数据

	testCase := []struct {
		req       *types.ReqCreateTransaction
		actual    *types.ReplyHash
		actualErr error
	}{
		{
			actualErr: types.ErrInputPara,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname: types.BTY,
				Type:      0,
			},
			actualErr: types.ErrInputPara,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       1,
				Amount:     10 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
			actualErr: types.ErrInsufficientBalance,
		},
	}

	for index, test := range testCase {
		_, err := wallet.createPrivacy2PrivacyTx(test.req)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
	}
}

func Test_createPrivacy2PublicTx(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet

	// TODO: 需要在这里插入测试数据

	testCase := []struct {
		req       *types.ReqCreateTransaction
		actual    *types.ReplyHash
		actualErr error
	}{
		{
			actualErr: types.ErrInputPara,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname: types.BTY,
				Type:      0,
			},
			actualErr: types.ErrInputPara,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       1,
				Amount:     10 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
			actualErr: types.ErrInsufficientBalance,
		},
	}

	for index, test := range testCase {
		_, err := wallet.createPrivacy2PublicTx(test.req)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
	}
}

func Test_procCreateTransaction(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet

	testCase := []struct {
		lockWallet bool
		req        *types.ReqCreateTransaction
		actual     *types.ReplyHash
		actualErr  error
	}{
		{
			lockWallet: true,
			actualErr:  types.ErrWalletIsLocked,
		},
		{
			actualErr: types.ErrInvalidParams,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname: types.BTY,
				Type:      0,
			},
			actualErr: types.ErrAmount,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       1,
				Amount:     10*types.Coin + 10,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
			actualErr: types.ErrAmount,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       1,
				Amount:     10 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}

	for index, test := range testCase {
		if test.lockWallet {
			wtd.lockWallet()
		}
		_, err := wallet.procCreateTransaction(test.req)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
		if test.lockWallet {
			wtd.unlockWallet()
		}
	}
}

func Test_transPub2PriV2(t *testing.T) {
	wtd := &walletTestData{mockMempool: true}
	wtd.init()
	wallet := wtd.wallet
	privacyKey, _ := wallet.getPrivKeyByAddr(testAddrs[0])

	testCase := []struct {
		req       *types.ReqPub2Pri
		actualErr error
	}{
		{
			req: &types.ReqPub2Pri{
				Tokenname:  types.BTY,
				Amount:     10 * types.Coin,
				Sender:     testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}

	for index, test := range testCase {
		_, err := wallet.transPub2PriV2(privacyKey, test.req)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
	}
}

func Test_procPublic2PrivacyV2(t *testing.T) {
	wtd := &walletTestData{mockMempool: true}
	wtd.init()
	wallet := wtd.wallet

	testCase := []struct {
		lockWallet bool
		req        *types.ReqPub2Pri
		actualErr  error
	}{
		{
			lockWallet: true,
			actualErr:  types.ErrWalletIsLocked,
		},
		{
			actualErr: types.ErrInputPara,
		},
	}

	for index, test := range testCase {
		if test.lockWallet {
			wtd.lockWallet()
		}
		_, err := wallet.procPublic2PrivacyV2(test.req)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
		if test.lockWallet {
			wtd.unlockWallet()
		}
	}
}

func Test_signatureTx(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	//wallet := wtd.wallet
	//
	//res := wtd.createPrivate2PrivacyTx(&types.ReqCreateTransaction{
	//	Tokenname:types.BTY,
	//	Type:2,
	//	Amount:10*types.Coin,
	//	From:testAddrs[0],
	//	Pubkeypair:testPubkeyPairs[0],
	//})
	//err := wallet.signatureTx(res.tx, res.privacyInput, res.UTXOs, res.realKeyInput)
	//require.Equal(t, err, nil)

}

func Test_buildInput(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()

}

func Test_procPrivacy2PrivacyV2(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()

}

func Test_procPrivacy2PublicV2(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()

}

func Test_transPri2PriV2(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()

}

func Test_transPri2PubV2(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()

}

func Test_signTxWithPrivacy(t *testing.T) {
	wtd := &walletTestData{
		mockMempool:    true,
		mockBlockChain: true,
	}
	wtd.init()
	wallet := wtd.wallet
	privateKey, _ := wallet.getPrivKeyByAddr(testAddrs[0])

	// Public -> Privacy
	tx, _ := wallet.procCreateTransaction(&types.ReqCreateTransaction{
		Tokenname:  types.BTY,
		Type:       1,
		Amount:     10 * types.Coin,
		From:       testAddrs[0],
		Pubkeypair: testPubkeyPairs[0],
	})
	txhex := common.ToHex(types.Encode(tx))
	reqSignRawTx := &types.ReqSignRawTx{
		Addr:  testAddrs[0],
		TxHex: txhex,
	}
	_, err := wallet.signTxWithPrivacy(privateKey, reqSignRawTx)
	require.Equal(t, err, nil)

	// Privacy -> Privacy
	wtd.createUTXOs(testAddrs[0], testPubkeyPairs[0], 10*types.Coin, 10000, 10)
	wtd.setBlockChainHeight(10020)
	tx, _ = wallet.procCreateTransaction(&types.ReqCreateTransaction{
		Tokenname:  types.BTY,
		Type:       2,
		Amount:     20 * types.Coin,
		From:       testAddrs[0],
		Pubkeypair: testPubkeyPairs[0],
	})
	txhex = common.ToHex(types.Encode(tx))
	reqSignRawTx = &types.ReqSignRawTx{
		Addr:  testAddrs[0],
		TxHex: txhex,
	}
	_, err = wallet.signTxWithPrivacy(privateKey, reqSignRawTx)
	require.Equal(t, err, nil)

	// Privacy -> Public
	wtd.createUTXOs(testAddrs[1], testPubkeyPairs[1], 10*types.Coin, 10000, 10)
	tx, _ = wallet.procCreateTransaction(&types.ReqCreateTransaction{
		Tokenname: types.BTY,
		Type:      3,
		Amount:    10 * types.Coin,
		From:      testAddrs[1],
		To:        testAddrs[2],
	})
	txhex = common.ToHex(types.Encode(tx))
	reqSignRawTx = &types.ReqSignRawTx{
		Addr:  testAddrs[1],
		TxHex: txhex,
	}
	_, err = wallet.signTxWithPrivacy(privateKey, reqSignRawTx)
	require.Equal(t, err, nil)
}

func Test_procReqCacheTxList(t *testing.T) {
	wtd := &walletTestData{}
	wtd.init()
	wallet := wtd.wallet

	testCase := []struct {
		lockWallet bool
		req        *types.ReqCreateTransaction
		actual     *types.ReplyHash
		actualErr  error
	}{
		{
			lockWallet: true,
			actualErr:  types.ErrWalletIsLocked,
		},
		{
			actualErr: types.ErrInvalidParams,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname: types.BTY,
				Type:      0,
			},
			actualErr: types.ErrAmount,
		},
		{
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       1,
				Amount:     10 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}

	for index, test := range testCase {
		if test.lockWallet {
			wtd.lockWallet()
		}
		_, err := wallet.procCreateTransaction(test.req)
		require.Equalf(t, err, test.actualErr, "testcase index %d", index)
		if test.lockWallet {
			wtd.unlockWallet()
		}
	}
}
