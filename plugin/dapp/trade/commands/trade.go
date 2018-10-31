package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	"gitlab.33.cn/chain33/chain33/types"
)

func TradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trade",
		Short: "Token trade management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		CreateRawTradeSellTxCmd(),
		CreateRawTradeBuyTxCmd(),
		CreateRawTradeRevokeTxCmd(),

		ShowOnesSellOrdersCmd(),
		ShowOnesSellOrdersStatusCmd(),
		ShowTokenSellOrdersStatusCmd(),

		ShowOnesBuyOrderCmd(),
		ShowOnesBuyOrdersStatusCmd(),
		ShowTokenBuyOrdersStatusCmd(),

		ShowOnesOrdersStatusCmd(),
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
	cmd.Flags().StringP("token", "t", "", "tokens, separated by space (not required)")
}

func showOnesSellOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	seller, _ := cmd.Flags().GetString("seller")
	token, _ := cmd.Flags().GetString("token")
	tokens := strings.Split(token, " ")
	var reqAddrtokens pty.ReqAddrAssets
	//reqAddrtokens.Status = types.TradeOrderStatusOnSale
	reqAddrtokens.Addr = seller
	if 0 != len(tokens) {
		reqAddrtokens.Token = append(reqAddrtokens.Token, tokens...)
	}
	params := types.Query4Cli{
		Execer:   "trade",
		FuncName: "GetOnesSellOrder",
		Payload:  reqAddrtokens,
	}
	var res pty.ReplySellOrders
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseSellOrders)
	ctx.Run()
}

// show one's sell order with status
func ShowOnesSellOrdersStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status_sell_order",
		Short: "Show selling orders of the status",
		Run:   showOnesSellOrdersStatus,
	}
	addShowOnesSellOrdersStatusFlags(cmd)
	return cmd
}

func addShowOnesSellOrdersStatusFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("status", "s", "", "sell order status (onsale, soldout or revoked)")
	cmd.MarkFlagRequired("status")
	cmd.Flags().StringP("address", "a", "", "seller address")
	cmd.MarkFlagRequired("address")
}

func showOnesSellOrdersStatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("address")
	status, _ := cmd.Flags().GetString("status")
	statusInt, ok := pty.MapSellOrderStatusStr2Int[status]
	if !ok {
		fmt.Fprintln(os.Stderr, types.ErrInvalidParam)
		return
	}
	var reqAddrtokens pty.ReqAddrAssets
	reqAddrtokens.Status = statusInt
	reqAddrtokens.Addr = addr

	var params types.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesSellOrderWithStatus"
	params.Payload = reqAddrtokens
	var res pty.ReplySellOrders
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseSellOrders)
	ctx.Run()
}

// show token sell order with status
func ShowTokenSellOrdersStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status_token_sell_order",
		Short: "Show token selling orders of a status",
		Run:   showTokenSellOrdersStatus,
	}
	addShowTokenSellOrdersStatusFlags(cmd)
	return cmd
}

func addShowTokenSellOrdersStatusFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("token", "t", "", "token name")
	cmd.MarkFlagRequired("token")
	cmd.Flags().Int32P("count", "c", 10, "order count")
	cmd.Flags().Int32P("direction", "d", 1, "direction must be 0 (previous-page) or 1(next-page)")
	cmd.Flags().StringP("from", "f", "", "start from sell id (not required)")
	cmd.Flags().StringP("status", "s", "", "sell order status (onsale, soldout or revoked)")
	cmd.MarkFlagRequired("status")
}

func showTokenSellOrdersStatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	token, _ := cmd.Flags().GetString("token")
	count, _ := cmd.Flags().GetInt32("count")
	dir, _ := cmd.Flags().GetInt32("direction")
	from, _ := cmd.Flags().GetString("from")
	status, _ := cmd.Flags().GetString("status")
	statusInt, ok := pty.MapSellOrderStatusStr2Int[status]
	if !ok {
		fmt.Fprintln(os.Stderr, types.ErrInvalidParam)
		return
	}
	if dir != 0 && dir != 1 {
		fmt.Fprintln(os.Stderr, "direction must be 0 (previous-page) or 1(next-page)")
		return
	}
	var req pty.ReqTokenSellOrder
	req.TokenSymbol = token
	req.Count = count
	req.Direction = dir
	req.FromKey = from
	req.Status = statusInt
	var params types.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetTokenSellOrderByStatus"
	params.Payload = req
	var res pty.ReplySellOrders
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseSellOrders)
	ctx.Run()
}

func parseSellOrders(arg interface{}) (interface{}, error) {
	res := arg.(*pty.ReplyTradeOrders)
	var result ReplySellOrdersResult
	for _, o := range res.Orders {
		order := &TradeOrderResult{
			TokenSymbol:    o.TokenSymbol,
			Owner:          o.Owner,
			BuyID:          o.BuyID,
			Status:         o.Status,
			SellID:         o.SellID,
			TxHash:         o.TxHash,
			Height:         o.Height,
			Key:            o.Key,
			BlockTime:      o.BlockTime,
			IsSellOrder:    o.IsSellOrder,
			MinBoardlot:    o.MinBoardlot,
			TotalBoardlot:  o.TotalBoardlot,
			TradedBoardlot: o.TradedBoardlot,
		}
		order.AmountPerBoardlot = strconv.FormatFloat(float64(o.AmountPerBoardlot)/float64(types.Coin), 'f', 4, 64)
		order.PricePerBoardlot = strconv.FormatFloat(float64(o.PricePerBoardlot)/float64(types.Coin), 'f', 4, 64)
		result.SellOrders = append(result.SellOrders, order)
	}
	return result, nil
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
	cmd.Flags().StringP("buyer", "b", "", "buyer address")
	cmd.MarkFlagRequired("buyer")
	cmd.Flags().StringP("token", "t", "", "tokens, separated by space (not required)")
}

func showOnesBuyOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	buyer, _ := cmd.Flags().GetString("buyer")
	token, _ := cmd.Flags().GetString("token")
	tokens := strings.Split(token, " ")
	var reqAddrtokens pty.ReqAddrAssets
	reqAddrtokens.Addr = buyer
	if 0 != len(tokens) {
		reqAddrtokens.Token = append(reqAddrtokens.Token, tokens...)
	}
	var params types.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesBuyOrder"
	params.Payload = reqAddrtokens
	var res pty.ReplyBuyOrders
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseBuyOrders)
	ctx.Run()
}

// show one's buy order with status
func ShowOnesBuyOrdersStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status_buy_order",
		Short: "Show one's buying orders of tokens",
		Run:   showOnesBuyOrdersStatus,
	}
	addShowOnesBuyTokenOrdersStatusFlags(cmd)
	return cmd
}

func addShowOnesBuyTokenOrdersStatusFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("buyer", "b", "", "buyer address")
	cmd.MarkFlagRequired("buyer")
	cmd.Flags().StringP("status", "s", "", "buy order status (onbuy, boughtout or buyrevoked)")
	cmd.MarkFlagRequired("status")
}

func showOnesBuyOrdersStatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	buyer, _ := cmd.Flags().GetString("buyer")
	status, _ := cmd.Flags().GetString("status")
	statusInt, ok := pty.MapBuyOrderStatusStr2Int[status]
	if !ok {
		fmt.Fprintln(os.Stderr, types.ErrInvalidParam)
		return
	}
	var reqAddrtokens pty.ReqAddrAssets
	reqAddrtokens.Addr = buyer
	reqAddrtokens.Status = statusInt
	var params types.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesBuyOrderWithStatus"
	params.Payload = reqAddrtokens
	var res pty.ReplyBuyOrders
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseBuyOrders)
	ctx.Run()
}

// show token buy order with status
func ShowTokenBuyOrdersStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status_token_buy_order",
		Short: "Show token buying orders of a status",
		Run:   showTokenBuyOrdersStatus,
	}
	addShowBuyTokenOrdersStatusFlags(cmd)
	return cmd
}

func addShowBuyTokenOrdersStatusFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("token", "t", "", "token name")
	cmd.MarkFlagRequired("token")
	cmd.Flags().Int32P("count", "c", 10, "order count")
	cmd.Flags().Int32P("direction", "d", 1, "direction must be 0 (previous-page) or 1(next-page)")
	cmd.Flags().StringP("from", "f", "", "start from sell id (not required)")
	cmd.Flags().StringP("status", "s", "", "buy order status (onbuy, boughtout or buyrevoked)")
	cmd.MarkFlagRequired("status")
}

func showTokenBuyOrdersStatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	token, _ := cmd.Flags().GetString("token")
	count, _ := cmd.Flags().GetInt32("count")
	dir, _ := cmd.Flags().GetInt32("direction")
	from, _ := cmd.Flags().GetString("from")
	status, _ := cmd.Flags().GetString("status")
	statusInt, ok := pty.MapBuyOrderStatusStr2Int[status]
	if !ok {
		fmt.Fprintln(os.Stderr, types.ErrInvalidParam)
		return
	}
	if dir != 0 && dir != 1 {
		fmt.Fprintln(os.Stderr, "direction must be 0 (previous-page) or 1(next-page)")
		return
	}
	var req pty.ReqTokenBuyOrder
	req.TokenSymbol = token
	req.Count = count
	req.Direction = dir
	req.FromKey = from
	req.Status = statusInt
	var params types.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetTokenBuyOrderByStatus"
	params.Payload = req
	var res pty.ReplyBuyOrders
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseBuyOrders)
	ctx.Run()
}

func parseBuyOrders(arg interface{}) (interface{}, error) {
	res := arg.(*pty.ReplyTradeOrders)
	var result ReplyBuyOrdersResult
	for _, o := range res.Orders {
		order := &TradeOrderResult{
			TokenSymbol:    o.TokenSymbol,
			Owner:          o.Owner,
			BuyID:          o.BuyID,
			Status:         o.Status,
			SellID:         o.SellID,
			TxHash:         o.TxHash,
			Height:         o.Height,
			Key:            o.Key,
			BlockTime:      o.BlockTime,
			IsSellOrder:    o.IsSellOrder,
			MinBoardlot:    o.MinBoardlot,
			TotalBoardlot:  o.TotalBoardlot,
			TradedBoardlot: o.TradedBoardlot,
		}
		order.AmountPerBoardlot = strconv.FormatFloat(float64(o.AmountPerBoardlot)/float64(types.Coin), 'f', 4, 64)
		order.PricePerBoardlot = strconv.FormatFloat(float64(o.PricePerBoardlot)/float64(types.Coin), 'f', 4, 64)
		result.BuyOrders = append(result.BuyOrders, order)
	}
	return result, nil
}

//
func ShowOnesOrdersStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status_order",
		Short: "Show one's orders with status",
		Run:   showOnesOrdersStatus,
	}
	addShowOnesOrdersStatusFlags(cmd)
	return cmd
}

func addShowOnesOrdersStatusFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("address", "a", "", "user address")
	cmd.MarkFlagRequired("address")
	cmd.Flags().Int32P("count", "c", 10, "order count")
	cmd.Flags().Int32P("direction", "d", 1, "direction must be 0 (previous-page) or 1(next-page)")
	cmd.Flags().StringP("from", "f", "", "start from sell id (not required)")
	cmd.Flags().Int32P("status", "s", 0, "order status (1: on, 2: done, 3: revoke)")
	cmd.MarkFlagRequired("status")
}

func showOnesOrdersStatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("address")
	count, _ := cmd.Flags().GetInt32("count")
	dir, _ := cmd.Flags().GetInt32("direction")
	from, _ := cmd.Flags().GetString("from")
	status, _ := cmd.Flags().GetInt32("status")
	if status < 1 || status > 3 {
		fmt.Fprintln(os.Stderr, types.ErrInvalidParam)
		return
	}
	var reqAddrtokens pty.ReqAddrAssets
	reqAddrtokens.Addr = addr
	reqAddrtokens.Count = count
	reqAddrtokens.Direction = dir
	reqAddrtokens.FromKey = from
	reqAddrtokens.Status = status
	var params types.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesOrderWithStatus"
	params.Payload = reqAddrtokens
	var res pty.ReplyTradeOrders
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseTradeOrders)
	ctx.Run()
}

func parseTradeOrders(arg interface{}) (interface{}, error) {
	res := arg.(*pty.ReplyTradeOrders)
	var result ReplyTradeOrdersResult
	for _, o := range res.Orders {
		order := &TradeOrderResult{
			TokenSymbol:    o.TokenSymbol,
			Owner:          o.Owner,
			BuyID:          o.BuyID,
			Status:         o.Status,
			SellID:         o.SellID,
			TxHash:         o.TxHash,
			Height:         o.Height,
			Key:            o.Key,
			BlockTime:      o.BlockTime,
			IsSellOrder:    o.IsSellOrder,
			MinBoardlot:    o.MinBoardlot,
			TotalBoardlot:  o.TotalBoardlot,
			TradedBoardlot: o.TradedBoardlot,
		}
		order.AmountPerBoardlot = strconv.FormatFloat(float64(o.AmountPerBoardlot)/float64(types.Coin), 'f', 4, 64)
		order.PricePerBoardlot = strconv.FormatFloat(float64(o.PricePerBoardlot)/float64(types.Coin), 'f', 4, 64)
		result.Orders = append(result.Orders, order)
	}
	return result, nil
}

/************* create trade transactions *************/

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

	cmd.Flags().Int64P("min", "m", 0, "min boardlot")
	cmd.MarkFlagRequired("min")

	cmd.Flags().Float64P("price", "p", 0, "price per boardlot")
	cmd.MarkFlagRequired("price")

	cmd.Flags().Float64P("fee", "f", 0, "transaction fee")

	cmd.Flags().Float64P("total", "t", 0, "total tokens to be sold")
	cmd.MarkFlagRequired("total")
}

func tokenSell(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	min, _ := cmd.Flags().GetInt64("min")
	price, _ := cmd.Flags().GetFloat64("price")
	fee, _ := cmd.Flags().GetFloat64("fee")
	total, _ := cmd.Flags().GetFloat64("total")

	priceInt64 := int64(price * 1e4)
	feeInt64 := int64(fee * 1e4)
	totalInt64 := int64(total * 1e8 / 1e6)
	params := &pty.TradeSellTx{
		TokenSymbol:       symbol,
		AmountPerBoardlot: 1e6,
		MinBoardlot:       min,
		PricePerBoardlot:  priceInt64 * 1e4,
		TotalBoardlot:     totalInt64,
		Fee:               feeInt64 * 1e4,
		AssetExec:         "token",
	}

	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "trade.CreateRawTradeSellTx", params, nil)
	ctx.RunWithoutMarshal()
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
}

func tokenBuy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	sellID, _ := cmd.Flags().GetString("sell_id")
	fee, _ := cmd.Flags().GetFloat64("fee")
	count, _ := cmd.Flags().GetInt64("count")

	feeInt64 := int64(fee * 1e4)
	params := &pty.TradeBuyTx{
		SellID:      sellID,
		BoardlotCnt: count,
		Fee:         feeInt64 * 1e4,
	}

	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "trade.CreateRawTradeBuyTx", params, nil)
	ctx.RunWithoutMarshal()
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
}

func tokenSellRevoke(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	sellID, _ := cmd.Flags().GetString("sell_id")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &pty.TradeRevokeTx{
		SellID: sellID,
		Fee:    feeInt64 * 1e4,
	}

	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "trade.CreateRawTradeRevokeTx", params, nil)
	ctx.RunWithoutMarshal()
}
