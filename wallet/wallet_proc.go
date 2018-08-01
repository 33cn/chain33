package wallet

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"bytes"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/wallet/bipwallet"
)

//input:
//type ReqSignRawTx struct {
//	Addr    string
//	Privkey string
//	TxHex   string
//	Expire  string
//}
//output:
//string
//签名交易
func (wallet *Wallet) ProcSignRawTx(unsigned *types.ReqSignRawTx) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	index := unsigned.Index

	if ok, err := wallet.IsRescanUtxosFlagScaning(); ok {
		return "", err
	}

	var key crypto.PrivKey
	if unsigned.GetAddr() != "" {
		ok, err := wallet.CheckWalletStatus()
		if !ok {
			return "", err
		}
		key, err = wallet.getPrivKeyByAddr(unsigned.GetAddr())
		if err != nil {
			return "", err
		}
	} else if unsigned.GetPrivkey() != "" {
		keyByte, err := common.FromHex(unsigned.GetPrivkey())
		if err != nil || len(keyByte) == 0 {
			return "", err
		}
		cr, err := crypto.New(types.GetSignatureTypeName(SignType))
		if err != nil {
			return "", err
		}
		key, err = cr.PrivKeyFromBytes(keyByte)
		if err != nil {
			return "", err
		}
	} else {
		return "", types.ErrNoPrivKeyOrAddr
	}

	var tx types.Transaction
	txByteData, err := common.FromHex(unsigned.GetTxHex())
	if err != nil {
		return "", err
	}
	err = types.Decode(txByteData, &tx)
	if err != nil {
		return "", err
	}
	expire, err := time.ParseDuration(unsigned.GetExpire())
	if err != nil {
		return "", err
	}
	tx.SetExpire(expire)
	if bytes.Equal(tx.Execer, types.ExecerPrivacy) {
		return wallet.signTxWithPrivacy(key, unsigned)
	}
	group, err := tx.GetTxGroup()
	if err != nil {
		return "", err
	}
	if group == nil {
		tx.Sign(int32(SignType), key)
		txHex := types.Encode(&tx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	if int(index) > len(group.GetTxs()) {
		return "", types.ErrIndex
	}
	if index <= 0 {
		for i := range group.Txs {
			group.SignN(i, int32(SignType), key)
		}
		grouptx := group.Tx()
		txHex := types.Encode(grouptx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	index--
	group.SignN(int(index), int32(SignType), key)
	grouptx := group.Tx()
	txHex := types.Encode(grouptx)
	signedTx := hex.EncodeToString(txHex)
	return signedTx, nil
}

//output:
//type WalletAccounts struct {
//	Wallets []*WalletAccount
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//获取钱包的地址列表
func (wallet *Wallet) ProcGetAccountList() (*types.WalletAccounts, error) {
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
	}
	//获取所有地址对应的账户详细信息从account模块
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcGetAccountList", "LoadAccounts:err", err)
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
func (wallet *Wallet) ProcCreateNewAccount(Label *types.ReqNewAccount) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if Label == nil || len(Label.GetLabel()) == 0 {
		walletlog.Error("ProcCreateNewAccount Label is nil")
		return nil, types.ErrInputPara
	}

	//首先校验label是否已被使用
	WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label.GetLabel())
	if WalletAccStores != nil {
		walletlog.Error("ProcCreateNewAccount Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}

	var Account types.Account
	var walletAccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	var cointype uint32
	if SignType == 1 {
		cointype = bipwallet.TypeBty
	} else if SignType == 2 {
		cointype = bipwallet.TypeYcc
	} else {
		cointype = bipwallet.TypeBty
	}

	//通过seed获取私钥, 首先通过钱包密码解锁seed然后通过seed生成私钥
	seed, err := wallet.getSeed(wallet.Password)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "getSeed err", err)
		return nil, err
	}
	privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.db, seed)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "GetPrivkeyBySeed err", err)
		return nil, err
	}
	privkeybyte, err := common.FromHex(privkeyhex)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcCreateNewAccount", "FromHex err", err)
		return nil, err
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, privkeybyte)
	if err != nil {
		seedlog.Error("ProcCreateNewAccount PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}
	addr, err := bipwallet.PubToAddress(cointype, pub)
	if err != nil {
		seedlog.Error("ProcCreateNewAccount PubToAddress", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	Account.Addr = addr
	Account.Currency = 0
	Account.Balance = 0
	Account.Frozen = 0

	walletAccount.Acc = &Account
	walletAccount.Label = Label.GetLabel()

	//使用钱包的password对私钥加密 aes cbc
	Encrypted := CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	WalletAccStore.Privkey = common.ToHex(Encrypted)
	WalletAccStore.Label = Label.GetLabel()
	WalletAccStore.Addr = addr

	//存储账户信息到wallet数据库中
	err = wallet.walletStore.SetWalletAccount(false, Account.Addr, &WalletAccStore)
	if err != nil {
		return nil, err
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "LoadAccounts err", err)
		return nil, err
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr
	}
	walletAccount.Acc = accounts[0]

	//从blockchain模块同步Account.Addr对应的所有交易详细信息
	wallet.wg.Add(1)
	go wallet.ReqTxDetailByAddr(addr)

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
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if TxList == nil {
		walletlog.Error("ProcWalletTxList TxList is nil!")
		return nil, types.ErrInputPara
	}
	if TxList.GetDirection() != 0 && TxList.GetDirection() != 1 {
		walletlog.Error("ProcWalletTxList Direction err!")
		return nil, types.ErrInputPara
	}
	WalletTxDetails, err := wallet.walletStore.getTxDetailByIter(TxList)
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
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if PrivKey == nil || len(PrivKey.GetLabel()) == 0 || len(PrivKey.GetPrivkey()) == 0 {
		walletlog.Error("ProcImportPrivKey input parameter is nil!")
		return nil, types.ErrInputPara
	}

	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(PrivKey.GetLabel())
	if Account != nil {
		walletlog.Error("ProcImportPrivKey Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
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
		walletlog.Error("ProcImportPrivKey", "FromHex err", err)
		return nil, types.ErrFromHex
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, privkeybyte)
	if err != nil {
		seedlog.Error("ProcImportPrivKey PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}
	addr, err := bipwallet.PubToAddress(cointype, pub)
	if err != nil {
		seedlog.Error("ProcImportPrivKey PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	//对私钥加密
	Encryptered := CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//校验PrivKey对应的addr是否已经存在钱包中
	Account, err = wallet.walletStore.GetAccountByAddr(addr)
	if Account != nil {
		if Account.Privkey == Encrypteredstr {
			walletlog.Error("ProcImportPrivKey Privkey is exist in wallet!")
			return nil, types.ErrPrivkeyExist
		} else {
			walletlog.Error("ProcImportPrivKey!", "Account.Privkey", Account.Privkey, "input Privkey", PrivKey.Privkey)
			return nil, types.ErrPrivkey
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
		walletlog.Error("ProcImportPrivKey", "SetWalletAccount err", err)
		return nil, err
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "LoadAccounts err", err)
		return nil, err
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr
	}
	walletaccount.Acc = accounts[0]
	walletaccount.Label = PrivKey.Label

	//从blockchain模块同步Account.Addr对应的所有交易详细信息
	wallet.wg.Add(1)
	go wallet.ReqTxDetailByAddr(addr)

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
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SendToAddress == nil {
		walletlog.Error("ProcSendToAddress input para is nil")
		return nil, types.ErrInputPara
	}
	if len(SendToAddress.From) == 0 || len(SendToAddress.To) == 0 {
		walletlog.Error("ProcSendToAddress input para From or To is nil!")
		return nil, types.ErrInputPara
	}

	ok, err := wallet.IsTransfer(SendToAddress.GetTo())
	if !ok {
		return nil, err
	}

	//获取from账户的余额从account模块，校验余额是否充足
	addrs := make([]string, 1)
	addrs[0] = SendToAddress.GetFrom()
	var accounts []*types.Account
	var tokenAccounts []*types.Account
	accounts, err = accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcSendToAddress", "LoadAccounts err", err)
		return nil, err
	}
	Balance := accounts[0].Balance
	amount := SendToAddress.GetAmount()
	if !SendToAddress.IsToken {
		if Balance < amount+wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}
	} else {
		//如果是token转账，一方面需要保证coin的余额满足fee，另一方面则需要保证token的余额满足转账操作
		if Balance < wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}

		if nil == accTokenMap[SendToAddress.TokenSymbol] {
			tokenAccDB, err := account.NewAccountDB("token", SendToAddress.TokenSymbol, nil)
			if err != nil {
				return nil, err
			}
			accTokenMap[SendToAddress.TokenSymbol] = tokenAccDB
		}
		tokenAccDB := accTokenMap[SendToAddress.TokenSymbol]
		tokenAccounts, err = tokenAccDB.LoadAccounts(wallet.api, addrs)
		if err != nil || len(tokenAccounts) == 0 {
			walletlog.Error("ProcSendToAddress", "Load Token Accounts err", err)
			return nil, err
		}
		tokenBalance := tokenAccounts[0].Balance
		if tokenBalance < amount {
			return nil, types.ErrInsufficientTokenBal
		}
	}
	addrto := SendToAddress.GetTo()
	note := SendToAddress.GetNote()
	priv, err := wallet.getPrivKeyByAddr(addrs[0])
	if err != nil {
		return nil, err
	}
	return wallet.sendToAddress(priv, addrto, amount, note, SendToAddress.IsToken, SendToAddress.TokenSymbol)
}

//type ReqWalletSetFee struct {
//	Amount int64
//设置钱包默认的手续费
func (wallet *Wallet) ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
	if WalletSetFee.Amount < minFee {
		walletlog.Error("ProcWalletSetFee err!", "Amount", WalletSetFee.Amount, "MinFee", minFee)
		return types.ErrInputPara
	}
	err := wallet.walletStore.SetFeeAmount(WalletSetFee.Amount)
	if err == nil {
		walletlog.Debug("ProcWalletSetFee success!")
		wallet.mtx.Lock()
		wallet.FeeAmount = WalletSetFee.Amount
		wallet.mtx.Unlock()
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
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SetLabel == nil || len(SetLabel.Addr) == 0 || len(SetLabel.Label) == 0 {
		walletlog.Error("ProcWalletSetLabel input parameter is nil!")
		return nil, types.ErrInputPara
	}
	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(SetLabel.GetLabel())
	if Account != nil {
		walletlog.Error("ProcWalletSetLabel Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
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
			accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
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
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if len(MergeBalance.GetTo()) == 0 {
		walletlog.Error("ProcMergeBalance input para is nil!")
		return nil, types.ErrInputPara
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
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcMergeBalance", "LoadAccounts err", err)
		return nil, err
	}

	//异常信息记录
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcMergeBalance", "AccStores", len(WalletAccStores), "accounts", len(accounts))
	}
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err)
		return nil, err
	}

	addrto := MergeBalance.GetTo()
	note := "MergeBalance"

	var ReplyHashes types.ReplyHashes

	for index, Account := range accounts {
		Privkey := WalletAccStores[index].Privkey
		//解密存储的私钥
		prikeybyte, err := common.FromHex(Privkey)
		if err != nil || len(prikeybyte) == 0 {
			walletlog.Error("ProcMergeBalance", "FromHex err", err, "index", index)
			continue
		}

		privkey := CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
		priv, err := cr.PrivKeyFromBytes(privkey)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "PrivKeyFromBytes err", err, "index", index)
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
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: addrto, Nonce: wallet.random.Int63()}
		tx.SetExpire(time.Second * 120)
		tx.Sign(int32(SignType), priv)
		//walletlog.Info("ProcMergeBalance", "tx.Nonce", tx.Nonce, "tx", tx, "index", index)

		//发送交易信息给mempool模块
		msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "Send tx err", err, "index", index)
			continue
		}
		//如果交易在mempool校验失败，不记录此交易
		reply := resp.GetData().(*types.Reply)
		if !reply.GetIsOk() {
			walletlog.Error("ProcMergeBalance", "Send tx err", string(reply.GetMsg()), "index", index)
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

	isok, err := wallet.CheckWalletStatus()
	if !isok && err == types.ErrSaveSeedFirst {
		return err
	}
	//保存钱包的锁状态，需要暂时的解锁，函数退出时再恢复回去
	tempislock := atomic.LoadInt32(&wallet.isWalletLocked)
	//wallet.isWalletLocked = false
	atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 1, 0)

	defer func() {
		//wallet.isWalletLocked = tempislock
		atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, tempislock)
	}()

	// 钱包已经加密需要验证oldpass的正确性
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(Passwd.OldPass)
		if !isok {
			walletlog.Error("ProcWalletSetPasswd Verify Oldpasswd fail!")
			return types.ErrVerifyOldpasswdFail
		}
	}

	if len(wallet.Password) != 0 && Passwd.OldPass != wallet.Password {
		walletlog.Error("ProcWalletSetPasswd Oldpass err!")
		return types.ErrVerifyOldpasswdFail
	}

	//使用新的密码生成passwdhash用于下次密码的验证
	newBatch := wallet.walletStore.db.NewBatch(true)
	err = wallet.walletStore.SetPasswordHash(Passwd.NewPass, newBatch)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetPasswordHash err", err)
		return err
	}
	//设置钱包加密标志位
	err = wallet.walletStore.SetEncryptionFlag(newBatch)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetEncryptionFlag err", err)
		return err
	}
	//使用old密码解密seed然后用新的钱包密码重新加密seed
	seed, err := wallet.getSeed(Passwd.OldPass)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "getSeed err", err)
		return err
	}
	ok, err := SaveSeedInBatch(wallet.walletStore.db, seed, Passwd.NewPass, newBatch)
	if !ok {
		walletlog.Error("ProcWalletSetPasswd", "SaveSeed err", err)
		return err
	}

	//对所有存储的私钥重新使用新的密码加密,通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Error("ProcWalletSetPasswd", "GetAccountByPrefix:err", err)
	}

	for _, AccStore := range WalletAccStores {
		//使用old Password解密存储的私钥
		storekey, err := common.FromHex(AccStore.GetPrivkey())
		if err != nil || len(storekey) == 0 {
			walletlog.Info("ProcWalletSetPasswd", "addr", AccStore.Addr, "FromHex err", err)
			continue
		}
		Decrypter := CBCDecrypterPrivkey([]byte(Passwd.OldPass), storekey)

		//使用新的密码重新加密私钥
		Encrypter := CBCEncrypterPrivkey([]byte(Passwd.NewPass), Decrypter)
		AccStore.Privkey = common.ToHex(Encrypter)
		err = wallet.walletStore.SetWalletAccountInBatch(true, AccStore.Addr, AccStore, newBatch)
		if err != nil {
			walletlog.Info("ProcWalletSetPasswd", "addr", AccStore.Addr, "SetWalletAccount err", err)
		}
	}

	newBatch.Write()
	wallet.Password = Passwd.NewPass
	return nil
}

