// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"

	"os"

	"github.com/spf13/cobra"
)

// StatCmd stat command
func StatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stat",
		Short: "Coin statistic",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		getTotalCoinsCmd(),
		getExecBalanceCmd(),
		totalFeeCmd(),
	)

	return cmd
}

// getTotalCoinsCmd get total coins
func getTotalCoinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total_coins",
		Short: "Get total amount of a token (default: bty of current height)",
		Run:   totalCoins,
	}
	addTotalCoinsCmdFlags(cmd)
	return cmd
}

func addTotalCoinsCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("symbol", "s", "bty", "token symbol")
	cmd.Flags().StringP("actual", "a", "", "actual statistics, any string")
	cmd.Flags().Int64P("height", "t", -1, `block height, "-1" stands for current height`)
}

func totalCoins(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	height, _ := cmd.Flags().GetInt64("height")
	actual, _ := cmd.Flags().GetString("actual")

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	cfg, err := commandtypes.GetChainConfig(rpcAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get chain config err=%s", err.Error())
		return
	}

	var stateHashHex string
	if height < 0 {

		header, err := getLastBlock(rpc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		height = header.Height
		stateHashHex = header.StateHash
	} else {
		blocks, err := getBlocks(height, height, rpc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetBlocksErr:%s", err.Error())
			return
		}
		stateHashHex = blocks.Items[0].Block.StateHash
	}

	stateHash, err := common.FromHex(stateHashHex)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// 查询高度哈希对应数据
	var totalAmount int64
	resp := commandtypes.GetTotalCoinsResult{}

	if symbol == "bty" {
		//查询历史总手续费
		fee, err := queryTotalFeeWithHeight(height, rpc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		resp.TxCount = fee.TxCount
		var issueCoins int64
		//只适用bty主网计算
		if height < 2270000 {
			issueCoins = 30 * height
		} else { //挖矿产量降低30->8
			issueCoins = 22*2269999 + height*8
		}
		totalAmount = (317430000+issueCoins)*cfg.CoinPrecision - fee.Fee
		resp.TotalAmount = types.FormatAmount2FloatDisplay(totalAmount, cfg.CoinPrecision, true)
	} else {
		var req types.ReqString
		req.Data = symbol
		var params rpctypes.Query4Jrpc
		params.Execer = "token"
		// 查询Token的总量
		params.FuncName = "GetTotalAmount"
		params.Payload = types.MustPBToJSON(&req)
		var res types.TotalAmount

		//查询Token总量
		//params.FuncName = "GetTokenInfo"
		//params.Payload = req
		//var res tokenty.Token
		err = rpc.Call("Chain33.Query", params, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		totalAmount = res.Total
		resp.TotalAmount = types.FormatAmount2FloatDisplay(totalAmount, cfg.TokenPrecision, true)
	}

	if actual != "" {
		var actualAmount int64
		var startKey []byte
		var count int64
		for count = 100; count == 100; {
			params := types.ReqGetTotalCoins{
				Symbol:    symbol,
				StateHash: stateHash,
				StartKey:  startKey,
				Count:     count,
			}
			if symbol == "bty" {
				params.Execer = "coins"
			} else {
				params.Execer = "token"
			}
			var res types.ReplyGetTotalCoins
			err = rpc.Call("Chain33.GetTotalCoins", &params, &res)
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
			resp.ActualAmount = types.FormatAmount2FloatDisplay(actualAmount, cfg.CoinPrecision, true)
			resp.DifferenceAmount = types.FormatAmount2FloatDisplay(totalAmount-actualAmount, cfg.CoinPrecision, true)
		} else {
			resp.ActualAmount = types.FormatAmount2FloatDisplay(actualAmount, cfg.TokenPrecision, true)
			resp.DifferenceAmount = types.FormatAmount2FloatDisplay(totalAmount-actualAmount, cfg.TokenPrecision, true)
		}

	}

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

// getExecBalanceCmd get exec-addr balance of specific addr
func getExecBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec_balance",
		Short: "Get the exec amount of a token of one address (default: all exec-addr bty of current height of one addr)",
		Run:   execBalance,
	}
	addExecBalanceCmdFlags(cmd)
	return cmd
}

func addExecBalanceCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("symbol", "s", "bty", "token symbol")
	cmd.Flags().StringP("exec", "e", "coins", "excutor name")
	cmd.Flags().StringP("addr", "a", "", "address")
	cmd.MarkFlagRequired("addr")
	cmd.Flags().StringP("exec_addr", "x", "", "exec address")
	cmd.Flags().Int64P("height", "t", -1, `block height, "-1" stands for current height`)
}

func execBalance(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	exec, _ := cmd.Flags().GetString("exec")
	addr, _ := cmd.Flags().GetString("addr")
	execAddr, _ := cmd.Flags().GetString("exec_addr")
	height, _ := cmd.Flags().GetInt64("height")

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	cfg, err := commandtypes.GetChainConfig(rpcAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get chain config err=%s", err.Error())
		return
	}

	var stateHashHex string
	if height < 0 {

		header, err := getLastBlock(rpc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		stateHashHex = header.StateHash
	} else {
		blocks, err := getBlocks(height, height, rpc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetBlocksErr:%s", err.Error())
			return
		}
		stateHashHex = blocks.Items[0].Block.StateHash
	}

	stateHash, err := common.FromHex(stateHashHex)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	resp := commandtypes.GetExecBalanceResult{}

	if symbol == "bty" {
		exec = "coins"
	}

	reqParam := types.ReqGetExecBalance{
		Symbol:    symbol,
		StateHash: stateHash,
		Addr:      []byte(addr),
		ExecAddr:  []byte(execAddr),
		Execer:    exec,
	}

	if len(execAddr) > 0 {
		reqParam.Count = 1 //由于精确匹配一条记录，所以这里设定为1
	} else {
		reqParam.Count = 100 //每次最多读取100条
	}

	var replys types.ReplyGetExecBalance
	for {
		var reply types.ReplyGetExecBalance
		var str string
		err = rpc.Call("Chain33.GetExecBalance", &reqParam, &str)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		data, err := common.FromHex(str)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		err = types.Decode(data, &reply)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		replys.Amount += reply.Amount
		replys.AmountActive += reply.AmountActive
		replys.AmountFrozen += reply.AmountFrozen
		replys.Items = append(replys.Items, reply.Items...)

		if reqParam.Count == 1 {
			break
		}

		if len(reply.NextKey) > 0 {
			reqParam.NextKey = reply.NextKey
		} else {
			break
		}
	}

	if symbol == "bty" {
		convertReplyToResult(&replys, &resp, cfg.CoinPrecision)
	} else {
		convertReplyToResult(&replys, &resp, cfg.TokenPrecision)
	}

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func convertReplyToResult(reply *types.ReplyGetExecBalance, result *commandtypes.GetExecBalanceResult, precision int64) {
	result.Amount = types.FormatAmount2FloatDisplay(reply.Amount, precision, true)
	result.AmountFrozen = types.FormatAmount2FloatDisplay(reply.AmountFrozen, precision, true)
	result.AmountActive = types.FormatAmount2FloatDisplay(reply.AmountActive, precision, true)

	for i := 0; i < len(reply.Items); i++ {
		item := &commandtypes.ExecBalance{}
		item.ExecAddr = string(reply.Items[i].ExecAddr)
		item.Frozen = types.FormatAmount2FloatDisplay(reply.Items[i].Frozen, precision, true)
		item.Active = types.FormatAmount2FloatDisplay(reply.Items[i].Active, precision, true)
		result.ExecBalances = append(result.ExecBalances, item)
	}
}

// totalFeeCmd query total fee command
func totalFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total_fee",
		Short: "query total transaction fee, support specific block height interval [start, end]",
		Run:   totalFee,
	}
	cmd.Flags().Int64P("start_height", "s", 0, "start block height, default 0")
	cmd.Flags().Int64P("end_height", "e", -1, "end block height, default current block height")
	return cmd
}

func totalFee(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	start, _ := cmd.Flags().GetInt64("start_height")
	end, _ := cmd.Flags().GetInt64("end_height")

	var startFeeAmount, endFeeAmount int64
	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewJsonClientErr:%s\n", err.Error())
		return
	}
	if start < 0 {
		start = 0
	}

	if start > 0 {
		totalFee, err := queryTotalFeeWithHeight(start-1, rpc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryStartFeeErr:%s\n", err.Error())
			return
		}
		startFeeAmount = totalFee.Fee
	}

	if end < 0 {
		//last block fee
		currentHeight, totalFee, err := queryCurrentTotalFee(rpc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryCurrentTotalFeeErr:%s\n", err.Error())
			return
		}
		endFeeAmount = totalFee.Fee
		end = currentHeight
	} else {
		totalFee, err := queryTotalFeeWithHeight(end, rpc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryEndFeeErr:%s\n", err.Error())
			return
		}
		endFeeAmount = totalFee.Fee
	}

	fee := endFeeAmount - startFeeAmount
	if fee < 0 {
		fee = 0
	}

	cfg, err := commandtypes.GetChainConfig(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	resp := fmt.Sprintf(`{"startHeight":%d,"endHeight":%d, "totalFee":%s}`, start, end, types.FormatAmount2FloatDisplay(fee, cfg.CoinPrecision, true))
	buf := &bytes.Buffer{}
	err = json.Indent(buf, []byte(resp), "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JsonIndentResultErr:%s\n", err.Error())
		return
	}

	fmt.Println(buf.String())
}

