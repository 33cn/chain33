// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"
	"os"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// AccountCmd account command
func AccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		DumpKeyCmd(),
		GetAccountListCmd(),
		GetBalanceCmd(),
		ImportKeyCmd(),
		NewAccountCmd(),
		NewRandAccountCmd(),
		PubKeyToAddrCmd(),
		SetLabelCmd(),
		DumpKeysFileCmd(),
		ImportKeysFileCmd(),
		GetAccountCmd(),
	)

	return cmd
}

// DumpKeyCmd dump private key
func DumpKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump_key",
		Short: "Dump private key for account address",
		Run:   dumpKey,
	}
	addDumpKeyFlags(cmd)
	return cmd
}

func addDumpKeyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "address of account")
	cmd.MarkFlagRequired("addr")
}

func dumpKey(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	params := types.ReqString{
		Data: addr,
	}
	var res types.ReplyString
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.DumpPrivkey", params, &res)
	ctx.Run()
}

// GetAccountListCmd get accounts of the wallet
func GetAccountListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get account list",
		Run:   listAccount,
	}
	return cmd
}

func listAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res rpctypes.WalletAccounts
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetAccounts", nil, &res)
	ctx.SetResultCbExt(parseListAccountRes)
	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	ctx.RunExt(cfg)
}

func parseListAccountRes(arg ...interface{}) (interface{}, error) {
	res := arg[0].(*rpctypes.WalletAccounts)
	cfg := arg[1].(*rpctypes.ChainConfigInfo)
	var result commandtypes.AccountsResult
	for _, r := range res.Wallets {
		balanceResult := types.GetFormatFloat(r.Acc.Balance, cfg.CoinPrecision)
		frozenResult := types.GetFormatFloat(r.Acc.Frozen, cfg.CoinPrecision)
		accResult := &commandtypes.AccountResult{
			Currency: r.Acc.Currency,
			Addr:     r.Acc.Addr,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result.Wallets = append(result.Wallets, &commandtypes.WalletResult{Acc: accResult, Label: r.Label})
	}
	return result, nil
}

// GetBalanceCmd get balance of an execer
func GetBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Get balance of a account address",
		Run:   balance,
	}
	addBalanceFlags(cmd)
	return cmd
}

func addBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account addr")
	cmd.MarkFlagRequired("addr")
	cmd.Flags().StringP("exec", "e", "", getExecuterNameString())
	cmd.Flags().IntP("height", "", -1, "block height")
}

func getExecuterNameString() string {
	str := "executer name (only "
	allowExeName := types.AllowUserExec
	nameLen := len(allowExeName)
	for i := 0; i < nameLen; i++ {
		if i > 0 {
			str += ", "
		}
		str += fmt.Sprintf("\"%s\"", string(allowExeName[i]))
	}
	str += " and user-defined type supported)"
	return str
}

