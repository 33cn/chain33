package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func WalletCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "Wallet management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		LockCmd(),
		UnlockCmd(),
		WalletStatusCmd(),
		SetPwdCmd(),
		WalletListTxsCmd(),
		MergeBalanceCmd(),
		AutoMineCmd(),
		SignRawTxCmd(),
		SetFeeCmd(),
	)

	return cmd
}

// lock the wallet
func LockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Lock wallet",
		Run:   lock,
	}
	return cmd
}

func lock(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res jsonrpc.Reply
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Lock", nil, &res)
	ctx.Run()
}

// unlock the wallet
func UnlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock wallet",
		Run:   unLock,
	}
	addUnlockFlags(cmd)
	return cmd
}

func addUnlockFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pwd", "p", "", "password needed to unlock")
	cmd.MarkFlagRequired("pwd")

	cmd.Flags().Int64P("time_out", "t", 0, "time out for unlock operation(0 for unlimited)")
	cmd.Flags().StringP("scope", "s", "wallet", "unlock scope(wallet/ticket)")
}

func unLock(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pwd, _ := cmd.Flags().GetString("pwd")
	timeOut, _ := cmd.Flags().GetInt64("time_out")
	scope, _ := cmd.Flags().GetString("scope")
	var walletOrTicket bool
	switch scope {
	case "wallet":
		walletOrTicket = false
	case "ticket":
		walletOrTicket = true
	default:
		fmt.Println("unlock scope code wrong: wallet or ticket")
		return
	}

	params := types.WalletUnLock{
		Passwd:         pwd,
		Timeout:        timeOut,
		WalletOrTicket: walletOrTicket,
	}
	var res jsonrpc.Reply
	ctx := NewRpcCtx(rpcLaddr, "Chain33.UnLock", params, &res)
	ctx.Run()
}

// get wallet status
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
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetWalletStatus", nil, &res)
	ctx.Run()
}

// set password
func SetPwdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set_pwd",
		Short: "Set password",
		Run:   setPwd,
	}
	addSetPwdFlags(cmd)
	return cmd
}

func addSetPwdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("old", "o", "", "old password")
	cmd.MarkFlagRequired("old")

	cmd.Flags().StringP("new", "n", "", "new password")
	cmd.MarkFlagRequired("new")
}

func setPwd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	oldPwd, _ := cmd.Flags().GetString("old")
	newPwd, _ := cmd.Flags().GetString("new")
	params := types.ReqWalletSetPasswd{
		OldPass: oldPwd,
		NewPass: newPwd,
	}
	var res jsonrpc.Reply
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SetPasswd", params, &res)
	ctx.Run()
}

// get wallet transactions
func WalletListTxsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list_txs",
		Short: "List transactions in wallet",
		Run:   walletListTxs,
	}
	addWalletListTxsFlags(cmd)
	return cmd
}

func addWalletListTxsFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "from which transaction begin")
	cmd.MarkFlagRequired("from")

	cmd.Flags().Int32P("count", "c", 0, "number of transactions")
	cmd.MarkFlagRequired("count")

	cmd.Flags().Int32P("direction", "d", 1, "query direction (0: pre page, 1: next page)")
}

func walletListTxs(cmd *cobra.Command, args []string) {
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
	ctx := NewRpcCtx(rpcLaddr, "Chain33.WalletTxList", params, &res)
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
			Blocktime:  v.BlockTime,
			Amount:     amountResult,
			Fromaddr:   v.FromAddr,
			Txhash:     v.TxHash,
			ActionName: v.ActionName,
		}
		result.TxDetails = append(result.TxDetails, wtxd)
	}
	return result, nil
}

// merge all balance to an account
func MergeBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge accounts' balance into address",
		Run:   mergeBalance,
	}
	addMergeBalanceFlags(cmd)
	return cmd
}

func addMergeBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "destination account address")
	cmd.MarkFlagRequired("to")
}

func mergeBalance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	toAddr, _ := cmd.Flags().GetString("to")
	params := types.ReqWalletMergeBalance{
		To: toAddr,
	}
	var res jsonrpc.ReplyHashes
	ctx := NewRpcCtx(rpcLaddr, "Chain33.MergeBalance", params, &res)
	ctx.Run()
}

// set auto mining
func AutoMineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto_mine",
		Short: "Set auto mine on/off",
		Run:   autoMine,
	}
	addAutoMineFlags(cmd)
	return cmd
}

func addAutoMineFlags(cmd *cobra.Command) {
	cmd.Flags().Int32P("flag", "f", 0, `auto mine(0: off, 1: on)`)
	cmd.MarkFlagRequired("flag")
}

func autoMine(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	flag, _ := cmd.Flags().GetInt32("flag")
	if flag != 0 && flag != 1 {
		cmd.UsageFunc()(cmd)
		return
	}
	params := types.MinerFlag{
		Flag: flag,
	}

	var res jsonrpc.Reply
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SetAutoMining", params, &res)
	ctx.Run()
}

// sign raw tx
func SignRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign transaction",
		Run:   signRawTx,
	}
	addSignRawTxFlags(cmd)
	return cmd
}

func addSignRawTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "raw transaction data")
	cmd.MarkFlagRequired("data")

	cmd.Flags().StringP("key", "k", "", "private key (optional)")
	cmd.Flags().StringP("addr", "a", "", "account address (optional)")
	cmd.Flags().StringP("expire", "e", "120s", "transaction expire time")
	// A duration string is a possibly signed sequence of
	// decimal numbers, each with optional fraction and a unit suffix,
	// such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
}

func signRawTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	data, _ := cmd.Flags().GetString("data")
	key, _ := cmd.Flags().GetString("key")
	addr, _ := cmd.Flags().GetString("addr")
	expire, _ := cmd.Flags().GetString("expire")
	params := types.ReqSignRawTx{
		Privkey: key,
		Addr:    addr,
		TxHex:   data,
		Expire:  expire,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SignRawTx", params, &res)
	ctx.Run()
}

// set tx fee
func SetFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set_fee",
		Short: "Set transaction fee",
		Run:   setFee,
	}
	addSetFeeFlags(cmd)
	return cmd
}

func addSetFeeFlags(cmd *cobra.Command) {
	cmd.Flags().Float64P("amount", "a", 0, "tx fee amount")
	cmd.MarkFlagRequired("amount")
}

func setFee(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount, _ := cmd.Flags().GetFloat64("amount")
	amountInt64 := int64(amount * 1e4)
	params := types.ReqWalletSetFee{
		Amount: amountInt64 * 1e4,
	}
	var res jsonrpc.Reply
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SetTxFee", params, &res)
	ctx.Run()
}