//锁定钱包
func (wallet *Wallet) ProcWalletLock() error {
	//判断钱包是否已保存seed
	has, _ := HasSeed(wallet.walletStore.db)
	if !has {
		return types.ErrSaveSeedFirst
	}

	atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
	atomic.CompareAndSwapInt32(&wallet.isTicketLocked, 0, 1)
	return nil
}

//input:
//type WalletUnLock struct {
//	Passwd  string
//	Timeout int64
//解锁钱包Timeout时间，超时后继续锁住
func (wallet *Wallet) ProcWalletUnLock(WalletUnLock *types.WalletUnLock) error {
	//判断钱包是否已保存seed
	has, _ := HasSeed(wallet.walletStore.db)
	if !has {
		return types.ErrSaveSeedFirst
	}
	// 钱包已经加密需要验证passwd的正确性
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(WalletUnLock.Passwd)
		if !isok {
			walletlog.Error("ProcWalletUnLock Verify Oldpasswd fail!")
			return types.ErrVerifyOldpasswdFail
		}
	}

	//内存中已经记录password时的校验
	if len(wallet.Password) != 0 && WalletUnLock.Passwd != wallet.Password {
		return types.ErrInputPassword
	}
	//本钱包没有设置密码加密过,只需要解锁不需要记录解锁密码
	wallet.Password = WalletUnLock.Passwd

	//walletlog.Error("ProcWalletUnLock !", "WalletOrTicket", WalletUnLock.WalletOrTicket)

	//只解锁挖矿转账
	if WalletUnLock.WalletOrTicket {
		//wallet.isTicketLocked = false
		atomic.CompareAndSwapInt32(&wallet.isTicketLocked, 1, 0)
	} else {
		//wallet.isWalletLocked = false
		atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 1, 0)
	}
	if WalletUnLock.Timeout != 0 {
		wallet.resetTimeout(WalletUnLock.WalletOrTicket, WalletUnLock.Timeout)
	}
	return nil

}

