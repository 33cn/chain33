// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/store"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/wallet/bipwallet"
	wcom "github.com/33cn/chain33/wallet/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	queue.DisableLog()
	SetLogLevel("err")
}

func initEnv() (*Wallet, queue.Module, queue.Queue, string) {
	cfg := types.NewChain33Config(types.ReadFile("../cmd/chain33/chain33.test.toml"))
	var q = queue.New("channel")
	q.SetConfig(cfg)
	wallet := New(cfg)
	wallet.SetQueueClient(q.Client())
	store := store.New(cfg)
	store.SetQueueClient(q.Client())

	return wallet, store, q, cfg.GetModuleConfig().Wallet.DbPath
}

var (
	Statehash      []byte
	CutHeight      int64
	FromAddr       string
	ToAddr1        string
	ToAddr2        string
	AddrPrivKey    string
	addr           string
	priv           crypto.PrivKey
	AllAccountlist *types.WalletAccounts
)

func blockchainModProc(q queue.Queue) {
	//store
	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			walletlog.Error("blockchain", "msg.Ty", msg.Ty, "name", types.GetEventName(int(msg.Ty)))
			if msg.Ty == types.EventGetLastHeader {
				header := &types.Header{StateHash: Statehash}
				msg.Reply(client.NewMessage("account", types.EventHeader, header))
			} else if msg.Ty == types.EventGetTransactionByAddr {
				addr := (msg.Data).(*types.ReqAddr)

				var replyTxInfos types.ReplyTxInfos
				total := 10
				replyTxInfos.TxInfos = make([]*types.ReplyTxInfo, total)
				for index := 0; index < total; index++ {
					var replyTxInfo types.ReplyTxInfo
					hashstr := fmt.Sprintf("hash:%s:%d", addr.Addr, index)
					replyTxInfo.Hash = []byte(hashstr)
					replyTxInfo.Height = CutHeight + 1
					replyTxInfo.Index = int64(index)
					replyTxInfos.TxInfos[index] = &replyTxInfo
					CutHeight++
				}
				msg.Reply(client.NewMessage("rpc", types.EventReplyTxInfo, &replyTxInfos))

			} else if msg.Ty == types.EventGetTransactionByHash {
				txhashs := (msg.Data).(*types.ReqHashes)

				var txDetails types.TransactionDetails
				txDetails.Txs = make([]*types.TransactionDetail, len(txhashs.Hashes))
				for index, txhash := range txhashs.Hashes {
					var txDetail types.TransactionDetail
					txDetail.Index = int64(index)
					txDetail.Receipt = &types.ReceiptData{Ty: 2, Logs: nil}
					txDetail.Tx = &types.Transaction{Execer: []byte("coins"), Payload: txhash, To: "14ZTV2wHG3uPHnA5cBJmNxAxxvbzS7Z5mE"}
					txDetails.Txs[index] = &txDetail
				}
				msg.Reply(client.NewMessage("rpc", types.EventTransactionDetails, &txDetails))
			} else if msg.Ty == types.EventGetBlockHeight {
				msg.Reply(client.NewMessage("", types.EventReplyBlockHeight, &types.ReplyBlockHeight{Height: 1}))
			} else if msg.Ty == types.EventIsSync {
				msg.Reply(client.NewMessage("", types.EventReplyIsSync, &types.IsCaughtUp{Iscaughtup: true}))
			} else if msg.Ty == types.EventQueryTx {
				msg.Reply(client.NewMessage("", types.EventTransactionDetail, &types.TransactionDetail{Receipt: &types.ReceiptData{Ty: types.ExecOk}}))
			}
		}
	}()
	go func() {
		client := q.Client()
		client.Sub("exec")
		for msg := range client.Recv() {
			walletlog.Error("execs", "msg.Ty", msg.Ty)
			if msg.Ty == types.EventBlockChainQuery {
				msg.Reply(client.NewMessage("", types.EventReplyQuery, types.ErrActionNotSupport))
			}
		}
	}()
	go func() {
		client := q.Client()
		client.Sub("consensus")
		for msg := range client.Recv() {
			walletlog.Error("execs", "msg.Ty", msg.Ty)
			if msg.Ty == types.EventConsensusQuery {
				msg.Reply(client.NewMessage("", types.EventReplyQuery, types.ErrActionNotSupport))
			}
		}
	}()
}

func mempoolModProc(q queue.Queue) {
	//store
	go func() {
		client := q.Client()
		client.Sub("mempool")
		for msg := range client.Recv() {
			//walletlog.Info("mempool", "msg.Ty", msg.Ty)
			if msg.Ty == types.EventTx {
				msg.Reply(client.NewMessage("wallet", types.EventReply, &types.Reply{IsOk: true}))
			} else if msg.Ty == types.EventGetProperFee {
				msg.Reply(client.NewMessage("wallet", types.EventReply, &types.ReplyProperFee{ProperFee: 1000000}))
			}
		}
	}()
}

func SaveAccountTomavl(wallet *Wallet, client queue.Client, prevStateRoot []byte, accs []*types.Account) []byte {
	var kvset []*types.KeyValue

	for _, acc := range accs {
		kvs := wallet.accountdb.GetKVSet(acc)
		kvset = append(kvset, kvs...)
	}
	hash, err := util.ExecKVMemSet(client, prevStateRoot, 0, kvset, true, false)
	if err != nil {
		panic(err)
	}
	Statehash = hash
	util.ExecKVSetCommit(client, Statehash, false)
	return hash
}

func TestAll(t *testing.T) {
	testAllWallet(t)
	testWalletImportPrivkeysFile(t)
	testWalletImportPrivkeysFile2(t)
}

