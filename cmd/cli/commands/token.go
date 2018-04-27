package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/rpc"
	jsonrpc "gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	tokenSymbol string
)

func TokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Token management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		CreateTokenTransferCmd(),
		CreateTokenWithdrawCmd(),
		GetTokensPreCreatedCmd(),
		GetTokensFinishCreatedCmd(),
		GetTokenAssetsCmd(),
		GetTokenBalanceCmd(),
		CreateRawTokenPreCreateTxCmd(),
		CreateRawTokenFinishTxCmd(),
		CreateRawTokenRevokeTxCmd(),
	)

	return cmd
}

// create raw transfer tx
func CreateTokenTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Create a token transfer transaction",
		Run:   createTokenTransfer,
	}
	addCreateTokenTransferFlags(cmd)
	return cmd
}

func addCreateTokenTransferFlags(cmd *cobra.Command) {
	//cmd.Flags().StringP("key", "k", "", "private key of sender")
	//cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.MarkFlagRequired("note")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")
}

func createTokenTransfer(cmd *cobra.Command, args []string) {
	//key, _ := cmd.Flags().GetString("key")
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")
	txHex, err := CreateRawTx(toAddr, amount, note, false, true, symbol)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create raw withdraw tx
func CreateTokenWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Create a token withdraw transaction",
		Run:   createTokenWithdraw,
	}
	addCreateTokenWithdrawFlags(cmd)
	return cmd
}

func addCreateTokenWithdrawFlags(cmd *cobra.Command) {
	//cmd.Flags().StringP("key", "k", "", "private key of user")
	//cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.MarkFlagRequired("note")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")
}

func createTokenWithdraw(cmd *cobra.Command, args []string) {
	//key, _ := cmd.Flags().GetString("key")
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := CreateRawTx(execAddr, amount, note, true, true, symbol)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// get precreated tokens
func GetTokensPreCreatedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_precreated",
		Short: "Get precreated tokens",
		Run:   getPreCreatedTokens,
	}
	return cmd
}

func getPreCreatedTokens(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var reqtokens types.ReqTokens
	reqtokens.Status = types.TokenStatusPreCreated
	reqtokens.Queryall = true
	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetTokens"
	params.Payload = reqtokens
	rpc, err := rpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyTokens
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, preCreatedToken := range res.Tokens {
		preCreatedToken.Price = preCreatedToken.Price / types.Coin
		preCreatedToken.Total = preCreatedToken.Total / types.TokenPrecision

		fmt.Printf("---The %dth precreated token is below--------------------\n", i)
		data, err := json.MarshalIndent(preCreatedToken, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fmt.Println(string(data))
	}
}

// get finish created tokens
func GetTokensFinishCreatedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_finish_created",
		Short: "Get finish created tokens",
		Run:   getFinishCreatedTokens,
	}
	return cmd
}

func getFinishCreatedTokens(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var reqtokens types.ReqTokens
	reqtokens.Status = types.TokenStatusCreated
	reqtokens.Queryall = true
	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetTokens"
	params.Payload = reqtokens
	rpc, err := rpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyTokens
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, createdToken := range res.Tokens {
		createdToken.Price = createdToken.Price / types.Coin
		createdToken.Total = createdToken.Total / types.TokenPrecision

		fmt.Printf("---The %dth Finish Created token is below--------------------\n", i)
		data, err := json.MarshalIndent(createdToken, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fmt.Println(string(data))
	}
}

// get token assets
func GetTokenAssetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_assets",
		Short: "Get token assets",
		Run:   tokenAssets,
	}
	addTokenAssetsFlags(cmd)
	return cmd
}

func addTokenAssetsFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", "execer name")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
}

func tokenAssets(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	req := types.ReqAccountTokenAssets{
		Address: addr,
		Execer:  execer,
	}

	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetAccountTokenAssets"
	params.Payload = req

	var res types.ReplyAccountTokenAssets
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseTokenAssetsRes)
	ctx.Run()
}

func parseTokenAssetsRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.ReplyAccountTokenAssets)
	var result []*TokenAccountResult
	for _, ta := range res.TokenAssets {
		balanceResult := strconv.FormatFloat(float64(ta.Account.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(ta.Account.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		tokenAccount := &TokenAccountResult{
			Token:    ta.Symbol,
			Addr:     ta.Account.Addr,
			Currency: ta.Account.Currency,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result = append(result, tokenAccount)
	}
	return result, nil
}

// get token balance
func GetTokenBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_balance",
		Short: "Get token balance of one or more addresses",
		Run:   tokenBalance,
	}
	addTokenBalanceFlags(cmd)
	return cmd
}

func addTokenBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&tokenSymbol, "symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().StringP("exec", "e", "", "execer name")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().StringP("address", "a", "", "account addresses, separated by space")
	cmd.MarkFlagRequired("address")
}

