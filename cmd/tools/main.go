package main

import (
	"os"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cmd/tools/commands"
	"gitlab.33.cn/chain33/chain33/common/log"
)

var (
	mlog = log15.New("module", "tools")
)

func main() {
	log.SetLogLevel("debug")
	runCommands()
}

func addCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(
		commands.SimpleCmd(),
		commands.AdvanceCmd(),
		commands.ImportCmd(),
	)
}

func runCommands() {
	rootCmd := &cobra.Command{
		Use:   "tools",
		Short: "chain33 tools",
	}

	addCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		mlog.Error("Execute command failed.", "error", err)
		os.Exit(1)
	}
}
