package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cmd/cli/commands"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/version"
)

var rootCmd = &cobra.Command{
	Use:   "chain33-cli",
	Short: "chain33 client tools",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetVersion())
	},
}

func init() {
	rootCmd.PersistentFlags().String("rpc_laddr", "http://localhost:8801", "http url")

	rootCmd.AddCommand(
		commands.AccountCmd(),
		commands.BlockCmd(),
		commands.BTYCmd(),
		commands.ConfigCmd(),
		commands.ExecCmd(),
		commands.MempoolCmd(),
		commands.NetCmd(),
		commands.SeedCmd(),
		commands.StatCmd(),
		commands.TicketCmd(),
		commands.TokenCmd(),
		commands.TradeCmd(),
		commands.TxCmd(),
		commands.WalletCmd(),
		commands.EvmCmd(),
		versionCmd)
}

func main() {
	log.SetLogLevel("error")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