//解锁超时处理，需要区分整个钱包的解锁或者只挖矿的解锁
func (wallet *Wallet) resetTimeout(IsTicket bool, Timeout int64) {
	//只挖矿的解锁超时
	if IsTicket {
		if wallet.minertimeout == nil {
			wallet.minertimeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
				//wallet.isTicketLocked = true
				atomic.CompareAndSwapInt32(&wallet.isTicketLocked, 0, 1)
			})
		} else {
			wallet.minertimeout.Reset(time.Second * time.Duration(Timeout))
		}
	} else { //整个钱包的解锁超时
		if wallet.timeout == nil {
			wallet.timeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
				//wallet.isWalletLocked = true
				atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
			})
		} else {
			wallet.timeout.Reset(time.Second * time.Duration(Timeout))
		}
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
	needflush := false
	for index := 0; index < txlen; index++ {
		tx := block.Block.Txs[index]

		//check whether the privacy tx belong to current wallet
		if types.PrivacyX != string(tx.Execer) {
			//获取from地址
			pubkey := block.Block.Txs[index].Signature.GetPubkey()
			addr := address.PubKeyToAddress(pubkey)
			param := &buildStoreWalletTxDetailParam{
				block:      block,
				tx:         tx,
				index:      index,
				newbatch:   newbatch,
				isprivacy:  false,
				addDelType: AddTx,
				utxos:      nil,
			}

			//from addr
			fromaddress := addr.String()
			param.senderRecver = fromaddress
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				param.sendRecvFlag = sendTx
				wallet.buildAndStoreWalletTxDetail(param)
				walletlog.Debug("ProcWalletAddBlock", "fromaddress", fromaddress)
				continue
			}
			//toaddr
			toaddr := tx.GetTo()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				param.sendRecvFlag = recvTx
				wallet.buildAndStoreWalletTxDetail(param)
				walletlog.Debug("ProcWalletAddBlock", "toaddr", toaddr)
				continue
			}

			if "ticket" == string(block.Block.Txs[index].Execer) {
				tx := block.Block.Txs[index]
				receipt := block.Receipts[index]
				if wallet.needFlushTicket(tx, receipt) {
					needflush = true
				}
			}
		} else {
			//TODO:当前不会出现扣掉交易费，而实际的交易不执行的情况，因为如果交易费得不到保障，交易将不被执行
			//确认隐私交易是否是ExecOk
			wallet.AddDelPrivacyTxsFromBlock(tx, int32(index), block, newbatch, AddTx)
			//wallet.onAddPrivacyTxFromBlock(tx, int32(index), block, newbatch)
		}
	}
	err := newbatch.Write()
	if err != nil {
		walletlog.Error("ProcWalletAddBlock newbatch.Write", "err", err)
		atomic.CompareAndSwapInt32(&wallet.fatalFailureFlag, 0, 1)
	}
	if needflush {
		//wallet.flushTicket()
	}
}

