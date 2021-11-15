// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package commands 管理插件命令
package commands

import (
	"github.com/33cn/chain33/util"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	mty "github.com/33cn/chain33/system/dapp/manage/types"
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
		ConfigApplyCmd(),
		ConfigApproveCmd(),
		QueryConfigCmd(),
		QueryConfigIDCmd(),
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
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	v := &types.ModifyConfig{Key: key, Op: op, Value: opAddr, Addr: ""}
	params := &rpctypes.CreateTxIn{
		Execer:     util.GetParaExecName(paraName, mty.ManageX),
		ActionName: "Modify",
		Payload:    types.MustPBToJSON(v),
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

// ConfigApplyCmd config transaction
func ConfigApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "apply system config",
		Run:   configApply,
	}
	addConfigApplyFlags(cmd)
	return cmd
}

func addConfigApplyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("config_key", "c", "", "config key string")
	cmd.MarkFlagRequired("config_key")

	cmd.Flags().StringP("operation", "o", "", "adding or deletion operation")
	cmd.MarkFlagRequired("operation")

	cmd.Flags().StringP("value", "v", "", "operating object")
	cmd.MarkFlagRequired("value")

}

func configApply(cmd *cobra.Command, args []string) {
	key, _ := cmd.Flags().GetString("config_key")
	op, _ := cmd.Flags().GetString("operation")
	opAddr, _ := cmd.Flags().GetString("value")
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	paraName, _ := cmd.Flags().GetString("paraName")
	v := &types.ModifyConfig{Key: key, Op: op, Value: opAddr, Addr: ""}
	apply := &mty.ApplyConfig{Modify: v}
	params := &rpctypes.CreateTxIn{
		Execer:     util.GetParaExecName(paraName, mty.ManageX),
		ActionName: "ApplyConfig",
		Payload:    types.MustPBToJSON(apply),
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

// ConfigApproveCmd config transaction
func ConfigApproveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "approve system config id",
		Run:   configApprove,
	}
	addConfigApproveFlags(cmd)
	return cmd
}

func addConfigApproveFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("config_id", "i", "", "config id string")
	cmd.MarkFlagRequired("config_id")

	cmd.Flags().StringP("approve_id", "a", "", "autonomy approved id string")
	cmd.MarkFlagRequired("approve_id")

}

func configApprove(cmd *cobra.Command, args []string) {
	id, _ := cmd.Flags().GetString("config_id")
	approveId, _ := cmd.Flags().GetString("approve_id")

	paraName, _ := cmd.Flags().GetString("paraName")
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	//cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, errors.Wrapf(err, "GetChainConfig"))
	//	return
	//}
	v := &mty.ApproveConfig{Id: id, ApproveId: approveId}
	//modify := &mty.ManageAction{
	//	Ty:    mty.ManageActionApproveConfig,
	//	Value: &mty.ManageAction_Approve{Approve: v},
	//}
	//txRaw := &types.Transaction{Payload: types.Encode(modify)}
	//tx, err := types.FormatTxExt(cfg.ChainID, len(paraName) > 0, cfg.MinTxFeeRate, util.GetParaExecName(paraName, "manage"), txRaw)
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, err)
	//	return
	//}
	//txHex := types.Encode(tx)
	//fmt.Println(hex.EncodeToString(txHex))

	params := &rpctypes.CreateTxIn{
		Execer:     util.GetParaExecName(paraName, mty.ManageX),
		ActionName: "ApproveConfig",
		Payload:    types.MustPBToJSON(v),
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()

}

// QueryConfigCmd  query config
func QueryConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
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

// QueryConfigIDCmd  query config id
func QueryConfigIDCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "id",
		Short: "Query config apply id",
		Run:   queryConfigId,
	}
	addQueryConfigIdFlags(cmd)
	return cmd
}

func addQueryConfigIdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("id", "i", "", "id string")
	cmd.MarkFlagRequired("id")
}

func queryConfigId(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	paraName, _ := cmd.Flags().GetString("paraName")
	key, _ := cmd.Flags().GetString("id")
	req := &types.ReqString{
		Data: key,
	}
	var params rpctypes.Query4Jrpc
	params.Execer = util.GetParaExecName(paraName, "manage")
	params.FuncName = "GetConfigID"
	params.Payload = types.MustPBToJSON(req)

	var res mty.ConfigStatus
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.Run()
}