func testAllWallet(t *testing.T) {
	wallet, store, q, datapath := initEnv()
	defer os.RemoveAll("datadir") // clean up
	defer wallet.Close()
	defer store.Close()

	//启动blockchain模块
	blockchainModProc(q)
	mempoolModProc(q)

	testSeed(t, wallet)

	testProcCreateNewAccount(t, wallet)

	testProcImportPrivKey(t, wallet)

	testProcDumpPrivkeysFile(t, wallet)

	//wait data sync
	testProcWalletTxList(t, wallet)

	testProcSendToAddress(t, wallet)

	testProcWalletSetFee(t, wallet)

	testProcWalletSetLabel(t, wallet)

	testProcMergeBalance(t, wallet)

	testProcWalletSetPasswd(t, wallet)

	testProcWalletLock(t, wallet)

	testProcWalletAddBlock(t, wallet)
	testSignRawTx(t, wallet)
	testsetFatalFailure(t, wallet)
	testgetFatalFailure(t, wallet)

	testWallet(t, wallet)
	testSendTx(t, wallet)
	testCreateNewAccountByIndex(t, wallet)

	t.Log(datapath)
}

func testWalletImportPrivkeysFile(t *testing.T) {
	wallet, store, q, _ := initEnv()
	defer os.RemoveAll("datadir") // clean up
	defer wallet.Close()
	defer store.Close()

	//启动blockchain模块
	blockchainModProc(q)
	mempoolModProc(q)

	testSeed(t, wallet)
	testProcImportPrivkeysFile(t, wallet)
}

func testWalletImportPrivkeysFile2(t *testing.T) {
	wallet, store, q, _ := initEnv()
	defer os.RemoveAll("datadir") // clean up
	defer wallet.Close()
	defer store.Close()

	//启动blockchain模块
	blockchainModProc(q)
	mempoolModProc(q)

	testSeed(t, wallet)
	testProcImportPrivkeysFile2(t, wallet)
}

// ProcWalletLock
func testSeed(t *testing.T, wallet *Wallet) {
	println("TestSeed begin")

	seedRes, _ := wallet.GetAPI().ExecWalletFunc("wallet", "GenSeed", &types.GenSeedLang{Lang: 1})
	seed := seedRes.(*types.ReplySeed).Seed
	println("seed: ", seed)

	password := "password123"
	saveSeedByPw := &types.SaveSeedByPw{Seed: "", Passwd: ""}
	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "SaveSeed", saveSeedByPw)
	assert.Nil(t, err)
	assert.Equal(t, string(resp.(*types.Reply).GetMsg()), types.ErrInvalidParam.Error())

	saveSeedByPw.Seed = "a b c d"
	saveSeedByPw.Passwd = password
	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "SaveSeed", saveSeedByPw)
	if string(resp.(*types.Reply).GetMsg()) != types.ErrSeedWordNum.Error() {
		t.Error("test input not enough seed failed")
	}

	saveSeedByPw.Seed = "a b c d e f g h i j k l m n"
	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "SaveSeed", saveSeedByPw)
	if string(resp.(*types.Reply).GetMsg()) != types.ErrSeedWord.Error() {
		t.Error("test input invalid seed failed")
	}

	saveSeedByPw.Seed = seed
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SaveSeed", saveSeedByPw)
	assert.Nil(t, err)

	seedstr, err := GetSeed(wallet.walletStore.GetDB(), password)
	require.NoError(t, err)
	if seed != seedstr {
		t.Error("testSaveSeed failed")
	}

	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "SaveSeed", saveSeedByPw)
	if string(resp.(*types.Reply).GetMsg()) != types.ErrSeedExist.Error() {
		t.Error("test seedExists failed")
	}

	walletUnLock := &types.WalletUnLock{
		Passwd:         password,
		Timeout:        0,
		WalletOrTicket: false,
	}
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletUnLock", walletUnLock)
	require.NoError(t, err)

	resp, err = wallet.GetAPI().ExecWalletFunc("wallet", "GetSeed", &types.GetSeedByPw{Passwd: password})
	require.NoError(t, err)

	reply := resp.(*types.ReplySeed)
	if reply.GetSeed() != seed {
		t.Error("testGetSeed failed")
	}

	println("TestSeed end")
	println("--------------------------")
}

func testProcCreateNewAccount(t *testing.T, wallet *Wallet) {
	println("TestProcCreateNewAccount begin")
	total := 10
	addrlist := make([]string, total)
	accs := make([]*types.Account, total+1)
	for i := 0; i < total; i++ {
		reqNewAccount := &types.ReqNewAccount{Label: fmt.Sprintf("account:%d", i)}
		resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "NewAccount", reqNewAccount)
		require.NoError(t, err)
		time.Sleep(time.Microsecond * 100)
		walletAcc := resp.(*types.WalletAccount)
		addrlist[i] = walletAcc.Acc.Addr
		walletAcc.Acc.Balance = int64(i)
		walletAcc.Acc.Currency = int32(i)
		walletAcc.Acc.Frozen = int64(i)
		accs[i] = walletAcc.Acc
		//FromAddr = Walletacc.Acc.Addr
		if i == 0 {
			ToAddr1 = walletAcc.Acc.Addr
		}
		if i == 1 {
			ToAddr2 = walletAcc.Acc.Addr
		}
	}

	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.Load(types.GetSignName("", wallet.SignType), wallet.lastHeader.GetHeight())
	require.NoError(t, err)

	Privkey := "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privkeybyte, _ := common.FromHex(Privkey)
	priv, err := cr.PrivKeyFromBytes(privkeybyte)
	require.NoError(t, err)

	addr := address.PubKeyToAddr(address.DefaultID, priv.PubKey().Bytes())
	FromAddr = addr
	var acc types.Account
	acc.Addr = addr
	acc.Balance = int64(1e16)
	acc.Currency = int32(10)
	acc.Frozen = int64(10)
	accs[total] = &acc

	//存入账户信息到mavl树中
	SaveAccountTomavl(wallet, wallet.client, nil, accs)

	//测试ProcGetAccountList函数
	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletGetAccountList", &types.ReqAccountList{})
	assert.Nil(t, err)
	accountlist := resp.(*types.WalletAccounts)
	for _, acc1 := range accountlist.Wallets {
		exist := false
		for _, acc2 := range accs[:10] {
			if equal(acc1.Acc, acc2) {
				exist = true
				break
			}
		}
		if !exist {
			t.Error("account list not match")
			return
		}
	}
	println("TestProcCreateNewAccount end")
	println("--------------------------")
}

