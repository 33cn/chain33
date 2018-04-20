package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/account"
)

func ExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Executor operation",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GetExecAddrCmd(),
	)

	return cmd
}

// get address of an execer
func GetExecAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addr",
		Short: "Get address of executor",
		Run:   getAddrByExec,
	}
	addGetAddrFlags(cmd)
	return cmd
}

func addGetAddrFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", `executor name ("none", "coins", "hashlock", "retrieve", "ticket", "token" and "trade" supported)`)
	cmd.MarkFlagRequired("exec")
}

func getAddrByExec(cmd *cobra.Command, args []string) {
	execer, _ := cmd.Flags().GetString("exec")
	switch execer {
	case "none", "coins", "hashlock", "retrieve", "ticket", "token", "trade":
		addrResult := account.ExecAddress(execer)
		result := addrResult.String()
		data, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fmt.Println(string(data))

	default:
		fmt.Println("only none, coins, hashlock, retrieve, ticket, token, trade supported")
	}
}
