package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
)

var (
	MinFee            int64 = 1000000
	MaxTxNumPerBlock  int64 = 100000
	MaxTxHashsPerTime int64 = 100
)

var walletlog = log.New("module", "wallet")
var ErrInputPara = errors.New("Input parameter error")
var WalletIsLocked = errors.New("WalletIsLocked")

type Wallet struct {
	qclient queue.IClient
	q       *queue.Queue
	mtx     sync.Mutex
	timeout *time.Timer

	isLocked    bool
	Password    string
	FeeAmount   int64
	EncryptFlag int64
	walletStore *WalletStore
}

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	walletlog.SetHandler(log.DiscardHandler())
	storelog.SetHandler(log.DiscardHandler())
}

func New(cfg *types.Wallet) *Wallet {

	//walletStore
	walletStoreDB := dbm.NewDB("wallet", "leveldb", cfg.DbPath)
	walletStore := NewWalletStore(walletStoreDB)
	MinFee = cfg.MinFee
	return &Wallet{
		walletStore: walletStore,
		isLocked:    false,
		FeeAmount:   walletStore.GetFeeAmount(),
		EncryptFlag: walletStore.GetEncryptionFlag(),
	}
}

func (wallet *Wallet) Close() {
	wallet.walletStore.db.Close()
	walletlog.Info("wallet module closed")
}

func (wallet *Wallet) IsLocked() bool {
	return wallet.isLocked
}

func (wallet *Wallet) SetQueue(q *queue.Queue) {
	wallet.qclient = q.GetClient()
	wallet.qclient.Sub("wallet")
	wallet.q = q
	go wallet.ProcRecvMsg()
}

