// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package rpc chain33 RPC模块包含JSONRpc以及grpc
package rpc

import (
	"encoding/hex"
	"time"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	ety "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
)

var log = log15.New("module", "rpc")

type channelClient struct {
	client.QueueProtocolAPI
	accountdb *account.DB
}

// Init channel client
func (c *channelClient) Init(q queue.Client, api client.QueueProtocolAPI) {
	if api == nil {
		var err error
		api, err = client.New(q, nil)
		if err != nil {
			panic(err)
		}
	}
	c.QueueProtocolAPI = api
	c.accountdb = account.NewCoinsAccount()
}

// CreateRawTransaction create rawtransaction
func (c *channelClient) CreateRawTransaction(param *types.CreateTx) ([]byte, error) {
	if param == nil {
		log.Error("CreateRawTransaction", "Error", types.ErrInvalidParam)
		return nil, types.ErrInvalidParam
	}
	//因为历史原因，这里还是有部分token 的字段，但是没有依赖token dapp
	//未来这个调用可能会被废弃
	execer := types.ExecName(ety.CoinsX)
	if param.IsToken {
		execer = types.ExecName("token")
	}
	if param.Execer != "" {
		execer = param.Execer
	}
	return types.CallCreateTx(execer, "", param)
}

func (c *channelClient) ReWriteRawTx(param *types.ReWriteRawTx) ([]byte, error) {
	if param == nil || param.Tx == "" {
		log.Error("ReWriteRawTx", "Error", types.ErrInvalidParam)
		return nil, types.ErrInvalidParam
	}

	tx, err := decodeTx(param.Tx)
	if err != nil {
		return nil, err
	}
	if param.To != "" {
		tx.To = param.To
	}
	if param.Fee != 0 {
		tx.Fee = param.Fee
	}
	if param.Expire != "" {
		expire, err := types.ParseExpire(param.Expire)
		if err != nil {
			return nil, err
		}
		tx.Expire = expire
	}

	return types.FormatTxEncode(string(tx.Execer), tx)
}

// CreateRawTxGroup create rawtransaction for group
func (c *channelClient) CreateRawTxGroup(param *types.CreateTransactionGroup) ([]byte, error) {
	if param == nil || len(param.Txs) <= 1 {
		return nil, types.ErrTxGroupCountLessThanTwo
	}
	var transactions []*types.Transaction
	for _, t := range param.Txs {
		txByte, err := hex.DecodeString(t)
		if err != nil {
			return nil, err
		}
		var transaction types.Transaction
		err = types.Decode(txByte, &transaction)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, &transaction)
	}
	group, err := types.CreateTxGroup(transactions)
	if err != nil {
		return nil, err
	}

	txGroup := group.Tx()
	txHex := types.Encode(txGroup)
	return txHex, nil
}

// CreateNoBalanceTransaction create the transaction with no balance
func (c *channelClient) CreateNoBalanceTransaction(in *types.NoBalanceTx) (*types.Transaction, error) {
	txNone := &types.Transaction{Execer: []byte(types.ExecName(types.NoneX)), Payload: []byte("no-fee-transaction")}
	txNone.To = address.ExecAddress(string(txNone.Execer))
	txNone, err := types.FormatTx(types.ExecName(types.NoneX), txNone)
	if err != nil {
		return nil, err
	}
	tx, err := decodeTx(in.TxHex)
	if err != nil {
		return nil, err
	}
	transactions := []*types.Transaction{txNone, tx}
	group, err := types.CreateTxGroup(transactions)
	if err != nil {
		return nil, err
	}
	err = group.Check(0, types.GInt("MinFee"), types.GInt("MaxFee"))
	if err != nil {
		return nil, err
	}
	newtx := group.Tx()
	//如果可能要做签名
	if in.PayAddr != "" || in.Privkey != "" {
		rawTx := hex.EncodeToString(types.Encode(newtx))
		req := &types.ReqSignRawTx{Addr: in.PayAddr, Privkey: in.Privkey, Expire: in.Expire, TxHex: rawTx, Index: 1}
		signedTx, err := c.SignRawTx(req)
		if err != nil {
			return nil, err
		}
		return decodeTx(signedTx.TxHex)
	}
	return newtx, nil
}

