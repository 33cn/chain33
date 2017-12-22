package rpc

import (
	l "github.com/inconshreveable/log15"
)

var log = l.New("module", "rpc")

type JTransparm struct {
	Execer    string     `json:"execer"`
	Payload   string     `json:"payload"`
	Signature *Signature `json:"signature"`
	Fee       int64      `json:"fee"`
}

type RawParm struct {
	Data string `json:"data"`
}
type QueryParm struct {
	Hash string `json:"hash"`
}
type BlockParam struct {
	Start    int64 `json:"start"`
	End      int64 `json:"end"`
	Isdetail bool  `json:"isdetail"`
}

type Header struct {
	Version    int64  `json:"version"`
	ParentHash string `json:"parenthash"`
	TxHash     string `json:"txhash"`
	StateHash  string `json:"statehash"`
	Height     int64  `json:"height"`
	BlockTime  int64  `json:"blocktime"`
}

type Signature struct {
	Ty        int32  `json:"ty"`
	Pubkey    string `json:"pubkey"`
	Signature string `json:"signature"`
}
type Transaction struct {
	Execer    string     `json:"execer"`
	Payload   string     `json:"payload"`
	Signature *Signature `json:"signature"`
	Fee       int64      `json:"fee"`
	Expire    int64      `json:"expire"`
	Nonce     int64      `json:"nonce"`
	To        string     `json:"to"`
}
type ReceiptLog struct {
	Ty  int32  `json:"ty"`
	Log string `json:"log"`
}
type ReceiptData struct {
	Ty   int32         `json:"ty"`
	Logs []*ReceiptLog `json:"logs"`
}
type Block struct {
	Version    int64          `json:"version"`
	ParentHash string         `json:"parenthash"`
	TxHash     string         `json:"txhash"`
	StateHash  string         `json:"statehash"`
	Height     int64          `json:"height"`
	BlockTime  int64          `json:"blocktime"`
	Txs        []*Transaction `json:"txs"`
}
type BlockDetail struct {
	Block    *Block         `json:"block"`
	Receipts []*ReceiptData `json:"recipts"`
}

type BlockDetails struct {
	Items []*BlockDetail `json:"items"`
}

type TransactionDetail struct {
	Tx      *Transaction `json:"tx"`
	Receipt *ReceiptData `json:"receipt"`
	Proofs  []string     `json:"proofs"`
}

type ReplyTxInfos struct {
	TxInfos []*ReplyTxInfo `protobuf:"bytes,1,rep,name=txInfos" json:"txinfos,omitempty"`
}

type ReplyTxInfo struct {
	Hash   string `json:"hash"`
	Height int64  `json:"height"`
	Index  int64  `json:"index"`
}
type TransactionDetails struct {
	Txs []*Transaction `json:"txs"`
}

type ReplyTxList struct {
	Txs []*Transaction `json:"txs"`
}
type ReplyHash struct {
	Hash string `json:"hash"`
}
type ReplyHashes struct {
	Hashes []string `json:hashes`
}
type PeerList struct {
	Peers []*Peer `json:"peers"`
}
type Peer struct {
	Addr        string  `json:"addr"`
	Port        int32   `json:"port"`
	Name        string  `json:"name"`
	MempoolSize int32   `json:"mempoolsize"`
	Header      *Header `json:"header"`
}

// Wallet Module
type WalletAccounts struct {
	Wallets []*WalletAccount `protobuf:"bytes,1,rep,name=wallets" json:"wallets"`
}
type WalletAccount struct {
	Acc   *Account `protobuf:"bytes,1,opt,name=acc" json:"acc"`
	Label string   `protobuf:"bytes,2,opt,name=label" json:"label"`
}

type Account struct {
	Currency int32  `protobuf:"varint,1,opt,name=currency" json:"currency"`
	Balance  int64  `protobuf:"varint,2,opt,name=balance" json:"balance"`
	Frozen   int64  `protobuf:"varint,3,opt,name=frozen" json:"frozen"`
	Addr     string `protobuf:"bytes,4,opt,name=addr" json:"addr"`
}
type Reply struct {
	IsOk bool   `protobuf:"varint,1,opt,name=isOk" json:"isok"`
	Msg  string `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg"`
}
type Headers struct {
	Items []*Header `protobuf:"bytes,1,rep,name=items" json:"items"`
}

type ReqAddr struct {
	Addr string `json:"items"`
}

type ReqHashes struct {
	Hashes []string `json:"items"`
}

type ReqWalletTransactionList struct {
	FromTx    string `json:"fromtx"`
	Count     int32  `json:"count"`
	Direction int32  `json:"direction"`
}
type WalletTxDetails struct {
	TxDetails []*WalletTxDetail `protobuf:"bytes,1,rep,name=txDetails" json:"txDetails"`
}
type WalletTxDetail struct {
	Tx      *Transaction `protobuf:"bytes,1,opt,name=tx" json:"tx"`
	Receipt *ReceiptData `protobuf:"bytes,2,opt,name=receipt" json:"receipt"`
	Height  int64        `protobuf:"varint,3,opt,name=height" json:"height"`
	Index   int64        `protobuf:"varint,4,opt,name=index" json:"index"`
}