func tokenBalance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	token, _ := cmd.Flags().GetString("symbol")
	execer, _ := cmd.Flags().GetString("exec")
	addresses := strings.Split(addr, " ")
	params := types.ReqTokenBalance{
		Addresses:   addresses,
		TokenSymbol: token,
		Execer:      execer,
	}
	var res []*jsonrpc.Account
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetTokenBalance", params, &res)
	ctx.SetResultCb(parseTokenBalanceRes)
	ctx.Run()
}

func parseTokenBalanceRes(arg interface{}) (interface{}, error) {
	res := arg.(*[]*jsonrpc.Account)
	var result []*TokenAccountResult
	for _, one := range *res {
		balanceResult := strconv.FormatFloat(float64(one.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(one.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		tokenAccount := &TokenAccountResult{
			Token:    tokenSymbol,
			Addr:     one.Addr,
			Currency: one.Currency,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result = append(result, tokenAccount)
	}
	return result, nil
}

// create raw token precreate transaction
func CreateRawTokenPreCreateTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "precreate",
		Short: "Create a precreated token transaction",
		Run:   tokenPrecreated,
	}
	addTokenPrecreatedFlags(cmd)
	return cmd
}

func addTokenPrecreatedFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "token name")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().StringP("introduction", "i", "", "token introduction")
	cmd.MarkFlagRequired("introduction")

	cmd.Flags().StringP("owner_addr", "a", "", "address of token owner")
	cmd.MarkFlagRequired("owner_addr")

	cmd.Flags().Float64P("price", "p", 0, "token price")
	cmd.MarkFlagRequired("price")

	cmd.Flags().Int64P("total", "t", 0, "total amount of the token")
	cmd.MarkFlagRequired("total")

	cmd.Flags().Float64P("fee", "f", 0, "token transaction fee")
	cmd.MarkFlagRequired("fee")
}

func tokenPrecreated(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	name, _ := cmd.Flags().GetString("name")
	symbol, _ := cmd.Flags().GetString("symbol")
	introduction, _ := cmd.Flags().GetString("introduction")
	ownerAddr, _ := cmd.Flags().GetString("owner_addr")
	price, _ := cmd.Flags().GetFloat64("price")
	fee, _ := cmd.Flags().GetFloat64("fee")
	total, _ := cmd.Flags().GetInt64("total")

	priceInt64 := int64(price * 1e4)
	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.TokenPreCreateTx{
		Price:        priceInt64 * 1e4,
		Name:         name,
		Symbol:       symbol,
		Introduction: introduction,
		OwnerAddr:    ownerAddr,
		Total:        total,
		Fee:          feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTokenPreCreateTx", params, &res)
	ctx.Run()
}

// create raw token finish create transaction
func CreateRawTokenFinishTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finish",
		Short: "Create a finish created token transaction",
		Run:   tokenFinish,
	}
	addTokenFinishFlags(cmd)
	return cmd
}

func addTokenFinishFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("owner_addr", "a", "", "address of token owner")
	cmd.MarkFlagRequired("owner_addr")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().Float64P("fee", "f", 0, "token transaction fee")
	cmd.MarkFlagRequired("fee")
}

func tokenFinish(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	ownerAddr, _ := cmd.Flags().GetString("owner_addr")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.TokenFinishTx{
		Symbol:    symbol,
		OwnerAddr: ownerAddr,
		Fee:       feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTokenFinishTx", params, &res)
	ctx.Run()
}

// create raw token revoke transaction
func CreateRawTokenRevokeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Create a token revoke transaction",
		Run:   tokenRevoke,
	}
	addTokenRevokeFlags(cmd)
	return cmd
}

func addTokenRevokeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("owner_addr", "a", "", "address of token owner")
	cmd.MarkFlagRequired("owner_addr")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().Float64P("fee", "f", 0, "token transaction fee")
	cmd.MarkFlagRequired("fee")
}

func tokenRevoke(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	ownerAddr, _ := cmd.Flags().GetString("owner_addr")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.TokenRevokeTx{
		Symbol:    symbol,
		OwnerAddr: ownerAddr,
		Fee:       feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTokenRevokeTx", params, &res)
	ctx.Run()
}
