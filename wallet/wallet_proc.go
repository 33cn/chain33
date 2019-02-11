// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	dbm "github.com/33cn/chain33/common/db"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet/bipwallet"
	wcom "github.com/33cn/chain33/wallet/common"
	"github.com/golang/protobuf/proto"
)

// ProcSignRawTx 用钱包对交易进行签名
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

	if ok, err := wallet.IsRescanUtxosFlagScaning(); ok || err != nil {
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
		cr, err := crypto.New(types.GetSignName("", SignType))
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

	txByteData, err := common.FromHex(unsigned.GetTxHex())
	if err != nil {
		return "", err
	}
	var tx types.Transaction
	err = types.Decode(txByteData, &tx)
	if err != nil {
		return "", err
	}

	if unsigned.NewToAddr != "" {
		tx.To = unsigned.NewToAddr
	}
	if unsigned.Fee != 0 {
		tx.Fee = unsigned.Fee
	}

	expire, err := types.ParseExpire(unsigned.GetExpire())
	if err != nil {
		return "", err
	}
	tx.SetExpire(time.Duration(expire))
	if policy, ok := wcom.PolicyContainer[string(types.GetParaExec(tx.Execer))]; ok {
		// 尝试让策略自己去完成签名
		needSysSign, signtx, err := policy.SignTransaction(key, unsigned)
		if !needSysSign {
			return signtx, err
		}
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
			err := group.SignN(i, int32(SignType), key)
			if err != nil {
				return "", err
			}
		}
		grouptx := group.Tx()
		txHex := types.Encode(grouptx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	index--
	err = group.SignN(int(index), int32(SignType), key)
	if err != nil {
		return "", err
	}
	grouptx := group.Tx()
	txHex := types.Encode(grouptx)
	signedTx := hex.EncodeToString(txHex)
	return signedTx, nil
}

// ProcGetAccountList 获取钱包账号列表
//output:
//type WalletAccounts struct {
//	Wallets []*WalletAccount
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//获取钱包的地址列表
func (wallet *Wallet) ProcGetAccountList(req *types.ReqAccountList) (*types.WalletAccounts, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("ProcGetAccountList", "GetAccountByPrefix:err", err)
		return nil, err
	}
	if req.WithoutBalance {
		return makeAccountWithoutBalance(WalletAccStores)
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

func makeAccountWithoutBalance(accountStores []*types.WalletAccountStore) (*types.WalletAccounts, error) {
	var WalletAccounts types.WalletAccounts
	WalletAccounts.Wallets = make([]*types.WalletAccount, len(accountStores))

	for index, account := range accountStores {
		var WalletAccount types.WalletAccount
		//此账户还没有参与交易所在account模块没有记录
		if len(account.Addr) == 0 {
			continue
		}
		WalletAccount.Acc = &types.Account{Addr: account.Addr}
		WalletAccount.Label = account.GetLabel()
		WalletAccounts.Wallets[index] = &WalletAccount
	}
	return &WalletAccounts, nil
}

// ProcCreateNewAccount 处理创建新账号
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
		return nil, types.ErrInvalidParam
	}

	//首先校验label是否已被使用
	WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label.GetLabel())
	if WalletAccStores != nil && err == nil {
		walletlog.Error("ProcCreateNewAccount Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}

	var Account types.Account
	var walletAccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	var cointype uint32
	var addr string
	var privkeybyte []byte

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

	for {
		privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.GetDB(), seed)
		if err != nil {
			walletlog.Error("ProcCreateNewAccount", "GetPrivkeyBySeed err", err)
			return nil, err
		}
		privkeybyte, err = common.FromHex(privkeyhex)
		if err != nil || len(privkeybyte) == 0 {
			walletlog.Error("ProcCreateNewAccount", "FromHex err", err)
			return nil, err
		}

		pub, err := bipwallet.PrivkeyToPub(cointype, privkeybyte)
		if err != nil {
			seedlog.Error("ProcCreateNewAccount PrivkeyToPub", "err", err)
			return nil, types.ErrPrivkeyToPub
		}
		addr, err = bipwallet.PubToAddress(cointype, pub)
		if err != nil {
			seedlog.Error("ProcCreateNewAccount PubToAddress", "err", err)
			return nil, types.ErrPrivkeyToPub
		}
		//通过新生成的账户地址查询钱包数据库，如果查询返回的账户信息是空，
		//说明新生成的账户没有被使用，否则继续使用下一个index生成私钥对
		account, err := wallet.walletStore.GetAccountByAddr(addr)
		if account == nil || err != nil {
			break
		}
	}

	Account.Addr = addr
	Account.Currency = 0
	Account.Balance = 0
	Account.Frozen = 0

	walletAccount.Acc = &Account
	walletAccount.Label = Label.GetLabel()

	//使用钱包的password对私钥加密 aes cbc
	Encrypted := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
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
	for _, policy := range wcom.PolicyContainer {
		policy.OnCreateNewAccount(walletAccount.Acc)
	}

	return &walletAccount, nil
}

