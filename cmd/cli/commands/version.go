package commands

import "github.com/spf13/cobra"

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

	ctx := NewRpcCtx(rpcLaddr, "Chain33.Version", nil, nil)
	ctx.RunWithoutMarshal()
}
