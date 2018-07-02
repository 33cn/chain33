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
	Mode  int32  `json:"mode"`
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
	Version    int64  `json:"version"`
	ParentHash string `json:"parentHash"`
	TxHash     string `json:"txHash"`
	StateHash  string `json:"stateHash"`
	Height     int64  `json:"height"`
	BlockTime  int64  `json:"blockTime"`
	TxCount    int64  `json:"txCount"`
	Hash       string `json:"hash"`
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
	TxInfos []*ReplyTxInfo `protobuf:"bytes,1,rep,name=txInfos" json:"txInfos"`
}

type ReplyTxInfo struct {
	Hash   string `json:"hash"`
	Height int64  `json:"height"`
	Index  int64  `json:"index"`
}

type TransactionDetails struct {
	//Txs []*Transaction `json:"txs"`
	Txs []*TransactionDetail `protobuf:"bytes,1,rep,name=txs" json:"txs"`
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
	IsOk bool   `protobuf:"varint,1,opt,name=isOk" json:"isOK"`
	Msg  string `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg"`
}
type Headers struct {
	Items []*Header `protobuf:"bytes,1,rep,name=items" json:"items"`
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
	TxDetails []*WalletTxDetail `protobuf:"bytes,1,rep,name=txDetails" json:"txDetails"`
}
type WalletTxDetail struct {
	Tx         *Transaction       `protobuf:"bytes,1,opt,name=tx" json:"tx"`
	Receipt    *ReceiptDataResult `protobuf:"bytes,2,opt,name=receipt" json:"receipt"`
	Height     int64              `protobuf:"varint,3,opt,name=height" json:"height"`
	Index      int64              `protobuf:"varint,4,opt,name=index" json:"index"`
	BlockTime  int64              `json:"blockTime"`
	Amount     int64              `json:"amount"`
	FromAddr   string             `json:"fromAddr"`
	TxHash     string             `json:"txHash"`
	ActionName string             `json:"actionName"`
}

type BlockOverview struct {
	Head     *Header  `protobuf:"bytes,1,opt,name=head" json:"head"`
	TxCount  int64    `protobuf:"varint,2,opt,name=txCount" json:"txCount"`
	TxHashes []string `protobuf:"bytes,3,rep,name=txHashes,proto3" json:"txHashes"`
}

type Query4Cli struct {
	Execer   string      `protobuf:"bytes,1,opt,name=execer,proto3" json:"execer"`
	FuncName string      `protobuf:"bytes,2,opt,name=funcName" json:"funcName"`
	Payload  interface{} `protobuf:"bytes,3,opt,name=payload" json:"payload"`
}

type Query4Jrpc struct {
	Execer   string          `protobuf:"bytes,1,opt,name=execer,proto3" json:"execer"`
	FuncName string          `protobuf:"bytes,2,opt,name=funcName" json:"funcName"`
	Payload  json.RawMessage `protobuf:"bytes,3,opt,name=payload" json:"payload"`
}

type WalletStatus struct {
	IsWalletLock bool `json:"isWalletLock"`
	IsAutoMining bool `json:"isAutoMining"`
	IsHasSeed    bool `json:"isHasSeed"`
	IsTicketLock bool `json:"isTicketLock"`
}

// Token Transaction
type TokenPreCreateTx struct {
	Price        int64  `json:"price"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Introduction string `json:"introduction"`
	OwnerAddr    string `json:"ownerAddr"`
	Total        int64  `json:"total"`
	Fee          int64  `json:"fee"`
}

type TokenFinishTx struct {
	OwnerAddr string `json:"ownerAddr"`
	Symbol    string `json:"symbol"`
	Fee       int64  `json:"fee"`
}

type TokenRevokeTx struct {
	OwnerAddr string `json:"ownerAddr"`
	Symbol    string `json:"symbol"`
	Fee       int64  `json:"fee"`
}

// Trade Transaction
type TradeSellTx struct {
	TokenSymbol       string `json:"tokenSymbol"`
	AmountPerBoardlot int64  `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  int64  `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	Fee               int64  `json:"fee"`
}

type TradeBuyTx struct {
	SellID      string `json:"sellID"`
	BoardlotCnt int64  `json:"boardlotCnt"`
	Fee         int64  `json:"fee"`
}

type TradeRevokeTx struct {
	SellID string `json:"sellID,"`
	Fee    int64  `json:"fee"`
}

type TradeBuyLimitTx struct {
	TokenSymbol       string `json:"tokenSymbol"`
	AmountPerBoardlot int64  `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  int64  `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	Fee               int64  `json:"fee"`
}

type TradeSellMarketTx struct {
	BuyID       string `json:"buyID"`
	BoardlotCnt int64  `json:"boardlotCnt"`
	Fee         int64  `json:"fee"`
}

type TradeRevokeBuyTx struct {
	BuyID string `json:"buyID,"`
	Fee   int64  `json:"fee"`
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

//Relay Transaction
type RelayOrderTx struct {
	Operation uint32 `json:"operation"`
	Coin      string `json:"coin"`
	Amount    uint64 `json:"coinamount"`
	Addr      string `json:"coinaddr"`
	BtyAmount uint64 `json:"btyamount"`
	Fee       int64  `json:"fee"`
}

type RelayAcceptTx struct {
	OrderId  string `json:"order_id"`
	CoinAddr string `json:"coinaddr"`
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