func (wallet *Wallet) ProcRecvMsg() {
	for msg := range wallet.qclient.Recv() {
		walletlog.Info("wallet recv", "msg", msg)
		msgtype := msg.Ty
		switch msgtype {
		case types.EventWalletGetAccountList:
			WalletAccounts, err := wallet.ProcGetAccountList()
			if err != nil {
				walletlog.Error("ProcGetAccountList", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccountList, err))
			} else {
				walletlog.Info("process WalletAccounts OK")
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccountList, WalletAccounts))
			}
		case types.EventWalletGetTickets:
			tickets, err := wallet.GetTickets()
			if err != nil {
				walletlog.Error("GetTickets", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("consensus", types.EventWalletTickets, err))
			} else {
				walletlog.Info("process GetTickets OK")
				msg.Reply(wallet.qclient.NewMessage("consensus", types.EventWalletTickets, tickets))
			}
		case types.EventNewAccount:
			NewAccount := msg.Data.(*types.ReqNewAccount)
			WalletAccount, err := wallet.ProcCreatNewAccount(NewAccount)
			if err != nil {
				walletlog.Error("ProcCreatNewAccount", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}

		case types.EventWalletTransactionList:
			WalletTxList := msg.Data.(*types.ReqWalletTransactionList)
			TransactionDetails, err := wallet.ProcWalletTxList(WalletTxList)
			if err != nil {
				walletlog.Error("ProcWalletTxList", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventTransactionDetails, err))
			} else {
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventTransactionDetails, TransactionDetails))
			}

		case types.EventWalletImportprivkey:
			ImportPrivKey := msg.Data.(*types.ReqWalletImportPrivKey)
			WalletAccount, err := wallet.ProcImportPrivKey(ImportPrivKey)
			if err != nil {
				walletlog.Error("ProcImportPrivKey", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}

		case types.EventWalletSendToAddress:
			SendToAddress := msg.Data.(*types.ReqWalletSendToAddress)
			ReplyHash, err := wallet.ProcSendToAddress(SendToAddress)
			if err != nil {
				walletlog.Error("ProcSendToAddress", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReplyHashes, ReplyHash))
			}

		case types.EventWalletSetFee:
			WalletSetFee := msg.Data.(*types.ReqWalletSetFee)

			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletSetFee(WalletSetFee)
			if err != nil {
				walletlog.Error("ProcWalletSetFee", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletSetLabel:
			WalletSetLabel := msg.Data.(*types.ReqWalletSetLabel)
			WalletAccount, err := wallet.ProcWalletSetLabel(WalletSetLabel)

			if err != nil {
				walletlog.Error("ProcWalletSetLabel", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}

		case types.EventWalletMergeBalance:
			MergeBalance := msg.Data.(*types.ReqWalletMergeBalance)
			ReplyHashes, err := wallet.ProcMergeBalance(MergeBalance)
			if err != nil {
				walletlog.Error("ProcMergeBalance", "err", err.Error())
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReplyHashes, ReplyHashes))
			}

		case types.EventWalletSetPasswd:
			SetPasswd := msg.Data.(*types.ReqWalletSetPasswd)

			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletSetPasswd(SetPasswd)
			if err != nil {
				walletlog.Error("ProcWalletSetPasswd", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletLock:
			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletLock()
			if err != nil {
				walletlog.Error("ProcWalletLock", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletUnLock:
			WalletUnLock := msg.Data.(*types.WalletUnLock)
			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletUnLock(WalletUnLock)
			if err != nil {
				walletlog.Error("ProcWalletUnLock", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.qclient.NewMessage("rpc", types.EventReply, &reply))

		case types.EventAddBlock:
			block := msg.Data.(*types.BlockDetail)
			wallet.ProcWalletAddBlock(block)

		default:
			walletlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
	}
}

//output:
//type WalletAccounts struct {
//	Wallets []*WalletAccount
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//获取钱包的地址列表
func (wallet *Wallet) ProcGetAccountList() (*types.WalletAccounts, error) {
	if wallet.IsLocked() {
		return nil, WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("ProcGetAccountList", "GetAccountByPrefix:err", err)
		return nil, err
	}

	addrs := make([]string, len(WalletAccStores))
	for index, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			addrs[index] = AccStore.Addr
		}
		walletlog.Info("ProcGetAccountList", "all AccStore", AccStore.String())
	}
	//获取所有地址对应的账户详细信息从account模块
	accounts, err := account.LoadAccounts(wallet.q, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Info("ProcGetAccountList", "LoadAccounts:err", err)
		return nil, err
	}

	//异常打印信息
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcGetAccountList err!", "AccStores)", len(WalletAccStores), "accounts", len(accounts))
	}

	var WalletAccounts types.WalletAccounts
	WalletAccounts.Wallets = make([]*types.WalletAccount, len(WalletAccStores))

	for index, Account := range accounts {
		var WalletAccount types.WalletAccount
		//此账户还没有参与交易所在account模块没有记录
		if len(Account.Addr) == 0 {
			Account.Addr = addrs[index]
		}
		WalletAccount.Acc = Account
		WalletAccount.Label = WalletAccStores[index].GetLabel()
		WalletAccounts.Wallets[index] = &WalletAccount

		//walletlog.Info("ProcGetAccountList", "LoadAccounts:account", Account.String())
	}
	return &WalletAccounts, nil
}

