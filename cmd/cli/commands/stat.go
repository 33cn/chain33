package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func StatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stat",
		Short: "Coin statistic",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GetTotalCoinsCmd(),
	)

	return cmd
}

// get total coins
func GetTotalCoinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total_coins",
		Short: "Get total amount of a token",
		Run:   totalCoins,
	}
	addTotalCoinsCmdFlags(cmd)
	return cmd
}

func addTotalCoinsCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().Int64P("height", "t", 0, "block height")
	cmd.MarkFlagRequired("height")
}

func totalCoins(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	height, _ := cmd.Flags().GetInt64("height")

	// 获取高度statehash
	params := jsonrpc.BlockParam{
		Start:    height,
		End:      height,
		Isdetail: false,
	}

	rpc, err := jsonrpc.NewJSONClient(rpcAddr)
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
	for count = 100; count == 100; {
		params := types.ReqGetTotalCoins{
			Symbol:    symbol,
			StateHash: stateHash,
			StartKey:  startKey,
			Count:     count,
		}
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
		params := types.ReqInt{height}
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
		expectedAmount = (3e+8+30000+30*height)*types.Coin - res2.Fee
		resp.ExpectedAmount = strconv.FormatFloat(float64(expectedAmount)/float64(types.Coin), 'f', 4, 64)
		resp.ActualAmount = strconv.FormatFloat(float64(actualAmount)/float64(types.Coin), 'f', 4, 64)
		resp.DifferenceAmount = strconv.FormatFloat(float64(expectedAmount-actualAmount)/float64(types.Coin), 'f', 4, 64)
	} else {
		//查询Token总量
		var req types.ReqString
		req.Data = symbol
		var params jsonrpc.Query4Cli
		params.Execer = "token"
		params.FuncName = "GetTokenInfo"
		params.Payload = req
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
