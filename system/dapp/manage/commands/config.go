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
		ListConfigItemCmd(),
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
	apply := &mty.ApplyConfig{Config: v}
	params := &rpctypes.CreateTxIn{
		Execer:     util.GetParaExecName(paraName, mty.ManageX),
		ActionName: "Apply",
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
	approveID, _ := cmd.Flags().GetString("approve_id")

	paraName, _ := cmd.Flags().GetString("paraName")
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	v := &mty.ApproveConfig{ApplyConfigId: id, AutonomyItemId: approveID}

	params := &rpctypes.CreateTxIn{
		Execer:     util.GetParaExecName(paraName, mty.ManageX),
		ActionName: "Approve",
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
		Short: "Query config apply id status",
		Run:   queryConfigID,
	}
	addQueryConfigIDFlags(cmd)
	return cmd
}

func addQueryConfigIDFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("id", "i", "", "config id string")
	cmd.MarkFlagRequired("id")
}

func queryConfigID(cmd *cobra.Command, args []string) {
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

// ListConfigItemCmd list config item
func ListConfigItemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list config id",
		Run:   listConfigItem,
	}
	listConfigItemflags(cmd)
	return cmd
}

func listConfigItemflags(cmd *cobra.Command) {
	cmd.Flags().Uint32P("status", "s", 0, "status 1:apply, 2:approved")
	cmd.Flags().StringP("addr", "a", "", "config proposer")
	cmd.Flags().Int32P("count", "c", 1, "count, default is 1")
	cmd.Flags().Int32P("direction", "d", 0, "direction, default is reserve")
	cmd.Flags().Int64P("height", "t", -1, "height, default is -1")
	cmd.Flags().Int32P("index", "i", -1, "index, default is -1")
}

func listConfigItem(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	status, _ := cmd.Flags().GetUint32("status")
	addr, _ := cmd.Flags().GetString("addr")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("direction")
	height, _ := cmd.Flags().GetInt64("height")
	index, _ := cmd.Flags().GetInt32("index")

	var params rpctypes.Query4Jrpc
	params.Execer = mty.ManageX
	req := &mty.ReqQueryConfigList{
		Status:    int32(status),
		Proposer:  addr,
		Count:     count,
		Direction: direction,
		Height:    height,
		Index:     index,
	}
	params.FuncName = "ListConfigID"
	params.Payload = types.MustPBToJSON(req)

	var rep mty.ReplyQueryConfigList
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &rep)
	ctx.Run()
}
