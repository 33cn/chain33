// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types rpc相关的一些结构体定义以及转化函数
package types

import (
	"encoding/json"
)

// TransParm transport parameter
type TransParm struct {
	Execer    string     `json:"execer"`
	Payload   string     `json:"payload"`
	Signature *Signature `json:"signature"`
	Fee       int64      `json:"fee"`
}

// SignedTx signature tx
type SignedTx struct {
	Unsign string `json:"unsignTx"`
	Sign   string `json:"sign"`
	Pubkey string `json:"pubkey"`
	Ty     int32  `json:"ty"`
}

// RawParm defines raw parameter command
type RawParm struct {
	Token string `json:"token"`
	Data  string `json:"data"`
}

// QueryParm Query parameter
type QueryParm struct {
	Hash string `json:"hash"`
}

// BlockParam block parameter
type BlockParam struct {
	Start    int64 `json:"start"`
	End      int64 `json:"end"`
	Isdetail bool  `json:"isDetail"`
}

// Header header parameter
type Header struct {
	Version    int64      `json:"version"`
	ParentHash string     `json:"parentHash"`
	TxHash     string     `json:"txHash"`
	StateHash  string     `json:"stateHash"`
	Height     int64      `json:"height"`
	BlockTime  int64      `json:"blockTime"`
	TxCount    int64      `json:"txCount"`
	Hash       string     `json:"hash"`
	Difficulty uint32     `json:"difficulty"`
	Signature  *Signature `json:"signature,omitempty"`
}

// Signature parameter
type Signature struct {
	Ty        int32  `json:"ty"`
	Pubkey    string `json:"pubkey"`
	Signature string `json:"signature"`
}

// Transaction parameter
type Transaction struct {
	Execer     string          `json:"execer"`
	Payload    json.RawMessage `json:"payload"`
	RawPayload string          `json:"rawPayload"`
	Signature  *Signature      `json:"signature"`
	Fee        int64           `json:"fee"`
	FeeFmt     string          `json:"feefmt"`
	Expire     int64           `json:"expire"`
	Nonce      int64           `json:"nonce"`
	From       string          `json:"from,omitempty"`
	To         string          `json:"to"`
	Amount     int64           `json:"amount,omitempty"`
	AmountFmt  string          `json:"amountfmt,omitempty"`
	GroupCount int32           `json:"groupCount,omitempty"`
	Header     string          `json:"header,omitempty"`
	Next       string          `json:"next,omitempty"`
	Hash       string          `json:"hash,omitempty"`
}

// ReceiptLog defines receipt log command
type ReceiptLog struct {
	Ty  int32  `json:"ty"`
	Log string `json:"log"`
}

// ReceiptData defines receipt data rpc command
type ReceiptData struct {
	Ty   int32         `json:"ty"`
	Logs []*ReceiptLog `json:"logs"`
}

// ReceiptDataResult receipt data result
type ReceiptDataResult struct {
	Ty     int32               `json:"ty"`
	TyName string              `json:"tyName"`
	Logs   []*ReceiptLogResult `json:"logs"`
}

// ReceiptLogResult receipt log result
type ReceiptLogResult struct {
	Ty     int32           `json:"ty"`
	TyName string          `json:"tyName"`
	Log    json.RawMessage `json:"log"`
	RawLog string          `json:"rawLog"`
}

// Block block information
type Block struct {
	Version    int64          `json:"version"`
	ParentHash string         `json:"parentHash"`
	TxHash     string         `json:"txHash"`
	StateHash  string         `json:"stateHash"`
	Height     int64          `json:"height"`
	BlockTime  int64          `json:"blockTime"`
	Txs        []*Transaction `json:"txs"`
}

// BlockDetail  block detail
type BlockDetail struct {
	Block    *Block               `json:"block"`
	Receipts []*ReceiptDataResult `json:"recipts"`
}

// BlockDetails block details
type BlockDetails struct {
	Items []*BlockDetail `json:"items"`
}