func equal(acc1, acc2 *types.Account) bool {
	if acc1.Currency != acc2.Currency {
		return false
	}
	if acc1.Balance != acc2.Balance {
		return false
	}
	if acc1.Frozen != acc2.Frozen {
		return false
	}
	if acc1.Addr != acc2.Addr {
		return false
	}
	return true
}

func testProcImportPrivKey(t *testing.T, wallet *Wallet) {
	println("TestProcImportPrivKey begin")

	//生成一个pubkey然后换算成对应的addr
	cr, err := crypto.Load(types.GetSignName("", wallet.SignType), wallet.lastHeader.GetHeight())
	require.NoError(t, err)

	priv, err := cr.GenKey()
	require.NoError(t, err)

	AddrPrivKey = common.ToHex(priv.Bytes())

	privKey := &types.ReqWalletImportPrivkey{Privkey: AddrPrivKey}
	privKey.Label = "account:0"
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletImportPrivkey", privKey)
	assert.Equal(t, err.Error(), types.ErrLabelHasUsed.Error())

	privKey.Label = "ImportPrivKey-Label"
	resp, _ := wallet.GetAPI().ExecWalletFunc("wallet", "WalletImportPrivkey", privKey)
	importedAcc := resp.(*types.WalletAccount)
	if importedAcc.Label != "ImportPrivKey-Label" || importedAcc.Acc.Addr != address.PubKeyToAddr(address.DefaultID, priv.PubKey().Bytes()) {
		t.Error("testProcImportPrivKey failed")
		return
	}

	//import privkey="0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privKey.Privkey = "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privKey.Label = "ImportPrivKey-Label-hyb"
	walletlog.Info("TestProcImportPrivKey", "Privkey", privKey.Privkey, "Label", privKey.Label)

	wallet.GetAPI().ExecWalletFunc("wallet", "WalletImportPrivkey", privKey)

	addr := &types.ReqString{Data: address.PubKeyToAddr(address.DefaultID, priv.PubKey().Bytes())}
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "DumpPrivkey", &types.ReqString{Data: "wrongaddr"})
	assert.Equal(t, err.Error(), types.ErrAddrNotExist.Error())

	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "DumpPrivkey", addr)
	if resp.(*types.ReplyString).Data != common.ToHex(priv.Bytes()) {
		t.Error("testDumpPrivKey failed")
	}

	//入参测试
	_, err = wallet.ProcImportPrivKey(nil)
	assert.Equal(t, err, types.ErrInvalidParam)
	privKey.Label = "hyb123"
	privKey.Privkey = "0xb694ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"

	_, err = wallet.ProcImportPrivKey(privKey)
	assert.Equal(t, err, types.ErrPrivkeyToPub)

	_, err = wallet.ProcImportPrivKey(nil)
	assert.Equal(t, err, types.ErrInvalidParam)

	privKey.Label = "hyb1234"
	privKey.Privkey = "lfllfllflflfllf"
	_, err = wallet.ProcImportPrivKey(privKey)
	assert.Equal(t, err, types.ErrFromHex)
	println("TestProcImportPrivKey end")
	println("--------------------------")
}

