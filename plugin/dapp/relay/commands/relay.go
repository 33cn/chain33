package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	"gitlab.33.cn/chain33/chain33/types"
)

func RelayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Cross chain relay management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ShowOnesCreateRelayOrdersCmd(),
		ShowOnesAcceptRelayOrdersCmd(),
		ShowOnesStatusOrdersCmd(),
		ShowBTCHeadHeightListCmd(),
		ShowBTCHeadCurHeightCmd(),
		CreateRawRelayOrderTxCmd(),
		CreateRawRelayAcceptTxCmd(),
		CreateRawRevokeTxCmd(),
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

	var reqList ty.ReqRelayBtcHeaderHeightList
	reqList.ReqHeight = base
	reqList.Counts = count
	reqList.Direction = direct

	params := types.Query4Cli{
		Execer:   "relay",
		FuncName: "GetBTCHeaderList",
		Payload:  reqList,
	}
	rpc, err := jsonclient.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res ty.ReplyRelayBtcHeadHeightList
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

	var reqList ty.ReqRelayQryBTCHeadHeight
	reqList.BaseHeight = base

	params := types.Query4Cli{
		Execer:   "relay",
		FuncName: "GetBTCHeaderCurHeight",
		Payload:  reqList,
	}
	rpc, err := jsonclient.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res ty.ReplayRelayQryBTCHeadHeight
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayBtcCurHeight(res)
}

func ShowOnesCreateRelayOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "creator_orders",
		Short: "Show one creator's relay orders, coins optional",
		Run:   showOnesRelayOrders,
	}
	addShowRelayOrdersFlags(cmd)
	return cmd
}

func addShowRelayOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("creator", "a", "", "coin order creator")
	cmd.MarkFlagRequired("creator")

	cmd.Flags().StringP("coin", "c", "", "coins, default BTC, separated by space")

}

func showOnesRelayOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	creator, _ := cmd.Flags().GetString("creator")
	coin, _ := cmd.Flags().GetString("coin")
	coins := strings.Split(coin, " ")
	var reqAddrCoins ty.ReqRelayAddrCoins
	reqAddrCoins.Status = ty.RelayOrderStatus_pending
	reqAddrCoins.Addr = creator
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := types.Query4Cli{
		Execer:   "relay",
		FuncName: "GetSellRelayOrder",
		Payload:  reqAddrCoins,
	}
	rpc, err := jsonclient.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res ty.ReplyRelayOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayOrders(res)
}

func ShowOnesAcceptRelayOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acceptor_orders",
		Short: "Show one acceptor's accept orders, coins optional",
		Run:   showRelayAcceptOrders,
	}
	addShowRelayAcceptOrdersFlags(cmd)
	return cmd
}

func addShowRelayAcceptOrdersFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("acceptor", "a", "", "coin order acceptor")
	cmd.MarkFlagRequired("acceptor")

	cmd.Flags().StringP("coin", "c", "", "coins, separated by space")
}

func showRelayAcceptOrders(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	acceptor, _ := cmd.Flags().GetString("acceptor")
	coin, _ := cmd.Flags().GetString("coin")
	coins := strings.Split(coin, " ")
	var reqAddrCoins ty.ReqRelayAddrCoins
	reqAddrCoins.Status = ty.RelayOrderStatus_locking
	reqAddrCoins.Addr = acceptor
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := types.Query4Cli{
		Execer:   "relay",
		FuncName: "GetBuyRelayOrder",
		Payload:  reqAddrCoins,
	}
	rpc, err := jsonclient.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res ty.ReplyRelayOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayOrders(res)
}

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
	var reqAddrCoins ty.ReqRelayAddrCoins
	reqAddrCoins.Status = ty.RelayOrderStatus(status)
	if 0 != len(coins) {
		reqAddrCoins.Coins = append(reqAddrCoins.Coins, coins...)
	}
	params := types.Query4Cli{
		Execer:   "relay",
		FuncName: "GetRelayOrderByStatus",
		Payload:  reqAddrCoins,
	}
	rpc, err := jsonclient.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res ty.ReplyRelayOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	parseRelayOrders(res)
}

func parseRelayOrders(res ty.ReplyRelayOrders) {
	for _, order := range res.Relayorders {
		var show RelayOrder2Show
		show.OrderId = order.Id
		show.Status = order.Status.String()
		show.Creator = order.CreaterAddr
		show.CoinOperation = ty.RelayOrderOperation[order.CoinOperation]
		show.Amount = strconv.FormatFloat(float64(order.Amount)/float64(types.Coin), 'f', 4, 64)
		show.Coin = order.Coin
		show.CoinAddr = order.CoinAddr
		show.CoinAmount = strconv.FormatFloat(float64(order.CoinAmount)/float64(types.Coin), 'f', 4, 64)
		show.CoinWaits = order.CoinWaits
		show.CreateTime = order.CreateTime
		show.AcceptAddr = order.AcceptAddr
		show.AcceptTime = order.AcceptTime
		show.ConfirmTime = order.ConfirmTime
		show.FinishTime = order.FinishTime
		show.FinishTxHash = order.FinishTxHash
		show.Height = order.Height

		data, err := json.MarshalIndent(show, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Println(string(data))
	}
}

func parseRelayBtcHeadHeightList(res ty.ReplyRelayBtcHeadHeightList) {
	data, _ := json.Marshal(res)
	fmt.Println(string(data))
}

func parseRelayBtcCurHeight(res ty.ReplayRelayQryBTCHeadHeight) {
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Println(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func CreateRawRelayOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
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

	cmd.Flags().Uint32P("coin_wait", "n", 6, "coin blocks to wait,default:6,min:1")

	cmd.Flags().Float64P("bty_amount", "b", 0, "exchange amount of BTY")
	cmd.MarkFlagRequired("bty_amount")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
}

func relayOrder(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	oper, _ := cmd.Flags().GetUint32("operation")
	coin, _ := cmd.Flags().GetString("coin")
	coinamount, _ := cmd.Flags().GetFloat64("coin_amount")
	coinaddr, _ := cmd.Flags().GetString("coin_addr")
	coinwait, _ := cmd.Flags().GetUint32("coin_wait")
	btyamount, _ := cmd.Flags().GetFloat64("bty_amount")

	if coinwait == 0 {
		coinwait = 1
	}

	btyUInt64 := uint64(btyamount * 1e4)
	coinUInt64 := uint64(coinamount * 1e4)

	params := &ty.RelayCreate{
		Operation: oper,
		Amount:    coinUInt64 * 1e4,
		Coin:      coin,
		Addr:      coinaddr,
		CoinWaits: coinwait,
		BtyAmount: btyUInt64 * 1e4,
	}

	var res string
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "relay.CreateRawRelayOrderTx", params, &res)
	ctx.RunWithoutMarshal()
}

func CreateRawRelayAcceptTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accept",
		Short: "Create a accept coin transaction",
		Run:   relayAccept,
	}
	addRelayAcceptFlags(cmd)
	return cmd
}

func addRelayAcceptFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "o", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().StringP("coin_addr", "a", "", "coin address in coin's block chain")
	cmd.MarkFlagRequired("coin_addr")

	cmd.Flags().Uint32P("coin_wait", "n", 6, "coin blocks to wait,default:6,min:1")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
}

func relayAccept(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderID, _ := cmd.Flags().GetString("order_id")
	coinaddr, _ := cmd.Flags().GetString("coin_addr")
	coinwait, _ := cmd.Flags().GetUint32("coin_wait")

	if coinwait == 0 {
		coinwait = 1
	}

	params := &ty.RelayAccept{
		OrderId:   orderID,
		CoinAddr:  coinaddr,
		CoinWaits: coinwait,
	}
	var res string
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "relay.CreateRawRelayAcceptTx", params, &res)
	ctx.RunWithoutMarshal()
}

func CreateRawRevokeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Create a revoke transaction",
		Run:   relayRevoke,
	}
	addRevokeFlags(cmd)
	return cmd
}

func addRevokeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("order_id", "i", "", "order id")
	cmd.MarkFlagRequired("order_id")

	cmd.Flags().Uint32P("target", "t", 0, "0:create, 1:accept")
	cmd.MarkFlagRequired("target")

	cmd.Flags().Uint32P("action", "a", 0, "0:unlock, 1:cancel(only for creator)")
	cmd.MarkFlagRequired("action")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
}

func relayRevoke(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderID, _ := cmd.Flags().GetString("order_id")
	target, _ := cmd.Flags().GetUint32("target")
	act, _ := cmd.Flags().GetUint32("action")

	params := &ty.RelayRevoke{
		OrderId: orderID,
		Target:  target,
		Action:  act,
	}
	var res string
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "relay.CreateRawRelayRevokeTx", params, &res)
	ctx.RunWithoutMarshal()
}

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
}

func relayConfirm(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderId, _ := cmd.Flags().GetString("order_id")
	txHash, _ := cmd.Flags().GetString("tx_hash")

	params := &ty.RelayConfirmTx{
		OrderId: orderId,
		TxHash:  txHash,
	}
	var res string
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "relay.CreateRawRelayConfirmTx", params, &res)
	ctx.RunWithoutMarshal()
}

func CreateRawRelayBtcHeaderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save_header",
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

	cmd.Flags().StringP("merkle_root", "m", "", "merkle root")
	cmd.MarkFlagRequired("merkle_root")

	cmd.Flags().Uint64P("height", "t", 0, "block height")
	cmd.MarkFlagRequired("height")

	cmd.Flags().Int32P("flag", "g", 0, "block height")

	cmd.Flags().Float64P("fee", "f", 0, "coin transaction fee")
}

func relaySaveBtcHead(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	blockhash, _ := cmd.Flags().GetString("block_hash")
	prehash, _ := cmd.Flags().GetString("pre_hash")
	merkleroot, _ := cmd.Flags().GetString("merkle_root")
	height, _ := cmd.Flags().GetUint64("height")
	flag, _ := cmd.Flags().GetInt32("flag")

	params := &ty.BtcHeader{
		Hash:         blockhash,
		PreviousHash: prehash,
		MerkleRoot:   merkleroot,
		Height:       height,
		IsReset:      flag == 1,
	}

	var res string
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "relay.CreateRawRelaySaveBTCHeadTx", params, &res)
	ctx.RunWithoutMarshal()
}

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
}

func relayVerifyBTC(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderid, _ := cmd.Flags().GetString("order_id")
	rawtx, _ := cmd.Flags().GetString("raw_tx")
	txindex, _ := cmd.Flags().GetUint32("tx_index")
	merkbranch, _ := cmd.Flags().GetString("merk_branch")
	blockhash, _ := cmd.Flags().GetString("block_hash")

	params := &ty.RelayVerifyCli{
		OrderId:    orderid,
		RawTx:      rawtx,
		TxIndex:    txindex,
		MerkBranch: merkbranch,
		BlockHash:  blockhash,
	}

	var res string
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "relay.CreateRawRelayVerifyBTCTx", params, &res)
	ctx.RunWithoutMarshal()
}