// Asset asset
type Asset struct {
	Exec   string `json:"exec"`
	Symbol string `json:"symbol"`
	Amount int64  `json:"amount"`
}

// TransactionDetail transaction detail
type TransactionDetail struct {
	Tx         *Transaction       `json:"tx"`
	Receipt    *ReceiptDataResult `json:"receipt"`
	Proofs     []string           `json:"proofs"`
	Height     int64              `json:"height"`
	Index      int64              `json:"index"`
	Blocktime  int64              `json:"blockTime"`
	Amount     int64              `json:"amount"`
	Fromaddr   string             `json:"fromAddr"`
	ActionName string             `json:"actionName"`
	Assets     []*Asset           `json:"assets"`
}

// ReplyTxInfos reply tx infos
type ReplyTxInfos struct {
	TxInfos []*ReplyTxInfo `json:"txInfos"`
}

// ReplyTxInfo reply tx information
type ReplyTxInfo struct {
	Hash   string   `json:"hash"`
	Height int64    `json:"height"`
	Index  int64    `json:"index"`
	Assets []*Asset `json:"assets"`
}

// TransactionDetails transaction details
type TransactionDetails struct {
	//Txs []*Transaction `json:"txs"`
	Txs []*TransactionDetail `json:"txs"`
}

// ReplyTxList reply tx list
type ReplyTxList struct {
	Txs []*Transaction `json:"txs"`
}

// ReplyProperFee reply proper fee
type ReplyProperFee struct {
	ProperFee int64 `json:"properFee"`
}

// ReplyHash reply hash string json
type ReplyHash struct {
	Hash string `json:"hash"`
}

// ReplyHashes reply hashes
type ReplyHashes struct {
	Hashes []string `json:"hashes"`
}

// PeerList peer list
type PeerList struct {
	Peers []*Peer `json:"peers"`
}

// Peer  information
type Peer struct {
	Addr        string  `json:"addr"`
	Port        int32   `json:"port"`
	Name        string  `json:"name"`
	MempoolSize int32   `json:"mempoolSize"`
	Self        bool    `json:"self"`
	Header      *Header `json:"header"`
}

// WalletAccounts Wallet Module
type WalletAccounts struct {
	Wallets []*WalletAccount `json:"wallets"`
}

// WalletAccount  wallet account
type WalletAccount struct {
	Acc   *Account `json:"acc"`
	Label string   `json:"label"`
}

// Account account information
type Account struct {
	Currency int32  `json:"currency"`
	Balance  int64  `json:"balance"`
	Frozen   int64  `json:"frozen"`
	Addr     string `json:"addr"`
}

// Reply info
type Reply struct {
	IsOk bool   `json:"isOK"`
	Msg  string `json:"msg"`
}

// Headers defines headers rpc command
type Headers struct {
	Items []*Header `json:"items"`
}

// ReqAddr require address
type ReqAddr struct {
	Addr string `json:"addr"`
}

// ReqHashes require hashes
type ReqHashes struct {
	Hashes        []string `json:"hashes"`
	DisableDetail bool     `json:"disableDetail"`
}

// ReqWalletTransactionList require wallet transaction list
type ReqWalletTransactionList struct {
	FromTx          string `json:"fromTx"`
	Count           int32  `json:"count"`
	Direction       int32  `json:"direction"`
	Mode            int32  `json:"mode,omitempty"`
	SendRecvPrivacy int32  `json:"sendRecvPrivacy,omitempty"`
	Address         string `json:"address,omitempty"`
	TokenName       string `json:"tokenname,omitempty"`
}

// WalletTxDetails wallet tx details
type WalletTxDetails struct {
	TxDetails []*WalletTxDetail `json:"txDetails"`
}

// WalletTxDetail wallet tx detail
type WalletTxDetail struct {
	Tx         *Transaction       `json:"tx"`
	Receipt    *ReceiptDataResult `json:"receipt"`
	Height     int64              `json:"height"`
	Index      int64              `json:"index"`
	BlockTime  int64              `json:"blockTime"`
	Amount     int64              `json:"amount"`
	FromAddr   string             `json:"fromAddr"`
	TxHash     string             `json:"txHash"`
	ActionName string             `json:"actionName"`
}