//
type buildStoreWalletTxDetailParam struct {
	tokenname    string
	block        *types.BlockDetail
	tx           *types.Transaction
	index        int
	newbatch     dbm.Batch
	senderRecver string
	isprivacy    bool
	addDelType   int32
	sendRecvFlag int32
	utxos        []*types.UTXO
}

func (wallet *Wallet) buildAndStoreWalletTxDetail(param *buildStoreWalletTxDetailParam) {
	blockheight := param.block.Block.Height*maxTxNumPerBlock + int64(param.index)
	heightstr := fmt.Sprintf("%018d", blockheight)
	walletlog.Debug("buildAndStoreWalletTxDetail", "heightstr", heightstr, "addDelType", param.addDelType)
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
			storelog.Error("ProcWalletAddBlock Marshal txdetail err", "Height", param.block.Block.Height, "index", param.index)
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

//wallet模块收到blockchain广播的delblock消息，需要解析钱包相关的tx并存db中删除
func (wallet *Wallet) ProcWalletDelBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletDelBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletDelBlock", "height", block.GetBlock().GetHeight())

	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.NewBatch(true)
	needflush := false
	for index := txlen - 1; index >= 0; index-- {
		blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)
		tx := block.Block.Txs[index]
		if "ticket" == string(tx.Execer) {
			receipt := block.Receipts[index]
			if wallet.needFlushTicket(tx, receipt) {
				needflush = true
			}
		}

		if types.PrivacyX != string(tx.Execer) {
			//获取from地址
			pubkey := tx.Signature.GetPubkey()
			addr := address.PubKeyToAddress(pubkey)
			fromaddress := addr.String()
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				newbatch.Delete(calcTxKey(heightstr))
				continue
			}
			//toaddr
			toaddr := tx.GetTo()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				newbatch.Delete(calcTxKey(heightstr))
			}
		} else {
			walletlog.Info("ProcWalletDelBlock going to call AddDelPrivacyTxsFromBlock")
			wallet.AddDelPrivacyTxsFromBlock(tx, int32(index), block, newbatch, DelTx)
			//wallet.onDelPrivacyTxFromBlock(tx, int32(index), block, newbatch)
		}
	}
	newbatch.Write()
	if needflush {
		wallet.flushTicket()
	}
}

