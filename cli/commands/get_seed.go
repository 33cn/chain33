package commands

import (
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func GetSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get seed by password",
		Run:   getSeed,
	}
	addGetSeedFlags(cmd)
	return cmd
}

func addGetSeedFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pwd", "p", "", "password used to fetch seed")
	cmd.MarkFlagRequired("pwd")
}

func getSeed(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pwd, _ := cmd.Flags().GetString("pwd")
	params := types.GetSeedByPw{
		Passwd: pwd,
	}
	var res types.ReplySeed
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetSeed", params, &res)
	ctx.Run()
}
