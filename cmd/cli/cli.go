package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cmd/cli/commands"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/log"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
)

var rootCmd = &cobra.Command{
	Use:   "chain33-cli",
	Short: "chain33 client tools",
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send transaction in one move",
	Run:   func(cmd *cobra.Command, args []string) {},
}

var closeCmd = &cobra.Command{
	Use:   "close",
	Short: "Close chain33",
	Run: func(cmd *cobra.Command, args []string) {
		rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
		//		rpc, _ := jsonrpc.NewJSONClient(rpcLaddr)
		//		rpc.Call("Chain33.CloseQueue", nil, nil)
		var res jsonrpc.Reply
		ctx := commands.NewRpcCtx(rpcLaddr, "Chain33.CloseQueue", nil, &res)
		ctx.Run()
	},
}

func init() {
	if config.RPCAddr == "" {
		config.RPCAddr = "http://localhost:8801"
	}
	rootCmd.PersistentFlags().String("rpc_laddr", config.RPCAddr, "http url")
	rootCmd.PersistentFlags().String("paraName", config.ParaName, "parachain")

	rootCmd.AddCommand(
		commands.AccountCmd(),
		commands.BlockCmd(),
		commands.BTYCmd(),
		commands.CoinsCmd(),
		commands.ConfigCmd(),
		commands.EvmCmd(),
		commands.ExecCmd(),
		commands.HashlockCmd(),
		commands.MempoolCmd(),
		commands.NetCmd(),
		commands.RelayCmd(),
		commands.RetrieveCmd(),
		commands.SeedCmd(),
		commands.StatCmd(),
		commands.TicketCmd(),
		commands.TokenCmd(),
		commands.TradeCmd(),
		commands.TxCmd(),
		commands.WalletCmd(),
		commands.PrivacyCmd(),
		commands.BlackwhiteCmd(),
		commands.VersionCmd(),
		sendCmd,
		closeCmd,
	)
}

func main() {
	log.SetLogLevel("error")
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
