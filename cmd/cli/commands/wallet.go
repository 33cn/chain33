package commands

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
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
		SendTxCmd(),
		QueryCacheTxByAddrCmd(),
		DeleteCacheTxCmd(),
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
		Oldpass: oldPwd,
		Newpass: newPwd,
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
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().Int32P("sendrecv", "s", 1, "send or recv flag (1: send, 2: recv)")
	cmd.Flags().Int32P("count", "c", 10, "number of transactions")
	cmd.Flags().StringP("starttxhash", "t", "", "from which transaction begin")
	cmd.Flags().Int32P("mode", "m", 0, "query mode. (0: normal, 1:privacy)")
	cmd.Flags().Int32P("direction", "d", 1, "query direction (0: pre page, 1: next page)")
	cmd.Flags().StringP("token", "n", "", "token name.(BTY supported)")
}

func walletListTxs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	txHash, _ := cmd.Flags().GetString("starttxhash")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("dir")
	mode, _ := cmd.Flags().GetInt32("mode")
	addr, _ := cmd.Flags().GetString("addr")
	sendRecvPrivacy, _ := cmd.Flags().GetInt32("sendrecv")
	tokenname, _ := cmd.Flags().GetString("token")
	params := jsonrpc.ReqWalletTransactionList{
		FromTx:          txHash,
		Count:           count,
		Direction:       direction,
		Mode:            mode,
		Address:         addr,
		SendRecvPrivacy: sendRecvPrivacy,
		TokenName:       tokenname,
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

	cmd.Flags().Int32P("index", "i", 0, "transaction index to be signed")
	cmd.Flags().StringP("key", "k", "", "private key (optional)")
	cmd.Flags().StringP("addr", "a", "", "account address (optional)")
	cmd.Flags().StringP("expire", "e", "120s", "transaction expire time")
	cmd.Flags().Int32P("mode", "m", 0, "transaction sign mode")
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
	index, _ := cmd.Flags().GetInt32("index")
	expire, _ := cmd.Flags().GetString("expire")
	expireTime, err := time.ParseDuration(expire)
	mode, _ := cmd.Flags().GetInt32("mode")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if expireTime < time.Minute*2 && expireTime != time.Second*0 {
		expire = "120s"
		fmt.Println("expire time must longer than 2 minutes, changed expire time into 2 minutes")
	}
	if key != "" {
		keyByte, err := common.FromHex(key)
		if err != nil || len(keyByte) == 0 {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		cr, err := crypto.New(types.GetSignatureTypeName(wallet.SignType))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		privKey, err := cr.PrivKeyFromBytes(keyByte)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		var tx types.Transaction
		bytes, err := common.FromHex(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		err = types.Decode(bytes, &tx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		tx.SetExpire(expireTime)
		group, err := tx.GetTxGroup()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if group == nil {
			tx.Sign(int32(wallet.SignType), privKey)
			txHex := types.Encode(&tx)
			signedTx := hex.EncodeToString(txHex)
			fmt.Println(signedTx)
			return
		}
		if int(index) > len(group.GetTxs()) {
			fmt.Fprintln(os.Stderr, types.ErrIndex)
			return
		}
		if index <= 0 {
			for i := range group.Txs {
				group.SignN(i, int32(wallet.SignType), privKey)
			}
			grouptx := group.Tx()
			txHex := types.Encode(grouptx)
			signedTx := hex.EncodeToString(txHex)
			fmt.Println(signedTx)
			return
		} else {
			index -= 1
			group.SignN(int(index), int32(wallet.SignType), privKey)
			grouptx := group.Tx()
			txHex := types.Encode(grouptx)
			signedTx := hex.EncodeToString(txHex)
			fmt.Println(signedTx)
			return
		}
	} else if addr != "" {
		params := types.ReqSignRawTx{
			Addr:   addr,
			TxHex:  data,
			Expire: expire,
			Index:  index,
			Mode:   mode,
		}
		ctx := NewRpcCtx(rpcLaddr, "Chain33.SignRawTx", params, nil)
		ctx.RunWithoutMarshal()
		return
	}
	fmt.Fprintln(os.Stderr, types.ErrNoPrivKeyOrAddr)
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

// send raw tx
func SendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a transaction",
		Run:   sendTx,
	}
	addSendTxFlags(cmd)
	return cmd
}

func addSendTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "transaction content")
	cmd.MarkFlagRequired("data")
	cmd.Flags().Int32P("mode", "m", 0, "transaction send mode")
}

func sendTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	data, _ := cmd.Flags().GetString("data")
	mode, _ := cmd.Flags().GetInt32("mode")
	params := jsonrpc.RawParm{
		Mode: mode,
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

// QueryCacheTxByAddrCmd 查询还未发送的隐私交易
func QueryCacheTxByAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queryctx",
		Short: "Query transaction in cache by address",
		Run:   queryCacheTx,
	}
	queryCacheTxFlags(cmd)
	return cmd
}

func queryCacheTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
}

func queryCacheTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	params := types.ReqCacheTxList{
		Addr: addr,
	}

	var res jsonrpc.ReplyCacheTxList
	ctx := NewRpcCtx(rpcLaddr, "Chain33.QueryCacheTransaction", params, &res)
	ctx.SetResultCb(parseQueryCacheTxRes)
	ctx.Run()
}

func parseQueryCacheTxRes(arg interface{}) (interface{}, error) {
	res := arg.(*jsonrpc.ReplyCacheTxList)
	var result TxListResult
	for _, v := range res.Txs {
		result.Txs = append(result.Txs, decodeTransaction(v))
	}
	return result, nil
}

// DeleteCacheTxCmd 删除位与缓存中未发送的隐私交易
func DeleteCacheTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deletectx",
		Short: "Delete transaction in cache by hash",
		Run:   deleteCacheTx,
	}
	deleteCacheTxFlags(cmd)
	return cmd
}

func deleteCacheTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("txhash", "t", "", "transaction hash")
	cmd.MarkFlagRequired("txhash")
}

func deleteCacheTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	txhash, _ := cmd.Flags().GetString("txhash")
	hash, err := common.FromHex(txhash)
	if err != nil {
		return
	}
	params := types.ReqHash{
		Hash: hash,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.DeleteCacheTransaction", params, nil)
	ctx.RunWithoutMarshal()
}
