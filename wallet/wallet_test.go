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
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

func init() {
	queue.DisableLog()
	log.SetLogLevel("err")
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

	testSaveSeed(t, wallet)

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
func testSaveSeed(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestSaveSeed begin --------------------")
	seed := "好 矩 一 否 买 左 丹 香 亚 共 舍 节 骂 殿 桥"
	password := "heyubin"
	ok, _ := wallet.saveSeed(password, seed)
	if ok {
		seedstr, err := GetSeed(wallet.walletStore.db, password)
		require.NoError(t, err)
		walletlog.Info("TestSaveSeed", "seed", seedstr)
	}
	walletlog.Info("TestSaveSeed end --------------------")
}

func testProcCreatNewAccount(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcCreatNewAccount begin --------------------")

	//先解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "heyubin" //wallet.Password
	WalletUnLock.Timeout = 0
	WalletUnLock.WalletOrTicket = false
	wallet.ProcWalletUnLock(&WalletUnLock)

	total := 10
	addres := make([]string, total)
	accs := make([]*types.Account, total+1)
	for i := 0; i < total; i++ {
		var ReqNewAccount types.ReqNewAccount
		ReqNewAccount.Label = fmt.Sprintf("hybaccount:%d", i)
		time.Sleep(time.Second * 1)
		Walletacc, err := wallet.ProcCreatNewAccount(&ReqNewAccount)
		require.NoError(t, err)
		addres[i] = Walletacc.Acc.Addr

		Walletacc.Acc.Balance = int64(i)
		Walletacc.Acc.Currency = int32(i)
		Walletacc.Acc.Frozen = int64(i)
		accs[i] = Walletacc.Acc
		//FromAddr = Walletacc.Acc.Addr
		if i == 0 {
			ToAddr1 = Walletacc.Acc.Addr
		}
		if i == 1 {
			ToAddr2 = Walletacc.Acc.Addr
		}
		walletlog.Info("ProcCreatNewAccount:", "Walletacc", Walletacc.String())
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

	// 存入账户信息到mavl树中
	hash := SaveAccountTomavl(wallet.client, nil, accs)
	walletlog.Info("TestProcCreatNewAccount", "hash", hash)

	//测试ProcGetAccountList函数
	Accounts, err := wallet.ProcGetAccountList()
	if err == nil && Accounts != nil {
		for _, Account := range Accounts.Wallets {
			walletlog.Info("TestProcCreatNewAccount:", "Account", Account.String())
		}
	}
	//测试GetAccountByLabel函数
	for i := 0; i < total; i++ {
		label := fmt.Sprintf("hybaccount:%d", i)
		Account, err := wallet.walletStore.GetAccountByLabel(label)
		if err == nil && Account != nil {
			walletlog.Info("TestProcCreatNewAccount:", "label", label, "label->Account", Account.String())
			Privkeynyte, _ := common.FromHex(Account.GetPrivkey())
			Decrypter := CBCDecrypterPrivkey([]byte(wallet.Password), Privkeynyte)
			walletlog.Info("TestProcCreatNewAccount:", "privkey", common.ToHex(Decrypter))

		}
	}
	//测试GetAccountByAddr函数
	for i := 0; i < total; i++ {
		Account, err := wallet.walletStore.GetAccountByAddr(addres[i])
		if err == nil && Account != nil {
			walletlog.Info("TestProcCreatNewAccount:", "addr", addres[i], "Addr->Account", Account.String())

		}
	}

	walletlog.Info("TestProcCreatNewAccount end --------------------")
}

func testProcImportPrivKey(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcImportPrivKey begin --------------------")

	//先解锁
	var WalletUnLock types.WalletUnLock
	WalletUnLock.Passwd = "heyubin" //wallet.Password
	WalletUnLock.Timeout = 0
	WalletUnLock.WalletOrTicket = false
	wallet.ProcWalletUnLock(&WalletUnLock)

	var PrivKey types.ReqWalletImportPrivKey

	//生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	require.NoError(t, err)

	priv, err := cr.GenKey()
	require.NoError(t, err)

	PrivKey.Privkey = common.ToHex(priv.Bytes())
	PrivKey.Label = "ImportPrivKey-Label"
	walletlog.Info("TestProcImportPrivKey", "Privkey", PrivKey.Privkey, "Label", PrivKey.Label)

	time.Sleep(time.Second * 1)
	WalletAccount, err := wallet.ProcImportPrivKey(&PrivKey)
	require.NoError(t, err)
	walletlog.Info("TestProcImportPrivKey", "WalletAccount", WalletAccount.String())

	//import privkey="0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	PrivKey.Privkey = "0xb94ae286a508e4bb3fbbcb61997822fea6f0a534510597ef8eb60a19d6b219a0"
	PrivKey.Label = "ImportPrivKey-Label-hyb"
	walletlog.Info("TestProcImportPrivKey", "Privkey", PrivKey.Privkey, "Label", PrivKey.Label)

	time.Sleep(time.Second * 1)
	WalletAccount, err = wallet.ProcImportPrivKey(&PrivKey)
	require.NoError(t, err)
	walletlog.Info("TestProcImportPrivKey", "WalletAccount", WalletAccount.String())

	time.Sleep(time.Second * 5)

	//测试ProcGetAccountList函数
	Accounts, err := wallet.ProcGetAccountList()
	if err == nil && Accounts != nil {
		for _, Account := range Accounts.Wallets {
			walletlog.Info("TestProcImportPrivKey:", "Account", Account.String())
		}
	}

	walletlog.Info("TestProcImportPrivKey end --------------------")
}

func testProcWalletTxList(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletTxList begin --------------------")
	var TxList types.ReqWalletTransactionList
	TxList.Count = 5

	TxList.Direction = 1 //
	TxList.FromTx = []byte("")
	var FromTxstr string

	walletlog.Info("TestProcWalletTxList dir last-------")
	WalletTxDetails, err := wallet.ProcWalletTxList(&TxList)
	require.NoError(t, err)
	for _, WalletTxDetail := range WalletTxDetails.TxDetails {
		walletlog.Info("TestProcWalletTxList", "Direction", TxList.Direction, "WalletTxDetail", WalletTxDetail.String())
		FromTxstr = fmt.Sprintf("%018d", WalletTxDetail.GetHeight()*100000+WalletTxDetail.GetIndex())
	}

	TxList.Direction = 1 //
	TxList.FromTx = []byte(FromTxstr)

	walletlog.Info("TestProcWalletTxList dir next-------")
	WalletTxDetails, err = wallet.ProcWalletTxList(&TxList)
	require.NoError(t, err)
	for _, WalletTxDetail := range WalletTxDetails.TxDetails {
		walletlog.Info("TestProcWalletTxList", "Direction", TxList.Direction, "WalletTxDetail", WalletTxDetail.String())
	}

	walletlog.Info("TestProcWalletTxList dir prv------")
	//TxList.Direction = 0
	TxList.Direction = 0
	WalletTxDetails, err = wallet.ProcWalletTxList(&TxList)
	require.NoError(t, err)
	for _, WalletTxDetail := range WalletTxDetails.TxDetails {
		walletlog.Info("TestProcWalletTxList", "Direction", TxList.Direction, "WalletTxDetail", WalletTxDetail.String())
	}
	walletlog.Info("TestProcWalletTxList end --------------------")
}

//(SendToAddress *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
func testProcSendToAddress(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcSendToAddress begin --------------------")

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
	walletlog.Info("TestProcSendToAddress end --------------------")
}

//ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
func testProcWalletSetFee(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletSetFee begin --------------------")

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

	walletlog.Info("TestProcWalletSetFee end --------------------")
}

//ProcWalletSetLabel(SetLabel *types.ReqWalletSetLabel) (*types.WalletAccount, error)
func testProcWalletSetLabel(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletSetLabel begin --------------------")
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

	walletlog.Info("TestProcWalletSetLabel end --------------------")
}

//ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
func testProcMergeBalance(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcMergeBalance begin --------------------")

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
	walletlog.Info("TestProcMergeBalance end --------------------")
}

//ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
func testProcWalletSetPasswd(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletSetPasswd begin --------------------")

	var Passwd types.ReqWalletSetPasswd
	Passwd.Oldpass = "heyubin"
	Passwd.Newpass = "Newpass"

	err := wallet.ProcWalletSetPasswd(&Passwd)
	require.NoError(t, err)

	walletlog.Info("TestProcWalletSetPasswd TestProcMergeBalance  --------------------")
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
	walletlog.Info("TestProcWalletSetPasswd end --------------------")
}

//ProcWalletLock
func testProcWalletLock(t *testing.T, wallet *Wallet) {
	walletlog.Info("TestProcWalletLock begin --------------------")

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
	walletlog.Info("TestProcWalletLock end --------------------")
}
