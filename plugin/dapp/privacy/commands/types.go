package commands

import (
	"encoding/json"

	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
)

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

type PrivacyAccountSpendResult struct {
	Txhash string                  `json:"Txhash,omitempty"`
	Res    []*PrivacyAccountResult `json:"Spend,omitempty"`
}

type ShowRescanResult struct {
	Addr       string `json:"addr"`
	FlagString string `json:"FlagString"`
}

type showRescanResults struct {
	RescanResults []*ShowRescanResult `json:"ShowRescanResults,omitempty"`
}

type ShowEnablePrivacy struct {
	Results []*ShowPriAddrResult `json:"results"`
}

type ShowPriAddrResult struct {
	Addr string `json:"addr"`
	IsOK bool   `json:"IsOK"`
	Msg  string `json:"msg"`
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

type ReceiptData struct {
	Ty     int32         `json:"ty"`
	TyName string        `json:"tyname"`
	Logs   []*ReceiptLog `json:"logs"`
}

type ReceiptLog struct {
	Ty     int32           `json:"ty"`
	TyName string          `json:"tyname"`
	Log    json.RawMessage `json:"log"`
	RawLog string          `json:"rawlog"`
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