// ProcWalletTxList 处理获取钱包交易列表
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
		return nil, types.ErrInvalidParam
	}
	if TxList.GetDirection() != 0 && TxList.GetDirection() != 1 {
		walletlog.Error("ProcWalletTxList Direction err!")
		return nil, types.ErrInvalidParam
	}
	WalletTxDetails, err := wallet.walletStore.GetTxDetailByIter(TxList)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "GetTxDetailByIter err", err)
		return nil, err
	}
	return WalletTxDetails, nil
}

// ProcImportPrivKey 处理导入私钥
//input:
//type ReqWalletImportPrivKey struct {
//	Privkey string
//	Label   string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//导入私钥，并且同时会导入交易
func (wallet *Wallet) ProcImportPrivKey(PrivKey *types.ReqWalletImportPrivkey) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if PrivKey == nil || len(PrivKey.GetLabel()) == 0 || len(PrivKey.GetPrivkey()) == 0 {
		walletlog.Error("ProcImportPrivKey input parameter is nil!")
		return nil, types.ErrInvalidParam
	}

	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(PrivKey.GetLabel())
	if Account != nil && err == nil {
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
	Encryptered := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//校验PrivKey对应的addr是否已经存在钱包中
	Account, err = wallet.walletStore.GetAccountByAddr(addr)
	if Account != nil && err == nil {
		if Account.Privkey == Encrypteredstr {
			walletlog.Error("ProcImportPrivKey Privkey is exist in wallet!")
			return nil, types.ErrPrivkeyExist
		}
		walletlog.Error("ProcImportPrivKey!", "Account.Privkey", Account.Privkey, "input Privkey", PrivKey.Privkey)
		return nil, types.ErrPrivkey

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

	for _, policy := range wcom.PolicyContainer {
		policy.OnImportPrivateKey(accounts[0])
	}
	return &walletaccount, nil
}

// ProcSendToAddress 响应发送到地址
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
		return nil, types.ErrInvalidParam
	}
	if len(SendToAddress.From) == 0 || len(SendToAddress.To) == 0 {
		walletlog.Error("ProcSendToAddress input para From or To is nil!")
		return nil, types.ErrInvalidParam
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
	//amount必须大于等于0
	if amount < 0 {
		return nil, types.ErrAmount
	}
	if !SendToAddress.IsToken {
		if Balance-amount < wallet.FeeAmount {
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

// ProcWalletSetFee 处理设置手续费
//type ReqWalletSetFee struct {
//	Amount int64
//设置钱包默认的手续费
func (wallet *Wallet) ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
	if WalletSetFee.Amount < minFee {
		walletlog.Error("ProcWalletSetFee err!", "Amount", WalletSetFee.Amount, "MinFee", minFee)
		return types.ErrInvalidParam
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

// ProcWalletSetLabel 处理设置账号标签
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
		return nil, types.ErrInvalidParam
	}
	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(SetLabel.GetLabel())
	if Account != nil && err == nil {
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

// ProcMergeBalance 处理
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
		return nil, types.ErrInvalidParam
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
	cr, err := crypto.New(types.GetSignName("", SignType))
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

		privkey := wcom.CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
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
		v := &cty.CoinsAction_Transfer{
			Transfer: &types.AssetsTransfer{Amount: amount, Note: []byte(note)},
		}
		transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
		//初始化随机数
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: addrto, Nonce: wallet.random.Int63()}
		tx.SetExpire(time.Second * 120)
		tx.Sign(int32(SignType), priv)
		//walletlog.Info("ProcMergeBalance", "tx.Nonce", tx.Nonce, "tx", tx, "index", index)

		//发送交易信息给mempool模块
		msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
		err = wallet.client.Send(msg, true)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "Send tx err", err, "index", index)
			continue
		}
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

// ProcWalletSetPasswd 处理钱包的保存密码
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
	newBatch := wallet.walletStore.NewBatch(true)
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
	ok, err := SaveSeedInBatch(wallet.walletStore.GetDB(), seed, Passwd.NewPass, newBatch)
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
		Decrypter := wcom.CBCDecrypterPrivkey([]byte(Passwd.OldPass), storekey)

		//使用新的密码重新加密私钥
		Encrypter := wcom.CBCEncrypterPrivkey([]byte(Passwd.NewPass), Decrypter)
		AccStore.Privkey = common.ToHex(Encrypter)
		err = wallet.walletStore.SetWalletAccountInBatch(true, AccStore.Addr, AccStore, newBatch)
		if err != nil {
			walletlog.Info("ProcWalletSetPasswd", "addr", AccStore.Addr, "SetWalletAccount err", err)
		}
	}

	err = newBatch.Write()
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd newBatch.Write", "err", err)
		return err
	}
	wallet.Password = Passwd.NewPass
	wallet.EncryptFlag = 1
	return nil
}

