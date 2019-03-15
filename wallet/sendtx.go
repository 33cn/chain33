// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/hex"
	"errors"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
)

func init() {
	rand.Seed(types.Now().UnixNano())
}

// GetBalance 根据地址和执行器类型获取对应的金额
func (wallet *Wallet) GetBalance(addr string, execer string) (*types.Account, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	return wallet.getBalance(addr, execer)
}

func (wallet *Wallet) getBalance(addr string, execer string) (*types.Account, error) {
	reqbalance := &types.ReqBalance{Addresses: []string{addr}, Execer: execer}
	reply, err := wallet.queryBalance(reqbalance)
	if err != nil {
		return nil, err
	}
	return reply[0], nil
}

// GetAllPrivKeys 获取所有私钥信息
func (wallet *Wallet) GetAllPrivKeys() ([]crypto.PrivKey, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	return wallet.getAllPrivKeys()
}

func (wallet *Wallet) getAllPrivKeys() ([]crypto.PrivKey, error) {
	accounts, err := wallet.GetWalletAccounts()
	if err != nil {
		return nil, err
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	ok, err := wallet.CheckWalletStatus()
	if !ok && err != types.ErrOnlyTicketUnLocked {
		return nil, err
	}
	var privs []crypto.PrivKey
	for _, acc := range accounts {
		priv, err := wallet.getPrivKeyByAddr(acc.Addr)
		if err != nil {
			return nil, err
		}
		privs = append(privs, priv)
	}
	return privs, nil
}

// GetHeight 获取当前区块最新高度
func (wallet *Wallet) GetHeight() int64 {
	if !wallet.isInited() {
		return 0
	}
	msg := wallet.client.NewMessage("blockchain", types.EventGetBlockHeight, nil)
	err := wallet.client.Send(msg, true)
	if err != nil {
		return 0
	}
	replyHeight, err := wallet.client.Wait(msg)
	h := replyHeight.GetData().(*types.ReplyBlockHeight).Height
	walletlog.Debug("getheight = ", "height", h)
	if err != nil {
		return 0
	}
	return h
}

func (wallet *Wallet) sendTransactionWait(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (err error) {
	hash, err := wallet.sendTransaction(payload, execer, priv, to)
	if err != nil {
		return err
	}
	txinfo := wallet.waitTx(hash)
	if txinfo.Receipt.Ty != types.ExecOk {
		return errors.New("sendTransactionWait error")
	}
	return nil
}

// SendTransaction 发送一笔交易
func (wallet *Wallet) SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	return wallet.sendTransaction(payload, execer, priv, to)
}

func (wallet *Wallet) sendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	if to == "" {
		to = address.ExecAddress(string(execer))
	}
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(payload), Fee: minFee, To: to}
	tx.Nonce = rand.Int63()
	tx.Fee, err = tx.GetRealFee(wallet.getFee())
	if err != nil {
		return nil, err
	}
	tx.SetExpire(time.Second * 120)
	tx.Sign(int32(SignType), priv)
	reply, err := wallet.sendTx(tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		walletlog.Info("wallet sendTransaction", "err", string(reply.GetMsg()))
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func (wallet *Wallet) sendTx(tx *types.Transaction) (*types.Reply, error) {
	if wallet.client == nil {
		panic("client not bind message queue.")
	}
	return wallet.api.SendTx(tx)
}

// WaitTx 等待交易确认
func (wallet *Wallet) WaitTx(hash []byte) *types.TransactionDetail {
	return wallet.waitTx(hash)
}

func (wallet *Wallet) waitTx(hash []byte) *types.TransactionDetail {
	i := 0
	for {
		if atomic.LoadInt32(&wallet.isclosed) == 1 {
			return nil
		}
		i++
		if i%100 == 0 {
			walletlog.Error("wait transaction timeout", "hash", hex.EncodeToString(hash))
			return nil
		}
		res, err := wallet.queryTx(hash)
		if err != nil {
			time.Sleep(time.Second)
		}
		if res != nil {
			return res
		}
	}
}

// WaitTxs 等待多个交易确认
func (wallet *Wallet) WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	return wallet.waitTxs(hashes)
}

