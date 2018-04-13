package commands

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/types"
)

func DumpKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump private key for account address",
		Run:   dumpKey,
	}
	addDumpKeyFlags(cmd)
	return cmd
}

func addDumpKeyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "address of account")
	cmd.MarkFlagRequired("addr")
}

func dumpKey(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	params := types.ReqStr{
		Reqstr: addr,
	}
	var res types.ReplyStr
	ctx := NewRPCCtx(rpcLaddr, "Chain33.DumpPrivkey", params, &res)
	ctx.Run()
}