//input:
//type ReqNewAccount struct {
//	Label string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//type Account struct {
//	Currency int32
//	Balance  int64
//	Frozen   int64
//	Addr     string
//创建一个新的账户
func (wallet *Wallet) ProcCreatNewAccount(Label *types.ReqNewAccount) (*types.WalletAccount, error) {
	if wallet.IsLocked() {
		return nil, WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if Label == nil || len(Label.GetLabel()) == 0 {
		walletlog.Error("ProcCreatNewAccount Label is nil")
		return nil, ErrInputPara
	}
	// 钱包已经加密需要先通过password 解锁钱包
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		err := errors.New("UnLock Wallet first!")
		return nil, err
	}
	//首先校验label是否已被使用
	WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label.GetLabel())
	if WalletAccStores != nil {
		walletlog.Error("ProcCreatNewAccount Label is exist in wallet!")
		Err := errors.New("Label Has been used in wallet!")
		return nil, Err
	}

	var Account types.Account
	var walletAccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore

	//生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		walletlog.Error("ProcCreatNewAccount", "err", err)
		return nil, err
	}

	priv, err := cr.GenKey()
	if err != nil {
		walletlog.Error("ProcCreatNewAccount GenKey", "err", err)
		return nil, err
	}

	addr := account.PubKeyToAddress(priv.PubKey().Bytes())
	Account.Addr = addr.String()
	Account.Currency = 0
	Account.Balance = 0
	Account.Frozen = 0

	walletAccount.Acc = &Account
	walletAccount.Label = Label.GetLabel()

	//使用钱包的password对私钥加密 aes cbc
	Encrypted := EncrypterPrivkey([]byte(wallet.Password), priv.Bytes())
	WalletAccStore.Privkey = common.ToHex(Encrypted)
	WalletAccStore.Label = Label.GetLabel()
	WalletAccStore.Addr = addr.String()

	//存储账户信息到wallet数据库中
	err = wallet.walletStore.SetWalletAccount(false, Account.Addr, &WalletAccStore)
	if err != nil {
		return nil, err
	}
	return &walletAccount, nil
}

//input:
//type ReqWalletTransactionList struct {
//	FromTx []byte
//	Count  int32
//output:
//type WalletTxDetails struct {
//	TxDetails []*WalletTxDetail
//type WalletTxDetail struct {
//	Tx      *Transaction
//	Receipt *ReceiptData
//	Height  int64
//	Index   int64
//获取所有钱包的交易记录
func (wallet *Wallet) ProcWalletTxList(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	if wallet.IsLocked() {
		return nil, WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if TxList == nil {
		walletlog.Error("ProcWalletTxList TxList is nil!")
		return nil, ErrInputPara
	}
	if TxList.GetDirection() != 0 && TxList.GetDirection() != 1 {
		walletlog.Error("ProcWalletTxList Direction err!")
		return nil, ErrInputPara
	}
	WalletTxDetails, err := wallet.walletStore.GetTxDetailByIter(TxList)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "GetTxDetailByIter err", err)
		return nil, err
	}
	return WalletTxDetails, nil
}

//input:
//type ReqWalletImportPrivKey struct {
//	Privkey string
//	Label   string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//导入私钥，并且同时会导入交易
func (wallet *Wallet) ProcImportPrivKey(PrivKey *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	if wallet.IsLocked() {
		return nil, WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if PrivKey == nil || len(PrivKey.GetLabel()) == 0 || len(PrivKey.GetPrivkey()) == 0 {
		walletlog.Error("ProcImportPrivKey input parameter is nil!")
		return nil, ErrInputPara
	}

	// 钱包已经加密需要先通过password 解锁钱包
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		err := errors.New("UnLock Wallet first!")
		return nil, err
	}

	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(PrivKey.GetLabel())
	if Account != nil {
		walletlog.Error("ProcImportPrivKey Label is exist in wallet!")
		Err := errors.New("Label Has been used in wallet!")
		return nil, Err
	}

	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "err", err)
		return nil, err
	}
	privkeybyte, err := common.FromHex(PrivKey.Privkey)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcImportPrivKey", "FromHex err", err)
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(privkeybyte)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "PrivKeyFromBytes err", err)
		return nil, err
	}
	addr := account.PubKeyToAddress(priv.PubKey().Bytes())

	//对私钥加密
	Encryptered := EncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//校验PrivKey对应的addr是否已经存在钱包中
	Account, err = wallet.walletStore.GetAccountByAddr(addr.String())
	if Account != nil {
		if Account.Privkey == Encrypteredstr {
			walletlog.Error("ProcImportPrivKey Privkey is exist in wallet!")
			Err := errors.New("Privkey Has exists in wallet!")
			return nil, Err
		} else {
			walletlog.Error("ProcImportPrivKey!", "Account.Privkey", Account.Privkey, "input Privkey", PrivKey.Privkey)
			Err := errors.New("ProcImportPrivKey PrivKey not equal!")
			return nil, Err
		}
	}

	var walletaccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	WalletAccStore.Privkey = Encrypteredstr //存储加密后的私钥
	WalletAccStore.Label = PrivKey.GetLabel()
	WalletAccStore.Addr = addr.String()
	//存储Addr:label+privkey+addr到数据库
	err = wallet.walletStore.SetWalletAccount(false, addr.String(), &WalletAccStore)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "SetWalletAccount err", err)
		return nil, err
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr.String()
	accounts, err := account.LoadAccounts(wallet.q, addrs)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "LoadAccounts err", err)
		return nil, err
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr.String()
	}
	walletaccount.Acc = accounts[0]
	walletaccount.Label = PrivKey.Label

	//从blockchain模块同步Account.Addr对应的所有交易详细信息
	go wallet.ReqTxDetailByAddr(addr.String())

	return &walletaccount, nil
}

