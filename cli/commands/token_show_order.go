package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

var (
	status string
	owner  string
	token  string
)

func ShowTokenOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show_order",
		Short: "Show token orders of user",
		Run:   showTokenOrder,
	}
	addShowTokenOrderFlags(cmd)
	return cmd
}

func addShowTokenOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&status, "status", "s", "", `token order status ("buy","onsale","soldout" or "revoked")`)
	cmd.Flags().StringVarP(&owner, "owner", "o", "", "token owner address")
	cmd.Flags().StringVarP(&token, "token", "t", "", "token symbols, seperated by ','")
}

func isStatusSet() bool {
	return status == "buy" || status == "onsale" || status == "soldout" || status == "revoked"
}

func isStatusBuy() bool {
	return status == "buy"
}

func isStatusSell() bool {
	return status == "onsale" || status == "soldout" || status == "revoked"
}

func showTokenOrder(cmd *cobra.Command, args []string) {
	rpc_laddr, _ := cmd.Flags().GetString("rpc_laddr")

	//fmt.Println("showsellorderwithstatus [onsale | soldout | revoked]           : 显示指定状态下的所有卖单")
	//fmt.Println("showonesbuyorder [buyer]                                       : 显示指定用户下所有token成交的购买单")
	//fmt.Println("showonesbuytokenorder [buyer, token0, [token1, token2]]        : 显示指定用户下指定token成交的购买单")
	//fmt.Println("showonesselltokenorder [seller, [token0, token1, token2]]      : 显示一个用户下的token卖单")

	// status  owner  token      ---> results
	//   s       x      x             showsellorderwithstatus
	//   b       √    both            showonesbuyorder/showonesbuytokenorder
	//   x       √    both            showonesselltokenorder
	if isStatusSell() && owner == "" && token == "" {
		showsellorderwithstatus(rpc_laddr)
	} else if isStatusBuy() && owner != "" {
		showonesbuyorder(rpc_laddr)
	} else if !isStatusSet() && owner != "" {
		showonesselltokenorder(rpc_laddr)
	} else {
		cmd.UsageFunc()(cmd)
		return
	}
}

func showsellorderwithstatus(rpcAddr string) {
	statusInt, ok := types.MapSellOrderStatusStr2Int[status]
	if !ok {
		fmt.Print(errors.New("status 参数错误\n").Error())
		return
	}
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Status = statusInt

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetAllSellOrdersWithStatus"
	params.Payload = reqAddrtokens
	rpc, err := jsonrpc.NewJsonClient(rpcAddr)
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

func showonesbuyorder(rpcAddr string) {
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Addr = owner
	if token != "" {
		reqAddrtokens.Token = strings.Split(token, ",")
	}

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesBuyOrder"
	params.Payload = reqAddrtokens
	rpc, err := jsonrpc.NewJsonClient(rpcAddr)
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

func showonesselltokenorder(rpcAddr string) {
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Status = types.OnSale
	reqAddrtokens.Addr = owner
	if token != "" {
		reqAddrtokens.Token = strings.Split(token, ",")
	}

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesSellOrder"
	params.Payload = reqAddrtokens

	rpc, err := jsonrpc.NewJsonClient(rpcAddr)
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
