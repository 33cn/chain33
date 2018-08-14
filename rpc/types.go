package rpc

import (
	"encoding/json"

	l "github.com/inconshreveable/log15"
)

var log = l.New("module", "rpc")

type userWrite struct {
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

type TransParm struct {
	Execer    string     `json:"execer"`
	Payload   string     `json:"payload"`
	Signature *Signature `json:"signature"`
	Fee       int64      `json:"fee"`
}

type SignedTx struct {
	Unsign string `json:"unsignTx"`
	Sign   string `json:"sign"`
	Pubkey string `json:"pubkey"`
	Ty     int32  `json:"ty"`
}

type RawParm struct {
	Token string `json:"token"`
	Data  string `json:"data"`
}

type QueryParm struct {
	Hash string `json:"hash"`
}

type BlockParam struct {
	Start    int64 `json:"start"`
	End      int64 `json:"end"`
	Isdetail bool  `json:"isDetail"`
}

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

type Signature struct {
	Ty        int32  `json:"ty"`
	Pubkey    string `json:"pubkey"`
	Signature string `json:"signature"`
}

type Transaction struct {
	Execer     string      `json:"execer"`
	Payload    interface{} `json:"payload"`
	RawPayload string      `json:"rawPayload"`
	Signature  *Signature  `json:"signature"`
	Fee        int64       `json:"fee"`
	Expire     int64       `json:"expire"`
	Nonce      int64       `json:"nonce"`
	From       string      `json:"from,omitempty"`
	To         string      `json:"to"`
	Amount     int64       `json:"amount,omitempty"`
	GroupCount int32       `json:"groupCount,omitempty"`
	Header     string      `json:"header,omitempty"`
	Next       string      `json:"next,omitempty"`
}

type ReceiptLog struct {
	Ty  int32  `json:"ty"`
	Log string `json:"log"`
}

type ReceiptData struct {
	Ty   int32         `json:"ty"`
	Logs []*ReceiptLog `json:"logs"`
}

type ReceiptDataResult struct {
	Ty     int32               `json:"ty"`
	TyName string              `json:"tyName"`
	Logs   []*ReceiptLogResult `json:"logs"`
}

type ReceiptLogResult struct {
	Ty     int32       `json:"ty"`
	TyName string      `json:"tyName"`
	Log    interface{} `json:"log"`
	RawLog string      `json:"rawLog"`
}

type Block struct {
	Version    int64          `json:"version"`
	ParentHash string         `json:"parentHash"`
	TxHash     string         `json:"txHash"`
	StateHash  string         `json:"stateHash"`
	Height     int64          `json:"height"`
	BlockTime  int64          `json:"blockTime"`
	Txs        []*Transaction `json:"txs"`
}

type BlockDetail struct {
	Block    *Block               `json:"block"`
	Receipts []*ReceiptDataResult `json:"recipts"`
}

type BlockDetails struct {
	Items []*BlockDetail `json:"items"`
}

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
}

type ReplyTxInfos struct {
	TxInfos []*ReplyTxInfo `json:"txInfos"`
}

type ReplyTxInfo struct {
	Hash   string `json:"hash"`
	Height int64  `json:"height"`
	Index  int64  `json:"index"`
}

type TransactionDetails struct {
	//Txs []*Transaction `json:"txs"`
	Txs []*TransactionDetail `json:"txs"`
}

type ReplyTxList struct {
	Txs []*Transaction `json:"txs"`
}

type ReplyHash struct {
	Hash string `json:"hash"`
}

type ReplyHashes struct {
	Hashes []string `json:"hashes"`
}
type PeerList struct {
	Peers []*Peer `json:"peers"`
}
type Peer struct {
	Addr        string  `json:"addr"`
	Port        int32   `json:"port"`
	Name        string  `json:"name"`
	MempoolSize int32   `json:"mempoolSize"`
	Self        bool    `json:"self"`
	Header      *Header `json:"header"`
}

// Wallet Module
type WalletAccounts struct {
	Wallets []*WalletAccount `json:"wallets"`
}
type WalletAccount struct {
	Acc   *Account `json:"acc"`
	Label string   `json:"label"`
}

type Account struct {
	Currency int32  `json:"currency"`
	Balance  int64  `json:"balance"`
	Frozen   int64  `json:"frozen"`
	Addr     string `json:"addr"`
}
type Reply struct {
	IsOk bool   `json:"isOK"`
	Msg  string `json:"msg"`
}
type Headers struct {
	Items []*Header `json:"items"`
}

type ReqAddr struct {
	Addr string `json:"addr"`
}