//input:
//type ReqWalletSendToAddress struct {
//	From   string
//	To     string
//	Amount int64
//	Note   string
//output:
//type ReplyHash struct {
//	Hashe []byte
//发送一笔交易给对方地址，返回交易hash
func (wallet *Wallet) ProcSendToAddress(SendToAddress *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	if wallet.IsLocked() {
		return nil, WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SendToAddress == nil {
		walletlog.Error("ProcSendToAddress input para is nil")
		return nil, ErrInputPara
	}
	if len(SendToAddress.From) == 0 || len(SendToAddress.To) == 0 {
		walletlog.Error("ProcSendToAddress input para From or To is nil!")
		return nil, ErrInputPara
	}

	// 钱包已经加密需要先通过password 解锁钱包
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		err := errors.New("UnLock Wallet first!")
		return nil, err
	}

	var hash types.ReplyHash

	//获取指定地址在钱包里的账户信息
	Accountstor, err := wallet.walletStore.GetAccountByAddr(SendToAddress.GetFrom())
	if err != nil {
		walletlog.Error("ProcSendToAddress", "GetAccountByAddr err:", err)
		return nil, err
	}

	//通过password解密存储的私钥
	prikeybyte, err := common.FromHex(Accountstor.GetPrivkey())
	if err != nil || len(prikeybyte) == 0 {
		walletlog.Error("ProcSendToAddress", "FromHex err", err)
		return nil, err
	}

	privkey := DecrypterPrivkey([]byte(wallet.Password), prikeybyte)
	walletlog.Error("ProcSendToAddress", "DecrypterPrivkey privkey", common.ToHex(privkey))
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		walletlog.Error("ProcSendToAddress", "err", err)
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(privkey)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "PrivKeyFromBytes err", err)
		return nil, err
	}
	//获取from账户的余额从account模块，校验余额是否充足
	addrs := make([]string, 1)
	addrs[0] = SendToAddress.GetFrom()

	accounts, err := account.LoadAccounts(wallet.q, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcMergeBalance", "LoadAccounts err", err)
		return nil, err
	}
	Balance := accounts[0].Balance
	amount := SendToAddress.GetAmount()
	if Balance < amount+wallet.FeeAmount {
		Err := errors.New("Insufficient balance!")
		return nil, Err
	}

	addrto := SendToAddress.GetTo()
	note := SendToAddress.GetNote()

	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount, Note: note}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}

	//初始化随机数
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: addrto, Nonce: r.Int63()}
	tx.Sign(types.SECP256K1, priv)

	//发送交易信息给mempool模块
	msg := wallet.qclient.NewMessage("mempool", types.EventTx, tx)
	wallet.qclient.Send(msg, true)
	resp, err := wallet.qclient.Wait(msg)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "Send err", err)
		return nil, err
	}
	reply := resp.GetData().(*types.Reply)
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}

	hash.Hash = tx.Hash()
	return &hash, nil
}

//type ReqWalletSetFee struct {
//	Amount int64
//设置钱包默认的手续费
func (wallet *Wallet) ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
	if wallet.IsLocked() {
		return WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if WalletSetFee.Amount < MinFee {
		walletlog.Error("ProcWalletSetFee err!", "Amount", WalletSetFee.Amount, "MinFee", MinFee)
		return ErrInputPara
	}
	err := wallet.walletStore.SetFeeAmount(WalletSetFee.Amount)
	if err == nil {
		walletlog.Info("ProcWalletSetFee success!")
		wallet.FeeAmount = WalletSetFee.Amount
	}
	return err
}

