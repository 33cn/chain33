package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func TradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trade",
		Short: "Token trade management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ShowOnesSellOrdersCmd(),
		ShowTokenSellOrderCmd(),
		ShowSellOrderWithStatusCmd(),
		ShowOnesBuyOrderCmd(),
		ShowOnesBuyTokenOrderCmd(),
		CreateRawTradeSellTxCmd(),
		CreateRawTradeBuyTxCmd(),
		CreateRawTradeRevokeTxCmd(),
	)

	return cmd
}

// show one's sell order
func ShowOnesSellOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell_order",
		Short: "Show one's token selling orders",
		Run:   showOnesSellOrders,
	}
	addShowOnesSellOrdersFlags(cmd)
	return cmd
}

func addShowOnesSellOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("seller", "s", "", "token seller")
	cmd.MarkFlagRequired("seller")

	cmd.Flags().StringP("token", "t", "", "tokens, separated by space")
	cmd.MarkFlagRequired("token")
}

func showOnesSellOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	seller, _ := cmd.Flags().GetString("seller")
	token, _ := cmd.Flags().GetString("token")
	tokens := strings.Split(token, " ")
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Status = types.OnSale
	reqAddrtokens.Addr = seller
	if 0 != len(tokens) {
		reqAddrtokens.Token = append(reqAddrtokens.Token, tokens...)
	}
	params := jsonrpc.Query4Cli{
		Execer:   "trade",
		FuncName: "GetOnesSellOrder",
		Payload:  reqAddrtokens,
	}
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplySellOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseSellOrders(res)
}

// show sell order of a token
func ShowTokenSellOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_order",
		Short: "Show selling orders of a token",
		Run:   showTokenSellOrders,
	}
	addShowTokenSellOrdersFlags(cmd)
	return cmd
}

func addShowTokenSellOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("token", "t", "", "token name")
	cmd.MarkFlagRequired("token")

	cmd.Flags().Int32P("count", "c", 0, "order count")
	cmd.MarkFlagRequired("count")

	cmd.Flags().Int32P("direction", "d", 1, "direction must be 0 (previous-page) or 1(next-page)")
	cmd.MarkFlagRequired("direction")

	cmd.Flags().StringP("from", "f", "", "start from sell id (not required)")
	cmd.MarkFlagRequired("from")
}

func showTokenSellOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	token, _ := cmd.Flags().GetString("token")
	count, _ := cmd.Flags().GetInt32("count")
	dir, _ := cmd.Flags().GetInt32("direction")
	from, _ := cmd.Flags().GetString("from")
	if dir != 0 && dir != 1 {
		fmt.Fprintln(os.Stderr, "direction must be 0 (previous-page) or 1(next-page)")
		return
	}
	var req types.ReqTokenSellOrder
	req.TokenSymbol = token
	req.Count = count
	req.Direction = dir
	req.FromSellId = from
	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetTokenSellOrderByStatus"
	params.Payload = req
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplySellOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseSellOrders(res)
}

// show sell order of a status
func ShowSellOrderWithStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status_sell_order",
		Short: "Show selling orders of the status",
		Run:   showSellOrderWithStatus,
	}
	addShowSellOrderWithStatusFlags(cmd)
	return cmd
}

func addShowSellOrderWithStatusFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("status", "s", "", "sell order status (onsale, soldout or revoked)")
	cmd.MarkFlagRequired("status")
}

func showSellOrderWithStatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	status, _ := cmd.Flags().GetString("status")
	statusInt, ok := types.MapSellOrderStatusStr2Int[status]
	if !ok {
		fmt.Fprintln(os.Stderr, types.ErrInvalidParam)
		return
	}
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Status = statusInt

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetAllSellOrdersWithStatus"
	params.Payload = reqAddrtokens
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplySellOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseSellOrders(res)
}

func parseSellOrders(res types.ReplySellOrders) {
	for i, sellorder := range res.Selloders {
		var sellOrders2show SellOrder2Show
		sellOrders2show.Tokensymbol = sellorder.Tokensymbol
		sellOrders2show.Seller = sellorder.Address
		sellOrders2show.Amountperboardlot = strconv.FormatFloat(float64(sellorder.Amountperboardlot)/float64(types.TokenPrecision), 'f', 4, 64)
		sellOrders2show.Minboardlot = sellorder.Minboardlot
		sellOrders2show.Priceperboardlot = strconv.FormatFloat(float64(sellorder.Priceperboardlot)/float64(types.Coin), 'f', 8, 64)
		sellOrders2show.Totalboardlot = sellorder.Totalboardlot
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Starttime = sellorder.Starttime
		sellOrders2show.Stoptime = sellorder.Stoptime
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Crowdfund = sellorder.Crowdfund
		sellOrders2show.SellID = sellorder.Sellid
		sellOrders2show.Status = types.SellOrderStatus[sellorder.Status]
		sellOrders2show.Height = sellorder.Height

		data, err := json.MarshalIndent(sellOrders2show, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("---The %dth sellorder is below--------------------\n", i)
		fmt.Println(string(data))
	}
}

// show one's buy order
func ShowOnesBuyOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy_order",
		Short: "Show one's buying orders",
		Run:   showOnesBuyOrders,
	}
	addShowBuyOrdersFlags(cmd)
	return cmd
}

func addShowBuyOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("buyer", "b", "", "token buyer")
	cmd.MarkFlagRequired("buyer")
}

// show one's buy order of tokens
func ShowOnesBuyTokenOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy_token_order",
		Short: "Show one's buying orders of tokens",
		Run:   showOnesBuyOrders,
	}
	addShowBuyTokenOrdersFlags(cmd)
	return cmd
}

func addShowBuyTokenOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("buyer", "b", "", "token buyer")
	cmd.MarkFlagRequired("buyer")

	cmd.Flags().StringP("token", "t", "", "tokens, separated by space")
}

func showOnesBuyOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	token, _ := cmd.Flags().GetString("token")
	buyer, _ := cmd.Flags().GetString("buyer")
	tokens := strings.Split(token, " ")
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Addr = buyer
	reqAddrtokens.Token = tokens

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesBuyOrder"
	params.Payload = reqAddrtokens
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyTradeBuyOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, buy := range res.Tradebuydones {
		data, err := json.MarshalIndent(buy, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("---The %dth buyorder is below--------------------\n", i)
		fmt.Println(string(data))
	}
}

// create raw sell token transaction
func CreateRawTradeSellTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell",
		Short: "Create a selling token transaction",
		Run:   tokenSell,
	}
	addTokenSellFlags(cmd)
	return cmd
}

func addTokenSellFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().Int64P("amount", "a", 0, "amount per boardlot")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().Int64P("min", "m", 0, "min boardlot")
	cmd.MarkFlagRequired("min")

	cmd.Flags().Float64P("price", "p", 0, "price per boardlot")
	cmd.MarkFlagRequired("price")

	cmd.Flags().Int64P("total", "t", 0, "total boardlot")
	cmd.MarkFlagRequired("total")

	cmd.Flags().Float64P("fee", "f", 0, "transaction fee")
	cmd.MarkFlagRequired("fee")
}

func tokenSell(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	amount, _ := cmd.Flags().GetInt64("amount")
	min, _ := cmd.Flags().GetInt64("min")
	price, _ := cmd.Flags().GetFloat64("price")
	fee, _ := cmd.Flags().GetFloat64("fee")
	total, _ := cmd.Flags().GetInt64("total")

	priceInt64 := int64(price * 1e4)
	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.TradeSellTx{
		TokenSymbol:       symbol,
		AmountPerBoardlot: amount,
		MinBoardlot:       min,
		PricePerBoardlot:  priceInt64 * 1e4,
		TotalBoardlot:     total,
		Fee:               feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTradeSellTx", params, &res)
	ctx.Run()
}

// create raw buy token transaction
func CreateRawTradeBuyTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy",
		Short: "Create a buying token transaction",
		Run:   tokenBuy,
	}
	addTokenBuyFlags(cmd)
	return cmd
}

func addTokenBuyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("sell_id", "s", "", "sell id")
	cmd.MarkFlagRequired("sell_id")

	cmd.Flags().Int64P("count", "c", 0, "count of buying (boardlot)")
	cmd.MarkFlagRequired("count")

	cmd.Flags().Float64P("fee", "f", 0, "transaction fee")
	cmd.MarkFlagRequired("fee")
}

func tokenBuy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	sellID, _ := cmd.Flags().GetString("sell_id")
	fee, _ := cmd.Flags().GetFloat64("fee")
	count, _ := cmd.Flags().GetInt64("count")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.TradeBuyTx{
		SellId:      sellID,
		BoardlotCnt: count,
		Fee:         feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTradeBuyTx", params, &res)
	ctx.Run()
}

// create raw revoke token transaction
func CreateRawTradeRevokeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Create a revoke token transaction",
		Run:   tokenSellRevoke,
	}
	addTokenSellRevokeFlags(cmd)
	return cmd
}

func addTokenSellRevokeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("sell_id", "s", "", "sell id")
	cmd.MarkFlagRequired("sell_id")

	cmd.Flags().Float64P("fee", "f", 0, "transaction fee")
	cmd.MarkFlagRequired("fee")
}

func tokenSellRevoke(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	sellID, _ := cmd.Flags().GetString("sell_id")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.TradeRevokeTx{
		SellId: sellID,
		Fee:    feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTradeRevokeTx", params, &res)
	ctx.Run()
}
