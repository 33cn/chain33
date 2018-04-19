package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func SaveSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save seed and encrypt with passwd",
		Run:   saveSeed,
	}
	addSaveSeedFlags(cmd)
	return cmd
}

func addSaveSeedFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("seed", "s", "", "15 seed characters seperated by space")
	cmd.MarkFlagRequired("seed")

	cmd.Flags().StringP("pwd", "p", "", "password used to encrypt seed")
	cmd.MarkFlagRequired("pwd")
}

func saveSeed(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	seed, _ := cmd.Flags().GetString("seed")
	pwd, _ := cmd.Flags().GetString("pwd")
	params := types.SaveSeedByPw{
		Seed:   seed,
		Passwd: pwd,
	}
	var res jsonrpc.Reply
	ctx := NewRPCCtx(rpcLaddr, "Chain33.SaveSeed", params, &res)
	ctx.Run()
}