// BlockOverview block overview
type BlockOverview struct {
	Head     *Header  `json:"head"`
	TxCount  int64    `json:"txCount"`
	TxHashes []string `json:"txHashes"`
}

// Query4Jrpc query jrpc
type Query4Jrpc struct {
	Execer   string          `json:"execer"`
	FuncName string          `json:"funcName"`
	Payload  json.RawMessage `json:"payload"`
}

// ChainExecutor chain executor
type ChainExecutor struct {
	Driver    string          `json:"execer"`
	FuncName  string          `json:"funcName"`
	StateHash string          `json:"stateHash"`
	Payload   json.RawMessage `json:"payload"`
}

// WalletStatus wallet status
type WalletStatus struct {
	IsWalletLock bool `json:"isWalletLock"`
	IsAutoMining bool `json:"isAutoMining"`
	IsHasSeed    bool `json:"isHasSeed"`
	IsTicketLock bool `json:"isTicketLock"`
}

// NodeNetinfo node net info
type NodeNetinfo struct {
	Externaladdr string `json:"externalAddr"`
	Localaddr    string `json:"localAddr"`
	Service      bool   `json:"service"`
	Outbounds    int32  `json:"outbounds"`
	Inbounds     int32  `json:"inbounds"`
}

// ReplyPrivacyPkPair   reply privekey pubkey pair
type ReplyPrivacyPkPair struct {
	ShowSuccessful bool   `json:"showSuccessful,omitempty"`
	ViewPub        string `json:"viewPub,omitempty"`
	SpendPub       string `json:"spendPub,omitempty"`
}

// ReplyCacheTxList reply cache tx list
type ReplyCacheTxList struct {
	Txs []*Transaction `json:"txs,omitempty"`
}

// TimeStatus time status
type TimeStatus struct {
	NtpTime   string `json:"ntpTime"`
	LocalTime string `json:"localTime"`
	Diff      int64  `json:"diff"`
}

// ReplyBlkSeqs reply block sequences
type ReplyBlkSeqs struct {
	BlkSeqInfos []*ReplyBlkSeq `json:"blkseqInfos"`
}

// ReplyBlkSeq reply block sequece
type ReplyBlkSeq struct {
	Hash string `json:"hash"`
	Type int64  `json:"type"`
}

// CreateTxIn create tx input
type CreateTxIn struct {
	Execer     string          `json:"execer"`
	ActionName string          `json:"actionName"`
	Payload    json.RawMessage `json:"payload"`
}

// AllExecBalance all exec balance
type AllExecBalance struct {
	Addr        string         `json:"addr"`
	ExecAccount []*ExecAccount `json:"execAccount"`
}

// ExecAccount exec account
type ExecAccount struct {
	Execer  string   `json:"execer"`
	Account *Account `json:"account"`
}

// ExecNameParm exec name parameter
type ExecNameParm struct {
	ExecName string `json:"execname"`
}

//CreateTx 为了简化Note 的创建过程，在json rpc 中，note 采用string 格式
type CreateTx struct {
	To          string `json:"to,omitempty"`
	Amount      int64  `json:"amount,omitempty"`
	Fee         int64  `json:"fee,omitempty"`
	Note        string `json:"note,omitempty"`
	IsWithdraw  bool   `json:"isWithdraw,omitempty"`
	IsToken     bool   `json:"isToken,omitempty"`
	TokenSymbol string `json:"tokenSymbol,omitempty"`
	ExecName    string `json:"execName,omitempty"` //TransferToExec and Withdraw 的执行器
	Execer      string `json:"execer,omitempty"`   //执行器名称
}

// ReWriteRawTx parameter
type ReWriteRawTx struct {
	Tx     string `json:"tx"`
	To     string `json:"to"`
	Fee    int64  `json:"fee"`
	Expire string `json:"expire"`
}