func decodeTx(hexstr string) (*types.Transaction, error) {
	var tx types.Transaction
	data, err := hex.DecodeString(hexstr)
	if err != nil {
		return nil, err
	}
	err = types.Decode(data, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// GetAddrOverview get overview of address
func (c *channelClient) GetAddrOverview(parm *types.ReqAddr) (*types.AddrOverview, error) {
	err := address.CheckAddress(parm.Addr)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}
	reply, err := c.QueueProtocolAPI.GetAddrOverview(parm)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	//获取地址账户的余额通过account模块
	addrs := make([]string, 1)
	addrs[0] = parm.Addr
	accounts, err := c.accountdb.LoadAccounts(c.QueueProtocolAPI, addrs)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	if len(accounts) != 0 {
		reply.Balance = accounts[0].Balance
	}
	return reply, nil
}

// GetBalance get balance
func (c *channelClient) GetBalance(in *types.ReqBalance) ([]*types.Account, error) {
	// in.AssetExec & in.AssetSymbol 新增参数，
	// 不填时兼容原来的调用
	if in.AssetExec == "" || in.AssetSymbol == "" {
		in.AssetSymbol = "bty"
		in.AssetExec = "coins"
		return c.accountdb.GetBalance(c.QueueProtocolAPI, in)
	}

	acc, err := account.NewAccountDB(in.AssetExec, in.AssetSymbol, nil)
	if err != nil {
		log.Error("GetBalance", "Error", err.Error())
		return nil, err
	}
	return acc.GetBalance(c.QueueProtocolAPI, in)
}

// GetAllExecBalance get balance of exec
func (c *channelClient) GetAllExecBalance(in *types.ReqAllExecBalance) (*types.AllExecBalance, error) {
	addr := in.Addr
	err := address.CheckAddress(addr)
	if err != nil {
		if err = address.CheckMultiSignAddress(addr); err != nil {
			return nil, types.ErrInvalidAddress
		}
	}
	var addrs []string
	addrs = append(addrs, addr)
	allBalance := &types.AllExecBalance{Addr: addr}
	for _, exec := range types.AllowUserExec {
		execer := types.ExecName(string(exec))
		params := &types.ReqBalance{
			Addresses:   addrs,
			Execer:      execer,
			StateHash:   in.StateHash,
			AssetExec:   in.AssetExec,
			AssetSymbol: in.AssetSymbol,
		}
		res, err := c.GetBalance(params)
		if err != nil {
			continue
		}
		if len(res) < 1 {
			continue
		}
		acc := res[0]
		if acc.Balance == 0 && acc.Frozen == 0 {
			continue
		}
		execAcc := &types.ExecAccount{Execer: execer, Account: acc}
		allBalance.ExecAccount = append(allBalance.ExecAccount, execAcc)
	}
	return allBalance, nil
}

// GetTotalCoins get total of coins
func (c *channelClient) GetTotalCoins(in *types.ReqGetTotalCoins) (*types.ReplyGetTotalCoins, error) {
	//获取地址账户的余额通过account模块
	resp, err := c.accountdb.GetTotalCoins(c.QueueProtocolAPI, in)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DecodeRawTransaction decode rawtransaction
func (c *channelClient) DecodeRawTransaction(param *types.ReqDecodeRawTransaction) (*types.Transaction, error) {
	var tx types.Transaction
	bytes, err := common.FromHex(param.TxHex)
	if err != nil {
		return nil, err
	}
	err = types.Decode(bytes, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// GetTimeStatus get status of time
func (c *channelClient) GetTimeStatus() (*types.TimeStatus, error) {
	ntpTime := common.GetRealTimeRetry(types.NtpHosts, 10)
	local := types.Now()
	if ntpTime.IsZero() {
		return &types.TimeStatus{NtpTime: "", LocalTime: local.Format("2006-01-02 15:04:05"), Diff: 0}, nil
	}
	diff := local.Sub(ntpTime) / time.Second
	return &types.TimeStatus{NtpTime: ntpTime.Format("2006-01-02 15:04:05"), LocalTime: local.Format("2006-01-02 15:04:05"), Diff: int64(diff)}, nil
}

// GetExecBalance get balance with exec by channelclient
func (c *channelClient) GetExecBalance(in *types.ReqGetExecBalance) (*types.ReplyGetExecBalance, error) {
	//通过account模块获取地址账户在合约中的余额
	resp, err := c.accountdb.GetExecBalance(c.QueueProtocolAPI, in)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