func (wallet *Wallet) AddDelPrivacyTxsFromBlock(tx *types.Transaction, index int32, block *types.BlockDetail, newbatch dbm.Batch, addDelType int32) {
	txhashstr := common.Bytes2Hex(tx.Hash())
	walletlog.Debug("PrivacyTrading AddDelPrivacyTxsFromBlock", "Enter AddDelPrivacyTxsFromBlock txhash", txhashstr)
	defer walletlog.Debug("PrivacyTrading AddDelPrivacyTxsFromBlock", "Leave AddDelPrivacyTxsFromBlock txhash", txhashstr)

	_, err := tx.Amount()
	if err != nil {
		walletlog.Error("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "tx.Amount() error", err)
		return
	}

	txExecRes := block.Receipts[index].Ty

	txhashInbytes := tx.Hash()
	txhash := common.Bytes2Hex(txhashInbytes)
	var privateAction types.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		walletlog.Error("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "Decode tx.GetPayload() error", err)
		return
	}
	walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "Enter AddDelPrivacyTxsFromBlock txhash", txhashstr, "index", index, "addDelType", addDelType)

	privacyInput := privateAction.GetInput()
	privacyOutput := privateAction.GetOutput()
	tokenname := privateAction.GetTokenName()
	RpubKey := privacyOutput.GetRpubKeytx()

	totalUtxosLeft := len(privacyOutput.Keyoutput)
	//处理output
	if privacyInfo, err := wallet.getPrivacyKeyPairsOfWallet(); err == nil {
		matchedCount := 0
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			privacykeyParirs := info.PrivacyKeyPair
			matched4addr := false
			var utxos []*types.UTXO
			if privacyInput != nil && types.ExecOk == txExecRes && AddTx == addDelType {
				// 如果输入中包含了已经设置的UTXO,需要直接移除
				for _, keyinput := range privacyInput.Keyinput {
					for _, utxogl := range keyinput.UtxoGlobalIndex {
						txhashstr := common.Bytes2Hex(utxogl.Txhash)
						wallet.walletStore.unlinkUTXO(info.Addr, &txhashstr, int(utxogl.Outindex), tokenname, newbatch)
					}
				}
			}
			for indexoutput, output := range privacyOutput.Keyoutput {
				if utxoProcessed[indexoutput] {
					continue
				}
				priv, err := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
				if err == nil {
					walletlog.Debug("PrivacyTrading AddDelPrivacyTxsFromBlock", "Enter RecoverOnetimePriKey txhash", txhashstr)
					recoverPub := priv.PubKey().Bytes()[:]
					if bytes.Equal(recoverPub, output.Onetimepubkey) {
						walletlog.Debug("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "bytes.Equal(recoverPub, output.Onetimepubkey) true")
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
									Txhash:           txhashInbytes,
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
								wallet.walletStore.setUTXO(info.Addr, &txhash, indexoutput, info2store, newbatch)
								walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "add tx txhash", txhashstr, "setUTXO addr ", *info.Addr, "indexoutput", indexoutput)
							} else {
								walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "delete tx txhash", txhashstr, "unsetUTXO addr ", *info.Addr, "indexoutput", indexoutput)
								wallet.walletStore.unsetUTXO(info.Addr, &txhash, indexoutput, tokenname, newbatch)
							}
						} else {
							//对于执行失败的交易，只需要将该交易记录在钱包就行
							walletlog.Error("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "txExecRes", txExecRes)
							break
						}
					}
				} else {
					walletlog.Error("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "RecoverOnetimePriKey error", err)
				}
			}
			if matched4addr {
				matchedCount++
				//匹配次数达到2次，不再对本钱包中的其他地址进行匹配尝试
				walletlog.Debug("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "address", *info.Addr, "totalUtxosLeft", totalUtxosLeft, "matchedCount", matchedCount)
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
				wallet.buildAndStoreWalletTxDetail(param)
				if 2 == matchedCount || 0 == totalUtxosLeft || types.ExecOk != txExecRes {
					walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "Get matched privacy transfer for address address", *info.Addr, "totalUtxosLeft", totalUtxosLeft, "matchedCount", matchedCount)
					break
				}
			}
		}
	}

	//处理input,对于公对私的交易类型，只会出现在output类型处理中
	//如果该隐私交易是本钱包中的地址发送出去的，则需要对相应的utxo进行处理
	if AddTx == addDelType {
		ftxos, keys := wallet.getFTXOlist()
		for i, ftxo := range ftxos {
			//查询确认该交易是否为记录的支付交易
			if ftxo.Txhash != txhash {
				continue
			}
			if types.ExecOk == txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveFTXO2STXO, key", string(keys[i]), "txExecRes", txExecRes)
				wallet.walletStore.moveFTXO2STXO(keys[i], txhash, newbatch)
			} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				//如果执行失败
				walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveFTXO2UTXO, key", string(keys[i]), "txExecRes", txExecRes)
				wallet.walletStore.moveFTXO2UTXO(keys[i], newbatch)
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
			wallet.buildAndStoreWalletTxDetail(param)
		}
	} else {
		//当发生交易回撤时，从记录的STXO中查找相关的交易，并将其重置为FTXO，因为该交易大概率会在其他区块中再次执行
		stxosInOneTx, _, _ := wallet.walletStore.GetWalletFtxoStxo(STXOs4Tx)
		for _, ftxo := range stxosInOneTx {
			if ftxo.Txhash == txhash {
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
					walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType, "moveSTXO2FTXO txExecRes", txExecRes)
					wallet.walletStore.moveSTXO2FTXO(tx, txhash, newbatch)
					wallet.buildAndStoreWalletTxDetail(param)
				} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
					walletlog.Info("PrivacyTrading AddDelPrivacyTxsFromBlock", "txhash", txhashstr, "addDelType", addDelType)
					wallet.buildAndStoreWalletTxDetail(param)
				}
			}
		}
	}
}

