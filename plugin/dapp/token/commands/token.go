package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
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
	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")
}

func createTokenTransfer(cmd *cobra.Command, args []string) {
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")
	txHex, err := CreateRawTx(cmd, toAddr, amount, note, false, symbol, "")
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
	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")
}

func createTokenWithdraw(cmd *cobra.Command, args []string) {
	exec, _ := cmd.Flags().GetString("exec")
	paraName, _ := cmd.Flags().GetString("paraName")
	exec = getRealExecName(paraName, exec)
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := CreateRawTx(cmd, execAddr, amount, note, true, symbol, exec)
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
	paraName, _ := cmd.Flags().GetString("paraName")
	var reqtokens tokenty.ReqTokens
	reqtokens.Status = tokenty.TokenStatusPreCreated
	reqtokens.QueryAll = true
	var params types.Query4Cli
	params.Execer = getRealExecName(paraName, "token")
	params.FuncName = "GetTokens"
	params.Payload = reqtokens
	rpc, err := jsonclient.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res tokenty.ReplyTokens
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, preCreatedToken := range res.Tokens {
		preCreatedToken.Price = preCreatedToken.Price / types.Coin
		preCreatedToken.Total = preCreatedToken.Total / types.TokenPrecision

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
	paraName, _ := cmd.Flags().GetString("paraName")
	var reqtokens tokenty.ReqTokens
	reqtokens.Status = tokenty.TokenStatusCreated
	reqtokens.QueryAll = true
	var params types.Query4Cli
	params.Execer = getRealExecName(paraName, "token")
	params.FuncName = "GetTokens"
	params.Payload = reqtokens
	rpc, err := jsonclient.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res tokenty.ReplyTokens
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, createdToken := range res.Tokens {
		createdToken.Price = createdToken.Price / types.Coin
		createdToken.Total = createdToken.Total / types.TokenPrecision

		//fmt.Printf("---The %dth Finish Created token is below--------------------\n", i)
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
	paraName, _ := cmd.Flags().GetString("paraName")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	execer = getRealExecName(paraName, execer)
	req := tokenty.ReqAccountTokenAssets{
		Address: addr,
		Execer:  execer,
	}

	var params types.Query4Cli
	params.Execer = getRealExecName(paraName, "token")
	params.FuncName = "GetAccountTokenAssets"
	params.Payload = req

	var res tokenty.ReplyAccountTokenAssets
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseTokenAssetsRes)
	ctx.Run()
}

func parseTokenAssetsRes(arg interface{}) (interface{}, error) {
	res := arg.(*tokenty.ReplyAccountTokenAssets)
	var result []*tokenty.TokenAccountResult
	for _, ta := range res.TokenAssets {
		balanceResult := strconv.FormatFloat(float64(ta.Account.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(ta.Account.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		tokenAccount := &tokenty.TokenAccountResult{
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
	addr, _ := cmd.Flags().GetString("address")
	token, _ := cmd.Flags().GetString("symbol")
	execer, _ := cmd.Flags().GetString("exec")
	paraName, _ := cmd.Flags().GetString("paraName")
	execer = getRealExecName(paraName, execer)
	addresses := strings.Split(addr, " ")
	params := tokenty.ReqTokenBalance{
		Addresses:   addresses,
		TokenSymbol: token,
		Execer:      execer,
	}
	var res []*rpctypes.Account
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "token.GetTokenBalance", params, &res)
	ctx.SetResultCb(parseTokenBalanceRes)
	ctx.Run()
}

func parseTokenBalanceRes(arg interface{}) (interface{}, error) {
	res := arg.(*[]*rpctypes.Account)
	var result []*tokenty.TokenAccountResult
	for _, one := range *res {
		balanceResult := strconv.FormatFloat(float64(one.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(one.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		tokenAccount := &tokenty.TokenAccountResult{
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

	cmd.Flags().Float64P("price", "p", 0, "token price(mini: 0.0001)")
	cmd.MarkFlagRequired("price")

	cmd.Flags().Int64P("total", "t", 0, "total amount of the token")
	cmd.MarkFlagRequired("total")

	cmd.Flags().Float64P("fee", "f", 0, "token transaction fee")
}

func tokenPrecreated(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	name, _ := cmd.Flags().GetString("name")
	symbol, _ := cmd.Flags().GetString("symbol")
	introduction, _ := cmd.Flags().GetString("introduction")
	ownerAddr, _ := cmd.Flags().GetString("owner_addr")
	price, _ := cmd.Flags().GetFloat64("price")
	total, _ := cmd.Flags().GetInt64("total")

	priceInt64 := int64((price + 0.000001) * 1e4)
	params := &tokenty.TokenPreCreate{
		Price:        priceInt64 * 1e4,
		Name:         name,
		Symbol:       symbol,
		Introduction: introduction,
		Owner:        ownerAddr,
		Total:        total * types.TokenPrecision,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "token.CreateRawTokenPreCreateTx", params, nil)
	ctx.RunWithoutMarshal()
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
}

func tokenFinish(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	ownerAddr, _ := cmd.Flags().GetString("owner_addr")
	params := &tokenty.TokenFinishCreate{
		Symbol: symbol,
		Owner:  ownerAddr,
	}

	ctx := jsonclient.NewRpcCtx(rpcLaddr, "token.CreateRawTokenFinishTx", params, nil)
	ctx.RunWithoutMarshal()
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
}

func tokenRevoke(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	ownerAddr, _ := cmd.Flags().GetString("owner_addr")

	params := &tokenty.TokenRevokeCreate{
		Symbol: symbol,
		Owner:  ownerAddr,
	}

	ctx := jsonclient.NewRpcCtx(rpcLaddr, "token.CreateRawTokenRevokeTx", params, nil)
	ctx.RunWithoutMarshal()
}
