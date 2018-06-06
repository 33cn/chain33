package commands

import (
	"fmt"
	"os"
	"strings"

	"encoding/json"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

////////////types.go//////////
type RelayOrder2Show struct {
	Orderid        string `json:"orderid"`
	Status         int32  `json:"status"`
	Seller         string `json:"address"`
	Sellamount     uint64 `json:"sellamount"`
	Exchgcoin      string `json:"exchangcoin"`
	Exchgamount    uint64 `json:"exchangamount"`
	Exchgaddr      string `json:"exchangaddr"`
	Waitcoinblocks uint32 `json:"waitcoinblocks"`
	Createtime     int64  `json:"createtime"`
	Buyeraddr      string `json:"buyeraddr"`
	Buyertime      int64  `json:"buyertime"`
	Finishtime     int64  `json:"finishtime"`
	Height         int64  `json:"height"`
}

type RelayBTCHeadHeightListShow struct {
	Height int64 `json:Height`
}

type RelayBTCHeadCurHeightShow struct {
	CurHeight  int64 `json:curHeight`
	BaseHeight int64 `json:baseHeight`
}

///////////////
func RelayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Cross chain relay management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ShowOnesSellRelayOrdersCmd(),
		ShowOnesBuyRelayOrdersCmd(),
		ShowOnesStatusOrdersCmd(),
		ShowBTCHeadHeightListCmd(),
		ShowBTCHeadCurHeightCmd(),
		CreateRawRelayOrderTxCmd(),
		CreateRawRevokeSellTxCmd(),
		CreateRawRelayBuyTxCmd(),
		CreateRawRevokeBuyTxCmd(),
		CreateRawRelayConfirmTxCmd(),
		CreateRawRelayVerifyBTCTxCmd(),
		CreateRawRelayBtcHeaderCmd(),
	)

	return cmd
}

func ShowBTCHeadHeightListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "btc_height_list",
		Short: "Show chain stored BTC head's height list",
		Run:   showBtcHeadHeightList,
	}
	addShowBtcHeadHeightListFlags(cmd)
	return cmd

}

func addShowBtcHeadHeightListFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("height_base", "b", 0, "height base")
	cmd.MarkFlagRequired("height_base")

	cmd.Flags().Int32P("counts", "c", 0, "height counts, default:0, means all")

	cmd.Flags().Int32P("direction", "d", 0, "0:desc,1:asc, default:0")

}

func showBtcHeadHeightList(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	base, _ := cmd.Flags().GetInt64("height_base")
	count, _ := cmd.Flags().GetInt32("counts")
	direct, _ := cmd.Flags().GetInt32("direction")

	var reqList types.ReqRelayBtcHeaderHeightList
	reqList.ReqHeight = base
	reqList.Counts = count
	reqList.Direction = direct

	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetBTCHeaderList",
		Payload:  reqList,
	}
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplyRelayBtcHeadHeightList
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayBtcHeadHeightList(res)
}

func ShowBTCHeadCurHeightCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "btc_cur_height",
		Short: "Show chain stored BTC head's current height",
		Run:   showBtcHeadCurHeight,
	}
	addShowBtcHeadCurHeightFlags(cmd)
	return cmd

}

func addShowBtcHeadCurHeightFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("height_base", "b", 0, "height base")
}

func showBtcHeadCurHeight(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	base, _ := cmd.Flags().GetInt64("height_base")

	var reqList types.ReqRelayQryBTCHeadHeight
	reqList.BaseHeight = base

	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetBTCHeaderCurHeight",
		Payload:  reqList,
	}
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplayRelayQryBTCHeadHeight
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayBtcCurHeight(res)
}

func ShowOnesSellRelayOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell_order",
		Short: "Show one seller's relay orders, coins optional",
		Run:   showOnesRelayOrders,
	}
	addShowRelayOrdersFlags(cmd)
	return cmd
}

func addShowRelayOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("seller", "s", "", "coin seller")
	cmd.MarkFlagRequired("seller")

	cmd.Flags().StringP("coin", "c", "", "coins, separated by space")

}

func showOnesRelayOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	seller, _ := cmd.Flags().GetString("seller")
	coin, _ := cmd.Flags().GetString("coin")
	coins := strings.Split(coin, " ")
	var reqAddrCoins types.ReqRelayAddrCoins
	reqAddrCoins.Status = types.RelayOrderStatus_pending
	reqAddrCoins.Addr = seller
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetSellRelayOrder",
		Payload:  reqAddrCoins,
	}
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplyRelayOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayOrders(res)
}

////
func ShowOnesBuyRelayOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy_order",
		Short: "Show one buyer's buy orders, coins optional",
		Run:   showRelayBuyOrders,
	}
	addShowRelayBuyOrdersFlags(cmd)
	return cmd
}

func addShowRelayBuyOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("buyer", "b", "", "coin buyer")
	cmd.MarkFlagRequired("buyer")

	cmd.Flags().StringP("coin", "c", "", "coins, separated by space")
	cmd.MarkFlagRequired("coin")
}

func showRelayBuyOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	buyer, _ := cmd.Flags().GetString("buyer")
	coin, _ := cmd.Flags().GetString("coin")
	coins := strings.Split(coin, " ")
	var reqAddrCoins types.ReqRelayAddrCoins
	reqAddrCoins.Status = types.RelayOrderStatus_locking
	reqAddrCoins.Addr = buyer
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetBuyRelayOrder",
		Payload:  reqAddrCoins,
	}
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplyRelayOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayOrders(res)
}

////
func ShowOnesStatusOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show ones status's orders",
		Run:   showCoinRelayOrders,
	}
	addShowCoinOrdersFlags(cmd)
	return cmd
}

func addShowCoinOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().Int32P("status", "s", 0, "order status (pending:1, locking:2, confirming:3, finished:4,cancled:5)")
	cmd.MarkFlagRequired("status")

	cmd.Flags().StringP("coin", "c", "", "coins, separated by space")
}

func showCoinRelayOrders(cmd *cobra.Command, args []string) {
	var coins = []string{}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	status, _ := cmd.Flags().GetInt32("status")
	coin, _ := cmd.Flags().GetString("coin")
	if coin == "" {
		coins = append(coins, []string{"BTC"}...)
	} else {
		spt := strings.Split(coin, " ")
		coins = append(coins, spt...)
	}
	var reqAddrCoins types.ReqRelayAddrCoins
	reqAddrCoins.Status = types.RelayOrderStatus(status)
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetRelayOrderByStatus",
		Payload:  reqAddrCoins,
	}
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplyRelayOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayOrders(res)
}

func parseRelayOrders(res types.ReplyRelayOrders) {
	for i, order := range res.Relayorders {
		var show RelayOrder2Show
		show.Orderid = order.Id
		show.Status = int32(order.Status)
		show.Seller = order.CreaterAddr
		show.Sellamount = order.Amount
		show.Exchgaddr = order.CoinAddr
		show.Exchgamount = order.CoinAmount
		show.Exchgcoin = order.Coin
		show.Createtime = order.CreateTime
		show.Buyeraddr = order.AcceptAddr
		show.Buyertime = order.AcceptTime
		show.Finishtime = order.FinishTime
		show.Height = order.Height

		data, err := json.MarshalIndent(show, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("---The %dth relay order is below--------------------\n", i)
		fmt.Println(string(data))
	}
}

func parseRelayBtcHeadHeightList(res types.ReplyRelayBtcHeadHeightList) {
	for i, height := range res.Heights {
		var show RelayBTCHeadHeightListShow
		show.Height = height

		data, err := json.MarshalIndent(show, "", "    ")
		if err != nil {
			fmt.Println(os.Stderr, err)
			return
		}

		fmt.Printf("---The %dth BTC height is below------\n", i)
		fmt.Println(string(data))
	}

}

func parseRelayBtcCurHeight(res types.ReplayRelayQryBTCHeadHeight) {
	var show RelayBTCHeadCurHeightShow
	show.CurHeight = res.CurHeight
	show.BaseHeight = res.BaseHeight

	data, err := json.MarshalIndent(show, "", "    ")
	if err != nil {
		fmt.Println(os.Stderr, err)
		return
	}

	fmt.Println("---The BTC height info is below------")
	fmt.Println(string(data))
}

//// create raw sell token transaction
func CreateRawRelayOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Create an exchange coin order",
		Run:   relayOrder,
	}
	addExchangeFlags(cmd)
	return cmd
}

