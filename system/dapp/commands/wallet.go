// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bufio"
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	ctype "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// WalletCmd wallet command
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
		NoBalanceCmd(),
		SetFeeCmd(),
		SendTxCmd(),
		SignRawTxWithCertCmd(),
	)

	return cmd
}

// LockCmd lock the wallet
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
	var res rpctypes.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Lock", nil, &res)
	ctx.Run()
}

// UnlockCmd unlock the wallet
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
	var res rpctypes.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.UnLock", &params, &res)
	ctx.Run()
}

// WalletStatusCmd get wallet status
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
	var res rpctypes.WalletStatus
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetWalletStatus", nil, &res)
	ctx.Run()
}

// SetPwdCmd set password
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

	cmd.Flags().StringP("new", "n", "", "new password,[8-30]letter and digit")
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
	var res rpctypes.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.SetPasswd", &params, &res)
	ctx.Run()
}

// WalletListTxsCmd  get wallet transactions
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

	cmd.Flags().Int32P("direction", "d", 0, "query direction (0: pre page, 1: next page)")
}

func walletListTxs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	txHash, _ := cmd.Flags().GetString("from")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("direction")
	params := rpctypes.ReqWalletTransactionList{
		FromTx:    txHash,
		Count:     count,
		Direction: direction,
	}
	var res rpctypes.WalletTxDetails
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.WalletTxList", params, &res)
	ctx.SetResultCbExt(parseWalletTxListRes)
	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	ctx.RunExt(cfg)
}

func parseWalletTxListRes(arg ...interface{}) (interface{}, error) {
	res := arg[0].(*rpctypes.WalletTxDetails)
	cfg := arg[1].(*rpctypes.ChainConfigInfo)
	var result commandtypes.WalletTxDetailsResult
	for _, v := range res.TxDetails {
		amountResult := types.FormatAmount2FloatDisplay(v.Amount, cfg.CoinPrecision, true)
		wtxd := &commandtypes.WalletTxDetailResult{
			Tx:         commandtypes.DecodeTransaction(v.Tx),
			Receipt:    v.Receipt,
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

// MergeBalanceCmd  merge all balance to an account
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
	var res rpctypes.ReplyHashes
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.MergeBalance", &params, &res)
	ctx.Run()
}

// AutoMineCmd  set auto mining: 为了兼容现在的命令行, 这个命令就放到wallet，实际上依赖 ticket
func AutoMineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto_mine",
		Short: "Set auto mine on/off. Deprecated",
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
		err := cmd.UsageFunc()(cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return
	}
	params := struct {
		Flag int32
	}{
		Flag: flag,
	}
	var res rpctypes.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "ticket.SetAutoMining", params, &res)
	ctx.Run()
}

// NoBalanceCmd sign raw tx
func NoBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nobalance",
		Short: "Create nobalance transaction",
		Run:   noBalanceTx,
	}
	addNoBalanceFlags(cmd)
	return cmd
}

func addNoBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "raw transaction data")
	cmd.MarkFlagRequired("data")
	cmd.Flags().StringP("key", "k", "", "private key of pay fee account (optional)")
	cmd.Flags().StringP("payaddr", "a", "", "account address of pay fee (optional)")
	cmd.Flags().StringP("expire", "e", "120s", "transaction expire time")
}

// SignRawTxCmd sign raw tx
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
	cmd.Flags().StringP("data", "d", "", "raw transaction hex data or data file")
	cmd.MarkFlagRequired("data")

	cmd.Flags().Int32P("index", "i", 0, "transaction index to be signed")
	cmd.Flags().StringP("key", "k", "", "private key (optional)")
	cmd.Flags().StringP("addr", "a", "", "account address (optional)")
	cmd.Flags().StringP("expire", "e", "120s", "transaction expire time")
	cmd.Flags().Float64P("fee", "f", 0, "transaction fee (optional), auto set proper fee if not set or zero fee")
	cmd.Flags().StringP("to", "t", "", "new to addr (optional)")
	cmd.Flags().Int32P("addressType", "p", -1, "address type ID, btc(0), btcMultiSign(1), eth(2)")
	cmd.Flags().Int32P("chainID", "c", 0, "for eth signn (optional)")
	cmd.Flags().StringP("out", "o", "sign.out", "output file, default sign.out")
	// A duration string is a possibly signed sequence of
	// decimal numbers, each with optional fraction and a unit suffix,
	// such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
}

