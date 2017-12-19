package rpc

import (
	l "github.com/inconshreveable/log15"
)

var log = l.New("module", "rpc")

type JTransparm struct {
	Execer    string
	Payload   string
	Signature *Signature
	Fee       int64
}

type RawParm struct {
	Data string
}

type Header struct {
	Version    int64
	ParentHash string
	TxHash     string
	StateHash  string
	Height     int64
	BlockTime  int64
}

type Signature struct {
	Ty        int32
	Pubkey    string
	Signature string
}
type Transaction struct {
	Execer    string
	Payload   string
	Signature *Signature
	Fee       int64
	Expire    int64
	Nonce     int64
	To        string
}
type ReceiptLog struct {
	Ty  int32
	Log string
}
type ReceiptData struct {
	Ty   int32
	Logs []*ReceiptLog
}
type Block struct {
	Version    int64
	ParentHash string
	TxHash     string
	StateHash  string
	Height     int64
	BlockTime  int64
	Txs        []*Transaction
}
type BlockDetail struct {
	Block    *Block
	Receipts []*ReceiptData
}

type BlockDetails struct {
	Items []*BlockDetail
}

type TransactionDetail struct {
	Tx      *Transaction
	Receipt *ReceiptData
	Proofs  []string
}

type ReplyTxInfos struct {
	TxInfos []*ReplyTxInfo `protobuf:"bytes,1,rep,name=txInfos" json:"txInfos,omitempty"`
}

type ReplyTxInfo struct {
	Hash   string
	Height int64
	Index  int64
}
type TransactionDetails struct {
	Txs []*Transaction
}

type ReplyTxList struct {
	Txs []*Transaction
}
type ReplyHash struct {
	Hash string
}
type ReplyHashes struct {
	Hashes []string
}
type PeerList struct {
	Peers []*Peer
}
type Peer struct {
	Addr        string
	Port        int32
	Name        string
	MempoolSize int32
	Header      *Header
}

type BlockParam struct {
	Start    int64
	End      int64
	Isdetail bool
}

type ReqAddr struct {
	Addr string
}

type QueryParm struct {
	Hash string
}

type ReqHashes struct {
	Hashes []string
}

type ReqWalletTransactionList struct {
	FromTx    string
	Count     int32
	Direction int32
}

// Wallet Module
type WalletAccounts struct {
	Wallets []*WalletAccount `protobuf:"bytes,1,rep,name=wallets" json:"wallets,omitempty"`
}
type WalletAccount struct {
	Acc   *Account `protobuf:"bytes,1,opt,name=acc" json:"acc,omitempty"`
	Label string   `protobuf:"bytes,2,opt,name=label" json:"label,omitempty"`
}

type Account struct {
	Currency int32  `protobuf:"varint,1,opt,name=currency" json:"currency,omitempty"`
	Balance  int64  `protobuf:"varint,2,opt,name=balance" json:"balance,omitempty"`
	Frozen   int64  `protobuf:"varint,3,opt,name=frozen" json:"frozen,omitempty"`
	Addr     string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
}
type Reply struct {
	IsOk bool   `protobuf:"varint,1,opt,name=isOk" json:"isOk,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
}
