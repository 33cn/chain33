package commands

import (
	"github.com/spf13/cobra"
	"github.com/33cn/chain33/rpc/jsonclient"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Get node version",
		Run:   version,
	}

	return cmd
}

func version(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.Version", nil, nil)
	ctx.RunWithoutMarshal()
}
