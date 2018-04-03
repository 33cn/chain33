package commands

import (
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func GenSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate seed",
		Run:   genSeed,
	}
	addGenSeedFlags(cmd)
	return cmd
}

func addGenSeedFlags(cmd *cobra.Command) {
	cmd.Flags().Int32P("lang", "l", 0, "seed language(0:English, 1:简体中文)")
}

func genSeed(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	lang, _ := cmd.Flags().GetInt32("lang")
	params := types.GenSeedLang{
		Lang: lang,
	}
	var res types.ReplySeed
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GenSeed", params, &res)
	ctx.Run()
}