//ProcWalletLock 锁定钱包
func (wallet *Wallet) ProcWalletLock() error {
	//判断钱包是否已保存seed
	has, err := wallet.walletStore.HasSeed()
	if !has || err != nil {
		return types.ErrSaveSeedFirst
	}

	atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
	for _, policy := range wcom.PolicyContainer {
		policy.OnWalletLocked()
	}
	return nil
}

// ProcWalletUnLock 处理钱包解锁
//input:
//type WalletUnLock struct {
//	Passwd  string
//	Timeout int64
//解锁钱包Timeout时间，超时后继续锁住
func (wallet *Wallet) ProcWalletUnLock(WalletUnLock *types.WalletUnLock) error {
	//判断钱包是否已保存seed
	has, err := wallet.walletStore.HasSeed()
	if !has || err != nil {
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
	//只解锁挖矿转账
	if !WalletUnLock.WalletOrTicket {
		//wallet.isTicketLocked = false
		atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 1, 0)
		if WalletUnLock.Timeout != 0 {
			wallet.resetTimeout(WalletUnLock.Timeout)
		}
	}
	for _, policy := range wcom.PolicyContainer {
		policy.OnWalletUnlocked(WalletUnLock)
	}
	return nil

}

//解锁超时处理，需要区分整个钱包的解锁或者只挖矿的解锁
func (wallet *Wallet) resetTimeout(Timeout int64) {
	if wallet.timeout == nil {
		wallet.timeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
			//wallet.isWalletLocked = true
			atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
		})
	} else {
		wallet.timeout.Reset(time.Second * time.Duration(Timeout))
	}
}