// SignRawTxWithCertCmd sign raw tx
func SignRawTxWithCertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signWithCert",
		Short: "SignWithCert transaction",
		Run:   signRawTxWithCert,
	}
	addSignRawTxWithCertFlags(cmd)
	return cmd
}

func addSignRawTxWithCertFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "raw transaction data")
	cmd.MarkFlagRequired("data")
	cmd.Flags().StringP("signType", "s", "sm2", "sign type")
	cmd.Flags().StringP("keyFilePath", "k", "", "private key file path")
	cmd.Flags().StringP("certFilePath", "c", "", "cert file path")
	// A duration string is a possibly signed sequence of
	// decimal numbers, each with optional fraction and a unit suffix,
	// such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
}

func signRawTxWithCert(cmd *cobra.Command, args []string) {
	//rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	data, _ := cmd.Flags().GetString("data")
	signType, _ := cmd.Flags().GetString("signType")
	keyFilePath, _ := cmd.Flags().GetString("keyFilePath")
	certFilePath, _ := cmd.Flags().GetString("certFilePath")

	privatekey, err := ctype.LoadPrivKeyFromLocal(signType, keyFilePath)
	if err != nil {
		fmt.Println("load account from local have err", err)
		return
	}
	certByte, err := ctype.ReadFile(certFilePath)
	if err != nil {
		fmt.Println("load cert file have err", err)
		return
	}
	signTx, err := ctype.CreateTxWithCert(signType, privatekey, data, certByte)
	if err != nil {
		fmt.Println("load cert file have err", err)
		return
	}
	fmt.Println(signTx)
}

func noBalanceTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	data, _ := cmd.Flags().GetString("data")
	addr, _ := cmd.Flags().GetString("payaddr")
	privkey, _ := cmd.Flags().GetString("key")
	expire, _ := cmd.Flags().GetString("expire")
	expireTime, err := time.ParseDuration(expire)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if expireTime < time.Minute*2 && expireTime != time.Second*0 {
		expire = "120s"
		fmt.Println("expire time must longer than 2 minutes, changed expire time into 2 minutes")
	}
	params := types.NoBalanceTx{
		PayAddr: addr,
		TxHex:   data,
		Expire:  expire,
		Privkey: privkey,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateNoBalanceTransaction", &params, nil)
	ctx.RunWithoutMarshal()
}

func signRawTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	data, _ := cmd.Flags().GetString("data")
	key, _ := cmd.Flags().GetString("key")
	addr, _ := cmd.Flags().GetString("addr")
	index, _ := cmd.Flags().GetInt32("index")
	to, _ := cmd.Flags().GetString("to")
	fee, _ := cmd.Flags().GetFloat64("fee")
	expire, _ := cmd.Flags().GetString("expire")
	addressType, _ := cmd.Flags().GetInt32("addressType")
	chainID, _ := cmd.Flags().GetInt32("chainID")
	outFile, _ := cmd.Flags().GetString("out")
	//check tx type
	ethTx := new(ethtypes.Transaction)
	if ethTx.UnmarshalBinary(common.FromHex(data)) == nil {
		//eth tx
		signer := ethtypes.NewEIP155Signer(big.NewInt(int64(chainID)))
		signKey, err := crypto.ToECDSA(common.FromHex(key))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		signedTx, err := ethtypes.SignTx(ethTx, signer, signKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		signedBytes, _ := signedTx.MarshalBinary()
		fmt.Println(common.Bytes2Hex(signedBytes))
		return
	}
	expire, err := commandtypes.CheckExpireOpt(expire)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	feeInt64, err := types.FormatFloatDisplay2Value(fee, cfg.CoinPrecision)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	params := types.ReqSignRawTx{
		Addr:      addr,
		Privkey:   key,
		TxHex:     data,
		Expire:    expire,
		Index:     index,
		Fee:       feeInt64,
		NewToAddr: to,
		AddressID: addressType,
	}

	if !isFileExist(data) {

		ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.SignRawTx", &params, nil)
		ctx.RunWithoutMarshal()
		return
	}

	// data为文件
	signTxsInFile(&params, data, outFile, rpcLaddr)

}