func addExchangeFlags(cmd *cobra.Command) {
	cmd.Flags().Uint32P("operation", "o", 0, "0:buy, 1:sell")
	cmd.MarkFlagRequired("operation")

	cmd.Flags().StringP("coin", "c", "", "coin to exchange by BTY, like BTC,ETH, default BTC")
	cmd.MarkFlagRequired("coin")

	cmd.Flags().Float64P("coin_amount", "m", 0, "coin amount to exchange")
	cmd.MarkFlagRequired("coin_amount")

	cmd.Flags().StringP("coin_addr", "a", "", "coin address in coin's block chain")
	cmd.MarkFlagRequired("coin_addr")

	cmd.Flags().Float64P("bty_amount", "b", 0, "sell amount of BTY")
	cmd.MarkFlagRequired("bty_amount")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")

}

func relayOrder(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	oper, _ := cmd.Flags().GetUint32("operation")
	coin, _ := cmd.Flags().GetString("coin")
	coinamount, _ := cmd.Flags().GetFloat64("coinamount")
	coinaddr, _ := cmd.Flags().GetString("coinaddr")
	btyamount, _ := cmd.Flags().GetFloat64("bty_amount")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	btyUInt64 := uint64(btyamount * 1e4)
	coinUInt64 := uint64(coinamount * 1e4)

	params := &jsonrpc.RelayOrderTx{
		Operation: oper,
		Amount:    coinUInt64 * 1e4,
		Coin:      coin,
		Addr:      coinaddr,
		BtyAmount: btyUInt64 * 1e4,
		Fee:       feeInt64 * 1e4,
	}

	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRelayOrderTx", params, &res)
	ctx.Run()
}

// create raw sell revoke transaction
func CreateRawRevokeSellTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell_revoke",
		Short: "Create a revoke sell transaction",
		Run:   relayRevokeSell,
	}
	addRevokeSellFlags(cmd)
	return cmd
}

func addRevokeSellFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "i", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")
}

func relayRevokeSell(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderID, _ := cmd.Flags().GetString("order_id")
	fee, _ := cmd.Flags().GetFloat64("fee")
	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.RelayRevokeSellTx{
		OrderId: orderID,
		Fee:     feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRevokeSellTx", params, &res)
	ctx.Run()
}

// create raw buy token transaction
func CreateRawRelayBuyTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy",
		Short: "Create a buying coin transaction",
		Run:   relayBuy,
	}
	addRelayBuyFlags(cmd)
	return cmd
}

func addRelayBuyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "o", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")
}

func relayBuy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderID, _ := cmd.Flags().GetString("order_id")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.RelayBuyTx{
		OrderId: orderID,
		Fee:     feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRelayBuyTx", params, &res)
	ctx.Run()
}

// create raw buy revoke transaction
func CreateRawRevokeBuyTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy_revoke",
		Short: "Create a revoke buy transaction",
		Run:   relayRevokeBuy,
	}
	addRevokeBuyFlags(cmd)
	return cmd
}

func addRevokeBuyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "i", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")
}

