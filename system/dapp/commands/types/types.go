// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types commands中结构体定义
package types

import (
	rpctypes "github.com/33cn/chain33/rpc/types"
)

// AccountsResult defines accountsresult command
type AccountsResult struct {
	Wallets []*WalletResult `json:"wallets"`
}

// WalletResult defines walletresult command
type WalletResult struct {
	Acc   *AccountResult `json:"acc,omitempty"`
	Label string         `json:"label,omitempty"`
}

// AccountResult defines account result command
type AccountResult struct {
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

// TokenAccountResult defines accounts result of token command
type TokenAccountResult struct {
	Token    string `json:"Token,omitempty"`
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

// TxListResult defines txlist result command
type TxListResult struct {
	Txs []*TxResult `json:"txs"`
}

// TxResult defines txresult command
type TxResult struct {
	Execer     string              `json:"execer"`
	Payload    interface{}         `json:"payload"`
	RawPayload string              `json:"rawpayload"`
	Signature  *rpctypes.Signature `json:"signature"`
	Fee        string              `json:"fee"`
	Expire     int64               `json:"expire"`
	Nonce      int64               `json:"nonce"`
	To         string              `json:"to"`
	Amount     string              `json:"amount,omitempty"`
	From       string              `json:"from,omitempty"`
	GroupCount int32               `json:"groupCount,omitempty"`
	Header     string              `json:"header,omitempty"`
	Next       string              `json:"next,omitempty"`
	Hash       string              `json:"hash,omitempty"`
	ChannelID  int32               `json:"channelID,"`
}

// ReceiptAccountTransfer defines receipt account transfer
type ReceiptAccountTransfer struct {
	Prev    *AccountResult `protobuf:"bytes,1,opt,name=prev" json:"prev,omitempty"`
	Current *AccountResult `protobuf:"bytes,2,opt,name=current" json:"current,omitempty"`
}

// ReceiptExecAccountTransfer defines account transfer of exec command
type ReceiptExecAccountTransfer struct {
	ExecAddr string         `protobuf:"bytes,1,opt,name=execAddr" json:"execAddr,omitempty"`
	Prev     *AccountResult `protobuf:"bytes,2,opt,name=prev" json:"prev,omitempty"`
	Current  *AccountResult `protobuf:"bytes,3,opt,name=current" json:"current,omitempty"`
}

// TxDetailResult defines txdetail result command
type TxDetailResult struct {
	Tx         *TxResult                   `json:"tx"`
	Receipt    *rpctypes.ReceiptDataResult `json:"receipt"`
	Proofs     []string                    `json:"proofs,omitempty"`
	Height     int64                       `json:"height"`
	Index      int64                       `json:"index"`
	Blocktime  int64                       `json:"blocktime"`
	Amount     string                      `json:"amount"`
	Fromaddr   string                      `json:"fromaddr"`
	ActionName string                      `json:"actionname"`
	Assets     []*rpctypes.Asset           `json:"assets"`
	TxProofs   []*rpctypes.TxProof         `json:"txProofs"`
	FullHash   string                      `json:"fullHash"`
}

// TxDetailsResult defines txdetails result command
type TxDetailsResult struct {
	Txs []*TxDetailResult `json:"txs"`
}

// BlockResult defines blockresult rpc command
type BlockResult struct {
	Version    int64       `json:"version"`
	ParentHash string      `json:"parenthash"`
	TxHash     string      `json:"txhash"`
	StateHash  string      `json:"statehash"`
	Height     int64       `json:"height"`
	BlockTime  int64       `json:"blocktime"`
	Txs        []*TxResult `json:"txs"`
}

// BlockDetailResult defines blockdetailresult rpc command
type BlockDetailResult struct {
	Block    *BlockResult                  `json:"block"`
	Receipts []*rpctypes.ReceiptDataResult `json:"receipts"`
}

// BlockDetailsResult defines blockdetails result rpc command
type BlockDetailsResult struct {
	Items []*BlockDetailResult `json:"items"`
}

// WalletTxDetailsResult defines walletexdetails result rpc command
type WalletTxDetailsResult struct {
	TxDetails []*WalletTxDetailResult `json:"txDetails"`
}

// WalletTxDetailResult  defines wallettxdetail result rpc command
type WalletTxDetailResult struct {
	Tx         *TxResult                   `json:"tx"`
	Receipt    *rpctypes.ReceiptDataResult `json:"receipt"`
	Height     int64                       `json:"height"`
	Index      int64                       `json:"index"`
	Blocktime  int64                       `json:"blocktime"`
	Amount     string                      `json:"amount"`
	Fromaddr   string                      `json:"fromaddr"`
	Txhash     string                      `json:"txhash"`
	ActionName string                      `json:"actionname"`
}

// AddrOverviewResult defines address overview result rpc command
type AddrOverviewResult struct {
	Receiver string `json:"receiver"`
	Balance  string `json:"balance"`
	TxCount  int64  `json:"txCount"`
}

// GetTotalCoinsResult defines totalcoinsresult rpc command
type GetTotalCoinsResult struct {
	TxCount          int64  `json:"txCount"`
	AccountCount     int64  `json:"accountCount"`
	TotalAmount      string `json:"totalAmount"`
	ActualAmount     string `json:"actualAmount,omitempty"`
	DifferenceAmount string `json:"differenceAmount,omitempty"`
}

// AllExecBalance defines all balance of exec command
type AllExecBalance struct {
	Addr        string         `json:"addr"`
	ExecAccount []*ExecAccount `json:"execAccount"`
}

// ExecAccount defines account of exec command
type ExecAccount struct {
	Execer  string         `json:"execer"`
	Account *AccountResult `json:"account"`
}

// GetExecBalanceResult  defines balance of exec result rpc command
type GetExecBalanceResult struct {
	Amount       string         `json:"totalAmount"`
	AmountFrozen string         `json:"frozenAmount"`
	AmountActive string         `json:"activeAmount"`
	ExecBalances []*ExecBalance `json:"execBalances,omitempty"`
}

// ExecBalance defines exec balance rpc command
type ExecBalance struct {
	ExecAddr string `json:"execAddr,omitempty"`
	Frozen   string `json:"frozen"`
	Active   string `json:"active"`
}
