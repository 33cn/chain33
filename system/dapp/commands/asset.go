// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"
	"os"

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
	cmd.MarkFlagRequired("asset_exec")
	cmd.Flags().StringP("asset_symbol", "", "bty", "the asset symbol")
	cmd.MarkFlagRequired("asset_symbol")
}

func assetBalance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	asset_symbol, _ := cmd.Flags().GetString("asset_symbol")
	asset_exec, _ := cmd.Flags().GetString("asset_exec")

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

	var addrs []string
	addrs = append(addrs, addr)
	params := types.ReqBalance{
		Addresses:   addrs,
		Execer:      execer,
		StateHash:   "",
		AssetExec:   asset_exec,
		AssetSymbol: asset_symbol,
	}
	var res []*rpcTypes.Account
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetAssetBalance", params, &res)
	ctx.SetResultCb(parseGetBalanceRes)
	ctx.Run()
}
