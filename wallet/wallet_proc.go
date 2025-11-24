// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/33cn/chain33/system/address/eth"

	"github.com/pkg/errors"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	dbm "github.com/33cn/chain33/common/db"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet/bipwallet"
	wcom "github.com/33cn/chain33/wallet/common"
)

// ProcSignRawTx 用钱包对交易进行签名
// input:
//
//	type ReqSignRawTx struct {
//		Addr    string
//		Privkey string
//		TxHex   string
//		Expire  string
//	}
//
// output:
// string
// 签名交易
func (wallet *Wallet) ProcSignRawTx(unsigned *types.ReqSignRawTx) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	index := unsigned.Index

	//Privacy交易功能，需要在隐私plugin里面检查，而不是所有交易签名都检查，在隐私启动scan时候会导致其他交易签名失败，同时这里在scaning时候，
	//需要返回一个错误，而不是nil
	//if ok, err := wallet.IsRescanUtxosFlagScaning(); ok || err != nil {
	//	return "", err types.ErrNotSupport
	//}

	var key crypto.PrivKey
	addressID := unsigned.GetAddressID()
	if unsigned.GetAddr() != "" {
		ok, err := wallet.checkWalletStatus()
		if !ok {
			return "", err
		}
		key, err = wallet.getPrivKeyByAddr(unsigned.GetAddr())
		if err != nil {
			return "", err
		}
		addressID, err = address.GetAddressType(unsigned.Addr)
		if err != nil {
			return "", types.ErrInvalidAddress
		}

	} else if unsigned.GetPrivkey() != "" {
		keyByte, err := common.FromHex(unsigned.GetPrivkey())
		if err != nil {
			return "", err
		}
		if len(keyByte) == 0 {
			return "", types.ErrPrivateKeyLen
		}
		cr, err := crypto.Load(types.GetSignName("", wallet.SignType), wallet.lastHeader.GetHeight())
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
	// signID integrate crypto ID with address ID
	signID := types.EncodeSignID(int32(wallet.SignType), addressID)

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

	//To to set fee based on bigger of two fees, on is from sign request and the other is calculated based on tx size
	if unsigned.Fee != 0 {
		tx.Fee = unsigned.Fee
	}

	proper, err := wallet.api.GetProperFee(nil)
	if err != nil {
		return "", err
	}
	fee, err := tx.GetRealFee(proper.ProperFee)
	if err != nil {
		return "", err
	}
	if fee > tx.Fee {
		tx.Fee = fee
	}

	expire, err := types.ParseExpire(unsigned.GetExpire())
	if err != nil {
		return "", err
	}
	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
	tx.SetExpire(cfg, time.Duration(expire))
	if policy, ok := wcom.PolicyContainer[string(cfg.GetParaExec(tx.Execer))]; ok {
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
		tx.Sign(signID, key)
		txHex := types.Encode(&tx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	if int(index) > len(group.GetTxs()) {
		return "", types.ErrIndex
	}
	if index <= 0 {
		//设置过期需要重构
		group.SetExpire(cfg, 0, time.Duration(expire))
		group.RebuiltGroup()
		for i := range group.Txs {
			err := group.SignN(i, signID, key)
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
	err = group.SignN(int(index), signID, key)
	if err != nil {
		return "", err
	}
	grouptx := group.Tx()
	txHex := types.Encode(grouptx)
	signedTx := hex.EncodeToString(txHex)
	return signedTx, nil
}

// ProcGetAccount 通过地址标签获取账户地址
func (wallet *Wallet) ProcGetAccount(req *types.ReqGetAccount) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	accStore, err := wallet.walletStore.GetAccountByLabel(req.GetLabel())
	if err != nil {
		return nil, err
	}

	accs, err := wallet.accountdb.LoadAccounts(wallet.api, []string{accStore.GetAddr()})
	if err != nil {
		return nil, err
	}
	return &types.WalletAccount{Label: accStore.GetLabel(), Acc: accs[0]}, nil

}

// ProcGetAccountList 获取钱包账号列表
// output:
//
//	type WalletAccounts struct {
//		Wallets []*WalletAccount
//
//	type WalletAccount struct {
//		Acc   *Account
//		Label string
//
// 获取钱包的地址列表
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
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
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
// input:
//
//	type ReqNewAccount struct {
//		Label string
//
// output:
//
//	type WalletAccount struct {
//		Acc   *Account
//		Label string
//
//	type Account struct {
//		Currency int32
//		Balance  int64
//		Frozen   int64
//		Addr     string
//
// 创建一个新的账户
func (wallet *Wallet) ProcCreateNewAccount(Label *types.ReqNewAccount) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
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
	var addr string
	var privkeybyte []byte

	cointype := wallet.GetCoinType()
	//通过seed获取私钥, 首先通过钱包密码解锁seed然后通过seed生成私钥
	seed, err := wallet.getSeed(wallet.Password)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "getSeed err", err)
		return nil, err
	}

	for {
		privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.GetDB(), seed, 0, wallet.SignType, wallet.CoinType)
		if err != nil {
			walletlog.Error("ProcCreateNewAccount", "GetPrivkeyBySeed err", err)
			return nil, err
		}
		privkeybyte, err = common.FromHex(privkeyhex)
		if err != nil || len(privkeybyte) == 0 {
			walletlog.Error("ProcCreateNewAccount", "FromHex err", err)
			return nil, err
		}

		pub, err := bipwallet.PrivkeyToPub(cointype, uint32(wallet.SignType), privkeybyte)
		if err != nil {
			seedlog.Error("ProcCreateNewAccount PrivkeyToPub", "err", err)
			return nil, types.ErrPrivkeyToPub
		}
		addr = address.PubKeyToAddr(Label.GetAddressID(), pub)
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
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
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

// ProcNewRandAccount 随机创建新账号
func (wallet *Wallet) ProcNewRandAccount(req *types.GenSeedLang) (*types.AccountInfo, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	seed, err := wallet.genSeed(req.Lang)
	if err != nil {
		return nil, errors.Wrapf(err, "genSeed")
	}

	privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.GetDB(), seed.Seed, 0, wallet.SignType, wallet.CoinType)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "GetPrivkeyBySeed err", err)
		return nil, errors.Wrapf(err, "GetPrivkeyBySeed")
	}

	privkeybyte, err := common.FromHex(privkeyhex)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcCreateNewAccount", "FromHex err", err)
		return nil, errors.Wrapf(err, "transfer privkey")
	}

	pub, err := bipwallet.PrivkeyToPub(wallet.CoinType, uint32(wallet.SignType), privkeybyte)
	if err != nil {
		seedlog.Error("ProcCreateNewAccount PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	addr, err := bipwallet.PubToAddress(pub)
	if err != nil {
		seedlog.Error("ProcCreateNewAccount PubToAddress", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	return &types.AccountInfo{
		Addr:       addr,
		PrivateKey: privkeyhex,
		PubKey:     hex.EncodeToString(pub),
		Seed:       seed.GetSeed(),
	}, nil

}

// ProcWalletTxList 处理获取钱包交易列表
// input:
//
//	type ReqWalletTransactionList struct {
//		FromTx []byte
//		Count  int32
//
// output:
//
//	type WalletTxDetails struct {
//		TxDetails []*WalletTxDetail
//
//	type WalletTxDetail struct {
//		Tx      *Transaction
//		Receipt *ReceiptData
//		Height  int64
//		Index   int64
//
// 获取所有钱包的交易记录
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
	//默认取10笔交易数据
	if TxList.Count == 0 {
		TxList.Count = 10
	}
	if int64(TxList.Count) > types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}

	WalletTxDetails, err := wallet.walletStore.GetTxDetailByIter(TxList)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "GetTxDetailByIter err", err)
		return nil, err
	}
	return WalletTxDetails, nil
}