func balance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	height, _ := cmd.Flags().GetInt("height")
	err := address.CheckAddress(addr)
	if err != nil {
		if err = address.CheckMultiSignAddress(addr); err != nil {
			fmt.Fprintln(os.Stderr, types.ErrInvalidAddress)
			return
		}
	}
	if execer == "" && height == -1 {
		req := types.ReqAllExecBalance{Addr: addr}
		var res rpctypes.AllExecBalance
		ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetAllExecBalance", req, &res)
		ctx.SetResultCbExt(parseGetAllBalanceRes)
		cfg, err := commandtypes.GetChainConfig(rpcLaddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		ctx.RunExt(cfg)
		return
	}

	stateHash := ""
	if height >= 0 {
		params := types.ReqBlocks{
			Start:    int64(height),
			End:      int64(height),
			IsDetail: false,
		}
		var res rpctypes.Headers
		ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetHeaders", params, &res)
		_, err := ctx.RunResult()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		h := res.Items[0]
		stateHash = h.StateHash
	}

	if execer == "" {
		req := types.ReqAllExecBalance{Addr: addr, StateHash: stateHash}
		var res rpctypes.AllExecBalance
		ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetAllExecBalance", req, &res)
		ctx.SetResultCbExt(parseGetAllBalanceRes)
		cfg, err := commandtypes.GetChainConfig(rpcLaddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		ctx.RunExt(cfg)
		return
	}

	if ok := types.IsAllowExecName([]byte(execer), []byte(execer)); !ok {
		fmt.Fprintln(os.Stderr, types.ErrExecNameNotAllow)
		return
	}

	var addrs []string
	addrs = append(addrs, addr)
	params := types.ReqBalance{
		Addresses: addrs,
		Execer:    execer,
		StateHash: stateHash,
	}
	var res []*rpctypes.Account
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetBalance", params, &res)
	ctx.SetResultCbExt(parseGetBalanceRes)
	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	ctx.RunExt(cfg)
}

func parseGetBalanceRes(arg ...interface{}) (interface{}, error) {
	res := *arg[0].(*[]*rpctypes.Account)
	cfg := arg[1].(*rpctypes.ChainConfigInfo)
	balanceResult := types.GetFormatFloat(res[0].Balance, cfg.CoinPrecision)
	frozenResult := types.GetFormatFloat(res[0].Frozen, cfg.CoinPrecision)
	result := &commandtypes.AccountResult{
		Addr:     res[0].Addr,
		Currency: res[0].Currency,
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return result, nil
}

func parseGetAllBalanceRes(arg ...interface{}) (interface{}, error) {
	res := *arg[0].(*rpctypes.AllExecBalance)
	cfg := arg[1].(*rpctypes.ChainConfigInfo)
	accs := res.ExecAccount
	result := commandtypes.AllExecBalance{Addr: res.Addr}
	for _, acc := range accs {
		balanceResult := types.GetFormatFloat(acc.Account.Balance, cfg.CoinPrecision)
		frozenResult := types.GetFormatFloat(acc.Account.Frozen, cfg.CoinPrecision)
		ar := &commandtypes.AccountResult{
			Currency: acc.Account.Currency,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result.ExecAccount = append(result.ExecAccount, &commandtypes.ExecAccount{Execer: acc.Execer, Account: ar})
	}
	return result, nil
}

// ImportKeyCmd  import private key
func ImportKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import_key",
		Short: "Import private key with label",
		Run:   importKey,
	}
	addImportKeyFlags(cmd)
	return cmd
}

func addImportKeyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "private key")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("label", "l", "", "label for private key")
	cmd.MarkFlagRequired("label")
}

func importKey(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	key, _ := cmd.Flags().GetString("key")
	label, _ := cmd.Flags().GetString("label")

	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqWalletImportPrivkey{
		Privkey: key,
		Label:   label,
	}
	var res types.WalletAccount
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.ImportPrivkey", params, &res)
	ctx.SetResultCbExt(parseImportKeyRes)
	ctx.RunExt(cfg)
}

func parseImportKeyRes(args ...interface{}) (interface{}, error) {
	res := args[0].(*types.WalletAccount)
	cfg := args[1].(*rpctypes.ChainConfigInfo)
	accResult := commandtypes.DecodeAccount(res.GetAcc(), cfg.CoinPrecision)
	result := commandtypes.WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}

// NewAccountCmd create an account
func NewAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new account with label",
		Run:   createAccount,
	}
	addCreateAccountFlags(cmd)
	return cmd
}

func addCreateAccountFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("label", "l", "", "account label")
	cmd.MarkFlagRequired("label")
}

func createAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	label, _ := cmd.Flags().GetString("label")

	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqNewAccount{
		Label: label,
	}
	var res types.WalletAccount
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.NewAccount", params, &res)
	ctx.SetResultCbExt(parseCreateAccountRes)
	ctx.RunExt(cfg)
}

func parseCreateAccountRes(arg ...interface{}) (interface{}, error) {
	res := arg[0].(*types.WalletAccount)
	cfg := arg[1].(*rpctypes.ChainConfigInfo)
	accResult := commandtypes.DecodeAccount(res.GetAcc(), cfg.CoinPrecision)
	result := commandtypes.WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}

