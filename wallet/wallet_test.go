// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"fmt"
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
	wcom "github.com/33cn/chain33/wallet/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	queue.DisableLog()
	SetLogLevel("err")
}

func initEnv() (*Wallet, queue.Module, queue.Queue) {
	var q = queue.New("channel")
	cfg, sub := types.InitCfg("../cmd/chain33/chain33.test.toml")

	wallet := New(cfg.Wallet, sub.Wallet)
	wallet.SetQueueClient(q.Client())

	store := store.New(cfg.Store, sub.Store)
	store.SetQueueClient(q.Client())

	return wallet, store, q
}

var (
	Statehash   []byte
	CutHeight   int64
	FromAddr    string
	ToAddr1     string
	ToAddr2     string
	AddrPrivKey string
	addr        string
	priv        crypto.PrivKey
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
			}
		}
	}()
}

func SaveAccountTomavl(client queue.Client, prevStateRoot []byte, accs []*types.Account) []byte {
	var kvset []*types.KeyValue

	for _, acc := range accs {
		kvs := accountdb.GetKVSet(acc)
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

func TestWallet(t *testing.T) {
	wallet, store, q := initEnv()
	defer wallet.Close()
	defer store.Close()

	//启动blockchain模块
	blockchainModProc(q)
	mempoolModProc(q)

	testSeed(t, wallet)

	testProcCreateNewAccount(t, wallet)

	testProcImportPrivKey(t, wallet)
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
}

//ProcWalletLock
func testSeed(t *testing.T, wallet *Wallet) {
	println("TestSeed begin")
	msgGen := wallet.client.NewMessage("wallet", types.EventGenSeed, &types.GenSeedLang{Lang: 1})
	wallet.client.Send(msgGen, true)
	seedRes, _ := wallet.client.Wait(msgGen)
	seed := seedRes.GetData().(*types.ReplySeed).Seed
	println("seed: ", seed)

	password := "password"
	saveSeedByPw := &types.SaveSeedByPw{Seed: "", Passwd: ""}
	msgSaveEmpty := wallet.client.NewMessage("wallet", types.EventSaveSeed, saveSeedByPw)
	wallet.client.Send(msgSaveEmpty, true)
	resp, err := wallet.client.Wait(msgSaveEmpty)
	assert.Nil(t, err)
	assert.Equal(t, string(resp.GetData().(*types.Reply).GetMsg()), types.ErrInvalidParam.Error())

	saveSeedByPw.Seed = "a b c d"
	saveSeedByPw.Passwd = password
	msgSaveShort := wallet.client.NewMessage("wallet", types.EventSaveSeed, saveSeedByPw)
	wallet.client.Send(msgSaveShort, true)
	resp, _ = wallet.client.Wait(msgSaveShort)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrSeedWordNum.Error() {
		t.Error("test input not enough seed failed")
	}

	saveSeedByPw.Seed = "a b c d e f g h i j k l m n"
	msgSaveInvalid := wallet.client.NewMessage("wallet", types.EventSaveSeed, saveSeedByPw)
	wallet.client.Send(msgSaveInvalid, true)
	resp, _ = wallet.client.Wait(msgSaveInvalid)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrSeedWord.Error() {
		t.Error("test input invalid seed failed")
	}

	saveSeedByPw.Seed = seed
	msgSave := wallet.client.NewMessage("wallet", types.EventSaveSeed, saveSeedByPw)
	wallet.client.Send(msgSave, true)
	_, err = wallet.client.Wait(msgSave)
	assert.Nil(t, err)

	seedstr, err := GetSeed(wallet.walletStore.GetDB(), password)
	require.NoError(t, err)
	if seed != seedstr {
		t.Error("testSaveSeed failed")
	}

	wallet.client.Send(msgSave, true)
	resp, _ = wallet.client.Wait(msgSave)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrSeedExist.Error() {
		t.Error("test seedExists failed")
	}

	walletUnLock := &types.WalletUnLock{
		Passwd:         password,
		Timeout:        0,
		WalletOrTicket: false,
	}
	msgUnlock := wallet.client.NewMessage("wallet", types.EventWalletUnLock, walletUnLock)
	wallet.client.Send(msgUnlock, true)
	_, err = wallet.client.Wait(msgUnlock)
	require.NoError(t, err)

	msgGet := wallet.client.NewMessage("wallet", types.EventGetSeed, &types.GetSeedByPw{Passwd: password})
	wallet.client.Send(msgGet, true)
	resp, err = wallet.client.Wait(msgGet)
	require.NoError(t, err)

	reply := resp.GetData().(*types.ReplySeed)
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
		msg := wallet.client.NewMessage("wallet", types.EventNewAccount, reqNewAccount)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
		require.NoError(t, err)
		time.Sleep(time.Microsecond * 100)
		walletAcc := resp.GetData().(*types.WalletAccount)
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
	cr, err := crypto.New(types.GetSignName("", SignType))
	require.NoError(t, err)

	Privkey := "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privkeybyte, _ := common.FromHex(Privkey)
	priv, err := cr.PrivKeyFromBytes(privkeybyte)
	require.NoError(t, err)

	addr := address.PubKeyToAddress(priv.PubKey().Bytes())
	FromAddr = addr.String()
	var acc types.Account
	acc.Addr = addr.String()
	acc.Balance = int64(1e16)
	acc.Currency = int32(10)
	acc.Frozen = int64(10)
	accs[total] = &acc

	//存入账户信息到mavl树中
	SaveAccountTomavl(wallet.client, nil, accs)

	//测试ProcGetAccountList函数
	msgGetAccList := wallet.client.NewMessage("wallet", types.EventWalletGetAccountList, &types.ReqAccountList{})
	wallet.client.Send(msgGetAccList, true)
	resp, err := wallet.client.Wait(msgGetAccList)
	assert.Nil(t, err)
	accountlist := resp.GetData().(*types.WalletAccounts)
	for _, acc1 := range accountlist.Wallets {
		exist := false
		for _, acc2 := range accs[:10] {
			if equal(*acc1.Acc, *acc2) {
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

func equal(acc1 types.Account, acc2 types.Account) bool {
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
	cr, err := crypto.New(types.GetSignName("", SignType))
	require.NoError(t, err)

	priv, err := cr.GenKey()
	require.NoError(t, err)

	AddrPrivKey = common.ToHex(priv.Bytes())

	privKey := &types.ReqWalletImportPrivkey{Privkey: AddrPrivKey}
	privKey.Label = "account:0"

	msgImport := wallet.client.NewMessage("wallet", types.EventWalletImportPrivkey, privKey)
	wallet.client.Send(msgImport, true)
	_, err = wallet.client.Wait(msgImport)
	assert.Equal(t, err.Error(), types.ErrLabelHasUsed.Error())

	privKey.Label = "ImportPrivKey-Label"
	msgImport = wallet.client.NewMessage("wallet", types.EventWalletImportPrivkey, privKey)
	wallet.client.Send(msgImport, true)
	resp, _ := wallet.client.Wait(msgImport)
	importedAcc := resp.GetData().(*types.WalletAccount)
	if importedAcc.Label != "ImportPrivKey-Label" || importedAcc.Acc.Addr != address.PubKeyToAddress(priv.PubKey().Bytes()).String() {
		t.Error("testProcImportPrivKey failed")
		return
	}

	//import privkey="0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privKey.Privkey = "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privKey.Label = "ImportPrivKey-Label-hyb"
	walletlog.Info("TestProcImportPrivKey", "Privkey", privKey.Privkey, "Label", privKey.Label)

	msgImport = wallet.client.NewMessage("wallet", types.EventWalletImportPrivkey, privKey)
	wallet.client.Send(msgImport, true)
	wallet.client.Wait(msgImport)

	addr := &types.ReqString{Data: address.PubKeyToAddress(priv.PubKey().Bytes()).String()}
	msgDump := wallet.client.NewMessage("wallet", types.EventDumpPrivkey, &types.ReqString{Data: "wrongaddr"})
	wallet.client.Send(msgDump, true)
	_, err = wallet.client.Wait(msgDump)
	assert.Equal(t, err.Error(), types.ErrAddrNotExist.Error())
	msgDump = wallet.client.NewMessage("wallet", types.EventDumpPrivkey, addr)
	wallet.client.Send(msgDump, true)
	resp, _ = wallet.client.Wait(msgDump)
	if resp.GetData().(*types.ReplyString).Data != common.ToHex(priv.Bytes()) {
		t.Error("testDumpPrivKey failed")
	}

	println("TestProcImportPrivKey end")
	println("--------------------------")
}

func testProcWalletTxList(t *testing.T, wallet *Wallet) {
	println("TestProcWalletTxList begin")
	txList := &types.ReqWalletTransactionList{
		Count:     3,
		Direction: 1,
		FromTx:    []byte(""),
	}
	msg := wallet.client.NewMessage("wallet", types.EventWalletTransactionList, txList)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	require.NoError(t, err)
	walletTxDetails := resp.GetData().(*types.WalletTxDetails)

	var FromTxstr string
	if len(walletTxDetails.TxDetails) != 3 {
		t.Error("testProcWalletTxList failed")
	}
	println("TestProcWalletTxList dir last-------")
	for _, walletTxDetail := range walletTxDetails.TxDetails {
		println("TestProcWalletTxList", "Direction", txList.Direction, "WalletTxDetail", walletTxDetail.String())
		FromTxstr = fmt.Sprintf("%018d", walletTxDetail.GetHeight()*100000+walletTxDetail.GetIndex())
	}

	txList.Direction = 1
	txList.Count = 2
	txList.FromTx = []byte(FromTxstr)

	println("TestProcWalletTxList dir next-------")
	msg = wallet.client.NewMessage("wallet", types.EventWalletTransactionList, txList)
	wallet.client.Send(msg, true)
	resp, err = wallet.client.Wait(msg)
	require.NoError(t, err)
	walletTxDetails = resp.GetData().(*types.WalletTxDetails)
	if len(walletTxDetails.TxDetails) != 2 {
		t.Error("testProcWalletTxList failed")
	}
	for _, walletTxDetail := range walletTxDetails.TxDetails {
		println("TestProcWalletTxList", "Direction", txList.Direction, "WalletTxDetail", walletTxDetail.String())
	}

	println("TestProcWalletTxList dir prv------")
	txList.Direction = 0
	msg = wallet.client.NewMessage("wallet", types.EventWalletTransactionList, txList)
	wallet.client.Send(msg, true)
	resp, err = wallet.client.Wait(msg)
	require.NoError(t, err)
	walletTxDetails = resp.GetData().(*types.WalletTxDetails)
	if len(walletTxDetails.TxDetails) != 2 {
		t.Error("testProcWalletTxList failed")
	}
	for _, walletTxDetail := range walletTxDetails.TxDetails {
		println("TestProcWalletTxList", "Direction", txList.Direction, "WalletTxDetail", walletTxDetail.String())
	}
	println("TestProcWalletTxList end")
	println("--------------------------")
}

//(SendToAddress *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
func testProcSendToAddress(t *testing.T, wallet *Wallet) {
	println("TestProcSendToAddress begin")
	transfer := &types.ReqWalletSendToAddress{
		Amount: 1000,
		From:   FromAddr,
		Note:   "test",
		To:     "1L1zEgVcjqdM2KkQixENd7SZTaudKkcyDu",
	}
	msg := wallet.client.NewMessage("wallet", types.EventWalletSendToAddress, transfer)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	require.NoError(t, err)
	replyHash := resp.GetData().(*types.ReplyHash)
	println("transfer tx", "ReplyHash", common.ToHex(replyHash.Hash))
	withdraw := &types.ReqWalletSendToAddress{
		Amount: -1000,
		From:   FromAddr,
		Note:   "test",
		To:     "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp",
	}
	msg = wallet.client.NewMessage("wallet", types.EventWalletSendToAddress, withdraw)
	wallet.client.Send(msg, true)
	resp, err = wallet.client.Wait(msg)
	//返回ErrAmount错误
	assert.Equal(t, string(err.Error()), types.ErrAmount.Error())
	require.Error(t, err)
	//replyHash = resp.GetData().(*types.ReplyHash)
	//println("withdraw tx", "ReplyHash", common.ToHex(replyHash.Hash))
	println("TestProcSendToAddress end")
	println("--------------------------")
}

//ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
func testProcWalletSetFee(t *testing.T, wallet *Wallet) {
	println("TestProcWalletSetFee begin")
	var fee int64 = 1000000
	walletSetFee := &types.ReqWalletSetFee{Amount: fee}
	msg := wallet.client.NewMessage("wallet", types.EventWalletSetFee, walletSetFee)
	wallet.client.Send(msg, true)
	wallet.client.Wait(msg)
	if wallet.FeeAmount != fee {
		t.Error("testProcWalletSetFee failed")
	}
	println("TestProcWalletSetFee end")
	println("--------------------------")
}

//ProcWalletSetLabel(SetLabel *types.ReqWalletSetLabel) (*types.WalletAccount, error)
func testProcWalletSetLabel(t *testing.T, wallet *Wallet) {
	println("TestProcWalletSetLabel begin")
	setLabel := &types.ReqWalletSetLabel{
		Addr:  FromAddr,
		Label: "account:000",
	}
	msg := wallet.client.NewMessage("wallet", types.EventWalletSetLabel, setLabel)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	require.NoError(t, err)
	walletAcc := resp.GetData().(*types.WalletAccount)
	if walletAcc.Acc.Addr != FromAddr || walletAcc.Label != "account:000" {
		t.Error("testProcWalletSetLabel failed")
	}

	//再次设置
	setLabel.Label = "account:000"
	msg = wallet.client.NewMessage("wallet", types.EventWalletSetLabel, setLabel)
	wallet.client.Send(msg, true)
	_, err = wallet.client.Wait(msg)
	assert.Equal(t, err.Error(), types.ErrLabelHasUsed.Error())

	setLabel.Label = "account:001"
	msg = wallet.client.NewMessage("wallet", types.EventWalletSetLabel, setLabel)
	wallet.client.Send(msg, true)
	wallet.client.Wait(msg)
	setLabel.Label = "account:000"
	msg = wallet.client.NewMessage("wallet", types.EventWalletSetLabel, setLabel)
	wallet.client.Send(msg, true)
	resp, _ = wallet.client.Wait(msg)
	walletAcc = resp.GetData().(*types.WalletAccount)
	if walletAcc.Acc.Addr != FromAddr || walletAcc.Label != "account:000" {
		t.Error("testProcWalletSetLabel failed")
	}

	println("TestProcWalletSetLabel end")
	println("--------------------------")
}

//ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
func testProcMergeBalance(t *testing.T, wallet *Wallet) {
	println("TestProcMergeBalance begin")
	mergeBalance := &types.ReqWalletMergeBalance{To: ToAddr2}
	msg := wallet.client.NewMessage("wallet", types.EventWalletMergeBalance, mergeBalance)
	wallet.client.Send(msg, true)
	resp, _ := wallet.client.Wait(msg)
	replyHashes := resp.GetData().(*types.ReplyHashes)

	for _, hash := range replyHashes.Hashes {
		println("hash:", common.ToHex(hash))
	}

	println("TestProcMergeBalance end")
	println("--------------------------")
}

//ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
func testProcWalletSetPasswd(t *testing.T, wallet *Wallet) {
	println("TestProcWalletSetPasswd begin")
	passwd := &types.ReqWalletSetPasswd{
		OldPass: "wrongpassword",
		NewPass: "Newpass",
	}
	msg := wallet.client.NewMessage("wallet", types.EventWalletSetPasswd, passwd)
	wallet.client.Send(msg, true)
	resp, _ := wallet.client.Wait(msg)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrVerifyOldpasswdFail.Error() {
		t.Error("testProcWalletSetPasswd failed")
	}

	passwd.OldPass = "password"
	msg = wallet.client.NewMessage("wallet", types.EventWalletSetPasswd, passwd)
	wallet.client.Send(msg, true)
	_, err := wallet.client.Wait(msg)
	require.NoError(t, err)
	println("TestProcWalletSetPasswd end")
	println("--------------------------")
}

//ProcWalletLock
func testProcWalletLock(t *testing.T, wallet *Wallet) {
	println("TestProcWalletLock begin")
	msg := wallet.client.NewMessage("wallet", types.EventWalletLock, nil)
	wallet.client.Send(msg, true)
	_, err := wallet.client.Wait(msg)
	require.NoError(t, err)

	transfer := &types.ReqWalletSendToAddress{
		Amount: 1000,
		From:   FromAddr,
		Note:   "test",
		To:     "1L1zEgVcjqdM2KkQixENd7SZTaudKkcyDu",
	}
	msg = wallet.client.NewMessage("wallet", types.EventWalletSendToAddress, transfer)
	wallet.client.Send(msg, true)
	resp, _ := wallet.client.Wait(msg)

	if resp.Err().Error() != types.ErrWalletIsLocked.Error() {
		t.Error("testProcWalletLock failed")
	}

	//解锁
	walletUnLock := &types.WalletUnLock{
		Passwd:         "",
		Timeout:        3,
		WalletOrTicket: false,
	}
	msg = wallet.client.NewMessage("wallet", types.EventWalletUnLock, walletUnLock)
	wallet.client.Send(msg, true)
	resp, _ = wallet.client.Wait(msg)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrInputPassword.Error() {
		t.Error("test input wrong password failed")
	}

	walletUnLock.Passwd = "Newpass"
	msg = wallet.client.NewMessage("wallet", types.EventWalletUnLock, walletUnLock)
	wallet.client.Send(msg, true)
	wallet.client.Wait(msg)

	msgGetSeed := wallet.client.NewMessage("wallet", types.EventGetSeed, &types.GetSeedByPw{Passwd: "Newpass"})
	wallet.client.Send(msgGetSeed, true)
	resp, _ = wallet.client.Wait(msgGetSeed)
	println("seed:", resp.GetData().(*types.ReplySeed).Seed)
	time.Sleep(time.Second * 5)
	wallet.client.Send(msgGetSeed, true)
	resp, _ = wallet.client.Wait(msgGetSeed)
	if resp.Err().Error() != types.ErrWalletIsLocked.Error() {
		t.Error("testProcWalletLock failed")
	}

	msg = wallet.client.NewMessage("wallet", types.EventGetWalletStatus, nil)
	wallet.client.Send(msg, true)
	resp, _ = wallet.client.Wait(msg)
	status := resp.GetData().(*types.WalletStatus)
	if !status.IsHasSeed || status.IsAutoMining || !status.IsWalletLock || (walletUnLock.GetWalletOrTicket() && status.IsTicketLock) {
		t.Error("testGetWalletStatus failed")
	}

	walletUnLock.Timeout = 0
	walletUnLock.WalletOrTicket = true
	err = wallet.ProcWalletUnLock(walletUnLock)
	require.NoError(t, err)
	wallet.client.Send(msg, true)
	resp, _ = wallet.client.Wait(msg)
	status = resp.GetData().(*types.WalletStatus)
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
	tx := &types.Transaction{Execer: []byte(types.NoneX)}
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
	msg := wallet.client.NewMessage("wallet", types.EventSignRawTx, unsigned)
	wallet.client.Send(msg, true)
	_, err := wallet.client.Wait(msg)
	require.NoError(t, err)

	unsigned.Privkey = AddrPrivKey
	msg = wallet.client.NewMessage("wallet", types.EventSignRawTx, unsigned)
	wallet.client.Send(msg, true)
	_, err = wallet.client.Wait(msg)
	require.NoError(t, err)
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

	msg := wallet.client.NewMessage("wallet", types.EventErrToFront, &reportErrEvent)
	wallet.client.Send(msg, false)
	println("testsetFatalFailure end")
	println("--------------------------")
}

// getFatalFailure
func testgetFatalFailure(t *testing.T, wallet *Wallet) {
	println("testgetFatalFailure begin")
	msg := wallet.client.NewMessage("wallet", types.EventFatalFailure, nil)
	wallet.client.Send(msg, true)
	_, err := wallet.client.Wait(msg)
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
	wallet.GetMutex()
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
	wallet.RegisterMineStatusReporter(nil)
}

func testSendTx(t *testing.T, wallet *Wallet) {
	ok := wallet.IsCaughtUp()
	assert.True(t, ok)

	_, err := wallet.GetBalance(addr, "coins")
	assert.NoError(t, err)

	_, err = wallet.GetAllPrivKeys()
	assert.NoError(t, err)
	hash, err := wallet.SendTransaction(&types.ReceiptAccountTransfer{}, []byte("coins"), priv, ToAddr1)
	assert.NoError(t, err)

	//wallet.WaitTx(hash)
	wallet.WaitTxs([][]byte{hash})
	hash, err = wallet.SendTransaction(&types.ReceiptAccountTransfer{}, []byte("test"), priv, ToAddr1)
	assert.NoError(t, err)
	t.Log(string(hash))

	err = wallet.sendTransactionWait(&types.ReceiptAccountTransfer{}, []byte("test"), priv, ToAddr1)
	assert.NoError(t, err)

	_, err = wallet.getMinerColdAddr(addr)
	assert.Equal(t, types.ErrActionNotSupport, err)

}