// get last block header
func getLastBlock(rpc *jsonclient.JSONClient) (*rpctypes.Header, error) {

	res := &rpctypes.Header{}
	err := rpc.Call("Chain33.GetLastHeader", nil, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// get block hash with height
func getBlockHash(height int64, rpc *jsonclient.JSONClient) (string, error) {

	params := types.ReqInt{Height: height}
	var res rpctypes.ReplyHash
	err := rpc.Call("Chain33.GetBlockHash", &params, &res)
	if err != nil {
		return "", err
	}
	return res.Hash, nil
}

func queryCurrentTotalFee(rpc *jsonclient.JSONClient) (int64, *types.TotalFee, error) {
	header, err := getLastBlock(rpc)
	if err != nil {
		return 0, nil, err
	}
	fee, err := queryTotalFeeWithHash(header.Hash, rpc)
	if err != nil {
		return 0, nil, err
	}
	return header.Height, fee, nil
}

func queryTotalFeeWithHeight(height int64, rpc *jsonclient.JSONClient) (*types.TotalFee, error) {
	hash, err := getBlockHash(height, rpc)
	if err != nil {
		return nil, err
	}
	fee, err := queryTotalFeeWithHash(hash, rpc)
	if err != nil {
		return nil, err
	}
	return fee, nil
}

func queryTotalFeeWithHash(blockHash string, rpc *jsonclient.JSONClient) (*types.TotalFee, error) {

	hash, err := common.FromHex(blockHash)
	if err != nil {
		return nil, err
	}

	//查询手续费
	hash = append([]byte("TotalFeeKey:"), hash...)
	params := types.LocalDBGet{Keys: [][]byte{hash[:]}}
	res := &types.TotalFee{}
	err = rpc.Call("Chain33.QueryTotalFee", &params, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func getBlocks(start, end int64, rpc *jsonclient.JSONClient) (*rpctypes.BlockDetails, error) {
	// 获取blocks
	params := rpctypes.BlockParam{
		Start: start,
		End:   end,
		//Isdetail: false,
		Isdetail: true,
	}
	res := &rpctypes.BlockDetails{}
	err := rpc.Call("Chain33.GetBlocks", params, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
