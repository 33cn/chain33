package commands

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	tokenName string
)

func TokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Token managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		CreateTokenTransferCmd(),
		CreateTokenWithdrawCmd(),
		GetTokensPreCreatedCmd(),
		GetTokensFinishCreatedCmd(),
		GetTokenAssetsCmd(),
		GetTokenBalanceCmd(),
		CreateRawTokenPreCreateCmd(),
		CreateRawTokenFinishCmd(),
		CreateRawTokenRevokeCmd(),


		TokenPreCreateCmd(),
		TokenFinishCreateCmd(),
		TokenRevokeCreateCmd(),
		ListTokenCmd(),
		SellTokenCmd(),
		RevokeSellTokenCmd(),
		BuyTokenCmd(),
		ShowTokenOrderCmd(),
		TokenAssetsCmd(),
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
	txHex, err := createRawTx(toAddr, amount, note, false, true, symbol)
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
	execAddr, err := getExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := createRawTx(execAddr, amount, note, true, true, symbol)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

func GetTokensPreCreatedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_precreated",
		Short: "get precreated tokens",
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
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
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

func GetTokensFinishCreatedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_finish_created",
		Short: "get finish created tokens",
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
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
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

func GetTokenAssetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_asset",
		Short: "Get token assets",
		Run:   tokenBalance,
	}
	addTokenBalanceFlags(cmd)
	return cmd
}





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
	cmd.Flags().StringVarP(&tokenName, "symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().StringP("exec", "e", "", "execer name")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().StringP("address", "a", "", "account addresses, seperated by ','")
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
			Token:    tokenName,
			Addr:     one.Addr,
			Currency: one.Currency,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result = append(result, tokenAccount)
	}
	return result, nil
}