// ProcImportPrivKey 处理导入私钥
// input:
//
//	type ReqWalletImportPrivKey struct {
//		Privkey string
//		Label   string
//
// output:
//
//	type WalletAccount struct {
//		Acc   *Account
//		Label string
//
// 导入私钥，并且同时会导入交易
func (wallet *Wallet) ProcImportPrivKey(PrivKey *types.ReqWalletImportPrivkey) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	walletaccount, err := wallet.procImportPrivKey(PrivKey)
	return walletaccount, err
}

func (wallet *Wallet) procImportPrivKey(PrivKey *types.ReqWalletImportPrivkey) (*types.WalletAccount, error) {
	ok, err := wallet.checkWalletStatus()
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

	cointype := wallet.GetCoinType()

	privkeybyte, err := common.FromHex(PrivKey.Privkey)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcImportPrivKey", "FromHex err", err)
		return nil, types.ErrFromHex
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, uint32(wallet.SignType), privkeybyte)
	if err != nil {
		seedlog.Error("ProcImportPrivKey PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	addr := address.PubKeyToAddr(PrivKey.GetAddressID(), pub)

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
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
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
// input:
//
//	type ReqWalletSendToAddress struct {
//		From   string
//		To     string
//		Amount int64
//		Note   string
//
// output:
//
//	type ReplyHash struct {
//		Hashe []byte
//
// 发送一笔交易给对方地址，返回交易hash
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

	ok, err := wallet.isTransfer(SendToAddress.GetTo())
	if !ok {
		return nil, err
	}

	//获取from账户的余额从account模块，校验余额是否充足
	addrs := make([]string, 1)
	addrs[0] = SendToAddress.GetFrom()
	var accounts []*types.Account
	var tokenAccounts []*types.Account
	accounts, err = wallet.accountdb.LoadAccounts(wallet.api, addrs)
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
		if nil == wallet.accTokenMap[SendToAddress.TokenSymbol] {
			tokenAccDB, err := account.NewAccountDB(wallet.api.GetConfig(), "token", SendToAddress.TokenSymbol, nil)
			if err != nil {
				return nil, err
			}
			wallet.accTokenMap[SendToAddress.TokenSymbol] = tokenAccDB
		}
		tokenAccDB := wallet.accTokenMap[SendToAddress.TokenSymbol]
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
	addressID := address.GetDefaultAddressID()
	if common.IsHex(addrs[0]) {
		addressID = eth.ID
	}
	return wallet.sendToAddress(priv, addressID, addrto, amount, note, SendToAddress.IsToken, SendToAddress.TokenSymbol)
}

// ProcWalletSetFee 处理设置手续费
//
//	type ReqWalletSetFee struct {
//		Amount int64
//
// 设置钱包默认的手续费
func (wallet *Wallet) ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
	if WalletSetFee.Amount < wallet.minFee {
		walletlog.Error("ProcWalletSetFee err!", "Amount", WalletSetFee.Amount, "MinFee", wallet.minFee)
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
// input:
//
//	type ReqWalletSetLabel struct {
//		Addr  string
//		Label string
//
// output:
//
//	type WalletAccount struct {
//		Acc   *Account
//		Label string
//
// 设置某个账户的标签
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
			accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
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
// input:
//
//	type ReqWalletMergeBalance struct {
//		To string
//
// output:
//
//	type ReplyHashes struct {
//		Hashes [][]byte
//
// 合并所有的balance 到一个地址
func (wallet *Wallet) ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
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
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcMergeBalance", "LoadAccounts err", err)
		return nil, err
	}

	//异常信息记录
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcMergeBalance", "AccStores", len(WalletAccStores), "accounts", len(accounts))
	}
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.Load(types.GetSignName("", wallet.SignType), wallet.lastHeader.GetHeight())
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err)
		return nil, err
	}

	addrto := MergeBalance.GetTo()
	note := "MergeBalance"

	var ReplyHashes types.ReplyHashes

	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
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
		if cfg.IsPara() {
			v.Transfer.To = MergeBalance.GetTo()
		}
		transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
		//初始化随机数
		exec := []byte(cfg.GetCoinExec())
		toAddr := addrto
		if cfg.IsPara() {
			exec = []byte(cfg.GetTitle() + cfg.GetCoinExec())
			toAddr = address.ExecAddress(string(exec))
		}
		tx := &types.Transaction{Execer: exec, Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: toAddr, Nonce: wallet.random.Int63()}
		tx.ChainID = cfg.GetChainID()

		tx.SetExpire(cfg, time.Second*120)
		tx.Sign(int32(wallet.SignType), priv)
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
// input:
//
//	type ReqWalletSetPasswd struct {
//		Oldpass string
//		Newpass string
//
// 设置或者修改密码
func (wallet *Wallet) ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	isok, err := wallet.checkWalletStatus()
	if !isok && err == types.ErrSaveSeedFirst {
		return err
	}
	// 新密码合法性校验
	if !isValidPassWord(Passwd.NewPass) {
		return types.ErrInvalidPassWord
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

// ProcWalletLock 锁定钱包
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
// input:
//
//	type WalletUnLock struct {
//		Passwd  string
//		Timeout int64
//
// 解锁钱包Timeout时间，超时后继续锁住
func (wallet *Wallet) ProcWalletUnLock(WalletUnLock *types.WalletUnLock) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

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

// 解锁超时处理，需要区分整个钱包的解锁或者只挖矿的解锁
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

// ProcWalletAddBlock wallet模块收到blockchain广播的addblock消息，需要解析钱包相关的tx并存储到db中
func (wallet *Wallet) ProcWalletAddBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletAddBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletAddBlock", "height", block.GetBlock().GetHeight())
	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.GetBlockBatch(true)
	defer wallet.walletStore.FreeBlockBatch()
	for index := 0; index < txlen; index++ {
		tx := block.Block.Txs[index]
		execer := string(cfg.GetParaExec(tx.Execer))
		// 执行钱包业务逻辑策略
		if policy, ok := wcom.PolicyContainer[execer]; ok {
			wtxdetail := policy.OnAddBlockTx(block, tx, int32(index), newbatch)
			if wtxdetail == nil {
				continue
			}
			if len(wtxdetail.Fromaddr) > 0 {
				txdetailbyte := types.Encode(wtxdetail)
				blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
				heightstr := fmt.Sprintf("%018d", blockheight)
				key := wcom.CalcTxKey(heightstr)
				newbatch.Set(key, txdetailbyte)
			}

		} else { // 默认的执行器类型处理
			// TODO: 钱包基础功能模块，将会重新建立一个处理策略，将钱包变成一个容器
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
			fromAddr := tx.From()
			param.senderRecver = fromAddr
			if len(fromAddr) != 0 && wallet.AddrInWallet(fromAddr) {
				param.sendRecvFlag = sendTx
				wallet.buildAndStoreWalletTxDetail(param)
				walletlog.Debug("ProcWalletAddBlock", "fromAddr", fromAddr)
				continue
			}
			//toaddr获取交易中真实的接收地址，主要是针对para
			toaddr := tx.GetRealToAddr()
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
		txdetail.Txhash = param.tx.Hash()
		txdetail.ActionName = txdetail.Tx.ActionName()
		txdetail.Amount, Err = param.tx.Amount()
		if Err != nil {
			walletlog.Debug("buildAndStoreWalletTxDetail Amount err", "Height", param.block.Block.Height, "index", param.index)
		}
		txdetail.Fromaddr = param.senderRecver
		//txdetail.Spendrecv = param.utxos

		txdetailbyte := types.Encode(&txdetail)
		param.newbatch.Set(key, txdetailbyte)
	} else {
		param.newbatch.Delete(wcom.CalcTxKey(heightstr))
	}
}

// ProcWalletDelBlock wallet模块收到blockchain广播的delblock消息，需要解析钱包相关的tx并存db中删除
func (wallet *Wallet) ProcWalletDelBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletDelBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletDelBlock", "height", block.GetBlock().GetHeight())
	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.GetBlockBatch(true)
	defer wallet.walletStore.FreeBlockBatch()
	for index := txlen - 1; index >= 0; index-- {
		blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)
		tx := block.Block.Txs[index]

		execer := string(cfg.GetParaExec(tx.Execer))
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
			fromAddr := tx.From()
			if len(fromAddr) != 0 && wallet.AddrInWallet(fromAddr) {
				newbatch.Delete(wcom.CalcTxKey(heightstr))
				continue
			}
			//toaddr
			toaddr := tx.GetRealToAddr()
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

		txdetailbyte := types.Encode(&txdetail)
		newbatch.Set(wcom.CalcTxKey(heightstr), txdetailbyte)
	}
	err = newbatch.Write()
	if err != nil {
		walletlog.Error("GetTxDetailByHashs newbatch.Write", "err", err)
	}
}