type ReqHashes struct {
	Hashes        []string `json:"hashes"`
	DisableDetail bool     `json:"disableDetail"`
}

type ReqWalletTransactionList struct {
	FromTx          string `json:"fromTx"`
	Count           int32  `json:"count"`
	Direction       int32  `json:"direction"`
	Mode            int32  `json:"mode,omitempty"`
	SendRecvPrivacy int32  `json:"sendRecvPrivacy,omitempty"`
	Address         string `json:"address,omitempty"`
	TokenName       string `json:"tokenname,omitempty"`
}

type WalletTxDetails struct {
	TxDetails []*WalletTxDetail `json:"txDetails"`
}
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

type BlockOverview struct {
	Head     *Header  `json:"head"`
	TxCount  int64    `json:"txCount"`
	TxHashes []string `json:"txHashes"`
}

type Query4Cli struct {
	Execer   string      `json:"execer"`
	FuncName string      `json:"funcName"`
	Payload  interface{} `json:"payload"`
}

type Query4Jrpc struct {
	Execer   string          `json:"execer"`
	FuncName string          `json:"funcName"`
	Payload  json.RawMessage `json:"payload"`
}

type WalletStatus struct {
	IsWalletLock bool `json:"isWalletLock"`
	IsAutoMining bool `json:"isAutoMining"`
	IsHasSeed    bool `json:"isHasSeed"`
	IsTicketLock bool `json:"isTicketLock"`
}

type NodeNetinfo struct {
	Externaladdr string `json:"externalAddr"`
	Localaddr    string `json:"localAddr"`
	Service      bool   `json:"service"`
	Outbounds    int32  `json:"outbounds"`
	Inbounds     int32  `json:"inbounds"`
}

type ReplyPrivacyPkPair struct {
	ShowSuccessful bool   `json:"showSuccessful,omitempty"`
	ViewPub        string `json:"viewPub,omitempty"`
	SpendPub       string `json:"spendPub,omitempty"`
}

type ReplyCacheTxList struct {
	Txs []*Transaction `json:"txs,omitempty"`
}

type TimeStatus struct {
	NtpTime   string `json:"ntpTime"`
	LocalTime string `json:"localTime"`
	Diff      int64  `json:"diff"`
}
type ReplyBlkSeqs struct {
	BlkSeqInfos []*ReplyBlkSeq `json:"blkseqInfos"`
}

type ReplyBlkSeq struct {
	Hash string `json:"hash"`
	Type int64  `json:"type"`
}

type TransactionCreate struct {
	Execer     string          `json:"execer"`
	ActionName string          `json:"actionName"`
	Payload    json.RawMessage `json:"payload"`
}

//Relay Transaction
type RelayOrderTx struct {
	Operation uint32 `json:"operation"`
	Coin      string `json:"coin"`
	Amount    uint64 `json:"coinamount"`
	Addr      string `json:"coinaddr"`
	CoinWait  uint32 `json:"waitblocks"`
	BtyAmount uint64 `json:"btyamount"`
	Fee       int64  `json:"fee"`
}

type RelayAcceptTx struct {
	OrderId  string `json:"order_id"`
	CoinAddr string `json:"coinaddr"`
	CoinWait uint32 `json:"waitblocks"`
	Fee      int64  `json:"fee"`
}

type RelayRevokeTx struct {
	OrderId string `json:"order_id"`
	Target  uint32 `json:"target"`
	Action  uint32 `json:"action"`
	Fee     int64  `json:"fee"`
}

type RelayConfirmTx struct {
	OrderId string `json:"order_id"`
	TxHash  string `json:"tx_hash"`
	Fee     int64  `json:"fee"`
}

type RelayVerifyBTCTx struct {
	OrderId     string `json:"order_id"`
	RawTx       string `json:"raw_tx"`
	TxIndex     uint32 `json:"tx_index"`
	MerklBranch string `json:"merkle_branch"`
	BlockHash   string `json:"block_hash"`
	Fee         int64  `json:"fee"`
}

type RelaySaveBTCHeadTx struct {
	Hash         string `json:"hash"`
	Height       uint64 `json:"height"`
	MerkleRoot   string `json:"merkleRoot"`
	PreviousHash string `json:"previousHash"`
	IsReset      bool   `json:"isReset"`
	Fee          int64  `json:"fee"`
}

type AllExecBalance struct {
	Addr        string         `json:"addr"`
	ExecAccount []*ExecAccount `json:"execAccount"`
}

type ExecAccount struct {
	Execer  string   `json:"execer"`
	Account *Account `json:"account"`
}

type ExecNameParm struct {
	ExecName    string `json:"execname"`
}
