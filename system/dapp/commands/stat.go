// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
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
		GetTotalCoinsCmd(),
		GetExecBalanceCmd(),
	)

	return cmd
}

// GetTotalCoinsCmd get total coins
func GetTotalCoinsCmd() *cobra.Command {
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

	if height == -1 {
		rpc, err := jsonclient.NewJSONClient(rpcAddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		var res rpctypes.Header
		err = rpc.Call("Chain33.GetLastHeader", nil, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		height = res.Height
	}

	// 获取高度statehash
	params := rpctypes.BlockParam{
		Start: height,
		End:   height,
		//Isdetail: false,
		Isdetail: true,
	}

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res rpctypes.BlockDetails
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
	var totalAmount int64
	resp := commandtypes.GetTotalCoinsResult{}

	if symbol == "bty" {
		//查询高度blockhash
		params := types.ReqInt{Height: height}
		var res1 rpctypes.ReplyHash
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
		key := append([]byte("TotalFeeKey:"), blockHash...)
		params2 := types.LocalDBGet{Keys: [][]byte{key}}
		var res2 types.TotalFee
		err = rpc.Call("Chain33.QueryTotalFee", params2, &res2)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		resp.TxCount = res2.TxCount
		totalAmount = (317430000+30*height)*types.Coin - res2.Fee
		resp.TotalAmount = strconv.FormatFloat(float64(totalAmount)/float64(types.Coin), 'f', 4, 64)
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
		resp.TotalAmount = strconv.FormatFloat(float64(totalAmount)/float64(types.TokenPrecision), 'f', 4, 64)
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
			resp.ActualAmount = strconv.FormatFloat(float64(actualAmount)/float64(types.Coin), 'f', 4, 64)
			resp.DifferenceAmount = strconv.FormatFloat(float64(totalAmount-actualAmount)/float64(types.Coin), 'f', 4, 64)
		} else {
			resp.ActualAmount = strconv.FormatFloat(float64(actualAmount)/float64(types.TokenPrecision), 'f', 4, 64)
			resp.DifferenceAmount = strconv.FormatFloat(float64(totalAmount-actualAmount)/float64(types.TokenPrecision), 'f', 4, 64)
		}

	}

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

// GetExecBalanceCmd get exec-addr balance of specific addr
func GetExecBalanceCmd() *cobra.Command {
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

	if height == -1 {
		rpc, err := jsonclient.NewJSONClient(rpcAddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		var res rpctypes.Header
		err = rpc.Call("Chain33.GetLastHeader", nil, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		height = res.Height
	}

	// 获取高度statehash
	params := rpctypes.BlockParam{
		Start: height,
		End:   height,
		//Isdetail: false,
		Isdetail: true,
	}

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res rpctypes.BlockDetails
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
	reqParam.StateHash = stateHash

	if len(execAddr) > 0 {
		reqParam.Count = 1 //由于精确匹配一条记录，所以这里设定为1
	} else {
		reqParam.Count = 100 //每次最多读取100条
	}

	var replys types.ReplyGetExecBalance
	for {
		var reply types.ReplyGetExecBalance
		var str string
		err = rpc.Call("Chain33.GetExecBalance", reqParam, &str)
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
		convertReplyToResult(&replys, &resp, types.Coin)
	} else {
		convertReplyToResult(&replys, &resp, types.TokenPrecision)
	}

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func convertReplyToResult(reply *types.ReplyGetExecBalance, result *commandtypes.GetExecBalanceResult, precision int64) {
	result.Amount = strconv.FormatFloat(float64(reply.Amount)/float64(precision), 'f', 4, 64)
	result.AmountFrozen = strconv.FormatFloat(float64(reply.AmountFrozen)/float64(precision), 'f', 4, 64)
	result.AmountActive = strconv.FormatFloat(float64(reply.AmountActive)/float64(precision), 'f', 4, 64)

	for i := 0; i < len(reply.Items); i++ {
		item := &commandtypes.ExecBalance{}
		item.ExecAddr = string(reply.Items[i].ExecAddr)
		item.Frozen = strconv.FormatFloat(float64(reply.Items[i].Frozen)/float64(precision), 'f', 4, 64)
		item.Active = strconv.FormatFloat(float64(reply.Items[i].Active)/float64(precision), 'f', 4, 64)
		result.ExecBalances = append(result.ExecBalances, item)
	}
}
