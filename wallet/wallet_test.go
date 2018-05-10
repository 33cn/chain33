package wallet

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	// "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

func init() {
	queue.DisableLog()
	SetLogLevel("err")
}

func initEnv() (*Wallet, queue.Module, queue.Queue) {
	var q = queue.New("channel")
	cfg := config.InitCfg("../cmd/chain33/chain33.test.toml")

	wallet := New(cfg.Wallet)
	wallet.SetQueueClient(q.Client())

	store := store.New(cfg.Store)
	store.SetQueueClient(q.Client())

	return wallet, store, q
}

var (
	Statehash []byte
	CutHeight int64
	FromAddr  string
	ToAddr1   string
	ToAddr2   string
)

func blockchainModProc(q queue.Queue) {
	//store
	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			//walletlog.Info("blockchain", "msg.Ty", msg.Ty)
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
				msg.Reply(client.NewMessage("wallet", types.EventReply, &types.Reply{true, nil}))
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
	hash := util.ExecKVMemSet(client, prevStateRoot, kvset, true)
	Statehash = hash
	util.ExecKVSetCommit(client, Statehash)
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

	testProcCreatNewAccount(t, wallet)

	testProcImportPrivKey(t, wallet)

	testProcWalletTxList(t, wallet)

	testProcSendToAddress(t, wallet)

	testProcWalletSetFee(t, wallet)

	testProcWalletSetLabel(t, wallet)

	testProcMergeBalance(t, wallet)

	testProcWalletSetPasswd(t, wallet)

	testProcWalletLock(t, wallet)
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
	msgSave := wallet.client.NewMessage("wallet", types.EventSaveSeed, &types.SaveSeedByPw{Seed: seed, Passwd: password})
	wallet.client.Send(msgSave, true)
	wallet.client.Wait(msgSave)

	seedstr, err := GetSeed(wallet.walletStore.db, password)
	require.NoError(t, err)

	if seed != seedstr {
		t.Error("testSaveSeed failed")
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
	resp, err := wallet.client.Wait(msgGet)
	require.NoError(t, err)

	reply := resp.GetData().(*types.ReplySeed)
	if reply.GetSeed() != seed {
		t.Error("testGetSeed failed")
	}

	println("TestSeed end")
	println("--------------------------")
}

func testProcCreatNewAccount(t *testing.T, wallet *Wallet) {
	println("TestProcCreatNewAccount begin")
	total := 10
	addres := make([]string, total)
	accs := make([]*types.Account, total+1)
	for i := 0; i < total; i++ {
		reqNewAccount := &types.ReqNewAccount{Label: fmt.Sprintf("account:%d", i)}
		msg := wallet.client.NewMessage("wallet", types.EventNewAccount, reqNewAccount)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
		require.NoError(t, err)
		time.Sleep(time.Microsecond * 100)
		walletAcc := resp.GetData().(*types.WalletAccount)
		addres[i] = walletAcc.Acc.Addr
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
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	require.NoError(t, err)

	Privkey := "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privkeybyte, _ := common.FromHex(Privkey)
	priv, err := cr.PrivKeyFromBytes(privkeybyte)
	require.NoError(t, err)

	addr := account.PubKeyToAddress(priv.PubKey().Bytes())
	FromAddr = addr.String()
	var acc types.Account
	acc.Addr = addr.String()
	acc.Balance = int64(1000000000)
	acc.Currency = int32(10)
	acc.Frozen = int64(10)
	accs[total] = &acc

	//存入账户信息到mavl树中
	SaveAccountTomavl(wallet.client, nil, accs)

	//测试ProcGetAccountList函数
	msgGetAccList := wallet.client.NewMessage("wallet", types.EventWalletGetAccountList, nil)
	wallet.client.Send(msgGetAccList, true)
	resp, _ := wallet.client.Wait(msgGetAccList)

	accountlist := resp.GetData().(*types.WalletAccounts)
	for _, acc1 := range accountlist.Wallets {
		exist := false
		for _, acc2 := range accs[:10] {
			if *acc1.Acc == *acc2 {
				exist = true
				break
			}
		}
		if !exist {
			t.Error("account list not match")
			return
		}
	}
	println("TestProcCreatNewAccount end")
	println("--------------------------")
}

func testProcImportPrivKey(t *testing.T, wallet *Wallet) {
	println("TestProcImportPrivKey begin")

	//生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	require.NoError(t, err)

	priv, err := cr.GenKey()
	require.NoError(t, err)

	privKey := &types.ReqWalletImportPrivKey{Privkey: common.ToHex(priv.Bytes())}
	privKey.Label = "account:0"

	msgImport := wallet.client.NewMessage("wallet", types.EventWalletImportprivkey, privKey)
	wallet.client.Send(msgImport, true)
	resp, err := wallet.client.Wait(msgImport)

	if resp.Err().Error() != types.ErrLabelHasUsed.Error() {
		t.Error("test used label failed")
		return
	}

	privKey.Label = "ImportPrivKey-Label"
	msgImport = wallet.client.NewMessage("wallet", types.EventWalletImportprivkey, privKey)
	wallet.client.Send(msgImport, true)
	resp, _ = wallet.client.Wait(msgImport)
	importedAcc := resp.GetData().(*types.WalletAccount)
	if importedAcc.Label != "ImportPrivKey-Label" || importedAcc.Acc.Addr != account.PubKeyToAddress(priv.PubKey().Bytes()).String() {
		t.Error("testProcImportPrivKey failed")
		return
	}

	//import privkey="0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privKey.Privkey = "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	privKey.Label = "ImportPrivKey-Label-hyb"
	walletlog.Info("TestProcImportPrivKey", "Privkey", privKey.Privkey, "Label", privKey.Label)

	msgImport = wallet.client.NewMessage("wallet", types.EventWalletImportprivkey, privKey)
	wallet.client.Send(msgImport, true)
	wallet.client.Wait(msgImport)

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
	walletlog.Info("TestProcSendToAddress begin")

	//先解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "heyubin" //wallet.Password
	WalletUnLock.Timeout = 0
	WalletUnLock.WalletOrTicket = false
	wallet.ProcWalletUnLock(&WalletUnLock)

	var SendToAddress types.ReqWalletSendToAddress
	SendToAddress.Amount = 1000
	SendToAddress.From = FromAddr
	SendToAddress.Note = "test"
	SendToAddress.To = "1L1zEgVcjqdM2KkQixENd7SZTaudKkcyDu"
	walletlog.Info("TestProcSendToAddress", "FromAddr", FromAddr)

	ReplyHash, err := wallet.ProcSendToAddress(&SendToAddress)
	require.NoError(t, err)
	walletlog.Info("TestProcSendToAddress", "ReplyHash", ReplyHash)
	walletlog.Info("TestProcSendToAddress end")
}

//ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
func testProcWalletSetFee(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletSetFee begin")

	//先解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "heyubin" //wallet.Password
	WalletUnLock.Timeout = 0
	WalletUnLock.WalletOrTicket = false
	wallet.ProcWalletUnLock(&WalletUnLock)

	var WalletSetFee types.ReqWalletSetFee
	WalletSetFee.Amount = 1000000
	err := wallet.ProcWalletSetFee(&WalletSetFee)
	require.NoError(t, err)
	walletlog.Info("TestProcWalletSetFee success")
	walletlog.Info("TestProcWalletSetFee!", "FeeAmount", wallet.FeeAmount)

	walletlog.Info("TestProcWalletSetFee end")
}

//ProcWalletSetLabel(SetLabel *types.ReqWalletSetLabel) (*types.WalletAccount, error)
func testProcWalletSetLabel(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletSetLabel begin")
	//先解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "heyubin" //wallet.Password
	WalletUnLock.Timeout = 0
	WalletUnLock.WalletOrTicket = false
	wallet.ProcWalletUnLock(&WalletUnLock)

	var SetLabel types.ReqWalletSetLabel
	SetLabel.Addr = FromAddr
	SetLabel.Label = "hybaccount:000"

	Acc, err := wallet.ProcWalletSetLabel(&SetLabel)
	require.NoError(t, err)
	if Acc != nil {
		walletlog.Info("TestProcWalletSetLabel success", "account", Acc.String())
	}
	//测试ProcGetAccountList函数
	Accounts, err := wallet.ProcGetAccountList()
	require.NoError(t, err)
	if Accounts != nil {
		for _, Account := range Accounts.Wallets {
			walletlog.Info("TestProcWalletSetLabel:", "Account", Account.String())
		}
	}

	//再次设置
	SetLabel.Label = "hybaccount:001"
	Acc, err = wallet.ProcWalletSetLabel(&SetLabel)
	require.NoError(t, err)
	if Acc != nil {
		walletlog.Info("TestProcWalletSetLabel success", "account", Acc.String())
	}

	//测试ProcGetAccountList函数
	Accounts, err = wallet.ProcGetAccountList()
	require.NoError(t, err)
	if Accounts != nil {
		for _, Account := range Accounts.Wallets {
			walletlog.Info("TestProcWalletSetLabel:", "Account", Account.String())
		}
	}

	//再次设置成
	SetLabel.Label = "hybaccount:000"
	Acc, err = wallet.ProcWalletSetLabel(&SetLabel)
	require.NoError(t, err)
	if Acc != nil {
		walletlog.Info("TestProcWalletSetLabel success", "account", Acc.String())
	}

	//测试ProcGetAccountList函数
	Accounts, err = wallet.ProcGetAccountList()
	require.NoError(t, err)
	if Accounts != nil {
		for _, Account := range Accounts.Wallets {
			walletlog.Info("TestProcWalletSetLabel:", "Account", Account.String())
		}
	}

	walletlog.Info("TestProcWalletSetLabel end")
}

//ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
func testProcMergeBalance(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcMergeBalance begin")

	//先解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "heyubin" //wallet.Password
	WalletUnLock.Timeout = 0
	WalletUnLock.WalletOrTicket = false
	wallet.ProcWalletUnLock(&WalletUnLock)

	var MergeBalance types.ReqWalletMergeBalance
	MergeBalance.To = ToAddr2 //"14ZTV2wHG3uPHnA5cBJmNxAxxvbzS7Z5mE"

	hashs, err := wallet.ProcMergeBalance(&MergeBalance)
	require.NoError(t, err)
	for _, hash := range hashs.Hashes {
		walletlog.Info("TestProcMergeBalance", "hash", hash)
	}
	walletlog.Info("TestProcMergeBalance end")
}

//ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
func testProcWalletSetPasswd(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletSetPasswd begin")

	var Passwd types.ReqWalletSetPasswd
	Passwd.Oldpass = "password"
	Passwd.Newpass = "Newpass"

	err := wallet.ProcWalletSetPasswd(&Passwd)
	require.NoError(t, err)

	walletlog.Info("TestProcWalletSetPasswd TestProcMergeBalance ")
	//新密码先解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "Newpass" //wallet.Password
	WalletUnLock.Timeout = 0
	WalletUnLock.WalletOrTicket = false
	wallet.ProcWalletUnLock(&WalletUnLock)

	var MergeBalance types.ReqWalletMergeBalance
	MergeBalance.To = ToAddr1 //"14ZTV2wHG3uPHnA5cBJmNxAxxvbzS7Z5mE"

	hashs, err := wallet.ProcMergeBalance(&MergeBalance)
	require.NoError(t, err)
	for _, hash := range hashs.Hashes {
		walletlog.Info("TestProcWalletSetPasswd TestProcMergeBalance", "hash", hash)
	}
	walletlog.Info("TestProcWalletSetPasswd end")
}

//ProcWalletLock
func testProcWalletLock(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletLock begin")

	err := wallet.ProcWalletLock()
	require.NoError(t, err)
	_, err = wallet.ProcGetAccountList()
	require.NoError(t, err)

	//解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "Newpass" //wallet.Password
	WalletUnLock.Timeout = 1
	WalletUnLock.WalletOrTicket = false
	err = wallet.ProcWalletUnLock(&WalletUnLock)
	require.NoError(t, err)
	walletlog.Info("ProcWalletUnLock ok")

	begin := time.Now()
	flag := 0
	//测试timeout
	for {
		var WalletSetFee types.ReqWalletSetFee
		WalletSetFee.Amount = 10000000
		seed, err := wallet.getSeed("Newpass")
		if err == nil {
			if flag == 0 {
				walletlog.Info("getSeed success", "seed", seed)
				flag = 1
			}
			if time.Since(begin) > 3*time.Second {
				t.Error("WalletUnLock.Timeout > 3")
			}
			//sleep 0.1 Second
			time.Sleep(100 * time.Millisecond)
		} else {
			walletlog.Info("getSeed", "err", err)
			break
		}
	}
	walletlog.Info("TestProcWalletLock end")
}
