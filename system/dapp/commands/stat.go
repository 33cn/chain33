// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/rpc/jsonclient"

	"math/big"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/difficulty"
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
		GetTicketStatCmd(),
		GetTicketInfoCmd(),
		GetTicketInfoListCmd(),
		GetMinerStatCmd(),
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

// GetTicketStatCmd get ticket stat
func GetTicketStatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket_stat",
		Short: "Get ticket statistics by addr",
		Run:   ticketStat,
	}
	addTicketStatCmdFlags(cmd)
	return cmd
}

func addTicketStatCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "addr")
	cmd.MarkFlagRequired("addr")
}

func ticketStat(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	key := []byte("Statistics:TicketStat:Addr:" + addr)
	fmt.Println(string(key))
	params := types.LocalDBGet{Keys: [][]byte{key}}
	var res types.TicketStatistic
	err = rpc.Call("Chain33.QueryTicketStat", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var resp commandtypes.GetTicketStatisticResult
	resp.CurrentOpenCount = res.CurrentOpenCount
	resp.TotalMinerCount = res.TotalMinerCount
	resp.TotalCloseCount = res.TotalCancleCount

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

// GetTicketInfoCmd get a ticket information
func GetTicketInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket_info",
		Short: "Get ticket info by ticket_id",
		Run:   ticketInfo,
	}
	addTicketInfoCmdFlags(cmd)
	return cmd
}

func addTicketInfoCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("ticket_id", "t", "", "ticket_id")
	cmd.MarkFlagRequired("ticket_id")
}

func ticketInfo(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	ticketID, _ := cmd.Flags().GetString("ticket_id")

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	key := []byte("Statistics:TicketInfo:TicketId:" + ticketID)
	fmt.Println(string(key))
	params := types.LocalDBGet{Keys: [][]byte{key}}
	var res types.TicketMinerInfo
	err = rpc.Call("Chain33.QueryTicketInfo", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var resp commandtypes.GetTicketMinerInfoResult
	resp.TicketID = res.TicketId
	switch res.Status {
	case 1:
		resp.Status = "openTicket"
	case 2:
		resp.Status = "minerTicket"
	case 3:
		resp.Status = "closeTicket"
	}
	switch res.PrevStatus {
	case 1:
		resp.PrevStatus = "openTicket"
	case 2:
		resp.PrevStatus = "minerTicket"
	case 3:
		resp.PrevStatus = "closeTicket"
	}
	resp.IsGenesis = res.IsGenesis
	resp.CreateTime = time.Unix(res.CreateTime, 0).Format("20060102150405")
	resp.MinerTime = time.Unix(res.MinerTime, 0).Format("20060102150405")
	resp.CloseTime = time.Unix(res.CloseTime, 0).Format("20060102150405")
	resp.MinerValue = res.MinerValue
	resp.MinerAddress = res.MinerAddress

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

// GetTicketInfoListCmd get ticket information list
func GetTicketInfoListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket_info_list",
		Short: "Get ticket info list by ticket_id",
		Run:   ticketInfoList,
	}
	addTicketInfoListCmdFlags(cmd)
	return cmd
}

func addTicketInfoListCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "addr")
	cmd.MarkFlagRequired("addr")
	cmd.Flags().Int32P("count", "c", 10, "count")
	//cmd.MarkFlagRequired("count")
	cmd.Flags().Int32P("direction", "d", 1, "query direction (0: pre page, 1: next page)")
	cmd.Flags().StringP("create_time", "r", "", "create_time")
	cmd.Flags().StringP("ticket_id", "t", "", "ticket_id")
}

