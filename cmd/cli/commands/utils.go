package commands

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
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

func sendToAddress(rpcAddr string, from string, to string, amount int64, note string, isToken bool, tokenSymbol string, isWithdraw bool) {
	amt := amount
	if isWithdraw {
		amt = -amount
	}
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amt, Note: note}
	if !isToken {
		params.Istoken = false
	} else {
		params.Istoken = true
		params.TokenSymbol = tokenSymbol
	}

	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcAddr, "Chain33.SendToAddress", params, &res)
	ctx.Run()
}

func createRawTx(to string, amount float64, note string, withdraw bool, isToken bool, tokenSymbol string) (string, error) {
	amountInt64 := int64(amount*1e4) * 1e4
	//c, err := crypto.New(types.GetSignatureTypeName(wallet.SignType))
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, err)
	//	return "", err
	//}
	//a, err := common.FromHex(priv)
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, err)
	//	return "", err
	//}
	//privKey, err := c.PrivKeyFromBytes(a)
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, err)
	//	return "", err
	//}
	// addrFrom := account.PubKeyToAddress(privKey.PubKey().Bytes()).String()
	var tx *types.Transaction
	if !isToken {
		transfer := &types.CoinsAction{}
		if !withdraw {
			v := &types.CoinsAction_Transfer{Transfer: &types.CoinsTransfer{Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionTransfer
		} else {
			v := &types.CoinsAction_Withdraw{Withdraw: &types.CoinsWithdraw{Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), To: to}
	} else {
		transfer := &types.TokenAction{}
		if !withdraw {
			v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Withdraw{Withdraw: &types.CoinsWithdraw{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("token"), Payload: types.Encode(transfer), To: to}
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinBalanceTransfer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", err
	}
	tx.Fee += types.MinBalanceTransfer
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	//tx.Sign(int32(wallet.SignType), privKey)
	txHex := types.Encode(tx)
	return hex.EncodeToString(txHex), nil
}

func getExecAddr(exec string) (string, error) {
	switch exec {
	case "none", "coins", "hashlock", "retrieve", "ticket", "token", "trade":
		addrResult := account.ExecAddress(exec)
		result := addrResult.String()
		return result, nil
	default:
		return "", errors.New("only none, coins, hashlock, retrieve, ticket, token, trade supported")
	}
}
