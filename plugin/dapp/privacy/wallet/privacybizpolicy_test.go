package wallet_test

import (
	"fmt"
	"testing"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/mempool"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util/testnode"
	"gitlab.33.cn/chain33/chain33/wallet"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
	"gitlab.33.cn/wallet/bipwallet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	privacy "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/wallet"
)

var (
	// 测试的私钥
	testPrivateKeys = []string{
		"0x8dea7332c7bb3e3b0ce542db41161fd021e3cfda9d7dabacf24f98f2dfd69558",
		"0x920976ffe83b5a98f603b999681a0bc790d97e22ffc4e578a707c2234d55cc8a",
		"0xb59f2b02781678356c231ad565f73699753a28fd3226f1082b513ebf6756c15c",
	}
	// 测试的地址
	testAddrs = []string{
		"1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t",
		"13cS5G1BDN2YfGudsxRxr7X25yu6ZdgxMU",
		"1JSRSwp16NvXiTjYBYK9iUQ9wqp3sCxz2p",
	}
	// 测试的隐私公钥对
	testPubkeyPairs = []string{
		"92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea85886b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389",
		"6326126c968a93a546d8f67d623ad9729da0e3e4b47c328a273dfea6930ffdc87bcc365822b80b90c72d30e955e7870a7a9725e9a946b9e89aec6db9455557eb",
		"44bf54abcbae297baf3dec4dd998b313eafb01166760f0c3a4b36509b33d3b50239de0a5f2f47c2fc98a98a382dcd95a2c5bf1f4910467418a3c2595b853338e",
	}
)

func setLogLevel(level string) {
	log.SetLogLevel(level)
}

func init() {
	queue.DisableLog()
	//setLogLevel("err")
	setLogLevel("crit")
}

type testDataMock struct {
	policy wcom.WalletBizPolicy

	wallet  *wallet.Wallet
	modules []queue.Module

	accdb            *account.DB
	mockMempool      bool
	mockBlockChain   bool
	blockChainHeight int64
	password         string
}

func (mock *testDataMock) init() {
	mock.initMember()
	mock.initAccounts()
}

func (mock *testDataMock) initMember() {
	var q = queue.New("channel")
	cfg, sub := testnode.GetDefaultConfig()

	wallet := wallet.New(cfg.Wallet, sub.Wallet)
	wallet.SetQueueClient(q.Client())
	mock.modules = append(mock.modules, wallet)
	mock.wallet = wallet

	store := store.New(cfg.Store, sub.Store)
	store.SetQueueClient(q.Client())
	mock.modules = append(mock.modules, store)

	if mock.mockBlockChain {
		mock.mockBlockChainProc(q)
	} else {
		chain := blockchain.New(cfg.BlockChain)
		chain.SetQueueClient(q.Client())
		mock.modules = append(mock.modules, chain)
	}

	if mock.mockMempool {
		mock.mockMempoolProc(q)
	} else {
		mempool := mempool.New(cfg.MemPool)
		mempool.SetQueueClient(q.Client())
		mempool.SetMinFee(1e5)
		mock.modules = append(mock.modules, mempool)
	}

	mock.accdb = account.NewCoinsAccount()
	mock.policy = privacy.New()
	mock.policy.Init(wallet, sub.Wallet["privacy"])
	mock.password = "123456"
}