func (wallet *Wallet) GetTxDetailByHashs(ReqHashes *types.ReqHashes) {
	//通过txhashs获取对应的txdetail
	msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByHash, ReqHashes)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("GetTxDetailByHashs EventGetTransactionByHash", "err", err)
		return
	}
	TxDetails := resp.GetData().(*types.TransactionDetails)
	if TxDetails == nil {
		walletlog.Info("GetTxDetailByHashs TransactionDetails is nil")
		return
	}

	//批量存储地址对应的所有交易的详细信息到wallet db中
	newbatch := wallet.walletStore.NewBatch(true)
	for _, txdetal := range TxDetails.Txs {
		height := txdetal.GetHeight()
		txindex := txdetal.GetIndex()

		blockheight := height*maxTxNumPerBlock + txindex
		heightstr := fmt.Sprintf("%018d", blockheight)
		var txdetail types.WalletTxDetail
		txdetail.Tx = txdetal.GetTx()
		txdetail.Height = txdetal.GetHeight()
		txdetail.Index = txdetal.GetIndex()
		txdetail.Receipt = txdetal.GetReceipt()
		txdetail.Blocktime = txdetal.GetBlocktime()
		txdetail.Amount = txdetal.GetAmount()
		txdetail.Fromaddr = txdetal.GetFromaddr()
		txdetail.ActionName = txdetal.GetTx().ActionName()

		//由于Withdraw的交易在blockchain模块已经做了from和to地址的swap的操作。
		//所以在此需要swap恢复回去。通过钱包的GetTxDetailByIter接口上送给前端时再做from和to地址的swap
		//确保保存在数据库中是的最原始的数据，提供给上层显示时可以做swap方便客户理解
		if txdetail.GetTx().IsWithdraw() {
			txdetail.Fromaddr, txdetail.Tx.To = txdetail.Tx.To, txdetail.Fromaddr
		}

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			storelog.Error("GetTxDetailByHashs Marshal txdetail err", "Height", height, "index", txindex)
			return
		}
		newbatch.Set(calcTxKey(heightstr), txdetailbyte)
		//walletlog.Debug("GetTxDetailByHashs", "heightstr", heightstr, "txdetail", txdetail.String())
	}
	newbatch.Write()
}