// 生成一个随机的seed种子, 目前支持英文单词和简体中文
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

// GetSeed 获取seed种子, 通过钱包密码
func (wallet *Wallet) GetSeed(password string) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.getSeed(password)
}

// 获取seed种子, 通过钱包密码
func (wallet *Wallet) getSeed(password string) (string, error) {
	ok, err := wallet.checkWalletStatus()
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

// 保存seed种子到数据库中, 并通过钱包密码加密, 钱包起来首先要设置seed
func (wallet *Wallet) saveSeed(password string, seed string) (bool, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//首先需要判断钱包是否已经设置seed，如果已经设置提示不需要再设置，一个钱包只能保存一个seed
	exit, err := wallet.walletStore.HasSeed()
	if exit && err == nil {
		return false, types.ErrSeedExist
	}
	//入参数校验，seed必须是大于等于12个单词或者汉字
	if len(password) == 0 || len(seed) == 0 {
		return false, types.ErrInvalidParam
	}

	// 密码合法性校验
	if !isValidPassWord(password) {
		return false, types.ErrInvalidPassWord
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
	have, err := VerifySeed(newseed, wallet.SignType, wallet.CoinType)
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

	ok, err := SaveSeedInBatch(wallet.walletStore.GetDB(), newseed, password, newBatch)
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

// ProcDumpPrivkey 获取地址对应的私钥
func (wallet *Wallet) ProcDumpPrivkey(addr string) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return "", err
	}
	if len(addr) == 0 {
		walletlog.Error("ProcDumpPrivkey input para is nil!")
		return "", types.ErrInvalidParam
	}

	priv, err := wallet.getPrivKeyFromStore(addr)
	if err != nil {
		walletlog.Error("ProcDumpPrivkey", "getPrivKeyFromStore err", err)
		return "", err
	}
	return common.ToHex(priv), nil
}

// 收到其他模块上报的系统有致命性故障，需要通知前端
func (wallet *Wallet) setFatalFailure(reportErrEvent *types.ReportErrEvent) {

	walletlog.Error("setFatalFailure", "reportErrEvent", reportErrEvent.String())
	if reportErrEvent.Error == "ErrDataBaseDamage" {
		atomic.StoreInt32(&wallet.fatalFailureFlag, 1)
	}
}

func (wallet *Wallet) getFatalFailure() int32 {
	return atomic.LoadInt32(&wallet.fatalFailureFlag)
}

// 密码合法性校验,密码长度在8-30位之间。必须是数字+字母的组合
func isValidPassWord(password string) bool {
	pwLen := len(password)
	if pwLen < 8 || pwLen > 30 {
		return false
	}

	var char bool
	var digit bool
	for _, s := range password {
		if unicode.IsLetter(s) {
			char = true
		} else if unicode.IsDigit(s) {
			digit = true
		} else {
			return false
		}
	}
	return char && digit
}

// CreateNewAccountByIndex 指定index创建公私钥对，主要用于空投地址。目前暂定一千万
func (wallet *Wallet) createNewAccountByIndex(index uint32) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return "", err
	}

	if !isValidIndex(index) {
		walletlog.Error("createNewAccountByIndex index err", "index", index)
		return "", types.ErrInvalidParam
	}

	//空投地址是否已经存在，存在就直接返回存储的值即可
	airDropAddr, err := wallet.walletStore.GetAirDropIndex()
	if airDropAddr != "" && err == nil {
		priv, err := wallet.getPrivKeyByAddr(airDropAddr)
		if err != nil {
			return "", err
		}
		return common.ToHex(priv.Bytes()), nil
	}

	var addr string
	var privkeybyte []byte
	var HexPubkey string
	var isUsed bool

	cointype := wallet.GetCoinType()

	//通过seed获取私钥, 首先通过钱包密码解锁seed然后通过seed生成私钥
	seed, err := wallet.getSeed(wallet.Password)
	if err != nil {
		walletlog.Error("createNewAccountByIndex", "getSeed err", err)
		return "", err
	}

	// 通过指定index生成公私钥对，并存入数据库中，如果账户已经存在就直接返回账户信息即可
	privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.GetDB(), seed, index, wallet.SignType, wallet.CoinType)
	if err != nil {
		walletlog.Error("createNewAccountByIndex", "GetPrivkeyBySeed err", err)
		return "", err
	}
	privkeybyte, err = common.FromHex(privkeyhex)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("createNewAccountByIndex", "FromHex err", err)
		return "", err
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, uint32(wallet.SignType), privkeybyte)
	if err != nil {
		seedlog.Error("createNewAccountByIndex PrivkeyToPub", "err", err)
		return "", types.ErrPrivkeyToPub
	}

	HexPubkey = hex.EncodeToString(pub)

	addr, err = bipwallet.PubToAddress(pub)
	if err != nil {
		seedlog.Error("createNewAccountByIndex PubToAddress", "err", err)
		return "", types.ErrPrivkeyToPub
	}
	//通过新生成的账户地址查询钱包数据库，如果查询返回的账户信息不为空，
	//说明此账户已经被使用,不需要再次存储账户信息
	account, err := wallet.walletStore.GetAccountByAddr(addr)
	if account != nil && err == nil {
		isUsed = true
	}

	//第一次创建此账户
	if !isUsed {
		Account := types.Account{
			Addr:     addr,
			Currency: 0,
			Balance:  0,
			Frozen:   0,
		}
		//首先校验label是否已被使用
		Label := "airdropaddr"
		for {
			i := 0
			WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label)
			if WalletAccStores != nil && err == nil {
				walletlog.Debug("createNewAccountByIndex Label is exist in wallet!", "WalletAccStores", WalletAccStores)
				i++
				Label = Label + fmt.Sprintf("%d", i)
			} else {
				break
			}
		}

		walletAccount := types.WalletAccount{
			Acc:   &Account,
			Label: Label,
		}

		//使用钱包的password对私钥加密 aes cbc
		Encrypted := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)

		var WalletAccStore types.WalletAccountStore
		WalletAccStore.Privkey = common.ToHex(Encrypted)
		WalletAccStore.Label = Label
		WalletAccStore.Addr = addr

		//存储账户信息到wallet数据库中
		err = wallet.walletStore.SetWalletAccount(false, Account.Addr, &WalletAccStore)
		if err != nil {
			return "", err
		}

		//获取地址对应的账户信息从account模块
		addrs := make([]string, 1)
		addrs[0] = addr
		accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
		if err != nil {
			walletlog.Error("createNewAccountByIndex", "LoadAccounts err", err)
			return "", err
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
	}
	//存贮空投地址的信息
	airfrop := &wcom.AddrInfo{
		Index:  index,
		Addr:   addr,
		Pubkey: HexPubkey,
	}
	err = wallet.walletStore.SetAirDropIndex(airfrop)
	if err != nil {
		walletlog.Error("createNewAccountByIndex", "SetAirDropIndex err", err)
	}
	return privkeyhex, nil
}