//input:
//type ReqWalletSetLabel struct {
//	Addr  string
//	Label string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//设置某个账户的标签
func (wallet *Wallet) ProcWalletSetLabel(SetLabel *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	if wallet.IsLocked() {
		return nil, WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SetLabel == nil || len(SetLabel.Addr) == 0 || len(SetLabel.Label) == 0 {
		walletlog.Error("ProcWalletSetLabel input parameter is nil!")
		return nil, ErrInputPara
	}
	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(SetLabel.GetLabel())
	if Account != nil {
		walletlog.Error("ProcWalletSetLabel Label is exist in wallet!")
		Err := errors.New("Label Has been used in wallet!")
		return nil, Err
	}
	//获取地址对应的账户信息从钱包中,然后修改label
	Account, err = wallet.walletStore.GetAccountByAddr(SetLabel.Addr)
	if err == nil && Account != nil {
		oldLabel := Account.Label
		Account.Label = SetLabel.GetLabel()
		err := wallet.walletStore.SetWalletAccount(true, SetLabel.Addr, Account)
		if err == nil {
			//新的label设置成功之后需要删除旧的label在db的数据
			wallet.walletStore.DelAccountByLabel(oldLabel)

			//获取地址对应的账户详细信息从account模块
			addrs := make([]string, 1)
			addrs[0] = SetLabel.Addr
			accounts, err := account.LoadAccounts(wallet.q, addrs)
			if err != nil || len(accounts) == 0 {
				walletlog.Error("ProcWalletSetLabel", "LoadAccounts err", err)
				return nil, err
			}
			var walletAccount types.WalletAccount
			walletAccount.Acc = accounts[0]
			walletAccount.Label = SetLabel.GetLabel()
			return &walletAccount, err
		}
	}
	return nil, err
}

//input:
//type ReqWalletMergeBalance struct {
//	To string
//output:
//type ReplyHashes struct {
//	Hashes [][]byte
//合并所有的balance 到一个地址
func (wallet *Wallet) ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	if wallet.IsLocked() {
		return nil, WalletIsLocked
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if len(MergeBalance.GetTo()) == 0 {
		walletlog.Error("ProcMergeBalance input para is nil!")
		return nil, ErrInputPara
	}

	// 钱包已经加密需要先通过password 解锁钱包
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		err := errors.New("UnLock Wallet first!")
		return nil, err
	}

	//获取钱包上的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Error("ProcMergeBalance", "GetAccountByPrefix err", err)
		return nil, err
	}

	addrs := make([]string, len(WalletAccStores))
	for index, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			addrs[index] = AccStore.Addr
		}
	}
	//获取所有地址对应的账户信息从account模块
	accounts, err := account.LoadAccounts(wallet.q, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcMergeBalance", "LoadAccounts err", err)
		return nil, err
	}

	//异常信息记录
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcMergeBalance", "AccStores", len(WalletAccStores), "accounts", len(accounts))
	}
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err)
		return nil, err
	}

	addrto := MergeBalance.GetTo()
	note := "MergeBalance"

	var ReplyHashes types.ReplyHashes
	//ReplyHashes.Hashes = make([][]byte, len(accounts))

	for index, Account := range accounts {
		Privkey := WalletAccStores[index].Privkey
		//解密存储的私钥
		prikeybyte, err := common.FromHex(Privkey)
		if err != nil || len(prikeybyte) == 0 {
			walletlog.Error("ProcMergeBalance", "FromHex err", err)
			return nil, err
		}

		privkey := DecrypterPrivkey([]byte(wallet.Password), prikeybyte)
		priv, err := cr.PrivKeyFromBytes(privkey)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "PrivKeyFromBytes err", err, "index", index)
			//ReplyHashes.Hashes[index] = common.Hash{}.Bytes()
			continue
		}
		//过滤掉to地址
		if Account.Addr == addrto {
			continue
		}
		//获取账户的余额，过滤掉余额不足的地址
		amount := Account.GetBalance()
		if amount < wallet.FeeAmount {
			continue
		}
		amount = amount - wallet.FeeAmount
		v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount, Note: note}}
		transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
		//初始化随机数
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: addrto, Nonce: r.Int63()}
		tx.Sign(types.SECP256K1, priv)

		//发送交易信息给mempool模块
		msg := wallet.qclient.NewMessage("mempool", types.EventTx, tx)
		wallet.qclient.Send(msg, true)
		_, err = wallet.qclient.Wait(msg)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "Send tx err", err, "index", index)
			//ReplyHashes.Hashes[index] = common.Hash{}.Bytes()
			continue
		}

		ReplyHashes.Hashes = append(ReplyHashes.Hashes, tx.Hash())
	}
	return &ReplyHashes, nil
}

