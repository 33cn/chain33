package commands

import (
	"encoding/json"
	"fmt"
	"os"

	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func TicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Ticket managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		CountTicketCmd(),
		CloseTicketCmd(),
	)

	return cmd
}

// count
func CountTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "count",
		Short: "Get ticket count",
		Run:   countTicket,
	}
	return cmd
}

func countTicket(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res int64
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetTicketCount", nil, &res)
	ctx.Run()
}

// close
func CloseTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close ticket ",
		Run:   closeTicket,
	}
	return cmd
}

func closeTicket(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	status, err := GetWalletStatus(rpcLaddr, true)
	if err != nil {
		return
	}

	isAutoMining := status.(jsonrpc.WalletStatus).IsAutoMining
	if isAutoMining {
		fmt.Fprintln(os.Stderr, types.ErrMinerNotClosed)
		return
	}

	var res types.ReplyHashes
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CloseTickets", nil, &res)
	ctx.Run()
}

// TODO
func GetWalletStatus(rpcAddr string, isCloseTickets bool) (interface{}, error) {
	rpc, err := jsonrpc.NewJsonClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	var res jsonrpc.WalletStatus
	err = rpc.Call("Chain33.GetWalletStatus", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	if !isCloseTickets {
		fmt.Println(string(data))
	}

	return res, nil
}
