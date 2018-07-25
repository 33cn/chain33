package commands

import (
	"github.com/spf13/cobra"
)

func RetriveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retrive",
		Short: "Wallet retrive operation",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		BackupCmd(),
		PrepareCmd(),
		PerformCmd(),
		CancelCmd(),
		QueryCmd(),
	)

	return cmd
}

func BackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup the wallet",
		Run:   backupCmd,
	}
	addBakupCmdFlags(cmd)
	return cmd
}

func PrepareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the wallet",
		Run:   prepareCmd,
	}
	addPrepareCmdFlags(cmd)
	return cmd
}

func PerformCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perform",
		Short: "Perform the retrive",
		Run:   performCmd,
	}
	addPerformCmdFlags(cmd)
	return cmd
}

func CancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the retrive",
		Run:   cancelCmd,
	}
	addCancelCmdFlags(cmd)
	return cmd
}

func QueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Backup the wallet",
		Run:   queryRetriveCmd,
	}
	addQueryRetriveCmdFlags(cmd)
	return cmd
}