func signTxsInFile(req *types.ReqSignRawTx, in, out, rpc string) {

	rawTxs := readLines(in)
	var signTxs []string
	var signTx string
	for line, rawTx := range rawTxs {

		if rawTx == "" {
			continue
		}
		req.TxHex = rawTx
		ctx := jsonclient.NewRPCCtx(rpc, "Chain33.SignRawTx", req, &signTx)
		_, err := ctx.RunResult()
		if err != nil {
			fmt.Fprintf(os.Stderr, "signRawTx err, line:%d, err:%s \n", line, err.Error())
			continue
		}
		signTxs = append(signTxs, signTx)
	}

	err := writeLines(out, signTxs)
	if err == nil {
		return
	}
	// output to stdout if write file failed
	for _, tx := range signTxs {
		fmt.Println(tx)
	}
}

func writeLines(filename string, lines []string) error {

	if len(lines) <= 0 {
		return nil
	}

	file, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Fprintf(os.Stderr, "writeLines open file err: %s \n", err.Error())
		return err
	}
	defer file.Close()
	datawriter := bufio.NewWriter(file)

	for _, data := range lines {
		_, _ = datawriter.WriteString(data + "\n")
	}

	datawriter.Flush()
	return nil
}

func readLines(file string) []string {

	var lines []string
	readFile, err := os.Open(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "readLines open file err:%s \n", err.Error())
		return nil
	}
	defer readFile.Close()
	fs := bufio.NewScanner(readFile)
	fs.Split(bufio.ScanLines)

	for fs.Scan() {
		lines = append(lines, fs.Text())
	}

	return lines
}

func isFileExist(file string) bool {

	if _, err := os.Stat(file); err == nil {
		return true
	}
	return false
}

// SetFeeCmd set tx fee
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

	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64, err := types.FormatFloatDisplay2Value(amount, cfg.CoinPrecision)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqWalletSetFee{
		Amount: amountInt64,
	}
	var res rpctypes.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.SetTxFee", &params, &res)
	ctx.Run()
}

// SendTxCmd  send raw tx
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
	cmd.Flags().StringP("data", "d", "", "tx data or tx file")
	cmd.MarkFlagRequired("data")
	cmd.Flags().StringP("out", "o", "send.out", "output file")
	cmd.Flags().StringP("token", "t", types.BTY, "token name. (BTY supported)")
	cmd.Flags().BoolP("ethType", "e", false, "")
}

func sendTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	data, _ := cmd.Flags().GetString("data")
	token, _ := cmd.Flags().GetString("token")
	ethType, _ := cmd.Flags().GetBool("ethType")
	outFile, _ := cmd.Flags().GetString("out")
	if ethType {
		c, err := rpc.DialContext(context.Background(), rpcLaddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		var etx = new(ethtypes.Transaction)
		err = etx.UnmarshalBinary(common.FromHex(data))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		var txHash string
		//调用eth 接口发送交易
		err = c.CallContext(context.Background(), &txHash, "eth_sendRawTransaction", data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Println(txHash)
		return
	}
	params := rpctypes.RawParm{
		Token: token,
		Data:  data,
	}

	if !isFileExist(data) {
		ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
		ctx.RunWithoutMarshal()
		return
	}

	sendTxsInFile(&params, data, outFile, rpcLaddr)
}

func sendTxsInFile(req *rpctypes.RawParm, in, out, rpc string) {

	signTxs := readLines(in)
	var hashes []string
	var hash string
	for line, tx := range signTxs {

		if tx == "" {
			continue
		}
		req.Data = tx
		ctx := jsonclient.NewRPCCtx(rpc, "Chain33.SendTransaction", req, &hash)
		_, err := ctx.RunResult()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sendTx, line:%d, err:%s \n", line, err.Error())
			continue
		}
		hashes = append(hashes, hash)
	}

	for _, h := range hashes {
		fmt.Println(h)
	}
}
