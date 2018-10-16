package main

import (
	"os"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"

	"gitlab.33.cn/chain33/chain33/cmd/exec_creator_guider/commands"
	"gitlab.33.cn/chain33/chain33/common/log"
)

var (
	mlog = log15.New("module", "ecg")
)

func addCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(
		commands.SimpleCmd(),
		commands.AdvanceCmd(),
	)
}

func runCommands() {
	rootCmd := &cobra.Command{
		Use:   "ecg",
		Short: "chain33 executor project create tools",
	}

	addCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		mlog.Error("Execute command failed.", "error", err)
		os.Exit(1)
	}
}

func main() {
	log.SetLogLevel("debug")
	runCommands()
}
