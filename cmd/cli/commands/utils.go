package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gitlab.33.cn/chain33/chain33/account"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func decodeTransaction(tx *jsonrpc.Transaction) *TxResult {
	feeResult := strconv.FormatFloat(float64(tx.Fee)/float64(types.Coin), 'f', 4, 64)
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

	if tx.Amount != 0 {
		result.Amount = strconv.FormatFloat(float64(tx.Amount)/float64(types.Coin), 'f', 4, 64)
	}
	if tx.From != "" {
		result.From = tx.From
	}

	payloadValue, ok := tx.Payload.(map[string]interface{})["Value"].(map[string]interface{})
	if !ok {
		return result
	}
	for _, e := range [4]string{"Transfer", "Withdraw", "Genesis", "Hlock"} {
		if _, ok := payloadValue[e]; ok {
			if amtValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"]; ok {
				amt := amtValue.(float64) / float64(types.Coin)
				amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
				result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"] = amtResult
				break
			}
		}
	}
	if _, ok := payloadValue["Miner"]; ok {
		if rwdValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"]; ok {
			rwd := rwdValue.(float64) / float64(types.Coin)
			rwdResult := strconv.FormatFloat(rwd, 'f', 4, 64)
			result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"] = rwdResult
		}
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
			types.TyLogTradeSellLimit, types.TyLogTradeBuyMarket, types.TyLogTradeSellRevoke,
			types.TyLogTradeBuyLimit, types.TyLogTradeSellMarket, types.TyLogTradeBuyRevoke:
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

func SendToAddress(rpcAddr string, from string, to string, amount int64, note string, isToken bool, tokenSymbol string, isWithdraw bool) {
	amt := amount
	if isWithdraw {
		amt = -amount
	}
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amt, Note: note}
	if !isToken {
		params.IsToken = false
	} else {
		params.IsToken = true
		params.TokenSymbol = tokenSymbol
	}

	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcAddr, "Chain33.SendToAddress", params, &res)
	ctx.Run()
}

func CreateRawTx(rpcAddr string, to string, amount float64, note string, isWithdraw bool, isToken bool, tokenSymbol string, execName string) {
	if amount < 0 {
		fmt.Fprintln(os.Stderr, types.ErrAmount)
		return
	}
	amountInt64 := int64(amount*1e4) * 1e4
	params := &types.CreateTx{
		To:          to,
		Amount:      amountInt64,
		Note:        note,
		IsWithdraw:  isWithdraw,
		IsToken:     isToken,
		TokenSymbol: tokenSymbol,
		ExecName:    execName,
	}

	ctx := NewRpcCtx(rpcAddr, "Chain33.CreateRawTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func GetExecAddr(exec string) (string, error) {
	if ok, err := isAllowExecName(exec); !ok {
		return "", err
	}

	addrResult := account.ExecAddress(exec)
	result := addrResult
	return result, nil
}

func isAllowExecName(exec string) (bool, error) {
	// exec name长度不能超过50
	if len(exec) > 50 {
		return false, errors.New("executor name too long")
	}
	// exec name中不允许有 "-"
	if strings.Contains(exec, "-") {
		return false, errors.New("executor name should not contain slash")
	}
	if strings.HasPrefix(exec, "user.") {
		return true, nil
	}
	for _, e := range []string{"none", "coins", "hashlock", "retrieve", "ticket", "token", "trade"} {
		if exec == e {
			return true, nil
		}
	}
	return false, errors.New("only none, coins, hashlock, retrieve, ticket, token, trade or user defined executor supported")
}