func testProcWalletTxList(t *testing.T, wallet *Wallet) {
	println("TestProcWalletTxList begin")

	//倒序获取最新的三笔交易
	txList := &types.ReqWalletTransactionList{
		Count:     3,
		Direction: 0,
		FromTx:    []byte(""),
	}
	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletTransactionList", txList)
	require.NoError(t, err)
	walletTxDetails := resp.(*types.WalletTxDetails)

	var FromTxstr string
	index := make([]int64, 3)

	if len(walletTxDetails.TxDetails) != 3 {
		t.Error("testProcWalletTxList failed")
	}
	println("TestProcWalletTxList dir last-------")
	for i, walletTxDetail := range walletTxDetails.TxDetails {
		println("TestProcWalletTxList", "Direction", txList.Direction, "WalletTxDetail", walletTxDetail.String())
		index[i] = walletTxDetail.GetHeight()*100000 + walletTxDetail.GetIndex()
		FromTxstr = fmt.Sprintf("%018d", walletTxDetail.GetHeight()*100000+walletTxDetail.GetIndex())
	}
	//倒序index值的判断，index[0]>index[1]>index[2]
	if index[0] <= index[1] {
		println("TestProcWalletTxList", "index[0]", index[0], "index[1]", index[1])
		t.Error("testProcWalletTxList:Reverse check fail!")
	}
	if index[1] <= index[2] {
		println("TestProcWalletTxList", "index[1]", index[1], "index[2]", index[2])
		t.Error("testProcWalletTxList:Reverse check fail!")
	}

	txList.Direction = 1
	txList.Count = 2
	txList.FromTx = []byte(FromTxstr)

	println("TestProcWalletTxList dir next-------")
	resp, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletTransactionList", txList)
	require.NoError(t, err)
	walletTxDetails = resp.(*types.WalletTxDetails)
	if len(walletTxDetails.TxDetails) != 2 {
		t.Error("testProcWalletTxList failed")
	}
	for _, walletTxDetail := range walletTxDetails.TxDetails {
		println("TestProcWalletTxList", "Direction", txList.Direction, "WalletTxDetail", walletTxDetail.String())
	}

	println("TestProcWalletTxList dir prv------")
	txList.Direction = 0
	resp, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletTransactionList", txList)
	require.NoError(t, err)
	walletTxDetails = resp.(*types.WalletTxDetails)
	if len(walletTxDetails.TxDetails) != 2 {
		t.Error("testProcWalletTxList failed")
	}
	for _, walletTxDetail := range walletTxDetails.TxDetails {
		println("TestProcWalletTxList", "Direction", txList.Direction, "WalletTxDetail", walletTxDetail.String())
	}

	//正序获取最早的三笔交易
	txList = &types.ReqWalletTransactionList{
		Count:     3,
		Direction: 1,
		FromTx:    []byte(""),
	}
	resp, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletTransactionList", txList)
	require.NoError(t, err)
	walletTxDetails = resp.(*types.WalletTxDetails)

	if len(walletTxDetails.TxDetails) != 3 {
		t.Error("testProcWalletTxList failed")
	}
	for i, walletTxDetail := range walletTxDetails.TxDetails {
		index[i] = walletTxDetail.GetHeight()*100000 + walletTxDetail.GetIndex()
	}
	//正序index值的判断，index[0]<index[1]<index[2]
	if index[0] >= index[1] {
		println("TestProcWalletTxList", "index[0]", index[0], "index[1]", index[1])
		t.Error("testProcWalletTxList:positive check fail!")
	}
	if index[1] >= index[2] {
		println("TestProcWalletTxList", "index[1]", index[1], "index[2]", index[2])
		t.Error("testProcWalletTxList:positive check fail!")
	}

	//count 大于1000个报错
	txList.Count = 1001
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletTransactionList", txList)
	assert.Equal(t, err, types.ErrMaxCountPerTime)

	//入参测试
	_, err = wallet.ProcWalletTxList(nil)
	assert.Equal(t, err, types.ErrInvalidParam)

	txList.Direction = 2
	_, err = wallet.ProcWalletTxList(txList)
	assert.Equal(t, err, types.ErrInvalidParam)
	println("TestProcWalletTxList end")
	println("--------------------------")
}

// (SendToAddress *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
func testProcSendToAddress(t *testing.T, wallet *Wallet) {
	println("TestProcSendToAddress begin")
	transfer := &types.ReqWalletSendToAddress{
		Amount: 1000,
		From:   FromAddr,
		Note:   "test",
		To:     "1L1zEgVcjqdM2KkQixENd7SZTaudKkcyDu",
	}
	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletSendToAddress", transfer)
	require.NoError(t, err)
	replyHash := resp.(*types.ReplyHash)
	println("transfer tx", "ReplyHash", common.ToHex(replyHash.Hash))
	withdraw := &types.ReqWalletSendToAddress{
		Amount: -1000,
		From:   FromAddr,
		Note:   "test",
		To:     "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp",
	}
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletSendToAddress", withdraw)
	//返回ErrAmount错误
	assert.Equal(t, err.Error(), types.ErrAmount.Error())
	require.Error(t, err)
	println("TestProcSendToAddress end")
	println("--------------------------")
}

// ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
func testProcWalletSetFee(t *testing.T, wallet *Wallet) {
	println("TestProcWalletSetFee begin")
	var fee int64 = 1000000
	walletSetFee := &types.ReqWalletSetFee{Amount: fee}
	wallet.GetAPI().ExecWalletFunc("wallet", "WalletSetFee", walletSetFee)
	if wallet.FeeAmount != fee {
		t.Error("testProcWalletSetFee failed")
	}
	println("TestProcWalletSetFee end")
	println("--------------------------")
}

// ProcWalletSetLabel(SetLabel *types.ReqWalletSetLabel) (*types.WalletAccount, error)
func testProcWalletSetLabel(t *testing.T, wallet *Wallet) {
	println("TestProcWalletSetLabel begin")
	setLabel := &types.ReqWalletSetLabel{
		Addr:  FromAddr,
		Label: "account:000",
	}
	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletSetLabel", setLabel)
	require.NoError(t, err)
	walletAcc := resp.(*types.WalletAccount)
	if walletAcc.Acc.Addr != FromAddr || walletAcc.Label != "account:000" {
		t.Error("testProcWalletSetLabel failed")
	}

	//再次设置
	setLabel.Label = "account:000"
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletSetLabel", setLabel)
	assert.Equal(t, err.Error(), types.ErrLabelHasUsed.Error())

	setLabel.Label = "account:001"
	wallet.GetAPI().ExecWalletFunc("wallet", "WalletSetLabel", setLabel)
	setLabel.Label = "account:000"
	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "WalletSetLabel", setLabel)
	walletAcc = resp.(*types.WalletAccount)
	if walletAcc.Acc.Addr != FromAddr || walletAcc.Label != "account:000" {
		t.Error("testProcWalletSetLabel failed")
	}

	println("TestProcWalletSetLabel end")
	println("--------------------------")
}

// ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
func testProcMergeBalance(t *testing.T, wallet *Wallet) {
	println("TestProcMergeBalance begin")
	mergeBalance := &types.ReqWalletMergeBalance{To: ToAddr2}
	resp, _ := wallet.GetAPI().ExecWalletFunc("wallet", "WalletMergeBalance", mergeBalance)
	replyHashes := resp.(*types.ReplyHashes)

	for _, hash := range replyHashes.Hashes {
		println("hash:", common.ToHex(hash))
	}

	println("TestProcMergeBalance end")
	println("--------------------------")
}

// ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
func testProcWalletSetPasswd(t *testing.T, wallet *Wallet) {
	println("TestProcWalletSetPasswd begin")
	passwd := &types.ReqWalletSetPasswd{
		OldPass: "wrongpassword",
		NewPass: "Newpass123",
	}
	resp, _ := wallet.GetAPI().ExecWalletFunc("wallet", "WalletSetPasswd", passwd)
	if string(resp.(*types.Reply).GetMsg()) != types.ErrVerifyOldpasswdFail.Error() {
		t.Error("testProcWalletSetPasswd failed")
	}

	passwd.OldPass = "password123"
	_, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletSetPasswd", passwd)
	require.NoError(t, err)
	println("TestProcWalletSetPasswd end")
	println("--------------------------")
}

// ProcWalletLock
func testProcWalletLock(t *testing.T, wallet *Wallet) {
	println("TestProcWalletLock begin")
	_, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletLock", &types.ReqNil{})
	require.NoError(t, err)

	transfer := &types.ReqWalletSendToAddress{
		Amount: 1000,
		From:   FromAddr,
		Note:   "test",
		To:     "1L1zEgVcjqdM2KkQixENd7SZTaudKkcyDu",
	}

	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "WalletSendToAddress", transfer)
	if err.Error() != types.ErrWalletIsLocked.Error() {
		t.Error("testProcWalletLock failed")
	}

	//解锁
	walletUnLock := &types.WalletUnLock{
		Passwd:         "",
		Timeout:        3,
		WalletOrTicket: false,
	}
	resp, _ := wallet.GetAPI().ExecWalletFunc("wallet", "WalletUnLock", walletUnLock)
	if string(resp.(*types.Reply).GetMsg()) != types.ErrInputPassword.Error() {
		t.Error("test input wrong password failed")
	}

	walletUnLock.Passwd = "Newpass123"
	wallet.GetAPI().ExecWalletFunc("wallet", "WalletUnLock", walletUnLock)

	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "GetSeed", &types.GetSeedByPw{Passwd: "Newpass123"})
	println("seed:", resp.(*types.ReplySeed).Seed)
	time.Sleep(time.Second * 5)
	wallet.GetAPI().ExecWalletFunc("wallet", "GetSeed", &types.GetSeedByPw{Passwd: "Newpass123"})
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "GetSeed", &types.GetSeedByPw{Passwd: "Newpass123"})
	if err.Error() != types.ErrWalletIsLocked.Error() {
		t.Error("testProcWalletLock failed")
	}

	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "GetWalletStatus", &types.ReqNil{})
	status := resp.(*types.WalletStatus)
	if !status.IsHasSeed || status.IsAutoMining || !status.IsWalletLock || (walletUnLock.GetWalletOrTicket() && status.IsTicketLock) {
		t.Error("testGetWalletStatus failed")
	}

	walletUnLock.Timeout = 0
	walletUnLock.WalletOrTicket = true
	err = wallet.ProcWalletUnLock(walletUnLock)
	require.NoError(t, err)
	resp, _ = wallet.GetAPI().ExecWalletFunc("wallet", "GetWalletStatus", &types.ReqNil{})
	status = resp.(*types.WalletStatus)
	if !status.IsHasSeed || status.IsAutoMining || !status.IsWalletLock {
		t.Error("testGetWalletStatus failed")
	}

	walletUnLock.WalletOrTicket = false
	err = wallet.ProcWalletUnLock(walletUnLock)
	require.NoError(t, err)

	println("TestProcWalletLock end")
	println("--------------------------")
}

// ProcWalletAddBlock
func testProcWalletAddBlock(t *testing.T, wallet *Wallet) {
	println("TestProcWalletAddBlock & TestProcWalletDelBlock begin")
	tx := &types.Transaction{Execer: []byte(types.NoneX), To: ToAddr1}
	blk := &types.Block{
		Version:    1,
		ParentHash: []byte("parent hash"),
		TxHash:     []byte("tx hash"),
		Height:     2,
		BlockTime:  1,
		Txs:        []*types.Transaction{tx},
	}
	blkDetail := &types.BlockDetail{
		Block:    blk,
		Receipts: []*types.ReceiptData{{Ty: types.ExecOk}},
	}
	msgAdd := wallet.client.NewMessage("wallet", types.EventAddBlock, blkDetail)
	err := wallet.client.Send(msgAdd, false)
	require.NoError(t, err)
	time.Sleep(time.Second * 10)
	msgDel := wallet.client.NewMessage("wallet", types.EventDelBlock, blkDetail)
	err = wallet.client.Send(msgDel, false)
	require.NoError(t, err)
	println("TestProcWalletAddBlock & TestProcWalletDelBlock end")
	println("--------------------------")
}