//从blockchain模块同步addr参与的所有交易详细信息
func (wallet *Wallet) reqTxDetailByAddr(addr string) {
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
		msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByAddr, &ReqAddr)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
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

//生成一个随机的seed种子, 目前支持英文单词和简体中文
func (wallet *Wallet) genSeed(lang int32) (*types.ReplySeed, error) {
	seed, err := CreateSeed("", lang)
	if err != nil {
		walletlog.Error("genSeed", "CreateSeed err", err)
		return nil, err
	}
	var ReplySeed types.ReplySeed
	ReplySeed.Seed = seed
	return &ReplySeed, nil
}

//获取seed种子, 通过钱包密码
func (wallet *Wallet) getSeed(password string) (string, error) {
	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return "", err
	}

	seed, err := GetSeed(wallet.walletStore.db, password)
	if err != nil {
		walletlog.Error("getSeed", "GetSeed err", err)
		return "", err
	}
	return seed, nil
}

//保存seed种子到数据库中, 并通过钱包密码加密, 钱包起来首先要设置seed
func (wallet *Wallet) saveSeed(password string, seed string) (bool, error) {

	//首先需要判断钱包是否已经设置seed，如果已经设置提示不需要再设置，一个钱包只能保存一个seed
	exit, err := HasSeed(wallet.walletStore.db)
	if exit {
		return false, types.ErrSeedExist
	}
	//入参数校验，seed必须是大于等于12个单词或者汉字
	if len(password) == 0 || len(seed) == 0 {
		return false, types.ErrInputPara
	}

	seedarry := strings.Fields(seed)
	curseedlen := len(seedarry)
	if curseedlen < SaveSeedLong {
		walletlog.Error("saveSeed VeriySeedwordnum", "curseedlen", curseedlen, "SaveSeedLong", SaveSeedLong)
		return false, types.ErrSeedWordNum
	}

	var newseed string
	for index, seedstr := range seedarry {
		if index != curseedlen-1 {
			newseed += seedstr + " "
		} else {
			newseed += seedstr
		}
	}

	//校验seed是否能生成钱包结构类型，从而来校验seed的正确性
	have, err := VerifySeed(newseed)
	if !have {
		walletlog.Error("saveSeed VerifySeed", "err", err)
		return false, types.ErrSeedWord
	}

	ok, err := SaveSeed(wallet.walletStore.db, newseed, password)
	//seed保存成功需要更新钱包密码
	if ok {
		var ReqWalletSetPasswd types.ReqWalletSetPasswd
		ReqWalletSetPasswd.OldPass = password
		ReqWalletSetPasswd.NewPass = password
		Err := wallet.ProcWalletSetPasswd(&ReqWalletSetPasswd)
		if Err != nil {
			walletlog.Error("saveSeed", "ProcWalletSetPasswd err", err)
		}
	}
	return ok, err
}