func relayRevokeBuy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderID, _ := cmd.Flags().GetString("order_id")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.RelayRevokeBuyTx{
		OrderId: orderID,
		Fee:     feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRelayRvkBuyTx", params, &res)
	ctx.Run()
}

// create raw verify transaction
func CreateRawRelayConfirmTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confirm",
		Short: "Create a confirm coin transaction",
		Run:   relayConfirm,
	}
	addConfirmFlags(cmd)
	return cmd
}

func addConfirmFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "o", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().StringP("tx_hash", "t", "", "coin tx hash")
	cmd.MarkFlagRequired("tx_hash")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")
}

func relayConfirm(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderId, _ := cmd.Flags().GetString("order_id")
	txHash, _ := cmd.Flags().GetString("tx_hash")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.RelayVerifyTx{
		OrderId: orderId,
		TxHash:  txHash,
		Fee:     feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRelayConfirmTx", params, &res)
	ctx.Run()
}

func CreateRawRelayBtcHeaderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "btcheader",
		Short: "save BTC header",
		Run:   relaySaveBtcHead,
	}
	addSaveBtcHeadFlags(cmd)
	return cmd
}

func addSaveBtcHeadFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("block_hash", "b", "", "block hash")
	cmd.MarkFlagRequired("block_hash")

	cmd.Flags().StringP("pre_hash", "p", "", "previous block hash")
	cmd.MarkFlagRequired("pre_hash")

	cmd.Flags().StringP("merkleroot", "m", "", "merkle root")
	cmd.MarkFlagRequired("merkleroot")

	cmd.Flags().Uint64P("height", "t", 0, "block height")
	cmd.MarkFlagRequired("height")

	cmd.Flags().Int32P("flag", "g", 0, "block height")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")
}

func relaySaveBtcHead(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	blockhash, _ := cmd.Flags().GetString("block_hash")
	prehash, _ := cmd.Flags().GetString("pre_hash")
	merkleroot, _ := cmd.Flags().GetString("merkleroot")
	height, _ := cmd.Flags().GetUint64("height")
	flag, _ := cmd.Flags().GetInt32("flag")

	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)

	params := &jsonrpc.RelaySaveBTCHeadTx{
		Hash:         blockhash,
		PreviousHash: prehash,
		MerkleRoot:   merkleroot,
		Height:       height,
		IsReset:      flag == 1,
		Fee:          feeInt64 * 1e4,
	}

	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRelaySaveBTCHeadTx", params, &res)
	ctx.Run()
}

// create raw verify transaction
func CreateRawRelayVerifyBTCTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Create a verify coin transaction",
		Run:   relayVerifyBTC,
	}
	addVerifyBTCFlags(cmd)
	return cmd
}

func addVerifyBTCFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "o", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().StringP("raw_tx", "t", "", "coin raw tx")
	cmd.MarkFlagRequired("raw_tx")

	cmd.Flags().Uint32P("tx_index", "i", 0, "raw tx index")
	cmd.MarkFlagRequired("tx_index")

	cmd.Flags().StringP("merk_branch", "m", "", "tx merkle branch")
	cmd.MarkFlagRequired("merk_branch")

	cmd.Flags().StringP("block_hash", "b", "", "block hash of tx ")
	cmd.MarkFlagRequired("block_hash")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")
}

func relayVerifyBTC(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderid, _ := cmd.Flags().GetString("order_id")
	rawtx, _ := cmd.Flags().GetString("raw_tx")
	txindex, _ := cmd.Flags().GetUint32("tx_index")
	merkbranch, _ := cmd.Flags().GetString("merk_branch")
	blockhash, _ := cmd.Flags().GetString("block_hash")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.RelayVerifyBTCTx{
		OrderId:     orderid,
		RawTx:       rawtx,
		TxIndex:     txindex,
		MerklBranch: merkbranch,
		BlockHash:   blockhash,
		Fee:         feeInt64 * 1e4,
	}

	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRelayVerifyBTCTx", params, &res)
	ctx.Run()
}