//ProcWalletAddBlock wallet模块收到blockchain广播的addblock消息，需要解析钱包相关的tx并存储到db中
func (wallet *Wallet) ProcWalletAddBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletAddBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletAddBlock", "height", block.GetBlock().GetHeight())
	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.NewBatch(true)
	for index := 0; index < txlen; index++ {
		tx := block.Block.Txs[index]
		execer := string(types.GetParaExec(tx.Execer))
		// 执行钱包业务逻辑策略
		if policy, ok := wcom.PolicyContainer[execer]; ok {
			wtxdetail := policy.OnAddBlockTx(block, tx, int32(index), newbatch)
			if wtxdetail == nil {
				continue
			}
			if len(wtxdetail.Fromaddr) > 0 {
				txdetailbyte, err := proto.Marshal(wtxdetail)
				if err != nil {
					walletlog.Error("ProcWalletAddBlock", "Marshal txdetail error", err, "Height", block.Block.Height, "index", index)
					continue
				}
				blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
				heightstr := fmt.Sprintf("%018d", blockheight)
				key := wcom.CalcTxKey(heightstr)
				newbatch.Set(key, txdetailbyte)
			}

		} else { // 默认的执行器类型处理
			// TODO: 钱包基础功能模块，将会重新建立一个处理策略，将钱包变成一个容器
			//获取from地址
			pubkey := block.Block.Txs[index].Signature.GetPubkey()
			addr := address.PubKeyToAddress(pubkey)
			param := &buildStoreWalletTxDetailParam{
				tokenname:  "",
				block:      block,
				tx:         tx,
				index:      index,
				newbatch:   newbatch,
				isprivacy:  false,
				addDelType: AddTx,
				//utxos:      nil,
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
		}
	}
	err := newbatch.Write()
	if err != nil {
		walletlog.Error("ProcWalletAddBlock newbatch.Write", "err", err)
		atomic.CompareAndSwapInt32(&wallet.fatalFailureFlag, 0, 1)
	}

	for _, policy := range wcom.PolicyContainer {
		policy.OnAddBlockFinish(block)
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
	//utxos        []*types.UTXO
}

func (wallet *Wallet) buildAndStoreWalletTxDetail(param *buildStoreWalletTxDetailParam) {
	blockheight := param.block.Block.Height*maxTxNumPerBlock + int64(param.index)
	heightstr := fmt.Sprintf("%018d", blockheight)
	walletlog.Debug("buildAndStoreWalletTxDetail", "heightstr", heightstr, "addDelType", param.addDelType)
	if AddTx == param.addDelType {
		var txdetail types.WalletTxDetail
		var Err error
		key := wcom.CalcTxKey(heightstr)
		txdetail.Tx = param.tx
		txdetail.Height = param.block.Block.Height
		txdetail.Index = int64(param.index)
		txdetail.Receipt = param.block.Receipts[param.index]
		txdetail.Blocktime = param.block.Block.BlockTime

		txdetail.ActionName = txdetail.Tx.ActionName()
		txdetail.Amount, Err = param.tx.Amount()
		if Err != nil {
			walletlog.Error("buildAndStoreWalletTxDetail Amount err", "Height", param.block.Block.Height, "index", param.index)
		}
		txdetail.Fromaddr = param.senderRecver
		//txdetail.Spendrecv = param.utxos

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			walletlog.Error("buildAndStoreWalletTxDetail Marshal txdetail err", "Height", param.block.Block.Height, "index", param.index)
			return
		}
		param.newbatch.Set(key, txdetailbyte)
	} else {
		param.newbatch.Delete(wcom.CalcTxKey(heightstr))
	}
}

//ProcWalletDelBlock wallet模块收到blockchain广播的delblock消息，需要解析钱包相关的tx并存db中删除
func (wallet *Wallet) ProcWalletDelBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletDelBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletDelBlock", "height", block.GetBlock().GetHeight())

	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.NewBatch(true)
	for index := txlen - 1; index >= 0; index-- {
		blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)
		tx := block.Block.Txs[index]

		execer := string(types.GetParaExec(tx.Execer))
		// 执行钱包业务逻辑策略
		if policy, ok := wcom.PolicyContainer[execer]; ok {
			wtxdetail := policy.OnDeleteBlockTx(block, tx, int32(index), newbatch)
			if wtxdetail == nil {
				continue
			}
			if len(wtxdetail.Fromaddr) > 0 {
				newbatch.Delete(wcom.CalcTxKey(heightstr))
			}

		} else { // 默认的合约处理流程
			// TODO:将钱包基础功能移动到专属钱包基础业务的模块中，将钱包模块变成容器
			//获取from地址
			pubkey := tx.Signature.GetPubkey()
			addr := address.PubKeyToAddress(pubkey)
			fromaddress := addr.String()
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				newbatch.Delete(wcom.CalcTxKey(heightstr))
				continue
			}
			//toaddr
			toaddr := tx.GetTo()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				newbatch.Delete(wcom.CalcTxKey(heightstr))
			}
		}
	}
	err := newbatch.Write()
	if err != nil {
		walletlog.Error("ProcWalletDelBlock newbatch.Write", "err", err)
	}
	for _, policy := range wcom.PolicyContainer {
		policy.OnDeleteBlockFinish(block)
	}
}

