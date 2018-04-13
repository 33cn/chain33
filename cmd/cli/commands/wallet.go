package commands

import (
	"strconv"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func WalletCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "Wallet managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		WalletStatusCmd(),
		WalletListTxsCmd(),
	)

	return cmd
}

// status
func WalletStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get wallet status",
		Run:   walletStatus,
	}
	return cmd
}

func walletStatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res jsonrpc.WalletStatus
	ctx := NewRPCCtx(rpcLaddr, "Chain33.GetWalletStatus", nil, &res)
	ctx.Run()
}

// list_txs
func WalletListTxsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list_txs",
		Short: "List transactions in wallet",
		Run:   WalletListTxs,
	}
	addWalletListTxsFlags(cmd)
	return cmd
}

func addWalletListTxsFlags(cmd *cobra.Command) {
	// TODO
	cmd.Flags().StringP("from", "f", "", "from which transaction begin (height:index)")
	cmd.MarkFlagRequired("from")

	cmd.Flags().Int32P("count", "c", 0, "number of transactions")
	cmd.MarkFlagRequired("count")

	cmd.Flags().Int32P("dir", "d", 0, "query direction (0: pre page, 1: next page)")
}

func WalletListTxs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	txHash, _ := cmd.Flags().GetString("from")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("dir")
	params := jsonrpc.ReqWalletTransactionList{
		FromTx:    txHash,
		Count:     count,
		Direction: direction,
	}
	var res jsonrpc.WalletTxDetails
	ctx := NewRPCCtx(rpcLaddr, "Chain33.WalletTxList", params, &res)
	ctx.SetResultCb(parseWalletTxListRes)
	ctx.Run()
}

func parseWalletTxListRes(arg interface{}) (interface{}, error) {
	res := arg.(*jsonrpc.WalletTxDetails)
	var result WalletTxDetailsResult
	for _, v := range res.TxDetails {
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(types.Coin), 'f', 4, 64)
		wtxd := &WalletTxDetailResult{
			Tx:         decodeTransaction(v.Tx),
			Receipt:    decodeLog(*(v.Receipt)),
			Height:     v.Height,
			Index:      v.Index,
			Blocktime:  v.Blocktime,
			Amount:     amountResult,
			Fromaddr:   v.Fromaddr,
			Txhash:     v.Txhash,
			ActionName: v.ActionName,
		}
		result.TxDetails = append(result.TxDetails, wtxd)
	}
	return result, nil
}