// SignRawTx
func testSignRawTx(t *testing.T, wallet *Wallet) {
	println("TestSignRawTx begin")
	unsigned := &types.ReqSignRawTx{
		Addr:   FromAddr,
		TxHex:  "0a05636f696e73120c18010a081080c2d72f1a01312080897a30c0e2a4a789d684ad443a0131",
		Expire: "0",
	}
	_, err := wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	require.NoError(t, err)

	unsigned.Privkey = AddrPrivKey
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	require.NoError(t, err)

	//交易组的签名
	group1 := "0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720c09a0c308dfddb82faf7dfc4113a2231444e615344524739524431397335396d65416f654e34613246365248393766536f40024aa5020aa3010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720c09a0c308dfddb82faf7dfc4113a2231444e615344524739524431397335396d65416f654e34613246365248393766536f40024a204d14e67e6123d8efee02bf0d707380e9b82e5bd8972d085974879a41190eba7c5220d41e1ba3a374424254f3f417de8175a34671238798a2c63b28a90ff0233679960a7d0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6730b8b082d799a4ddc93a3a2231444e615344524739524431397335396d65416f654e34613246365248393766536f40024a204d14e67e6123d8efee02bf0d707380e9b82e5bd8972d085974879a41190eba7c5220d41e1ba3a374424254f3f417de8175a34671238798a2c63b28a90ff023367996"
	unsigned.TxHex = group1
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	require.NoError(t, err)

	unsigned.TxHex = "0a05636f696e73120c18010a081080c2d72f1a01312080897a30c0e2a4a789d684ad443a0131"

	//重新设置toaddr和fee
	unsigned.NewToAddr = "1JzFKyrvSP5xWUkCMapUvrKDChgPDX1EN6"
	unsigned.Fee = 10000
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	require.NoError(t, err)

	//地址和私钥都为空
	unsigned.Privkey = ""
	unsigned.Addr = ""
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	assert.Equal(t, err, types.ErrNoPrivKeyOrAddr)

	unsigned.Privkey = "0x"
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	assert.Equal(t, err, types.ErrPrivateKeyLen)

	unsigned.Privkey = "0x5Z"
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	assert.NotNil(t, err)

	signTy := wallet.SignType
	wallet.SignType = 0xff
	unsigned.Privkey = "0x55"
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "SignRawTx", unsigned)
	assert.NotNil(t, err)
	wallet.SignType = signTy

	println("TestSignRawTx end")
	println("--------------------------")
}

// setFatalFailure
func testsetFatalFailure(t *testing.T, wallet *Wallet) {
	println("testsetFatalFailure begin")
	var reportErrEvent types.ReportErrEvent
	reportErrEvent.Frommodule = "wallet"
	reportErrEvent.Tomodule = "wallet"
	reportErrEvent.Error = "ErrDataBaseDamage"

	wallet.GetAPI().ExecWalletFunc("wallet", "ErrToFront", &reportErrEvent)
	println("testsetFatalFailure end")
	println("--------------------------")
}

// getFatalFailure
func testgetFatalFailure(t *testing.T, wallet *Wallet) {
	println("testgetFatalFailure begin")
	_, err := wallet.GetAPI().ExecWalletFunc("wallet", "FatalFailure", &types.ReqNil{})
	require.NoError(t, err)
	println("testgetFatalFailure end")
	println("--------------------------")
}

func testWallet(t *testing.T, wallet *Wallet) {
	println("test wallet begin")
	addr, priv = util.Genaddress()
	bpriv := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), priv.Bytes())
	was := &types.WalletAccountStore{Privkey: common.ToHex(bpriv), Label: "test", Addr: addr, TimeStamp: time.Now().String()}
	err := wallet.SetWalletAccount(false, addr, was)
	assert.NoError(t, err)
	was1, err := wallet.GetAccountByAddr(addr)
	assert.NoError(t, err)
	assert.Equal(t, was.Privkey, was1.Privkey)
	was2, err := wallet.GetAccountByLabel("test")
	assert.NoError(t, err)
	assert.Equal(t, was.Privkey, was2.Privkey)
	priv2, err := wallet.GetPrivKeyByAddr(addr)
	assert.NoError(t, err)
	assert.Equal(t, priv, priv2)
	_, err = wallet.GetWalletAccounts()
	assert.NoError(t, err)
	t.Log("password:", wallet.Password)

	wallet.walletStore.SetWalletPassword("Newpass2")
	assert.Equal(t, "Newpass2", wallet.walletStore.GetWalletPassword())

	err = wallet.walletStore.SetFeeAmount(1e5)
	assert.NoError(t, err)
	fee := wallet.walletStore.GetFeeAmount(1e4)
	assert.Equal(t, int64(1e5), fee)

	println("test wallet end")

	wallet.GetConfig()
	wallet.GetDBStore()
	wallet.GetSignType()
	wallet.GetPassword()
	wallet.Nonce()
	wallet.GetRandom()
	wallet.GetBlockHeight()
	wallet.GetWalletDone()
	wallet.GetLastHeader()
	wallet.IsClose()
	wallet.AddWaitGroup(1)
	wallet.WaitGroupDone()
	report := &WalletReport{}
	err = wallet.RegisterMineStatusReporter(report)
	assert.NoError(t, err)

}

func testSendTx(t *testing.T, wallet *Wallet) {
	ok := wallet.IsCaughtUp()
	assert.True(t, ok)

	_, err := wallet.GetBalance(addr, "coins")
	assert.NoError(t, err)

	_, err = wallet.GetAllPrivKeys()
	assert.NoError(t, err)

	hash, err := wallet.SendTransaction(&types.ReceiptAccountTransfer{}, []byte("coins"), priv, address.GetDefaultAddressID(), ToAddr1)
	assert.NoError(t, err)

	//wallet.WaitTx(hash)
	wallet.WaitTxs([][]byte{hash})
	hash, err = wallet.SendTransaction(&types.ReceiptAccountTransfer{}, []byte("test"), priv, address.GetDefaultAddressID(), ToAddr1)
	assert.NoError(t, err)
	t.Log(common.ToHex(hash))

	err = wallet.sendTransactionWait(&types.ReceiptAccountTransfer{}, []byte("test"), priv, address.GetDefaultAddressID(), ToAddr1)
	assert.NoError(t, err)

	_, err = wallet.getMinerColdAddr(addr)
	assert.Equal(t, types.ErrActionNotSupport, err)

}

