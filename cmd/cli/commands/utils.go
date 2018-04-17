package commands

import (
	"fmt"
	"strconv"

	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
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
	Reciver string `json:"reciver"`
	Balance string `json:"balance"`
	TxCount int64  `json:"txCount"`
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

func decodeTransaction(tx *jsonrpc.Transaction) *TxResult {
	feeResult := strconv.FormatFloat(float64(tx.Fee)/float64(types.Coin), 'f', 4, 64)
	amountResult := ""
	if tx.Amount != 0 {
		amountResult = strconv.FormatFloat(float64(tx.Amount)/float64(types.Coin), 'f', 4, 64)
	}
	result := &TxResult{
		Execer:     tx.Execer,
		Payload:    tx.Payload,
		RawPayload: tx.RawPayload,
		Signature:  tx.Signature,
		Fee:        feeResult,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.To,
	}
	payloacValue := tx.Payload.(map[string]interface{})["Value"].(map[string]interface{})
	for _, e := range [4]string{"Transfer", "Withdraw", "Genesis", "Hlock"} {
		if _, ok := payloacValue[e]; ok {
			if amtValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"]; ok {
				amt := amtValue.(float64) / float64(types.Coin)
				amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
				result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"] = amtResult
				break
			}
		}
	}
	if _, ok := payloacValue["Miner"]; ok {
		if rwdValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"]; ok {
			rwd := rwdValue.(float64) / float64(types.Coin)
			rwdResult := strconv.FormatFloat(rwd, 'f', 4, 64)
			result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"] = rwdResult
		}
	}

	if tx.Amount != 0 {
		result.Amount = amountResult
	}
	if tx.From != "" {
		result.From = tx.From
	}
	return result
}

func decodeAccount(acc *types.Account, precision int64) *AccountResult {
	balanceResult := strconv.FormatFloat(float64(acc.GetBalance())/float64(precision), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(acc.GetFrozen())/float64(precision), 'f', 4, 64)
	accResult := &AccountResult{
		Addr:     acc.GetAddr(),
		Currency: acc.GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return accResult
}

func constructAccFromLog(l *jsonrpc.ReceiptLogResult, key string) *types.Account {
	var cur int32
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["currency"]; ok {
		cur = int32(tmp.(float32))
	}
	var bal int64
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["balance"]; ok {
		bal = int64(tmp.(float64))
	}
	var fro int64
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["frozen"]; ok {
		fro = int64(tmp.(float64))
	}
	var ad string
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["addr"]; ok {
		ad = tmp.(string)
	}
	return &types.Account{
		Currency: cur,
		Balance:  bal,
		Frozen:   fro,
		Addr:     ad,
	}
}

func decodeLog(rlog jsonrpc.ReceiptDataResult) *ReceiptData {
	rd := &ReceiptData{Ty: rlog.Ty, TyName: rlog.TyName}

	for _, l := range rlog.Logs {
		rl := &ReceiptLog{Ty: l.Ty, TyName: l.TyName, RawLog: l.RawLog}
		switch l.Ty {
		//case 1, 4, 111, 112, 113, 114:
		case types.TyLogErr, types.TyLogGenesis, types.TyLogNewTicket, types.TyLogCloseTicket, types.TyLogMinerTicket,
			types.TyLogTicketBind, types.TyLogPreCreateToken, types.TyLogFinishCreateToken, types.TyLogRevokeCreateToken,
			types.TyLogTradeSell, types.TyLogTradeBuy, types.TyLogTradeRevoke:
			rl.Log = l.Log
		//case 2, 3, 5, 11:
		case types.TyLogFee, types.TyLogTransfer, types.TyLogDeposit, types.TyLogGenesisTransfer,
			types.TyLogTokenTransfer, types.TyLogTokenDeposit:
			rl.Log = &ReceiptAccountTransfer{
				Prev:    decodeAccount(constructAccFromLog(l, "prev"), types.Coin),
				Current: decodeAccount(constructAccFromLog(l, "current"), types.Coin),
			}
		//case 6, 7, 8, 9, 10, 12:
		case types.TyLogExecTransfer, types.TyLogExecWithdraw, types.TyLogExecDeposit, types.TyLogExecFrozen, types.TyLogExecActive, types.TyLogGenesisDeposit:
			var execaddr string
			if tmp, ok := l.Log.(map[string]interface{})["execaddr"].(string); ok {
				execaddr = tmp
			}
			rl.Log = &ReceiptExecAccountTransfer{
				ExecAddr: execaddr,
				Prev:     decodeAccount(constructAccFromLog(l, "prev"), types.Coin),
				Current:  decodeAccount(constructAccFromLog(l, "current"), types.Coin),
			}
		case types.TyLogTokenExecTransfer, types.TyLogTokenExecWithdraw, types.TyLogTokenExecDeposit, types.TyLogTokenExecFrozen, types.TyLogTokenExecActive,
			types.TyLogTokenGenesisTransfer, types.TyLogTokenGenesisDeposit:
			var execaddr string
			if tmp, ok := l.Log.(map[string]interface{})["execaddr"].(string); ok {
				execaddr = tmp
			}
			rl.Log = &ReceiptExecAccountTransfer{
				ExecAddr: execaddr,
				Prev:     decodeAccount(constructAccFromLog(l, "prev"), types.TokenPrecision),
				Current:  decodeAccount(constructAccFromLog(l, "current"), types.TokenPrecision),
			}
		default:
			fmt.Printf("---The log with vlaue:%d is not decoded --------------------\n", l.Ty)
			return nil
		}
		rd.Logs = append(rd.Logs, rl)
	}
	return rd
}