// isValidIndex校验index的合法性
func isValidIndex(index uint32) bool {
	if types.AirDropMinIndex <= index && index <= types.AirDropMaxIndex {
		return true
	}
	return false
}

// ProcDumpPrivkeysFile 获取全部私钥保存到文件
func (wallet *Wallet) ProcDumpPrivkeysFile(fileName, passwd string) error {
	_, err := os.Stat(fileName)
	if err == nil {
		walletlog.Error("ProcDumpPrivkeysFile file already exists!", "fileName", fileName)
		return types.ErrFileExists
	}

	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return err
	}

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		walletlog.Error("ProcDumpPrivkeysFile create file error!", "fileName", fileName, "err", err)
		return err
	}
	defer f.Close()

	accounts, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(accounts) == 0 {
		walletlog.Info("ProcDumpPrivkeysFile GetWalletAccounts", "GetAccountByPrefix:err", err)
		return err
	}

	for i, acc := range accounts {
		priv, err := wallet.getPrivKeyByAddr(acc.Addr)
		if err != nil {
			walletlog.Info("getPrivKeyByAddr", acc.Addr, err)
			continue
		}

		privkey := common.ToHex(priv.Bytes())
		content := privkey + "& *.prickey.+.label.* &" + acc.Label

		Encrypter, err := AesgcmEncrypter([]byte(passwd), []byte(content))
		if err != nil {
			walletlog.Error("ProcDumpPrivkeysFile AesgcmEncrypter fileContent error!", "fileName", fileName, "err", err)
			continue
		}

		f.WriteString(string(Encrypter))

		if i < len(accounts)-1 {
			f.WriteString("&ffzm.&**&")
		}
	}

	return nil
}