// GetTxDetailByHashs 根据交易哈希获取对应的交易详情
func (wallet *Wallet) GetTxDetailByHashs(ReqHashes *types.ReqHashes) {
	//通过txhashs获取对应的txdetail
	msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByHash, ReqHashes)
	err := wallet.client.Send(msg, true)
	if err != nil {
		walletlog.Error("GetTxDetailByHashs Send EventGetTransactionByHash", "err", err)
		return
	}
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
			walletlog.Error("GetTxDetailByHashs Marshal txdetail err", "Height", height, "index", txindex)
			return
		}
		newbatch.Set(wcom.CalcTxKey(heightstr), txdetailbyte)
	}
	err = newbatch.Write()
	if err != nil {
		walletlog.Error("GetTxDetailByHashs newbatch.Write", "err", err)
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

// GenSeed 获取随机种子
func (wallet *Wallet) GenSeed(lang int32) (*types.ReplySeed, error) {
	return wallet.genSeed(lang)
}

//获取seed种子, 通过钱包密码
func (wallet *Wallet) getSeed(password string) (string, error) {
	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return "", err
	}

	seed, err := GetSeed(wallet.walletStore.GetDB(), password)
	if err != nil {
		walletlog.Error("getSeed", "GetSeed err", err)
		return "", err
	}
	return seed, nil
}

// SaveSeed 保存种子
func (wallet *Wallet) SaveSeed(password string, seed string) (bool, error) {
	return wallet.saveSeed(password, seed)
}

//保存seed种子到数据库中, 并通过钱包密码加密, 钱包起来首先要设置seed
func (wallet *Wallet) saveSeed(password string, seed string) (bool, error) {

	//首先需要判断钱包是否已经设置seed，如果已经设置提示不需要再设置，一个钱包只能保存一个seed
	exit, err := wallet.walletStore.HasSeed()
	if exit && err == nil {
		return false, types.ErrSeedExist
	}
	//入参数校验，seed必须是大于等于12个单词或者汉字
	if len(password) == 0 || len(seed) == 0 {
		return false, types.ErrInvalidParam
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
	//批量处理seed和password的存储
	newBatch := wallet.walletStore.NewBatch(true)
	err = wallet.walletStore.SetPasswordHash(password, newBatch)
	if err != nil {
		walletlog.Error("saveSeed", "SetPasswordHash err", err)
		return false, err
	}
	//设置钱包加密标志位
	err = wallet.walletStore.SetEncryptionFlag(newBatch)
	if err != nil {
		walletlog.Error("saveSeed", "SetEncryptionFlag err", err)
		return false, err
	}

	ok, err := SaveSeedInBatch(wallet.walletStore.GetDB(), seed, password, newBatch)
	if !ok {
		walletlog.Error("saveSeed", "SaveSeed err", err)
		return false, err
	}

	err = newBatch.Write()
	if err != nil {
		walletlog.Error("saveSeed newBatch.Write", "err", err)
		return false, err
	}
	wallet.Password = password
	wallet.EncryptFlag = 1
	return true, nil
}

//ProcDumpPrivkey 获取地址对应的私钥
func (wallet *Wallet) ProcDumpPrivkey(addr string) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return "", err
	}
	if len(addr) == 0 {
		walletlog.Error("ProcDumpPrivkey input para is nil!")
		return "", types.ErrInvalidParam
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