func testCreateNewAccountByIndex(t *testing.T, wallet *Wallet) {
	println("testCreateNewAccountByIndex begin")

	//首先创建一个airdropaddr标签的账户
	reqNewAccount := &types.ReqNewAccount{Label: "airdropaddr"}
	respp, err := wallet.GetAPI().ExecWalletFunc("wallet", "NewAccount", reqNewAccount)
	require.NoError(t, err)
	walletAcc := respp.(*types.WalletAccount)
	addrtmp := walletAcc.GetAcc().Addr
	if walletAcc.GetLabel() != "airdropaddr" {
		t.Error("testCreateNewAccountByIndex", "walletAcc.GetLabel()", walletAcc.GetLabel(), "Label", "airdropaddr")
	}

	//index参数的校验。目前只支持10000000
	reqIndex := &types.Int32{Data: 0}
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "NewAccountByIndex", reqIndex)
	assert.Equal(t, types.ErrInvalidParam, err)

	//创建一个空投地址
	reqIndex = &types.Int32{Data: 100000000}
	resp1, err := wallet.GetAPI().ExecWalletFunc("wallet", "NewAccountByIndex", reqIndex)

	require.NoError(t, err)
	pubkey := resp1.(*types.ReplyString)

	//通过pubkey换算成addr然后获取账户信息
	privkeybyte, err := common.FromHex(pubkey.Data)
	require.NoError(t, err)
	pub, err := bipwallet.PrivkeyToPub(wallet.CoinType, uint32(wallet.SignType), privkeybyte)
	require.NoError(t, err)

	addr, err := bipwallet.PubToAddress(pub)
	require.NoError(t, err)
	if addr != "" {
		//测试ProcGetAccountList函数
		resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletGetAccountList", &types.ReqAccountList{})
		assert.Nil(t, err)
		accountlist := resp.(*types.WalletAccounts)
		for _, acc := range accountlist.Wallets {
			if addr == acc.Acc.Addr && addr != addrtmp {
				if acc.GetLabel() != ("airdropaddr" + fmt.Sprintf("%d", 1)) {
					t.Error("testCreateNewAccountByIndex", "addr", addr, "acc.GetLabel()", acc.GetLabel())
				}
			}
		}
	}

	//已经存在，和上一次获取的地址是一致的
	reqIndex = &types.Int32{Data: 100000000}
	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "NewAccountByIndex", reqIndex)

	require.NoError(t, err)
	pubkey = resp.(*types.ReplyString)

	//通过pubkey换算成addr然后获取账户信息
	privkeybyte, err = common.FromHex(pubkey.Data)
	require.NoError(t, err)
	pub2, err := bipwallet.PrivkeyToPub(wallet.CoinType, uint32(wallet.SignType), privkeybyte)
	require.NoError(t, err)
	addr2, err := bipwallet.PubToAddress(pub2)
	require.NoError(t, err)
	if addr != addr2 {
		t.Error("TestProcCreateNewAccount", "addr", addr, "addr2", addr2)
	}

	privstr := "0x78a8c993abf85d2a452233033c19fac6b3bd4fe2c805615b337ef75dacd86ac9"
	pubstr := "0277786ddef164b594f7db40d9a563f1ef1733cf34f1592f4c3bf1b344bd8f059b"
	addrstr := "19QtNuUS9UN4hQPLrnYr3UhJsQYy4z4TMT"
	privkeybyte, err = common.FromHex(privstr)
	require.NoError(t, err)
	pub3, err := bipwallet.PrivkeyToPub(wallet.CoinType, uint32(wallet.SignType), privkeybyte)
	require.NoError(t, err)
	pubtmp := hex.EncodeToString(pub3)
	if pubtmp != pubstr {
		t.Error("TestProcCreateNewAccount", "pubtmp", pubtmp, "pubstr", pubstr)
	}
	addr3, err := bipwallet.PubToAddress(pub3)
	require.NoError(t, err)
	if addr3 != addrstr {
		t.Error("TestProcCreateNewAccount", "addr3", addr3, "addrstr", addrstr)
	}
	println("TestProcCreateNewAccount end")
	println("--------------------------")
}