func (mock *testDataMock) importPrivateKey(PrivKey *types.ReqWalletImportPrivkey) {
	wallet := mock.wallet
	wallet.GetMutex().Lock()
	defer wallet.GetMutex().Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return
	}

	if PrivKey == nil || len(PrivKey.GetLabel()) == 0 || len(PrivKey.GetPrivkey()) == 0 {
		return
	}

	//校验label是否已经被使用
	Account, err := wallet.GetAccountByLabel(PrivKey.GetLabel())
	if Account != nil {
		return
	}

	var cointype uint32
	signType := wallet.GetSignType()
	if signType == 1 {
		cointype = bipwallet.TypeBty
	} else if signType == 2 {
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
	Encryptered := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//校验PrivKey对应的addr是否已经存在钱包中
	Account, err = wallet.GetAccountByAddr(addr)
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
	err = wallet.SetWalletAccount(false, addr, &WalletAccStore)
	if err != nil {
		return
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := mock.accdb.LoadAccounts(wallet.GetAPI(), addrs)
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

func (mock *testDataMock) initAccounts() {
	wallet := mock.wallet
	replySeed, _ := wallet.GenSeed(1)
	wallet.SaveSeed(mock.password, replySeed.Seed)
	wallet.ProcWalletUnLock(&types.WalletUnLock{
		Passwd: mock.password,
	})

	for index, key := range testPrivateKeys {
		privKey := &types.ReqWalletImportPrivkey{
			Label:   fmt.Sprintf("Label%d", index+1),
			Privkey: key,
		}
		mock.importPrivateKey(privKey)
	}
	accCoin := account.NewCoinsAccount()
	accCoin.SetDB(wallet.GetDBStore())
	accounts, _ := mock.accdb.LoadAccounts(wallet.GetAPI(), testAddrs)
	for _, account := range accounts {
		account.Balance = 1000 * types.Coin
		accCoin.SaveAccount(account)
	}
}

func (mock *testDataMock) enablePrivacy() {
	mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "EnablePrivacy", &ty.ReqEnablePrivacy{Addrs: testAddrs})
}

func (mock *testDataMock) setBlockChainHeight(height int64) {
	mock.blockChainHeight = height
}

func (mock *testDataMock) mockBlockChainProc(q queue.Queue) {
	// blockchain
	go func() {
		topic := "blockchain"
		client := q.Client()
		client.Sub(topic)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetBlockHeight:
				msg.Reply(client.NewMessage(topic, types.EventReplyBlockHeight, &types.ReplyBlockHeight{Height: mock.blockChainHeight}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (mock *testDataMock) mockMempoolProc(q queue.Queue) {
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

func Test_EnablePrivacy(t *testing.T) {
	mock := &testDataMock{}
	mock.init()

	testCases := []struct {
		req       *ty.ReqEnablePrivacy
		needReply *ty.RepEnablePrivacy
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqEnablePrivacy{Addrs: []string{testAddrs[0]}},
			needReply: &ty.RepEnablePrivacy{
				Results: []*ty.PriAddrResult{
					{IsOK: true, Addr: testAddrs[0]}},
			},
		},
	}
	for index, testCase := range testCases {
		getReply, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "EnablePrivacy", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "EnablePrivacy test case index %d", index)
		if testCase.needReply == nil {
			assert.Nil(t, getReply)
		} else {
			require.Equal(t, getReply, testCase.needReply)
		}
	}
}

func Test_ShowPrivacyKey(t *testing.T) {
	mock := &testDataMock{}
	mock.init()
	// 设置第0个地址开启隐私交易
	mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "EnablePrivacy", &ty.ReqEnablePrivacy{Addrs: []string{testAddrs[0]}})

	testCases := []struct {
		req       *types.ReqString
		needReply *ty.ReplyPrivacyPkPair
		needError error
	}{
		{
			req:       &types.ReqString{Data: testAddrs[1]},
			needError: ty.ErrPrivacyNotEnabled,
		},
		{
			req: &types.ReqString{Data: testAddrs[0]},
			needReply: &ty.ReplyPrivacyPkPair{
				ShowSuccessful: true,
				Pubkeypair:     "92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea85886b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389",
			},
		},
	}

	for index, testCase := range testCases {
		getReply, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "ShowPrivacyKey", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "ShowPrivacyKey test case index %d", index)
		if testCase.needReply == nil {
			assert.Nil(t, getReply)
			continue
		}
		require.Equal(t, getReply, testCase.needReply)
	}
}

func Test_CreateUTXOs(t *testing.T) {
	mock := &testDataMock{mockMempool: true}
	mock.init()
	mock.enablePrivacy()

	testCases := []struct {
		req       *ty.ReqCreateUTXOs
		needReply *types.Reply
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqCreateUTXOs{
				Tokenname:  types.BTY,
				Amount:     10 * types.Coin,
				Note:       "say something",
				Count:      16,
				Sender:     testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}

	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "CreateUTXOs", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "CreateUTXOs test case index %d", index)
	}
}

func Test_SendPublic2PrivacyTransaction(t *testing.T) {
	mock := &testDataMock{mockMempool: true}
	mock.init()
	mock.enablePrivacy()

	testCases := []struct {
		req       *ty.ReqPub2Pri
		needReply *types.Reply
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqPub2Pri{
				Tokenname:  types.BTY,
				Amount:     10 * types.Coin,
				Sender:     testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
			needReply: &types.Reply{IsOk: true},
		},
	}

	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "Public2Privacy", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "Publick2Privacy test case index %d", index)
	}
}

func Test_SendPrivacy2PrivacyTransaction(t *testing.T) {
	mock := &testDataMock{
		mockMempool:    true,
		mockBlockChain: true,
	}
	mock.init()
	mock.enablePrivacy()
	// 创建辅助对象
	privacyMock := privacy.PrivacyMock{}
	privacyMock.Init(mock.wallet, mock.password)
	// 创建几条可用UTXO
	privacyMock.CreateUTXOs(testAddrs[0], testPubkeyPairs[0], 17*types.Coin, 10000, 5)
	mock.setBlockChainHeight(10020)

	testCases := []struct {
		req       *ty.ReqPri2Pri
		needReply *types.Reply
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqPri2Pri{
				Tokenname:  types.BTY,
				Amount:     10 * types.Coin,
				Sender:     testAddrs[0],
				Pubkeypair: testPubkeyPairs[1],
			},
			needReply: &types.Reply{IsOk: true},
		},
	}

	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "Privacy2Privacy", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "Privacy2Privacy test case index %d", index)
	}
}

