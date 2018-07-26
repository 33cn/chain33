package commands

import jsonrpc "gitlab.33.cn/chain33/chain33/rpc"

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
	Execer     string             `json:"execer"`
	Payload    interface{}        `json:"payload"`
	RawPayload string             `json:"rawpayload"`
	Signature  *jsonrpc.Signature `json:"signature"`
	Fee        string             `json:"fee"`
	Expire     int64              `json:"expire"`
	Nonce      int64              `json:"nonce"`
	To         string             `json:"to"`
	Amount     string             `json:"amount,omitempty"`
	From       string             `json:"from,omitempty"`
	GroupCount int32              `json:"groupCount,omitempty"`
	Header     string             `json:"header,omitempty"`
	Next       string             `json:"next,omitempty"`
}

type ReceiptData struct {
	Ty     int32         `json:"ty"`
	TyName string        `json:"tyname"`
	Logs   []*ReceiptLog `json:"logs"`
}

type ReceiptLog struct {
	Ty     int32       `json:"ty"`
	TyName string      `json:"tyname"`
	Log    interface{} `json:"log"`
	RawLog string      `json:"rawlog"`
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
	Tx         *TxResult    `json:"tx"`
	Receipt    *ReceiptData `json:"receipt"`
	Proofs     []string     `json:"proofs,omitempty"`
	Height     int64        `json:"height"`
	Index      int64        `json:"index"`
	Blocktime  int64        `json:"blocktime"`
	Amount     string       `json:"amount"`
	Fromaddr   string       `json:"fromaddr"`
	ActionName string       `json:"actionname"`
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
	Block    *BlockResult   `json:"block"`
	Receipts []*ReceiptData `json:"receipts"`
}

type BlockDetailsResult struct {
	Items []*BlockDetailResult `json:"items"`
}

type WalletTxDetailsResult struct {
	TxDetails []*WalletTxDetailResult `json:"txDetails"`
}

type WalletTxDetailResult struct {
	Tx         *TxResult    `json:"tx"`
	Receipt    *ReceiptData `json:"receipt"`
	Height     int64        `json:"height"`
	Index      int64        `json:"index"`
	Blocktime  int64        `json:"blocktime"`
	Amount     string       `json:"amount"`
	Fromaddr   string       `json:"fromaddr"`
	Txhash     string       `json:"txhash"`
	ActionName string       `json:"actionname"`
}

type AddrOverviewResult struct {
	Receiver string `json:"receiver"`
	Balance  string `json:"balance"`
	TxCount  int64  `json:"txCount"`
}

type SellOrder2Show struct {
	Tokensymbol       string `json:"tokensymbol"`
	Seller            string `json:"address"`
	Amountperboardlot string `json:"amountperboardlot"`
	Minboardlot       int64  `json:"minboardlot"`
	Priceperboardlot  string `json:"priceperboardlot"`
	Totalboardlot     int64  `json:"totalboardlot"`
	Soldboardlot      int64  `json:"soldboardlot"`
	Starttime         int64  `json:"starttime"`
	Stoptime          int64  `json:"stoptime"`
	Crowdfund         bool   `json:"crowdfund"`
	SellID            string `json:"sellid"`
	Status            string `json:"status"`
	Height            int64  `json:"height"`
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

type PrivacyAccountResult struct {
	Token         string `json:"Token,omitempty"`
	Txhash        string `json:"Txhash,omitempty"`
	OutIndex      int32  `json:"OutIndex,omitempty"`
	Amount        string `json:"Amount,omitempty"`
	OnetimePubKey string `json:"OnetimePubKey,omitempty"`
}

type PrivacyAccountInfoResult struct {
	AvailableDetail []*PrivacyAccountResult `json:"AvailableDetail,omitempty"`
	FrozenDetail    []*PrivacyAccountResult `json:"FrozenDetail,omitempty"`
	AvailableAmount string                  `json:"AvailableAmount,omitempty"`
	FrozenAmount    string                  `json:"FrozenAmount,omitempty"`
	TotalAmount     string                  `json:"TotalAmount,omitempty"`
}

type UTXOGlobalIndex struct {
	Height   int64  `json:"height,omitempty"`
	Txindex  int32  `json:"txindex,omitempty"`
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

type PrivacyAccountSpendResult struct {
	Txhash string                  `json:"Txhash,omitempty"`
	Res    []*PrivacyAccountResult `json:"Spend,omitempty"`
}

type RelayOrder2Show struct {
	OrderId       string `json:"orderid"`
	Status        string `json:"status"`
	Creator       string `json:"address"`
	Amount        string `json:"amount"`
	CoinOperation string `json:"coinoperation"`
	Coin          string `json:"coin"`
	CoinAmount    string `json:"coinamount"`
	CoinAddr      string `json:"coinaddr"`
	CoinWaits     uint32 `json:"coinwaits"`
	CreateTime    int64  `json:"createtime"`
	AcceptAddr    string `json:"acceptaddr"`
	AcceptTime    int64  `json:"accepttime"`
	ConfirmTime   int64  `json:"confirmtime"`
	FinishTime    int64  `json:"finishtime"`
	FinishTxHash  string `json:"finishtxhash"`
	Height        int64  `json:"height"`
}

type AllExecBalance struct {
	Addr        string         `json:"addr"`
	ExecAccount []*ExecAccount `json:"execAccount"`
}

type ExecAccount struct {
	Execer  string         `json:"execer"`
	Account *AccountResult `json:"account"`
}

type ShowRescanResult struct {
	Addr       string `json:"addr"`
	FlagString string `json:"FlagString"`
}

type showRescanResults struct {
	RescanResults []*ShowRescanResult `json:"ShowRescanResults,omitempty"`
}
