package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.33.cn/chain33/chain33/cmd/cpm/commands"
)

func initCommand(cmd *cobra.Command)  {
	cmd.AddCommand(commands.ImportCmd())
}

func main()  {
	rootCmd := &cobra.Command{
		Use:"cpm",
		Short:"chain33 package manager",
	}
	initCommand(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}