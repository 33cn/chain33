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

func RelayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Cross chain relay management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ShowOnesRelayOrdersCmd(),
		ShowOnesRelayBuyOrdersCmd(),
		ShowOrdersByCoinStatusCmd(),
		CreateRawRelaySellTxCmd(),
		CreateRawRevokeSellTxCmd(),
		CreateRawRelayBuyTxCmd(),
		CreateRawRevokeBuyTxCmd(),
		CreateRawRelayVerifyTxCmd(),
		CreateRawRelayVerifyBTCTxCmd(),
	)

	return cmd
}

func ShowOnesRelayOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay_order",
		Short: "Show one coin's relay orders",
		Run:   showOnesRelayOrders,
	}
	addShowRelayOrdersFlags(cmd)
	return cmd
}

func addShowRelayOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("seller", "s", "", "coin seller")
	cmd.MarkFlagRequired("seller")

	cmd.Flags().StringP("coin", "c", "", "coins, separated by space")
	cmd.MarkFlagRequired("coin")
}

func showOnesRelayOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	seller, _ := cmd.Flags().GetString("seller")
	coin, _ := cmd.Flags().GetString("coin")
	coins := strings.Split(coin, " ")
	var reqAddrCoins types.ReqRelayAddrCoins
	reqAddrCoins.Status = types.Relay_OnSell
	reqAddrCoins.Addr = seller
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetRelayOrder",
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
func ShowOnesRelayBuyOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy_order",
		Short: "Show one coin's buy orders",
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
	reqAddrCoins.Status = types.Relay_Deal
	reqAddrCoins.Addr = buyer
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetRelayBuyOrder",
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
func ShowOrdersByCoinStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coin_order",
		Short: "Show all coin's status",
		Run:   showCoinRelayOrders,
	}
	addShowCoinOrdersFlags(cmd)
	return cmd
}

func addShowCoinOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("coin", "c", "", "coins, separated by space")
	cmd.MarkFlagRequired("coin")

	cmd.Flags().Int32P("status", "s", 0, "relay order status (onsale:0, deal:1, finished:2,revoked:3)")
	cmd.MarkFlagRequired("status")
}

func showCoinRelayOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	status, _ := cmd.Flags().GetInt32("status")
	coin, _ := cmd.Flags().GetString("coin")
	coins := strings.Split(coin, " ")
	var reqAddrCoins types.ReqRelayAddrCoins
	reqAddrCoins.Status = status
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := jsonrpc.Query4Cli{
		Execer:   "relay",
		FuncName: "GetRelayOrderByCoinStatus",
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
		show.Orderid = order.Orderid
		show.Status = order.Status
		show.Sellamount = order.Sellamount
		show.Exchgaddr = order.Exchgaddr
		show.Exchgamount = order.Exchgamount
		show.Exchgcoin = order.Exchgcoin
		show.Waitcoinblocks = order.Waitcoinblocks
		show.Createtime = order.Createtime
		show.Buyeraddr = order.Buyeraddr
		show.Buyertime = order.Buyertime
		show.Buyercoinheight = order.Buyercoinheight
		show.Finishtime = order.Finishtime
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

//// create raw sell token transaction
func CreateRawRelaySellTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell",
		Short: "Create a exchange coin transaction",
		Run:   relaysell,
	}
	addExchangeFlags(cmd)
	return cmd
}

func addExchangeFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("sellamount", "s", 0, "amount of BTY")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("coin", "c", "", "coin to exchange by BTY, separated by space")
	cmd.MarkFlagRequired("coin")

	cmd.Flags().Int64P("coinamount", "m", 0, "coin amount to exchange")
	cmd.MarkFlagRequired("coin_amount")

	cmd.Flags().StringP("coinaddr", "a", "", "coin address in coin's block chain")
	cmd.MarkFlagRequired("coin_addr")

	cmd.Flags().Int32P("coinwait", "w", 0, "wait n blocks of coin to verify, min:1")
	cmd.MarkFlagRequired("total")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")

}

func relaysell(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount, _ := cmd.Flags().GetInt64("sellamount")
	coin, _ := cmd.Flags().GetString("coin")
	coinamount, _ := cmd.Flags().GetInt64("coinamount")
	coinaddr, _ := cmd.Flags().GetString("coinaddr")
	coinwait, _ := cmd.Flags().GetInt32("coinwait")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)

	params := &jsonrpc.RelaySellTx{
		SellAmount: amount,
		Coin:       coin,
		CoinAmount: coinamount,
		CoinAddr:   coinaddr,
		CoinWait:   coinwait,
		Fee:        feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTradeSellTx", params, &res)
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
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRevokeBuyTx", params, &res)
	ctx.Run()
}

// create raw verify transaction
func CreateRawRelayVerifyTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Create a verify coin transaction",
		Run:   relayVerify,
	}
	addVerifyFlags(cmd)
	return cmd
}

func addVerifyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "i", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().StringP("tx_hash", "h", "", "coin tx hash")
	cmd.MarkFlagRequired("tx_hash")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
	cmd.MarkFlagRequired("fee")
}

func relayVerify(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderid, _ := cmd.Flags().GetString("order_id")
	txhash, _ := cmd.Flags().GetString("tx_hash")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	params := &jsonrpc.RelayVerifyTx{
		OrderId: orderid,
		Txhash:  txhash,
		Fee:     feeInt64 * 1e4,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawRelayVerifyTx", params, &res)
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

type RelayVerifyBTCTx struct {
	OrderId     string `json:"order_id"`
	RawTx       string `json:"raw_tx"`
	TxIndex     uint32 `json:"tx_index"`
	MerklBranch string `json:"merkle_branch"`
	BlockHash   string `json:"block_hash"`
	Fee         int64  `json:"fee"`
}

func addVerifyBTCFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "o", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().StringP("raw_tx", "t", "", "coin raw tx")
	cmd.MarkFlagRequired("raw_tx")

	cmd.Flags().Int32P("tx_index", "i", 0, "raw tx index")
	cmd.MarkFlagRequired("tx_index")

	cmd.Flags().StringP("merk_branch", "m", "", "tx merkle branch")
	cmd.MarkFlagRequired("merk_branch")

	cmd.Flags().StringP("block_hash", "h", "", "block hash of tx ")
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
