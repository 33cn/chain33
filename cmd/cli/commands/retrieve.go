package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	retrievetype "gitlab.33.cn/chain33/chain33/types/executor/retrieve"
)

func RetrieveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retrieve",
		Short: "Wallet retrieve operation",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		BackupCmd(),
		PrepareCmd(),
		PerformCmd(),
		CancelCmd(),
		QueryCmd(),
	)

	return cmd
}

// 备份
func BackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup the wallet",
		Run:   backupCmd,
	}
	addBakupCmdFlags(cmd)
	return cmd
}

func addBakupCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("backup", "b", "", "backup address")
	cmd.MarkFlagRequired("backup")
	cmd.Flags().StringP("default", "t", "", "default address")
	cmd.MarkFlagRequired("default")
	cmd.Flags().Int64P("delay", "d", 0, "delay period")
	cmd.MarkFlagRequired("delay")

	defaultFee := float64(types.MinFee) / float64(types.Coin)
	cmd.Flags().Float64P("fee", "f", defaultFee, "transaction fee")
}

func backupCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	backup, _ := cmd.Flags().GetString("backup")
	defaultAddr, _ := cmd.Flags().GetString("default")
	delay, _ := cmd.Flags().GetInt64("delay")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := retrievetype.RetrieveBackupTx{
		BackupAddr:  backup,
		DefaultAddr: defaultAddr,
		DelayPeriod: delay,
		Fee:         feeInt64,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrieveBackupTx", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

// 准备
func PrepareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the wallet",
		Run:   prepareCmd,
	}
	addRetrieveCmdFlags(cmd)
	return cmd
}

func addRetrieveCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("backup", "b", "", "backup address")
	cmd.MarkFlagRequired("backup")
	cmd.Flags().StringP("default", "t", "", "default address")
	cmd.MarkFlagRequired("default")

	defaultFee := float64(types.MinFee) / float64(types.Coin)
	cmd.Flags().Float64P("fee", "f", defaultFee, "sign address")
}

func prepareCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	backup, _ := cmd.Flags().GetString("backup")
	defaultAddr, _ := cmd.Flags().GetString("default")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := retrievetype.RetrievePrepareTx{
		BackupAddr:  backup,
		DefaultAddr: defaultAddr,
		Fee:         feeInt64,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrievePrepareTx", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

// 执行
func PerformCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perform",
		Short: "Perform the retrieve",
		Run:   performCmd,
	}
	addRetrieveCmdFlags(cmd)
	return cmd
}

func performCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	backup, _ := cmd.Flags().GetString("backup")
	defaultAddr, _ := cmd.Flags().GetString("default")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := retrievetype.RetrievePerformTx{
		BackupAddr:  backup,
		DefaultAddr: defaultAddr,
		Fee:         feeInt64,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrievePerformTx", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

// 取消
func CancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the retrieve",
		Run:   cancelCmd,
	}
	addRetrieveCmdFlags(cmd)
	return cmd
}

func cancelCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	backup, _ := cmd.Flags().GetString("backup")
	defaultAddr, _ := cmd.Flags().GetString("default")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := retrievetype.RetrieveCancelTx{
		BackupAddr:  backup,
		DefaultAddr: defaultAddr,
		Fee:         feeInt64,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrieveCancelTx", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

// 查询
func QueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Backup the wallet",
		Run:   queryRetrieveCmd,
	}
	addQueryRetrieveCmdFlags(cmd)
	return cmd
}

func addQueryRetrieveCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("backup", "b", "", "backup address")
	cmd.MarkFlagRequired("backup")
	cmd.Flags().StringP("default", "t", "", "default address")
	cmd.MarkFlagRequired("default")
}

func queryRetrieveCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	backup, _ := cmd.Flags().GetString("backup")
	defaultAddr, _ := cmd.Flags().GetString("default")

	req := &types.ReqRetrieveInfo{
		BackupAddress:  backup,
		DefaultAddress: defaultAddr,
	}

	var params jsonrpc.Query4Cli
	params.Execer = "retrieve"
	params.FuncName = "GetRetrieveInfo"
	params.Payload = req

	var res types.RetrieveQuery
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}