func ticketInfoList(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("direction")
	createTime, _ := cmd.Flags().GetString("create_time")
	ticketID, _ := cmd.Flags().GetString("ticket_id")

	if count <= 0 {
		fmt.Fprintln(os.Stderr, fmt.Errorf("input err, count:%v", count))
		return
	}

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var key []byte
	prefix := []byte("Statistics:TicketInfoOrder:Addr:" + addr)
	if ticketID != "" && createTime != "" {
		key = []byte("Statistics:TicketInfoOrder:Addr:" + addr + ":CreateTime:" + createTime + ":TicketId:" + ticketID)
	}
	fmt.Println(string(prefix))
	fmt.Println(string(key))
	params := types.LocalDBList{Prefix: prefix, Key: key, Direction: direction, Count: count}
	var res []types.TicketMinerInfo
	err = rpc.Call("Chain33.QueryTicketInfoList", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var resp []commandtypes.GetTicketMinerInfoResult
	for _, v := range res {
		var ticket commandtypes.GetTicketMinerInfoResult
		ticket.TicketID = v.TicketId

		switch v.Status {
		case 1:
			ticket.Status = "openTicket"
		case 2:
			ticket.Status = "minerTicket"
		case 3:
			ticket.Status = "closeTicket"
		}
		switch v.PrevStatus {
		case 1:
			ticket.PrevStatus = "openTicket"
		case 2:
			ticket.PrevStatus = "minerTicket"
		case 3:
			ticket.PrevStatus = "closeTicket"
		}
		ticket.IsGenesis = v.IsGenesis
		ticket.CreateTime = time.Unix(v.CreateTime, 0).Format("20060102150405")
		if v.MinerTime != 0 {
			ticket.MinerTime = time.Unix(v.MinerTime, 0).Format("20060102150405")
		}
		if v.CloseTime != 0 {
			ticket.CloseTime = time.Unix(v.CloseTime, 0).Format("20060102150405")
		}
		ticket.MinerValue = v.MinerValue
		ticket.MinerAddress = v.MinerAddress

		resp = append(resp, ticket)
	}

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

// GetMinerStatCmd get miner stat
func GetMinerStatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "miner",
		Short: "Get miner statistic",
		Run:   minerStat,
	}
	addMinerStatCmdFlags(cmd)
	return cmd
}

func addMinerStatCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "addr")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().Int64P("height", "t", 0, "start block height")
	cmd.MarkFlagRequired("height")
}