func Test_SendPrivacy2PublicTransaction(t *testing.T) {
	mock := &testDataMock{
		mockMempool:    true,
		mockBlockChain: true,
	}
	mock.init()
	mock.enablePrivacy()
	// 创建辅助对象
	privacyMock := privacy.PrivacyMock{}
	privacyMock.Init(mock.wallet, mock.password)
	// 创建几条可用UTXO
	privacyMock.CreateUTXOs(testAddrs[0], testPubkeyPairs[0], 17*types.Coin, 10000, 5)
	mock.setBlockChainHeight(10020)

	testCases := []struct {
		req       *ty.ReqPri2Pub
		needReply *types.Reply
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqPri2Pub{
				Tokenname: types.BTY,
				Amount:    10 * types.Coin,
				Sender:    testAddrs[0],
				Receiver:  testAddrs[0],
			},
			needReply: &types.Reply{IsOk: true},
		},
	}

	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "Privacy2Public", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "Privacy2Public test case index %d", index)
	}
}

func Test_CreateTransaction(t *testing.T) {
	mock := &testDataMock{
		mockMempool:    true,
		mockBlockChain: true,
	}
	mock.init()
	mock.enablePrivacy()
	// 创建辅助对象
	privacyMock := privacy.PrivacyMock{}
	privacyMock.Init(mock.wallet, mock.password)
	// 创建几条可用UTXO
	privacyMock.CreateUTXOs(testAddrs[0], testPubkeyPairs[0], 17*types.Coin, 10000, 5)
	mock.setBlockChainHeight(10020)

	testCases := []struct {
		req       *types.ReqCreateTransaction
		needReply *types.Transaction
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{ // 公对私测试
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       1,
				Amount:     100 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
		{ // 私对私测试
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       2,
				Amount:     10 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[1],
			},
		},
		{ // 私对公测试
			req: &types.ReqCreateTransaction{
				Tokenname:  types.BTY,
				Type:       3,
				Amount:     10 * types.Coin,
				From:       testAddrs[0],
				Pubkeypair: testPubkeyPairs[0],
			},
		},
	}
	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "CreateTransaction", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "CreateTransaction test case index %d", index)
	}
}

func Test_PrivacyAccountInfo(t *testing.T) {
	mock := &testDataMock{}
	mock.init()

	testCases := []struct {
		req       *ty.ReqPPrivacyAccount
		needReply *ty.ReplyPrivacyAccount
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqPPrivacyAccount{
				Addr:        testAddrs[0],
				Token:       types.BTY,
				Displaymode: 0,
			},
		},
	}
	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "ShowPrivacyAccountInfo", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "ShowPrivacyAccoPrivacyAccountInfountInfo test case index %d", index)
	}
}

func Test_ShowPrivacyAccountSpend(t *testing.T) {
	mock := &testDataMock{}
	mock.init()

	testCases := []struct {
		req       *ty.ReqPrivBal4AddrToken
		needReply *ty.UTXOHaveTxHashs
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqPrivBal4AddrToken{
				Addr:  testAddrs[0],
				Token: types.BTY,
			},
			needError: types.ErrNotFound,
		},
	}
	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "ShowPrivacyAccountSpend", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "ShowPrivacyAccountSpend test case index %d", index)
	}
}

func Test_PrivacyTransactionList(t *testing.T) {
	mock := &testDataMock{}
	mock.init()

	testCases := []struct {
		req       *ty.ReqPrivacyTransactionList
		needReply *types.WalletTxDetails
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqPrivacyTransactionList{
				Tokenname:    types.BTY,
				SendRecvFlag: 1,
				Direction:    0,
				Count:        10,
				Address:      testAddrs[0],
			},
			needError: types.ErrTxNotExist,
		},
	}
	for index, testCase := range testCases {
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "PrivacyTransactionList", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "PrivacyTransactionList test case index %d", index)
	}
}

func Test_RescanUTXOs(t *testing.T) {
	mock := &testDataMock{}
	mock.init()

	testCases := []struct {
		enable    bool
		req       *ty.ReqRescanUtxos
		needReply *ty.RepRescanUtxos
		needError error
	}{
		{
			needError: types.ErrInvalidParam,
		},
		{
			req: &ty.ReqRescanUtxos{
				Addrs: testAddrs,
				Flag:  0,
			},
			needError: ty.ErrPrivacyNotEnabled,
		},
		{
			enable: true,
			req: &ty.ReqRescanUtxos{
				Addrs: testAddrs,
				Flag:  0,
			},
		},
	}
	for index, testCase := range testCases {
		if testCase.enable {
			mock.enablePrivacy()
		}
		_, getErr := mock.wallet.GetAPI().ExecWalletFunc(ty.PrivacyX, "RescanUtxos", testCase.req)
		require.Equalf(t, getErr, testCase.needError, "RescanUtxos test case index %d", index)
	}
}
