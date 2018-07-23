package commands

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
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
		From:       tx.From,
		GroupCount: tx.GroupCount,
		Header:     tx.Header,
		Next:       tx.Next,
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
	// 隐私交易
	if pub2priv, ok := payloadValue["Public2Privacy"]; ok {
		decodePrivacyPayload(pub2priv)
	} else if priv2priv, ok := payloadValue["Privacy2Privacy"]; ok {
		decodePrivacyPayload(priv2priv)
	} else if priv2priv, ok := payloadValue["Privacy2Public"]; ok {
		decodePrivacyPayload(priv2priv)
	}
	return result
}

func decodePrivacyPayload(pub2priv interface{}) {
	if amountVal, ok := getStrMapValue(pub2priv, "amount"); ok {
		amount := amountVal.(float64) / float64(types.Coin)
		getStrMapPair(pub2priv)["amount"] = strconv.FormatFloat(amount, 'f', 4, 64)
	}

	if inputVal, ok := getStrMapValue(pub2priv, "input"); ok {
		if keyinputVals, ok := getStrMapValue(inputVal, "keyinput"); ok {
			for _, value := range keyinputVals.([]interface{}) {
				if amountVal, ok := getStrMapValue(value, "amount"); ok {
					amount := amountVal.(float64) / float64(types.Coin)
					getStrMapPair(value)["amount"] = strconv.FormatFloat(amount, 'f', 4, 64)
				}
			}
		}
	}

	if outputVal, ok := getStrMapValue(pub2priv, "output"); ok {
		if keyoutputVals, ok := getStrMapValue(outputVal, "keyoutput"); ok {
			for _, value := range keyoutputVals.([]interface{}) {
				if amountVal, ok := getStrMapValue(value, "amount"); ok {
					amount := amountVal.(float64) / float64(types.Coin)
					getStrMapPair(value)["amount"] = strconv.FormatFloat(amount, 'f', 4, 64)
				}
			}
		}
	}
}

func getStrMapPair(m interface{}) map[string]interface{} {
	return m.(map[string]interface{})
}

func getStrMapValue(m interface{}, key string) (interface{}, bool) {
	ret, ok := m.(map[string]interface{})[key]
	return ret, ok
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
			types.TyLogTradeBuyLimit, types.TyLogTradeSellMarket, types.TyLogTradeBuyRevoke,
			types.TyLogRelayCreate, types.TyLogRelayRevokeCreate, types.TyLogRelayAccept, types.TyLogRelayRevokeAccept,
			types.TyLogRelayRcvBTCHead, types.TyLogRelayConfirmTx, types.TyLogRelayFinishTx:
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
			// EVM合约日志处理逻辑
		case types.TyLogCallContract:
			rl.Log = buildCallContractResult(l)
		case types.TyLogContractData:
			rl.Log = buildContractDataResult(l)
		case types.TyLogContractState:
			rl.Log = buildContractStateResult(l)
			// 隐私交易
		case types.TyLogPrivacyInput:
			rl.Log = buildPrivacyInputResult(l)
		case types.TyLogPrivacyOutput:
			rl.Log = buildPrivacyOutputResult(l)

		case types.TyLogEVMStateChangeItem:
			rl.Log = buildStateChangeItemResult(l)
		default:
			fmt.Printf("---The log with vlaue:%d is not decoded --------------------\n", l.Ty)
			return nil
		}
		rd.Logs = append(rd.Logs, rl)
	}
	return rd
}

func buildCallContractResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	receipt := &types.ReceiptEVMContract{}
	proto.Unmarshal(data, receipt)
	rlog := &types.ReceiptEVMContractCmd{Caller: receipt.Caller, ContractAddr: receipt.ContractAddr, ContractName: receipt.ContractName, UsedGas: receipt.UsedGas}
	rlog.Ret = common.ToHex(receipt.Ret)
	return rlog
}

func buildContractDataResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	receipt := &types.EVMContractData{}
	proto.Unmarshal(data, receipt)
	rlog := &types.EVMContractDataCmd{Creator: receipt.Creator, Name: receipt.Name, Addr: receipt.Addr, Alias: receipt.Alias}
	rlog.Code = common.ToHex(receipt.Code)
	rlog.CodeHash = common.ToHex(receipt.CodeHash)
	return rlog
}

func buildContractStateResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	receipt := &types.EVMContractState{}
	proto.Unmarshal(data, receipt)
	rlog := &types.EVMContractStateCmd{Nonce: receipt.Nonce, Suicided: receipt.Suicided}
	rlog.StorageHash = common.ToHex(receipt.StorageHash)
	if receipt.Storage != nil {
		rlog.Storage = make(map[string]string)
		for k, v := range receipt.Storage {
			rlog.Storage[k] = common.ToHex(v)
		}
	}
	return rlog
}

func buildStateChangeItemResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	receipt := &types.EVMStateChangeItem{}
	proto.Unmarshal(data, receipt)
	return receipt
}

func buildPrivacyInputResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	srcInput := &types.PrivacyInput{}
	dstInput := &PrivacyInput{}
	if proto.Unmarshal(data, srcInput) != nil {
		return dstInput
	}
	for _, srcKeyInput := range srcInput.Keyinput {
		var dstUtxoGlobalIndex []*UTXOGlobalIndex
		for _, srcUTXOGlobalIndex := range srcKeyInput.UtxoGlobalIndex {
			dstUtxoGlobalIndex = append(dstUtxoGlobalIndex, &UTXOGlobalIndex{
				Outindex: srcUTXOGlobalIndex.GetOutindex(),
				Txhash:   common.ToHex(srcUTXOGlobalIndex.GetTxhash()),
			})
		}
		dstKeyInput := &KeyInput{
			Amount:          strconv.FormatFloat(float64(srcKeyInput.GetAmount())/float64(types.Coin), 'f', 4, 64),
			UtxoGlobalIndex: dstUtxoGlobalIndex,
			KeyImage:        common.ToHex(srcKeyInput.GetKeyImage()),
		}
		dstInput.Keyinput = append(dstInput.Keyinput, dstKeyInput)
	}
	return dstInput
}

func buildPrivacyOutputResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	srcOutput := &types.ReceiptPrivacyOutput{}
	dstOutput := &ReceiptPrivacyOutput{}
	if proto.Unmarshal(data, srcOutput) != nil {
		return dstOutput
	}
	dstOutput.Token = srcOutput.Token
	for _, srcKeyoutput := range srcOutput.Keyoutput {
		dstKeyoutput := &KeyOutput{
			Amount:        strconv.FormatFloat(float64(srcKeyoutput.GetAmount())/float64(types.Coin), 'f', 4, 64),
			Onetimepubkey: common.ToHex(srcKeyoutput.Onetimepubkey),
		}
		dstOutput.Keyoutput = append(dstOutput.Keyoutput, dstKeyoutput)
	}
	return dstOutput
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

func CreateRawTx(cmd *cobra.Command, to string, amount float64, note string, isWithdraw, isToken bool, tokenSymbol, execName string) (string, error) {
	if amount < 0 {
		return "", types.ErrAmount
	}
	paraName, _ := cmd.Flags().GetString("paraName")
	amountInt64 := int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4
	execName = getRealExecName(paraName, execName)
	if execName != "" && !types.IsAllowExecName(execName) {
		return "", types.ErrExecNameNotMatch
	}
	var tx *types.Transaction
	if !isToken {
		transfer := &types.CoinsAction{}
		if !isWithdraw {
			if execName != "" {
				v := &types.CoinsAction_TransferToExec{TransferToExec: &types.CoinsTransferToExec{Amount: amountInt64, Note: note, ExecName: execName}}
				transfer.Value = v
				transfer.Ty = types.CoinsActionTransferToExec
			} else {
				v := &types.CoinsAction_Transfer{Transfer: &types.CoinsTransfer{Amount: amountInt64, Note: note, To: to}}
				transfer.Value = v
				transfer.Ty = types.CoinsActionTransfer
			}
		} else {
			v := &types.CoinsAction_Withdraw{Withdraw: &types.CoinsWithdraw{Amount: amountInt64, Note: note, ExecName: execName}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionWithdraw
		}
		execer := []byte(getRealExecName(paraName, "coins"))
		if paraName == "" {
			tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: to}
		} else {
			tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: address.ExecAddress(string(execer))}
		}
	} else {
		transfer := &types.TokenAction{}
		if !isWithdraw {
			v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{Cointoken: tokenSymbol, Amount: amountInt64, Note: note, To: to}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Withdraw{Withdraw: &types.CoinsWithdraw{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionWithdraw
		}
		execer := []byte(getRealExecName(paraName, "token"))
		if paraName == "" {
			tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: to}
		} else {
			tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: address.ExecAddress(string(execer))}
		}
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		return "", err
	}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	txHex := types.Encode(tx)
	return hex.EncodeToString(txHex), nil
}

func GetExecAddr(exec string) (string, error) {
	if ok := types.IsAllowExecName(exec); !ok {
		return "", types.ErrExecNameNotAllow
	}

	addrResult := address.ExecAddress(exec)
	result := addrResult
	return result, nil
}

func getExecuterNameString() string {
	str := "executer name (only "
	allowExeName := types.AllowUserExec
	nameLen := len(allowExeName)
	for i := 0; i < nameLen; i++ {
		if i > 0 {
			str += ", "
		}
		str += fmt.Sprintf("\"%s\"", string(allowExeName[i]))
	}
	str += " and user-defined type supported)"
	return str
}

// FormatAmountValue2Display 将传输、计算的amount值格式化成显示值
func FormatAmountValue2Display(amount int64) string {
	return strconv.FormatFloat(float64(amount)/float64(types.Coin), 'f', 4, 64)
}

// FormatAmountDisplay2Value 将显示、输入的amount值格式话成传输、计算值
func FormatAmountDisplay2Value(amount float64) int64 {
	return int64(amount*types.InputPrecision) * types.Multiple1E4
}

// GetAmountValue 将命令行中的amount值转换成int64
func GetAmountValue(cmd *cobra.Command, field string) int64 {
	amount, _ := cmd.Flags().GetFloat64(field)
	return FormatAmountDisplay2Value(amount)
}

func getRealExecName(paraName string, name string) string {
	if strings.HasPrefix(name, "user.p.") {
		return name
	}
	return paraName + name
}