//input:
//type ReqWalletSetPasswd struct {
//	Oldpass string
//	Newpass string
//设置或者修改密码
func (wallet *Wallet) ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	// 钱包已经加密需要验证oldpass的正确性
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(Passwd.Oldpass)
		if !isok {
			walletlog.Error("ProcWalletSetPasswd Verify Oldpasswd fail!")
			err := errors.New("Verify Oldpasswd fail!")
			return err
		}
	}

	if len(wallet.Password) != 0 && Passwd.Oldpass != wallet.Password {
		walletlog.Error("ProcWalletSetPasswd Oldpass err!")
		err := errors.New("ProcWalletSetPasswd Oldpass err!")
		return err
	}

	//使用新的密码生成passwdhash用于下次密码的验证
	err := wallet.walletStore.SetPasswordHash(Passwd.Newpass)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetPasswordHash err", err)
		return err
	}
	//设置钱包加密标志位
	err = wallet.walletStore.SetEncryptionFlag()
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetEncryptionFlag err", err)
		return err
	}
	//对所有存储的私钥重新使用新的密码加密
	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("ProcWalletSetPasswd", "GetAccountByPrefix:err", err)
		//return err
	}
	if WalletAccStores != nil {
		for _, AccStore := range WalletAccStores {

			//使用old Password解密存储的私钥
			storekey, err := common.FromHex(AccStore.GetPrivkey())
			if err != nil || len(storekey) == 0 {
				walletlog.Info("ProcWalletSetPasswd", "addr", AccStore.Addr, "FromHex err", err)
				continue
			}
			Decrypter := DecrypterPrivkey([]byte(wallet.Password), storekey)

			//使用新的密码重新加密私钥
			Encrypter := EncrypterPrivkey([]byte(Passwd.Newpass), Decrypter)
			AccStore.Privkey = common.ToHex(Encrypter)
			err = wallet.walletStore.SetWalletAccount(true, AccStore.Addr, AccStore)
			if err != nil {
				walletlog.Info("ProcWalletSetPasswd", "addr", AccStore.Addr, "SetWalletAccount err", err)
			}
		}
	}
	wallet.Password = Passwd.Newpass
	return nil
}

//锁定钱包
func (wallet *Wallet) ProcWalletLock() error {
	wallet.isLocked = true
	return nil
}

//input:
//type WalletUnLock struct {
//	Passwd  string
//	Timeout int64
//解锁钱包Timeout时间，超时后继续锁住
func (wallet *Wallet) ProcWalletUnLock(WalletUnLock *types.WalletUnLock) error {
	// 钱包已经加密需要验证passwd的正确性
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(WalletUnLock.Passwd)
		if !isok {
			walletlog.Error("ProcWalletUnLock Verify Oldpasswd fail!")
			err := errors.New("Verify Oldpasswd fail!")
			return err
		}
	}

	//内存中已经记录password时的校验
	if len(wallet.Password) != 0 && WalletUnLock.Passwd != wallet.Password {
		err := errors.New("Input Password error!")
		return err
	}
	//本钱包没有设置密码加密过,只需要解锁不需要记录解锁密码
	if len(wallet.Password) != 0 || wallet.EncryptFlag != 0 {
		wallet.Password = WalletUnLock.Passwd
	}
	wallet.isLocked = false
	if WalletUnLock.Timeout != 0 {
		wallet.resetTimeout(WalletUnLock.Timeout)
	}
	return nil

}