func (wallet *Wallet) waitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	for _, hash := range hashes {
		result := wallet.waitTx(hash)
		ret = append(ret, result)
	}
	return ret
}

func (wallet *Wallet) queryTx(hash []byte) (*types.TransactionDetail, error) {
	msg := wallet.client.NewMessage("blockchain", types.EventQueryTx, &types.ReqHash{Hash: hash})
	err := wallet.client.Send(msg, true)
	if err != nil {
		walletlog.Error("QueryTx", "Error", err.Error())
		return nil, err
	}
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.TransactionDetail), nil
}

// SendToAddress 想合约地址转账
func (wallet *Wallet) SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	return wallet.sendToAddress(priv, addrto, amount, note, Istoken, tokenSymbol)
}

func (wallet *Wallet) createSendToAddress(addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.Transaction, error) {
	var tx *types.Transaction
	var isWithdraw = false
	if amount < 0 {
		amount = -amount
		isWithdraw = true
	}
	create := &types.CreateTx{
		To:          addrto,
		Amount:      amount,
		Note:        []byte(note),
		IsWithdraw:  isWithdraw,
		IsToken:     Istoken,
		TokenSymbol: tokenSymbol,
	}

	exec := cty.CoinsX
	//历史原因，token是作为系统合约的,但是改版后，token变成非系统合约
	//这样的情况下，的方案是做一些特殊的处理
	if create.IsToken {
		exec = "token"
	}
	ety := types.LoadExecutorType(exec)
	if ety == nil {
		return nil, types.ErrActionNotSupport
	}
	tx, err := ety.AssertCreate(create)
	if err != nil {
		return nil, err
	}
	tx.SetExpire(time.Second * 120)
	fee, err := tx.GetRealFee(wallet.getFee())
	if err != nil {
		return nil, err
	}
	tx.Fee = fee
	if tx.To == "" {
		tx.To = addrto
	}
	if len(tx.Execer) == 0 {
		tx.Execer = []byte(exec)
	}
	tx.Nonce = rand.Int63()
	return tx, nil
}

func (wallet *Wallet) sendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	tx, err := wallet.createSendToAddress(addrto, amount, note, Istoken, tokenSymbol)
	if err != nil {
		return nil, err
	}
	tx.Sign(int32(SignType), priv)

	reply, err := wallet.api.SendTx(tx)
	if err != nil {
		return nil, err
	}
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()
	return &hash, nil
}

func (wallet *Wallet) queryBalance(in *types.ReqBalance) ([]*types.Account, error) {

	switch in.GetExecer() {
	case "coins":
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := address.CheckAddress(addr); err != nil {
				addr = address.ExecAddress(addr)
			}
			exaddrs = append(exaddrs, addr)
		}
		accounts, err := accountdb.LoadAccounts(wallet.api, exaddrs)
		if err != nil {
			walletlog.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := address.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			acc, err := accountdb.LoadExecAccountQueue(wallet.api, addr, execaddress)
			if err != nil {
				walletlog.Error("GetBalance", "err", err.Error())
				return nil, err
			}
			accounts = append(accounts, acc)
		}
		return accounts, nil
	}
}

func (wallet *Wallet) getMinerColdAddr(addr string) ([]string, error) {
	reqaddr := &types.ReqString{Data: addr}
	req := types.ChainExecutor{
		Driver:   "ticket",
		FuncName: "MinerSourceList",
		Param:    types.Encode(reqaddr),
	}

	msg := wallet.client.NewMessage("exec", types.EventBlockChainQuery, &req)
	err := wallet.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	reply := resp.GetData().(types.Message).(*types.ReplyStrings)
	return reply.Datas, nil
}

// IsCaughtUp 检测当前区块是否已经同步完成
func (wallet *Wallet) IsCaughtUp() bool {
	if !wallet.isInited() {
		return false
	}
	if wallet.client == nil {
		panic("wallet client not bind message queue.")
	}
	reply, err := wallet.api.IsSync()
	if err != nil {
		return false
	}
	return reply.IsOk
}
