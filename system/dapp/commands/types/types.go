package types

import (
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
)

type AccountsResult struct {
	Wallets []*WalletResult `json:"wallets"`
}

type WalletResult struct {
	Acc   *AccountResult `json:"acc,omitempty"`
	Label string         `json:"label,omitempty"`
}

type AccountResult struct {
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

type TokenAccountResult struct {
	Token    string `json:"Token,omitempty"`
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

type TxListResult struct {
	Txs []*TxResult `json:"txs"`
}

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
}

type ReceiptAccountTransfer struct {
	Prev    *AccountResult `protobuf:"bytes,1,opt,name=prev" json:"prev,omitempty"`
	Current *AccountResult `protobuf:"bytes,2,opt,name=current" json:"current,omitempty"`
}

type ReceiptExecAccountTransfer struct {
	ExecAddr string         `protobuf:"bytes,1,opt,name=execAddr" json:"execAddr,omitempty"`
	Prev     *AccountResult `protobuf:"bytes,2,opt,name=prev" json:"prev,omitempty"`
	Current  *AccountResult `protobuf:"bytes,3,opt,name=current" json:"current,omitempty"`
}

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
}

type TxDetailsResult struct {
	Txs []*TxDetailResult `json:"txs"`
}

type BlockResult struct {
	Version    int64       `json:"version"`
	ParentHash string      `json:"parenthash"`
	TxHash     string      `json:"txhash"`
	StateHash  string      `json:"statehash"`
	Height     int64       `json:"height"`
	BlockTime  int64       `json:"blocktime"`
	Txs        []*TxResult `json:"txs"`
}

type BlockDetailResult struct {
	Block    *BlockResult                  `json:"block"`
	Receipts []*rpctypes.ReceiptDataResult `json:"receipts"`
}

type BlockDetailsResult struct {
	Items []*BlockDetailResult `json:"items"`
}

type WalletTxDetailsResult struct {
	TxDetails []*WalletTxDetailResult `json:"txDetails"`
}

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

type AddrOverviewResult struct {
	Receiver string `json:"receiver"`
	Balance  string `json:"balance"`
	TxCount  int64  `json:"txCount"`
}

type GetTotalCoinsResult struct {
	TxCount          int64  `json:"txCount"`
	AccountCount     int64  `json:"accountCount"`
	TotalAmount      string `json:"totalAmount"`
	ActualAmount     string `json:"actualAmount,omitempty"`
	DifferenceAmount string `json:"differenceAmount,omitempty"`
}

type GetTicketStatisticResult struct {
	CurrentOpenCount int64 `json:"currentOpenCount"`
	TotalMinerCount  int64 `json:"totalMinerCount"`
	TotalCloseCount  int64 `json:"totalCloseCount"`
}

type GetTicketMinerInfoResult struct {
	TicketId     string `json:"ticketId"`
	Status       string `json:"status"`
	PrevStatus   string `json:"prevStatus"`
	IsGenesis    bool   `json:"isGenesis"`
	CreateTime   string `json:"createTime"`
	MinerTime    string `json:"minerTime"`
	CloseTime    string `json:"closeTime"`
	MinerValue   int64  `json:"minerValue,omitempty"`
	MinerAddress string `json:"minerAddress,omitempty"`
}

type UTXOGlobalIndex struct {
	// Height   int64  `json:"height,omitempty"`
	// Txindex  int32  `json:"txindex,omitempty"`
	Outindex int32  `json:"outindex,omitempty"`
	Txhash   string `json:"txhash,omitempty"`
}

type KeyInput struct {
	Amount          string             `json:"amount,omitempty"`
	UtxoGlobalIndex []*UTXOGlobalIndex `json:"utxoGlobalIndex,omitempty"`
	KeyImage        string             `json:"keyImage,omitempty"`
}

type PrivacyInput struct {
	Keyinput []*KeyInput `json:"keyinput,omitempty"`
}

// privacy output
type KeyOutput struct {
	Amount        string `json:"amount,omitempty"`
	Onetimepubkey string `json:"onetimepubkey,omitempty"`
}

type ReceiptPrivacyOutput struct {
	Token     string       `json:"token,omitempty"`
	Keyoutput []*KeyOutput `json:"keyoutput,omitempty"`
}

type AllExecBalance struct {
	Addr        string         `json:"addr"`
	ExecAccount []*ExecAccount `json:"execAccount"`
}

type ExecAccount struct {
	Execer  string         `json:"execer"`
	Account *AccountResult `json:"account"`
}

// decodetx
type CoinsTransferCLI struct {
	Cointoken string `json:"cointoken,omitempty"`
	Amount    string `json:"amount,omitempty"`
	Note      string `json:"note,omitempty"`
	To        string `json:"to,omitempty"`
}

type CoinsWithdrawCLI struct {
	Cointoken string `json:"cointoken,omitempty"`
	Amount    string `json:"amount,omitempty"`
	Note      string `json:"note,omitempty"`
	ExecName  string `json:"execName,omitempty"`
	To        string `json:"to,omitempty"`
}

type CoinsGenesisCLI struct {
	Amount        string `json:"amount,omitempty"`
	ReturnAddress string `json:"returnAddress,omitempty"`
}

type CoinsTransferToExecCLI struct {
	Cointoken string `json:"cointoken,omitempty"`
	Amount    string `json:"amount,omitempty"`
	Note      string `json:"note,omitempty"`
	ExecName  string `json:"execName,omitempty"`
	To        string `json:"to,omitempty"`
}

type HashlockLockCLI struct {
	Amount        string `json:"amount,omitempty"`
	Time          int64  `json:"time,omitempty"`
	Hash          []byte `json:"hash,omitempty"`
	ToAddress     string `json:"toAddress,omitempty"`
	ReturnAddress string `json:"returnAddress,omitempty"`
}

type TicketMinerCLI struct {
	Bits     uint32 `json:"bits,omitempty"`
	Reward   string `json:"reward,omitempty"`
	TicketId string `json:"ticketId,omitempty"`
	Modify   []byte `json:"modify,omitempty"`
}

type TokenPreCreateCLI struct {
	Name         string `json:"name,omitempty"`
	Symbol       string `json:"symbol,omitempty"`
	Introduction string `json:"introduction,omitempty"`
	Total        int64  `json:"total,omitempty"`
	Price        string `json:"price,omitempty"`
	Owner        string `json:"owner,omitempty"`
}

type Public2PrivacyCLI struct {
	Tokenname string         `json:"tokenname,omitempty"`
	Amount    string         `json:"amount,omitempty"`
	Note      string         `json:"note,omitempty"`
	Output    *PrivacyOutput `json:"output,omitempty"`
}

type Privacy2PrivacyCLI struct {
	Tokenname string         `json:"tokenname,omitempty"`
	Amount    string         `json:"amount,omitempty"`
	Note      string         `json:"note,omitempty"`
	Input     *PrivacyInput  `json:"input,omitempty"`
	Output    *PrivacyOutput `json:"output,omitempty"`
}

type Privacy2PublicCLI struct {
	Tokenname string         `json:"tokenname,omitempty"`
	Amount    string         `json:"amount,omitempty"`
	Note      string         `json:"note,omitempty"`
	Input     *PrivacyInput  `json:"input,omitempty"`
	Output    *PrivacyOutput `json:"output,omitempty"`
}

type PrivacyOutput struct {
	RpubKeytx string       `protobuf:"bytes,1,opt,name=RpubKeytx,proto3" json:"RpubKeytx,omitempty"`
	Keyoutput []*KeyOutput `protobuf:"bytes,2,rep,name=keyoutput" json:"keyoutput,omitempty"`
}