func TestInitSeedLibrary(t *testing.T) {
	wallet, store, q, _ := initEnv()
	defer os.RemoveAll("datadir") // clean up
	defer wallet.Close()
	defer store.Close()

	//启动blockchain模块
	blockchainModProc(q)
	mempoolModProc(q)

	InitSeedLibrary()
	you := ChineseSeedCache["有"]
	assert.Equal(t, "有", you)
	abandon := EnglishSeedCache["abandon"]
	assert.Equal(t, "abandon", abandon)

	_, err := wallet.IsTransfer("1JzFKyrvSP5xWUkCMapUvrKDChgPDX1EN6")
	assert.Equal(t, err, types.ErrWalletIsLocked)
	//获取随机种子
	replySeed, err := wallet.GenSeed(0)
	require.NoError(t, err)
	replySeed, err = wallet.GenSeed(1)
	require.NoError(t, err)
	password := "heyubin123"
	wallet.SaveSeed(password, replySeed.Seed)
	wallet.ProcWalletUnLock(&types.WalletUnLock{Passwd: password})

	//16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp ticket合约地址
	_, err = wallet.IsTransfer("16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
	require.NoError(t, err)

	_, err = GetPrivkeyBySeed(wallet.walletStore.GetDB(), replySeed.Seed, 0, 2, wallet.CoinType)
	require.NoError(t, err)

	// 指定生成私钥的index值
	_, err = GetPrivkeyBySeed(wallet.walletStore.GetDB(), replySeed.Seed, 10000, 2, wallet.CoinType)
	require.NoError(t, err)

	// cointype不支持
	_, err = GetPrivkeyBySeed(wallet.walletStore.GetDB(), replySeed.Seed, 10000, 50, wallet.CoinType)
	assert.Equal(t, err, types.ErrNotSupport)

	acc, err := wallet.GetBalance("1JzFKyrvSP5xWUkCMapUvrKDChgPDX1EN6", "token")
	require.NoError(t, err)
	assert.Equal(t, acc.Addr, "1JzFKyrvSP5xWUkCMapUvrKDChgPDX1EN6")
	assert.Equal(t, int64(0), acc.Balance)

	err = wallet.RegisterMineStatusReporter(nil)
	assert.Equal(t, err, types.ErrInvalidParam)

	in := wallet.AddrInWallet("")
	assert.Equal(t, in, false)
	policy := wcom.PolicyContainer[walletBizPolicyX]
	if policy != nil {
		policy.OnAddBlockTx(nil, nil, 0, nil)
		policy.OnDeleteBlockTx(nil, nil, 0, nil)
		need, _, _ := policy.SignTransaction(nil, nil)
		assert.Equal(t, true, need)
		account := types.Account{
			Addr:     "1JzFKyrvSP5xWUkCMapUvrKDChgPDX1EN6",
			Currency: 0,
			Balance:  0,
			Frozen:   0,
		}
		policy.OnCreateNewAccount(&account)
		policy.OnImportPrivateKey(&account)
		policy.OnAddBlockFinish(nil)
		policy.OnDeleteBlockFinish(nil)
		policy.OnClose()
		policy.OnSetQueueClient()
	}
}

func testProcDumpPrivkeysFile(t *testing.T, wallet *Wallet) {
	println("testProcDumpPrivkeysFile begin")
	fileName := "Privkeys"
	_, err := os.Stat(fileName)
	if err == nil {
		os.Remove(fileName)
	}

	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "DumpPrivkeysFile", &types.ReqPrivkeysFile{FileName: fileName, Passwd: "123456"})
	require.NoError(t, err)

	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletGetAccountList", &types.ReqAccountList{})
	assert.Nil(t, err)

	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "DumpPrivkeysFile", &types.ReqPrivkeysFile{FileName: fileName, Passwd: "123456"})
	assert.Equal(t, err, types.ErrFileExists)

	// 后面要对比
	AllAccountlist = resp.(*types.WalletAccounts)
	println("testProcDumpPrivkeysFile end")
	println("--------------------------")
}

func testProcImportPrivkeysFile(t *testing.T, wallet *Wallet) {
	println("testProcImportPrivkeysFile begin")
	fileName := "Privkeys"

	_, err := wallet.GetAPI().ExecWalletFunc("wallet", "ImportPrivkeysFile", &types.ReqPrivkeysFile{FileName: fileName, Passwd: "123456"})
	assert.Nil(t, err)

	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletGetAccountList", &types.ReqAccountList{})
	assert.Nil(t, err)

	// 与之前的 AllAccountlist 对比
	accountlist := resp.(*types.WalletAccounts)
	assert.Equal(t, len(accountlist.GetWallets()), len(AllAccountlist.GetWallets()))

	for _, acc1 := range AllAccountlist.GetWallets() {
		isEqual := false
		for _, acc2 := range accountlist.GetWallets() {
			if acc1.Acc.Addr == acc2.Acc.Addr {
				isEqual = true
				break
			}
		}
		if !isEqual {
			assert.Nil(t, errors.New(acc1.Acc.Addr+" not find in new list."))
		}
	}

	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "ImportPrivkeysFile", &types.ReqPrivkeysFile{FileName: fileName, Passwd: "12345678"})
	assert.Equal(t, err, types.ErrVerifyOldpasswdFail)

	println("testProcImportPrivkeysFile end")
	println("--------------------------")
}

func testProcImportPrivkeysFile2(t *testing.T, wallet *Wallet) {
	println("testProcImportPrivkeysFile begin")
	reqNewAccount := &types.ReqNewAccount{Label: "account:0"}
	_, err := wallet.GetAPI().ExecWalletFunc("wallet", "NewAccount", reqNewAccount)
	assert.NoError(t, err)

	privKey := &types.ReqWalletImportPrivkey{}
	privKey.Privkey = "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privKey.Label = "ImportPrivKey-Label-hyb-new"
	wallet.GetAPI().ExecWalletFunc("wallet", "WalletImportPrivkey", privKey)

	fileName := "Privkeys"
	_, err = wallet.GetAPI().ExecWalletFunc("wallet", "ImportPrivkeysFile", &types.ReqPrivkeysFile{FileName: fileName, Passwd: "123456"})
	assert.Nil(t, err)

	resp, err := wallet.GetAPI().ExecWalletFunc("wallet", "WalletGetAccountList", &types.ReqAccountList{})
	assert.Nil(t, err)

	// 与之前的 AllAccountlist 对比
	accountlist := resp.(*types.WalletAccounts)
	assert.Equal(t, len(accountlist.GetWallets()), len(AllAccountlist.GetWallets())+1)

	os.Remove(fileName)
	println("testProcImportPrivkeysFile end")
	println("--------------------------")
}

type WalletReport struct {
}

func (report WalletReport) IsAutoMining() bool {
	return true

}
func (report WalletReport) IsTicketLocked() bool {
	return true

}
func (report WalletReport) PolicyName() string {
	return "ticket"
}