func (wallet *Wallet) resetTimeout(Timeout int64) {
	if wallet.timeout == nil {
		wallet.timeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
			wallet.isLocked = true
		})
	} else {
		wallet.timeout.Reset(time.Second * time.Duration(Timeout))
	}
}

//wallet模块收到blockchain广播的addblock消息，需要解析钱包相关的tx并存储到db中
func (wallet *Wallet) ProcWalletAddBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletAddBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletAddBlock", "height", block.GetBlock().GetHeight())
	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.NewBatch(true)

	for index := 0; index < txlen; index++ {
		if "coins" == string(block.Block.Txs[index].Execer) {
			blockheight := block.Block.Height*MaxTxNumPerBlock + int64(index)
			heightstr := fmt.Sprintf("%018d", blockheight)

			var txdetail types.WalletTxDetail
			txdetail.Tx = block.Block.Txs[index]
			txdetail.Height = block.Block.Height
			txdetail.Index = int64(index)
			txdetail.Receipt = block.Receipts[index]
			txdetail.Blocktime = block.Block.BlockTime

			//获取Amount
			var action types.CoinsAction
			err := types.Decode(txdetail.Tx.GetPayload(), &action)
			if err != nil {
				walletlog.Error("ProcWalletAddBlock Decode err!", "Height", txdetail.Height, "txindex", index, "err", err)
				continue
			}
			if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
				transfer := action.GetTransfer()
				txdetail.Amount = transfer.Amount
			}
			//获取from地址
			pubkey := block.Block.Txs[index].Signature.GetPubkey()
			addr := account.PubKeyToAddress(pubkey)
			txdetail.Fromaddr = addr.String()

			txdetailbyte, err := proto.Marshal(&txdetail)
			if err != nil {
				storelog.Error("ProcWalletAddBlock Marshal txdetail err", "Height", block.Block.Height, "index", index)
				continue
			}

			//from addr
			fromaddress := addr.String()
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				newbatch.Set([]byte(calcTxKey(heightstr)), txdetailbyte)
				walletlog.Debug("ProcWalletAddBlock", "fromaddress", fromaddress, "heightstr", heightstr)
				continue
			}
			//toaddr
			toaddr := block.Block.Txs[index].GetTo()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				newbatch.Set([]byte(calcTxKey(heightstr)), txdetailbyte)
				walletlog.Debug("ProcWalletAddBlock", "toaddr", toaddr, "heightstr", heightstr)
			}
		}
	}
	newbatch.Write()
}

//地址对应的账户是否属于本钱包
func (wallet *Wallet) AddrInWallet(addr string) bool {
	if len(addr) == 0 {
		return false
	}
	account, err := wallet.walletStore.GetAccountByAddr(addr)
	if err == nil && account != nil {
		return true
	}
	return false
}
func (wallet *Wallet) GetTxDetailByHashs(ReqHashes *types.ReqHashes) {
	//通过txhashs获取对应的txdetail
	msg := wallet.qclient.NewMessage("blockchain", types.EventGetTransactionByHash, ReqHashes)
	wallet.qclient.Send(msg, true)
	resp, err := wallet.qclient.Wait(msg)
	if err != nil {
		walletlog.Error("ReqTxInfosByAddr EventGetTransactionByHash", "err", err)
		return
	}
	TxDetails := resp.GetData().(*types.TransactionDetails)
	if TxDetails == nil {
		walletlog.Info("ReqTxInfosByAddr TransactionDetails is nil")
		return
	}

	//批量存储地址对应的所有交易的详细信息到wallet db中
	newbatch := wallet.walletStore.NewBatch(true)
	for _, txdetal := range TxDetails.Txs {
		height := txdetal.GetHeight()
		txindex := txdetal.GetIndex()

		blockheight := height*MaxTxNumPerBlock + int64(txindex)
		heightstr := fmt.Sprintf("%018d", blockheight)
		var txdetail types.WalletTxDetail
		txdetail.Tx = txdetal.GetTx()
		txdetail.Height = txdetal.GetHeight()
		txdetail.Index = txdetal.GetIndex()
		txdetail.Receipt = txdetal.GetReceipt()
		txdetail.Blocktime = txdetal.GetBlocktime()
		txdetail.Amount = txdetal.GetAmount()
		txdetail.Fromaddr = txdetal.GetFromaddr()

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			storelog.Error("ReqTxDetailByAddr Marshal txdetail err", "Height", height, "index", txindex)
			return
		}
		newbatch.Set([]byte(calcTxKey(heightstr)), txdetailbyte)
		walletlog.Debug("ReqTxInfosByAddr", "heightstr", heightstr, "txdetail", txdetail.String())
	}
	newbatch.Write()
}

