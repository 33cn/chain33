package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/system/dapp/commands"
	"gitlab.33.cn/chain33/chain33/types"
)

var rootCmd = &cobra.Command{
	Use:   types.GetTitle() + "-cli",
	Short: types.GetTitle() + " client tools",
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send transaction in one move",
	Run:   func(cmd *cobra.Command, args []string) {},
}

var closeCmd = &cobra.Command{
	Use:   "close",
	Short: "Close " + types.GetTitle(),
	Run: func(cmd *cobra.Command, args []string) {
		rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
		//		rpc, _ := jsonrpc.NewJSONClient(rpcLaddr)
		//		rpc.Call("Chain33.CloseQueue", nil, nil)
		var res rpctypes.Reply
		ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CloseQueue", nil, &res)
		ctx.Run()
	},
}

func init() {
	rootCmd.AddCommand(
		commands.AccountCmd(),
		commands.BlockCmd(),
		commands.BTYCmd(),
		commands.CoinsCmd(),
		commands.ExecCmd(),
		commands.MempoolCmd(),
		commands.NetCmd(),
		commands.SeedCmd(),
		commands.StatCmd(),
		commands.TxCmd(),
		commands.WalletCmd(),
		commands.VersionCmd(),
		sendCmd,
		closeCmd,
	)
}

func Run(RPCAddr, ParaName string) {
	pluginmgr.AddCmd(rootCmd)
	log.SetLogLevel("error")
	types.S("RPCAddr", RPCAddr)
	types.S("ParaName", ParaName)
	rootCmd.PersistentFlags().String("rpc_laddr", types.GStr("RPCAddr"), "http url")
	rootCmd.PersistentFlags().String("paraName", types.GStr("ParaName"), "parachain")
	if len(os.Args) > 1 {
		if os.Args[1] == "send" {
			commands.OneStepSend(os.Args)
			return
		}
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