func minerStat(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	height, _ := cmd.Flags().GetInt64("height")

	if err := address.CheckAddress(addr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if height <= 0 {
		fmt.Fprintln(os.Stderr, fmt.Errorf("input err, height:%v", height))
		return
	}

	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// 获取最新高度
	var lastheader rpctypes.Header
	err = rpc.Call("Chain33.GetLastHeader", nil, &lastheader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// 循环取难度区间
	var diffList []difficultyRange
	var diff difficultyRange

	startH := height
diffListLoop:
	for {
		endH := startH + 10
		if endH > lastheader.Height {
			endH = lastheader.Height
		}

		params := types.ReqBlocks{
			Start:    startH,
			End:      endH,
			IsDetail: false,
		}
		var respHeaders rpctypes.Headers
		err = rpc.Call("Chain33.GetHeaders", params, &respHeaders)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		for _, h := range respHeaders.Items {
			if diff.bits != h.Difficulty {
				diff.timestamp = h.BlockTime
				diff.height = h.Height
				diff.bits = h.Difficulty
				diff.diff = difficulty.CompactToBig(diff.bits)
				diffList = append(diffList, diff)
			}
		}

		if len(respHeaders.Items) < 10 || endH+1 > lastheader.Height {
			// 最后一个点，记录截止时间
			diff.timestamp = respHeaders.Items[len(respHeaders.Items)-1].BlockTime
			diffList = append(diffList, diff)
			break diffListLoop
		}
		startH = endH + 1
	}

	// 统计挖矿概率
	bigOne := big.NewInt(1)
	oneLsh256 := new(big.Int).Lsh(bigOne, 256)
	oneLsh256.Sub(oneLsh256, bigOne)
	resp := new(MinerResult)
	resp.Expect = new(big.Float).SetInt64(0)

	var key []byte
	prefix := []byte("Statistics:TicketInfoOrder:Addr:" + addr)
	for {
		params := types.LocalDBList{Prefix: prefix, Key: key, Direction: 1, Count: 10}
		var res []types.TicketMinerInfo
		err = rpc.Call("Chain33.QueryTicketInfoList", params, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("QueryTicketInfoList failed:%v", err))
			fmt.Fprintln(os.Stderr, err)
			return
		}

		for _, v := range res {
			if v.CreateTime < diffList[0].timestamp-types.GetP(diffList[0].height).TicketFrozenTime {
				continue
			}
			//找到开始挖矿的难度坐标
			var pos int
			for i := range diffList {
				//创建时间+冻结时间<右节点
				if v.CreateTime >= diffList[i].timestamp-types.GetP(diffList[i].height).TicketFrozenTime {
					continue
				} else {
					pos = i - 1
					break
				}
			}
			diffList2 := diffList[pos:]

			//没有挖到，还在挖
			if v.CloseTime == 0 && v.MinerTime == 0 {
				for i := range diffList2 {
					if i == 0 {
						continue
					}
					probability := new(big.Float).Quo(new(big.Float).SetInt(diffList2[i-1].diff), new(big.Float).SetInt(oneLsh256))
					if i == 1 {
						resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(diffList2[i].timestamp-v.CreateTime-types.GetP(diffList2[i-1].height).TicketFrozenTime)))
					} else {
						resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(diffList2[i].timestamp-diffList2[i-1].timestamp)))
					}
				}
			}

			//没有挖到，主动关闭
			if v.MinerTime == 0 && v.CloseTime != 0 {
				for i := range diffList2 {
					if i == 0 {
						continue
					}
					probability := new(big.Float).Quo(new(big.Float).SetInt(diffList2[i-1].diff), new(big.Float).SetInt(oneLsh256))
					if i == 1 {
						resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(diffList2[i].timestamp-v.CreateTime-types.GetP(diffList2[i-1].height).TicketFrozenTime)))
					} else {
						if v.CloseTime <= diffList2[i].timestamp {
							resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(v.CloseTime-diffList2[i-1].timestamp)))
							break
						} else {
							resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(diffList2[i].timestamp-diffList2[i-1].timestamp)))
						}
					}
				}
			}

			//挖到
			if v.MinerTime != 0 {
				for i := range diffList2 {
					if i == 0 {
						continue
					}
					probability := new(big.Float).Quo(new(big.Float).SetInt(diffList2[i-1].diff), new(big.Float).SetInt(oneLsh256))
					if i == 1 {
						resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(diffList2[i].timestamp-v.CreateTime-types.GetP(diffList2[i-1].height).TicketFrozenTime)))
					} else {
						if v.MinerTime <= diffList2[i].timestamp {
							resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(v.MinerTime-diffList2[i-1].timestamp)))
							break
						} else {
							resp.Expect = resp.Expect.Add(resp.Expect, probability.Mul(probability, new(big.Float).SetInt64(diffList2[i].timestamp-diffList2[i-1].timestamp)))
						}
					}
				}

				resp.Actual++
			}
		}

		if len(res) < 10 {
			break
		}
		key = []byte("Statistics:TicketInfoOrder:Addr:" + addr + ":CreateTime:" + time.Unix(res[len(res)-1].CreateTime, 0).Format("20060102150405") + ":TicketId:" + res[len(res)-1].TicketId)
	}

	expect := resp.Expect.Text('f', 2)
	fmt.Println("Expect:", expect)
	fmt.Println("Actual:", resp.Actual)

	e, _ := strconv.ParseFloat(expect, 64)
	fmt.Printf("Deviation:%+.3f%%\n", (float64(resp.Actual)/e-1.0)*100.0)
}

type difficultyRange struct {
	timestamp int64
	height    int64
	bits      uint32
	diff      *big.Int
}

// MinerResult defiles miner command
type MinerResult struct {
	Expect *big.Float
	Actual int64
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