//获取地址对应的私钥
func (wallet *Wallet) ProcDumpPrivkey(addr string) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return "", err
	}
	if len(addr) == 0 {
		walletlog.Error("ProcDumpPrivkey input para is nil!")
		return "", types.ErrInputPara
	}

	priv, err := wallet.getPrivKeyByAddr(addr)
	if err != nil {
		return "", err
	}
	return common.ToHex(priv.Bytes()), nil
	//return strings.ToUpper(common.ToHex(priv.Bytes())), nil
}

//收到其他模块上报的系统有致命性故障，需要通知前端
func (wallet *Wallet) setFatalFailure(reportErrEvent *types.ReportErrEvent) {

	walletlog.Error("setFatalFailure", "reportErrEvent", reportErrEvent.String())
	if reportErrEvent.Error == "ErrDataBaseDamage" {
		atomic.StoreInt32(&wallet.fatalFailureFlag, 1)
	}
}

func (wallet *Wallet) getFatalFailure() int32 {
	return atomic.LoadInt32(&wallet.fatalFailureFlag)
}

func (wallet *Wallet) procPrivacyTransactionList(req *types.ReqPrivacyTransactionList) (*types.WalletTxDetails, error) {
	walletlog.Info("call procPrivacyTransactionList")
	if req == nil {
		walletlog.Error("procPrivacyTransactionList", "param is nil")
		return nil, types.ErrInvalidParams
	}
	if req.Direction != 0 && req.Direction != 1 {
		walletlog.Error("procPrivacyTransactionList", "invalid direction ", req.Direction)
		return nil, types.ErrInvalidParams
	}
	// convert to sendTx / recvTx
	sendRecvFlag := req.SendRecvFlag + sendTx
	if sendRecvFlag != sendTx && sendRecvFlag != recvTx {
		walletlog.Error("procPrivacyTransactionList", "invalid sendrecvflag ", req.SendRecvFlag)
		return nil, types.ErrInvalidParams
	}
	req.SendRecvFlag = sendRecvFlag

	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	reply, err := wallet.walletStore.getWalletPrivacyTxDetails(req)
	if err != nil {
		walletlog.Error("procPrivacyTransactionList", "getWalletPrivacyTxDetails error", err)
		return nil, err
	}
	return reply, nil
}

func (wallet *Wallet) procRescanUtxos(req *types.ReqRescanUtxos) (*types.RepRescanUtxos, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if 0 == req.Flag { // Rescan请求
		var repRescanUtxos types.RepRescanUtxos
		repRescanUtxos.Flag = req.Flag

		if wallet.IsWalletLocked() {
			return nil, types.ErrWalletIsLocked
		}
		if ok, err := wallet.IsRescanUtxosFlagScaning(); ok {
			return nil, err
		}
		atomic.StoreInt32(&wallet.rescanUTXOflag, types.UtxoFlagScaning)
		wallet.wg.Add(1)
		go wallet.RescanReqUtxosByAddr(req.Addrs)
		return &repRescanUtxos, nil
	} else { // 查询地址对应扫描状态
		repRescanUtxos, err := wallet.walletStore.GetRescanUtxosFlag4Addr(req)
		return repRescanUtxos, err
	}
}
