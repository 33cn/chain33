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

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common/address"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/types"

)

func CreateRawTx(cmd *cobra.Command, to string, amount float64, note string, isWithdraw bool, tokenSymbol, execName string) (string, error) {
	if amount < 0 {
		return "", types.ErrAmount
	}
	paraName, _ := cmd.Flags().GetString("paraName")
	amountInt64 := int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4
	execName = getRealExecName(paraName, execName)
	if execName != "" && !types.IsAllowExecName([]byte(execName), []byte(execName)) {
		return "", types.ErrExecNameNotMatch
	}
	var tx *types.Transaction
		transfer := &tokenty.TokenAction{}
		if !isWithdraw {
			v := &tokenty.TokenAction_Transfer{Transfer: &types.AssetsTransfer{Cointoken: tokenSymbol, Amount: amountInt64, Note: note, To: to}}
			transfer.Value = v
			transfer.Ty = tokenty.ActionTransfer
		} else {
			v := &tokenty.TokenAction_Withdraw{Withdraw: &types.AssetsWithdraw{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = tokenty.ActionWithdraw
		}
		execer := []byte(getRealExecName(paraName, "token"))
		if paraName == "" {
			tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: to}
		} else {
			tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: address.ExecAddress(string(execer))}
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
