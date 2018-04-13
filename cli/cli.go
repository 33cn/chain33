package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cli/commands"
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
	rootCmd.PersistentFlags().String("rpc_laddr", "http://localhost:8801", "RPC listen address")

	rootCmd.AddCommand(
		commands.CommonCmd(),
		commands.SeedCmd(),
		commands.KeyCmd(),
		commands.AccountCmd(),
		commands.TxCmd(),
		commands.TokenCmd(),
		commands.TicketCmd(),
		commands.MempoolCmd(),
		commands.WalletCmd(),
		commands.AddressCmd(),
		commands.BlockCmd(),
		versionCmd)
}

func main() {
	log.SetLogLevel("error")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
