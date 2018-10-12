package commands

import (
	"encoding/hex"
	//	"encoding/json"
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
	bwty "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
	// TODO: 暂时将插件中的类型引用起来，后续需要修改
	hlt "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
)

func decodeTransaction(tx *rpctypes.Transaction) *TxResult {
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

	switch tx.Execer {
	case cty.CoinsX:
		var action cty.CoinsAction
		bt, _ := common.FromHex(tx.RawPayload)
		types.Decode(bt, &action)
		if pl, ok := action.Value.(*cty.CoinsAction_Transfer); ok {
			amt := float64(pl.Transfer.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsTransferCLI{
				Cointoken: pl.Transfer.Cointoken,
				Amount:    amtResult,
				Note:      pl.Transfer.Note,
				To:        pl.Transfer.To,
			}
		} else if pl, ok := action.Value.(*cty.CoinsAction_Withdraw); ok {
			amt := float64(pl.Withdraw.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsWithdrawCLI{
				Cointoken: pl.Withdraw.Cointoken,
				Amount:    amtResult,
				Note:      pl.Withdraw.Note,
				ExecName:  pl.Withdraw.ExecName,
				To:        pl.Withdraw.To,
			}
		} else if pl, ok := action.Value.(*cty.CoinsAction_Genesis); ok {
			amt := float64(pl.Genesis.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsGenesisCLI{
				Amount:        amtResult,
				ReturnAddress: pl.Genesis.ReturnAddress,
			}
		} else if pl, ok := action.Value.(*cty.CoinsAction_TransferToExec); ok {
			amt := float64(pl.TransferToExec.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsTransferToExecCLI{
				Cointoken: pl.TransferToExec.Cointoken,
				Amount:    amtResult,
				Note:      pl.TransferToExec.Note,
				ExecName:  pl.TransferToExec.ExecName,
				To:        pl.TransferToExec.To,
			}
		}
	case types.HashlockX:
		var action hlt.HashlockAction
		bt, _ := common.FromHex(tx.RawPayload)
		types.Decode(bt, &action)
		if pl, ok := action.Value.(*hlt.HashlockAction_Hlock); ok {
			amt := float64(pl.Hlock.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &HashlockLockCLI{
				Amount:        amtResult,
				Time:          pl.Hlock.Time,
				Hash:          pl.Hlock.Hash,
				ToAddress:     pl.Hlock.ToAddress,
				ReturnAddress: pl.Hlock.ReturnAddress,
			}
		}
	case types.TicketX:
		var action types.TicketAction
		bt, _ := common.FromHex(tx.RawPayload)
		types.Decode(bt, &action)
		if pl, ok := action.Value.(*types.TicketAction_Miner); ok {
			amt := float64(pl.Miner.Reward) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &TicketMinerCLI{
				Bits:     pl.Miner.Bits,
				Reward:   amtResult,
				TicketId: pl.Miner.TicketId,
				Modify:   pl.Miner.Modify,
			}
		}
	case types.TokenX:
		var action types.TokenAction
		bt, _ := common.FromHex(tx.RawPayload)
		types.Decode(bt, &action)
		if pl, ok := action.Value.(*types.TokenAction_Tokenprecreate); ok {
			amt := float64(pl.Tokenprecreate.Price) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &TokenPreCreateCLI{
				Name:         pl.Tokenprecreate.Name,
				Symbol:       pl.Tokenprecreate.Symbol,
				Introduction: pl.Tokenprecreate.Introduction,
				Total:        pl.Tokenprecreate.Total,
				Price:        amtResult,
				Owner:        pl.Tokenprecreate.Owner,
			}
		} else if pl, ok := action.Value.(*types.TokenAction_Transfer); ok {
			amt := float64(pl.Transfer.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsTransferCLI{
				Cointoken: pl.Transfer.Cointoken,
				Amount:    amtResult,
				Note:      pl.Transfer.Note,
				To:        pl.Transfer.To,
			}
		} else if pl, ok := action.Value.(*types.TokenAction_Withdraw); ok {
			amt := float64(pl.Withdraw.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsWithdrawCLI{
				Cointoken: pl.Withdraw.Cointoken,
				Amount:    amtResult,
				Note:      pl.Withdraw.Note,
				ExecName:  pl.Withdraw.ExecName,
				To:        pl.Withdraw.To,
			}
		} else if pl, ok := action.Value.(*types.TokenAction_Genesis); ok {
			amt := float64(pl.Genesis.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsGenesisCLI{
				Amount:        amtResult,
				ReturnAddress: pl.Genesis.ReturnAddress,
			}
		} else if pl, ok := action.Value.(*types.TokenAction_TransferToExec); ok {
			amt := float64(pl.TransferToExec.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &CoinsTransferToExecCLI{
				Cointoken: pl.TransferToExec.Cointoken,
				Amount:    amtResult,
				Note:      pl.TransferToExec.Note,
				ExecName:  pl.TransferToExec.ExecName,
				To:        pl.TransferToExec.To,
			}
		}
	case types.PrivacyX:
		var action types.PrivacyAction
		bt, _ := common.FromHex(tx.RawPayload)
		types.Decode(bt, &action)
		if pl, ok := action.Value.(*types.PrivacyAction_Public2Privacy); ok {
			amt := float64(pl.Public2Privacy.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &Public2PrivacyCLI{
				Tokenname: pl.Public2Privacy.Tokenname,
				Amount:    amtResult,
				Note:      pl.Public2Privacy.Note,
				Output:    decodePrivOutput(pl.Public2Privacy.Output),
			}
		} else if pl, ok := action.Value.(*types.PrivacyAction_Privacy2Privacy); ok {
			amt := float64(pl.Privacy2Privacy.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &Privacy2PrivacyCLI{
				Tokenname: pl.Privacy2Privacy.Tokenname,
				Amount:    amtResult,
				Note:      pl.Privacy2Privacy.Note,
				Input:     decodePrivInput(pl.Privacy2Privacy.Input),
				Output:    decodePrivOutput(pl.Privacy2Privacy.Output),
			}
		} else if pl, ok := action.Value.(*types.PrivacyAction_Privacy2Public); ok {
			amt := float64(pl.Privacy2Public.Amount) / float64(types.Coin)
			amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
			result.Payload = &Privacy2PublicCLI{
				Tokenname: pl.Privacy2Public.Tokenname,
				Amount:    amtResult,
				Note:      pl.Privacy2Public.Note,
				Input:     decodePrivInput(pl.Privacy2Public.Input),
				Output:    decodePrivOutput(pl.Privacy2Public.Output),
			}
		}
	default:
	}
	return result
}

func decodePrivInput(input *types.PrivacyInput) *PrivacyInput {
	inputResult := &PrivacyInput{}
	for _, value := range input.Keyinput {
		amt := float64(value.Amount) / float64(types.Coin)
		amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
		var ugis []*UTXOGlobalIndex
		for _, u := range value.UtxoGlobalIndex {
			ugis = append(ugis, &UTXOGlobalIndex{Outindex: u.Outindex, Txhash: common.ToHex(u.Txhash)})
		}
		kin := &KeyInput{
			Amount:          amtResult,
			KeyImage:        common.ToHex(value.KeyImage),
			UtxoGlobalIndex: ugis,
		}
		inputResult.Keyinput = append(inputResult.Keyinput, kin)
	}
	return inputResult
}

func decodePrivOutput(output *types.PrivacyOutput) *PrivacyOutput {
	outputResult := &PrivacyOutput{RpubKeytx: common.ToHex(output.RpubKeytx)}
	for _, value := range output.Keyoutput {
		amt := float64(value.Amount) / float64(types.Coin)
		amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
		kout := &KeyOutput{
			Amount:        amtResult,
			Onetimepubkey: common.ToHex(value.Onetimepubkey),
		}
		outputResult.Keyoutput = append(outputResult.Keyoutput, kout)
	}
	return outputResult
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

func constructAccFromLog(l *rpctypes.ReceiptLogResult, key string) *types.Account {
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

//这里需要重构
func decodeLog(rlog rpctypes.ReceiptDataResult) *ReceiptData {
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
			types.TyLogRelayRcvBTCHead, types.TyLogRelayConfirmTx, types.TyLogRelayFinishTx,
			bwty.TyLogBlackwhiteCreate, bwty.TyLogBlackwhiteShow, bwty.TyLogBlackwhitePlay,
			bwty.TyLogBlackwhiteTimeout, bwty.TyLogBlackwhiteDone, bwty.TyLogBlackwhiteLoopInfo,
			types.TyLogLotteryCreate, types.TyLogLotteryBuy, types.TyLogLotteryDraw, types.TyLogLotteryClose,
			pt.TyLogParacrossMiner, pt.TyLogParaAssetTransfer, pt.TyLogParaAssetWithdraw, pt.TyLogParacrossCommit:

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
			// 隐私交易
		case types.TyLogPrivacyInput:
			rl.Log = buildPrivacyInputResult(l)
		case types.TyLogPrivacyOutput:
			rl.Log = buildPrivacyOutputResult(l)
		default:
			fmt.Printf("---The log with vlaue:%d is not decoded --------------------\n", l.Ty)
			return nil
		}
		rd.Logs = append(rd.Logs, rl)
	}
	return rd
}

func buildPrivacyInputResult(l *rpctypes.ReceiptLogResult) interface{} {
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

func buildPrivacyOutputResult(l *rpctypes.ReceiptLogResult) interface{} {
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

	var res rpctypes.ReplyHash
	ctx := NewRpcCtx(rpcAddr, "Chain33.SendToAddress", params, &res)
	ctx.Run()
}

func CreateRawTx(cmd *cobra.Command, to string, amount float64, note string, isWithdraw, isToken bool, tokenSymbol, execName string) (string, error) {
	if amount < 0 {
		return "", types.ErrAmount
	}
	paraName, _ := cmd.Flags().GetString("paraName")
	amountInt64 := int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4
	initExecName := execName
	execName = getRealExecName(paraName, execName)
	if execName != "" && !types.IsAllowExecName([]byte(execName), []byte(execName)) {
		return "", types.ErrExecNameNotMatch
	}
	var tx *types.Transaction
	if !isToken {
		transfer := &cty.CoinsAction{}
		if !isWithdraw {
			if initExecName != "" {
				v := &cty.CoinsAction_TransferToExec{TransferToExec: &types.AssetsTransferToExec{Amount: amountInt64, Note: note, ExecName: execName}}
				transfer.Value = v
				transfer.Ty = cty.CoinsActionTransferToExec
			} else {
				v := &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amountInt64, Note: note, To: to}}
				transfer.Value = v
				transfer.Ty = cty.CoinsActionTransfer
			}
		} else {
			v := &cty.CoinsAction_Withdraw{Withdraw: &types.AssetsWithdraw{Amount: amountInt64, Note: note, ExecName: execName}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionWithdraw
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
			v := &types.TokenAction_Transfer{Transfer: &types.AssetsTransfer{Cointoken: tokenSymbol, Amount: amountInt64, Note: note, To: to}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Withdraw{Withdraw: &types.AssetsWithdraw{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
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
	if ok := types.IsAllowExecName([]byte(exec), []byte(exec)); !ok {
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