// ProcImportPrivkeysFile 处理导入私钥
// input:
//
//	type  struct {
//		fileName string
//	 passwd   string
//
// 导入私钥，并且同时会导入交易
func (wallet *Wallet) ProcImportPrivkeysFile(fileName, passwd string) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		walletlog.Error("ProcImportPrivkeysFile file is not exist!", "fileName", fileName)
		return err
	}

	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return err
	}

	f, err := os.Open(fileName)
	if err != nil {
		walletlog.Error("ProcImportPrivkeysFile Open file error", "fileName", fileName, "err", err)
		return err
	}
	defer f.Close()

	fileContent, err := ioutil.ReadAll(f)
	if err != nil {
		walletlog.Error("ProcImportPrivkeysFile read file error", "fileName", fileName, "err", err)
		return err
	}
	accounts := strings.Split(string(fileContent), "&ffzm.&**&")
	for _, value := range accounts {
		Decrypter, err := AesgcmDecrypter([]byte(passwd), []byte(value))
		if err != nil {
			walletlog.Error("ProcImportPrivkeysFile AesgcmDecrypter fileContent error", "fileName", fileName, "err", err)
			return types.ErrVerifyOldpasswdFail
		}

		acc := strings.Split(string(Decrypter), "& *.prickey.+.label.* &")
		if len(acc) != 2 {
			walletlog.Error("ProcImportPrivkeysFile len(acc) != 2, File format error.", "Decrypter", string(Decrypter), "len", len(acc))
			continue
		}
		privKey := acc[0]
		label := acc[1]

		//校验label是否已经被使用
		Account, err := wallet.walletStore.GetAccountByLabel(label)
		if Account != nil && err == nil {
			walletlog.Info("ProcImportPrivKey Label is exist in wallet, label = label + _2!")
			label = label + "_2"
		}

		PrivKey := &types.ReqWalletImportPrivkey{
			Privkey: privKey,
			Label:   label,
		}

		_, err = wallet.procImportPrivKey(PrivKey)
		if err == types.ErrPrivkeyExist {
			// 修改标签名称 是否需要?
			// wallet.ProcWalletSetLabel()
		} else if err != nil {
			walletlog.Info("ProcImportPrivKey procImportPrivKey error")
			return err
		}
	}

	return nil
}
