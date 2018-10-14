package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	retrievetype "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	"gitlab.33.cn/chain33/chain33/types"
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
		RetrieveQueryCmd(),
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
	cmd.Flags().Int64P("delay", "d", 60, "delay period (minimum 60 seconds)")
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

	if delay < 60 {
		fmt.Println("delay period changed to 60")
		delay = 60
	}
	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := retrievetype.RetrieveBackupTx{
		BackupAddr:  backup,
		DefaultAddr: defaultAddr,
		DelayPeriod: delay,
		Fee:         feeInt64,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrieveBackupTx", params, nil)
	ctx.RunWithoutMarshal()
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
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrievePrepareTx", params, nil)
	ctx.RunWithoutMarshal()
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
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrievePerformTx", params, nil)
	ctx.RunWithoutMarshal()
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
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CreateRawRetrieveCancelTx", params, nil)
	ctx.RunWithoutMarshal()
}

// 查询
func RetrieveQueryCmd() *cobra.Command {
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

func parseRerieveDetail(arg interface{}) (interface{}, error) {
	res := arg.(*retrievetype.RetrieveQuery)

	result := RetrieveResult{
		DelayPeriod: res.DelayPeriod,
	}
	switch res.Status {
	case RetrieveBackup:
		result.Status = "backup"
	case RetrievePreapred:
		result.Status = "prepared"
	case RetrievePerformed:
		result.Status = "performed"
	case RetrieveCanceled:
		result.Status = "canceled"
	default:
		result.Status = "unknown"
	}

	return result, nil
}

func queryRetrieveCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	backup, _ := cmd.Flags().GetString("backup")
	defaultAddr, _ := cmd.Flags().GetString("default")

	req := &retrievetype.ReqRetrieveInfo{
		BackupAddress:  backup,
		DefaultAddress: defaultAddr,
	}

	var params types.Query4Cli
	params.Execer = "retrieve"
	params.FuncName = "GetRetrieveInfo"
	params.Payload = req

	var res retrievetype.RetrieveQuery
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseRerieveDetail)
	ctx.Run()
}
