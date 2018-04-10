package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cli/commands"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/log"
)

var rootCmd = &cobra.Command{
	Use:   "chain33-cli",
	Short: "chain33 client tools",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(common.GetVersion())
	},
}

func init() {
	rootCmd.PersistentFlags().String("rpc_laddr", "http://localhost:8801", "RPC listen address")

	rootCmd.AddCommand(
		commands.CommonCmd(),
		commands.SeedCmd(),
		commands.KeyCmd(),
		commands.AccountCmd(),
		commands.TxCmd(),
		commands.TokenCmd(),
		commands.TicketCmd(),
		commands.MempoolCmd(),
		commands.WalletCmd(),
		commands.AddressCmd(),
		commands.BlockCmd(),
		versionCmd)
}

func main() {
	log.SetLogLevel("error")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GetTotalCoins(symbol string, height string) {
	// 获取高度statehash
	heightInt64, err := strconv.ParseInt(height, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	params := jsonrpc.BlockParam{Start: heightInt64, End: heightInt64, Isdetail: false}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.BlockDetails
	err = rpc.Call("Chain33.GetBlocks", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	stateHash, err := common.FromHex(res.Items[0].Block.StateHash)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// 查询高度哈希对应数据
	var expectedAmount int64
	var actualAmount int64
	resp := GetTotalCoinsResult{}

	var startKey []byte
	var count int64
	for count = 1000; count == 1000; {
		params := types.ReqGetTotalCoins{Symbol: symbol, StateHash: stateHash, StartKey: startKey, Count: count}
		var res types.ReplyGetTotalCoins
		err = rpc.Call("Chain33.GetTotalCoins", params, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		count = res.Num
		resp.AccountCount += res.Num
		actualAmount += res.Amount
		startKey = res.NextKey
	}

	if symbol == "bty" {
		//查询高度blockhash
		params := types.ReqInt{heightInt64}
		var res1 jsonrpc.ReplyHash
		err = rpc.Call("Chain33.GetBlockHash", params, &res1)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		blockHash, err := common.FromHex(res1.Hash)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		//查询手续费
		params2 := types.ReqHash{Hash: blockHash}
		var res2 types.TotalFee
		err = rpc.Call("Chain33.QueryTotalFee", params2, &res2)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		resp.TxCount = res2.TxCount
		expectedAmount = (3e+8+30000+30*heightInt64)*types.Coin - res2.Fee
		resp.ExpectedAmount = strconv.FormatFloat(float64(expectedAmount)/float64(types.Coin), 'f', 4, 64)
		resp.ActualAmount = strconv.FormatFloat(float64(actualAmount)/float64(types.Coin), 'f', 4, 64)
		resp.DifferenceAmount = strconv.FormatFloat(float64(expectedAmount-actualAmount)/float64(types.Coin), 'f', 4, 64)
	} else {
		//查询Token总量
		var req types.ReqString
		req.Data = symbol
		var params jsonrpc.Query
		params.Execer = "token"
		params.FuncName = "GetTokenInfo"
		params.Payload = hex.EncodeToString(types.Encode(&req))
		var res types.Token
		err = rpc.Call("Chain33.Query", params, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		expectedAmount = res.Total
		resp.ExpectedAmount = strconv.FormatFloat(float64(expectedAmount)/float64(types.TokenPrecision), 'f', 4, 64)
		resp.ActualAmount = strconv.FormatFloat(float64(actualAmount)/float64(types.TokenPrecision), 'f', 4, 64)
		resp.DifferenceAmount = strconv.FormatFloat(float64(expectedAmount-actualAmount)/float64(types.TokenPrecision), 'f', 4, 64)
	}

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}
