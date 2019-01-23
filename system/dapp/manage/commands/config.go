// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package commands 管理插件命令
package commands

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/33cn/chain33/util"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	pty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// ConfigCmd config command
func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ConfigTxCmd(),
		QueryConfigCmd(),
	)

	return cmd
}

// ConfigTxCmd config transaction
func ConfigTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config_tx",
		Short: "set system config",
		Run:   configTx,
	}
	addConfigTxFlags(cmd)
	return cmd
}

func addConfigTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("config_key", "c", "", "config key string")
	cmd.MarkFlagRequired("config_key")

	cmd.Flags().StringP("operation", "o", "", "adding or deletion operation")
	cmd.MarkFlagRequired("operation")

	cmd.Flags().StringP("value", "v", "", "operating object")
	cmd.MarkFlagRequired("value")

}

func configTx(cmd *cobra.Command, args []string) {
	paraName, _ := cmd.Flags().GetString("paraName")
	key, _ := cmd.Flags().GetString("config_key")
	op, _ := cmd.Flags().GetString("operation")
	opAddr, _ := cmd.Flags().GetString("value")

	v := &types.ModifyConfig{Key: key, Op: op, Value: opAddr, Addr: ""}
	modify := &pty.ManageAction{
		Ty:    pty.ManageActionModifyConfig,
		Value: &pty.ManageAction_Modify{Modify: v},
	}
	tx := &types.Transaction{Payload: types.Encode(modify)}
	var err error
	tx, err = types.FormatTx(util.GetParaExecName(paraName, "manage"), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}

// QueryConfigCmd  query config
func QueryConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query_config",
		Short: "Query config item",
		Run:   queryConfig,
	}
	addQueryConfigFlags(cmd)
	return cmd
}

func addQueryConfigFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "key string")
	cmd.MarkFlagRequired("key")
}

func queryConfig(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	paraName, _ := cmd.Flags().GetString("paraName")
	key, _ := cmd.Flags().GetString("key")
	req := &types.ReqString{
		Data: key,
	}
	var params rpctypes.Query4Jrpc
	params.Execer = util.GetParaExecName(paraName, "manage")
	params.FuncName = "GetConfigItem"
	params.Payload = types.MustPBToJSON(req)

	var res types.ReplyConfig
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.Run()
}