//从blockchain模块同步addr参与的所有交易详细信息
func (wallet *Wallet) ReqTxDetailByAddr(addr string) {
	if len(addr) == 0 {
		walletlog.Error("ReqTxInfosByAddr input addr is nil!")
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
		msg := wallet.qclient.NewMessage("blockchain", types.EventGetTransactionByAddr, &ReqAddr)
		wallet.qclient.Send(msg, true)
		resp, err := wallet.qclient.Wait(msg)
		if err != nil {
			walletlog.Error("ReqTxInfosByAddr EventGetTransactionByAddr", "err", err, "addr", addr)
			return
		}

		ReplyTxInfos := resp.GetData().(*types.ReplyTxInfos)
		if ReplyTxInfos == nil {
			walletlog.Info("ReqTxInfosByAddr ReplyTxInfos is nil")
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
		wallet.GetTxDetailByHashs(&ReqHashes)
		if txcount < int(MaxTxHashsPerTime) {
			return
		}
	}
}

//使用钱包的password对私钥进行aes cbc加密,返回加密后的privkey
func EncrypterPrivkey(password []byte, privkey []byte) []byte {
	key := make([]byte, 32)
	Encrypted := make([]byte, len(privkey))
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, _ := aes.NewCipher(key)
	iv := key[:block.BlockSize()]

	//walletlog.Info("EncrypterPrivkey", "password", string(key), "Privkey", common.ToHex(privkey))

	encrypter := cipher.NewCBCEncrypter(block, iv)
	encrypter.CryptBlocks(Encrypted, privkey)

	//walletlog.Info("EncrypterPrivkey", "Encrypted", common.ToHex(Encrypted))
	return Encrypted
}

//使用钱包的password对私钥进行aes cbc解密,返回解密后的privkey
func DecrypterPrivkey(password []byte, privkey []byte) []byte {
	key := make([]byte, 32)
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, _ := aes.NewCipher(key)
	iv := key[:block.BlockSize()]
	decryptered := make([]byte, len(privkey))
	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypter.CryptBlocks(decryptered, privkey)
	//walletlog.Info("DecrypterPrivkey", "password", string(key), "Encrypted", common.ToHex(privkey), "decryptered", common.ToHex(decryptered))
	return decryptered
}

func (wallet *Wallet) GetTickets() ([]*types.Ticket, error) {
	accounts, err := wallet.ProcGetAccountList()
	if err != nil {
		return nil, err
	}
	//循环遍历所有的账户-->保证钱包已经解锁
	var tickets []*types.Ticket
	for _, account := range accounts.Wallets {
		t, err := loadTicket(account.Acc.Addr)
		if err != nil {
			return nil, err
		}
		if t != nil {
			tickets = append(tickets, t...)
		}
	}
	if len(tickets) == 0 {
		return nil, types.ErrNoTicket
	}
	return tickets, nil
}

func loadTicket(addr string) ([]*types.Ticket, error) {
	return nil, nil
}