//NewRandAccountCmd get rand account
func NewRandAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rand",
		Short: "Get account by label",
		Run:   getRandAccount,
	}
	addRandAccountFlags(cmd)
	return cmd
}
func addRandAccountFlags(cmd *cobra.Command) {
	cmd.Flags().Int32P("lang", "l", 0, "seed language(0:English, 1:简体中文)")
	cmd.MarkFlagRequired("lang")
}
func getRandAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	lang, _ := cmd.Flags().GetInt32("lang")

	params := types.GenSeedLang{
		Lang: lang,
	}
	var res types.AccountInfo
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.NewRandAccount", params, &res)
	ctx.Run()
}

//PubKeyToAddrCmd get rand account
func PubKeyToAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "p2addr",
		Short: "Get addr by pubkey",
		Run:   getPubToAddr,
	}
	addPubKeyFlags(cmd)
	return cmd
}
func addPubKeyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pub", "p", "", "pub key string")
	cmd.MarkFlagRequired("pub")
}
func getPubToAddr(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pub, _ := cmd.Flags().GetString("pub")

	params := types.ReqString{
		Data: pub,
	}
	var res types.ReplyString
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.PubKeyToAddr", params, &res)
	ctx.Run()
}

//GetAccountCmd get account by label
func GetAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get account by label",
		Run:   getAccount,
	}
	addGetAccountFlags(cmd)
	return cmd
}

// SetLabelCmd set label of an account
func SetLabelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set_label",
		Short: "Set label for account address",
		Run:   setLabel,
	}
	addSetLabelFlags(cmd)
	return cmd
}
func addGetAccountFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("label", "l", "", "account label")
	cmd.MarkFlagRequired("label")
}
func addSetLabelFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("label", "l", "", "account label")
	cmd.MarkFlagRequired("label")
}

func getAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	label, _ := cmd.Flags().GetString("label")

	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqGetAccount{
		Label: label,
	}
	var res types.WalletAccount
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetAccount", params, &res)
	ctx.SetResultCbExt(parseSetLabelRes)
	ctx.RunExt(cfg)
}

func setLabel(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	label, _ := cmd.Flags().GetString("label")

	cfg, err := commandtypes.GetChainConfig(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	params := types.ReqWalletSetLabel{
		Addr:  addr,
		Label: label,
	}
	var res types.WalletAccount
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.SetLabl", params, &res)
	ctx.SetResultCbExt(parseSetLabelRes)
	ctx.RunExt(cfg)
}

func parseSetLabelRes(arg ...interface{}) (interface{}, error) {
	res := arg[0].(*types.WalletAccount)
	cfg := arg[1].(*rpctypes.ChainConfigInfo)
	accResult := commandtypes.DecodeAccount(res.GetAcc(), cfg.CoinPrecision)
	result := commandtypes.WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}

//DumpKeysFileCmd dump file
func DumpKeysFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump_keys",
		Short: "Dump private keys to file",
		Run:   dumpKeys,
	}
	cmd.Flags().StringP("file", "f", "", "file name")
	cmd.MarkFlagRequired("file")
	cmd.Flags().StringP("pwd", "p", "", "password needed to encrypt")
	cmd.MarkFlagRequired("pwd")
	return cmd
}

//ImportKeysFileCmd import key
func ImportKeysFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import_keys",
		Short: "Import private keys from file",
		Run:   importKeys,
	}
	cmd.Flags().StringP("file", "f", "", "file name")
	cmd.MarkFlagRequired("file")
	cmd.Flags().StringP("pwd", "p", "", "password needed to decode")
	cmd.MarkFlagRequired("pwd")
	return cmd
}

func dumpKeys(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	file, _ := cmd.Flags().GetString("file")
	pwd, _ := cmd.Flags().GetString("pwd")
	params := types.ReqPrivkeysFile{
		FileName: file,
		Passwd:   pwd,
	}
	var res types.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.DumpPrivkeysFile", params, &res)
	ctx.Run()
}

func importKeys(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	file, _ := cmd.Flags().GetString("file")
	pwd, _ := cmd.Flags().GetString("pwd")
	params := types.ReqPrivkeysFile{
		FileName: file,
		Passwd:   pwd,
	}
	var res types.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.ImportPrivkeysFile", params, &res)
	ctx.Run()
}
