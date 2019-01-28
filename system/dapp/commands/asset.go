// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpcTypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// AssetCmd command
func AssetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset",
		Short: "Asset query",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		GetAssetBalanceCmd(),
	)
	return cmd
}

// GetAssetBalanceCmd query asset balance
func GetAssetBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Query asset balance",
		Run:   assetBalance,
	}
	addAssetBalanceFlags(cmd)
	return cmd
}

func addAssetBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account addr")
	cmd.MarkFlagRequired("addr")
	cmd.Flags().StringP("exec", "e", "", getExecuterNameString())
	cmd.Flags().StringP("asset_exec", "", "coins", "the asset executor")
	cmd.Flags().StringP("asset_symbol", "", "bty", "the asset symbol")
	cmd.Flags().IntP("height", "", -1, "block height")
}

func assetBalance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	asset_symbol, _ := cmd.Flags().GetString("asset_symbol")
	asset_exec, _ := cmd.Flags().GetString("asset_exec")
	height, _ := cmd.Flags().GetInt("height")

	err := address.CheckAddress(addr)
	if err != nil {
		if err = address.CheckMultiSignAddress(addr); err != nil {
			fmt.Fprintln(os.Stderr, types.ErrInvalidAddress)
			return
		}
	}
	if execer == "" {
		execer = asset_exec
	}

	if ok := types.IsAllowExecName([]byte(execer), []byte(execer)); !ok {
		fmt.Fprintln(os.Stderr, types.ErrExecNameNotAllow)
		return
	}

	stateHash := ""
	if height >= 0 {
		params := types.ReqBlocks{
			Start:    int64(height),
			End:      int64(height),
			IsDetail: false,
		}
		var res rpcTypes.Headers
		ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetHeaders", params, &res)
		_, err := ctx.RunResult()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		h := res.Items[0]
		stateHash = h.StateHash
	}

	var addrs []string
	addrs = append(addrs, addr)
	params := types.ReqBalance{
		Addresses:   addrs,
		Execer:      execer,
		StateHash:   stateHash,
		AssetExec:   asset_exec,
		AssetSymbol: asset_symbol,
	}
	var res []*rpcTypes.Account
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetBalance", params, &res)
	ctx.SetResultCb(parseGetBalanceRes)
	ctx.Run()
}

// CreateAssetSendToExec 通用的创建 send_exec 交易， 额外指定资产合约
func CreateAssetSendToExec(cmd *cobra.Command, args []string, fromExec string) {
	paraName, _ := cmd.Flags().GetString("paraName")
	exec, _ := cmd.Flags().GetString("exec")
	exec = getRealExecName(paraName, exec)
	to, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")

	payload := &types.AssetsTransferToExec{
		To:        to,
		Amount:    int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4,
		Note:      []byte(note),
		Cointoken: symbol,
		ExecName:  exec,
	}

	params := &rpcTypes.CreateTxIn{
		Execer:     types.ExecName(fromExec),
		ActionName: "TransferToExec",
		Payload:    types.MustPBToJSON(payload),
	}

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

// CreateAssetWithdraw 通用的创建 withdraw 交易， 额外指定资产合约
func CreateAssetWithdraw(cmd *cobra.Command, args []string, fromExec string) {
	exec, _ := cmd.Flags().GetString("exec")
	paraName, _ := cmd.Flags().GetString("paraName")
	exec = getRealExecName(paraName, exec)
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")

	exec = getRealExecName(paraName, exec)
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	payload := &types.AssetsWithdraw{
		To:        execAddr,
		Amount:    int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4,
		Note:      []byte(note),
		Cointoken: symbol,
		ExecName:  exec,
	}
	params := &rpcTypes.CreateTxIn{
		Execer:     types.ExecName(fromExec),
		ActionName: "Withdraw",
		Payload:    types.MustPBToJSON(payload),
	}

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

// CreateAssetTransfer 通用的创建 transfer 交易， 额外指定资产合约
func CreateAssetTransfer(cmd *cobra.Command, args []string, fromExec string) {
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")

	payload := &types.AssetsTransfer{
		To:        toAddr,
		Amount:    int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4,
		Note:      []byte(note),
		Cointoken: symbol,
	}
	params := &rpcTypes.CreateTxIn{
		Execer:     types.ExecName(fromExec),
		ActionName: "Transfer",
		Payload:    types.MustPBToJSON(payload),
	}

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

// GetExecAddr 获取执行器地址
func GetExecAddr(exec string) (string, error) {
	if ok := types.IsAllowExecName([]byte(exec), []byte(exec)); !ok {
		return "", types.ErrExecNameNotAllow
	}

	addrResult := address.ExecAddress(exec)
	result := addrResult
	return result, nil
}

func getRealExecName(paraName string, name string) string {
	if strings.HasPrefix(name, "user.p.") {
		return name
	}
	return paraName + name
}
