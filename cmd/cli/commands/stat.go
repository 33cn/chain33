package commands

import (
	"encoding/json"
	"fmt"
	"errors"
	"os"
	"strconv"
	"time"

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
		GetTicketStatCmd(),
		GetTicketInfoCmd(),
		GetTicketInfoListCmd(),
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
	cmd.Flags().StringP("actual", "a", "", "actual statistics, any string")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().Int64P("height", "t", 0, "block height")
	cmd.MarkFlagRequired("height")
}

func totalCoins(cmd *cobra.Command, args []string) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	symbol, _ := cmd.Flags().GetString("symbol")
	height, _ := cmd.Flags().GetInt64("height")
	actual, _ := cmd.Flags().GetString("actual")

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
	var totalAmount int64
	resp := GetTotalCoinsResult{}

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
		key := append([]byte("Statistics:TotalFeeKey:"), blockHash...)
		params2 := types.LocalDBGet{Keys: [][]byte{key}}
		var res2 types.TotalFee
		err = rpc.Call("Chain33.QueryTotalFee", params2, &res2)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		resp.TxCount = res2.TxCount
		totalAmount = (3e+8+30000+30*height)*types.Coin - res2.Fee
		resp.TotalAmount = strconv.FormatFloat(float64(totalAmount)/float64(types.Coin), 'f', 4, 64)
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

// get ticket stat
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

	rpc, err := jsonrpc.NewJSONClient(rpcAddr)
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

	var resp GetTicketStatisticResult
	resp.CurrentOpenCount = res.CurrentOpenCount
	resp.TotalMinerCount = res.TotalMinerCount
	resp.TotalCancleCount = res.TotalCancleCount

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

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
	ticketId, _ := cmd.Flags().GetString("ticket_id")

	rpc, err := jsonrpc.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	key := []byte("Statistics:TicketInfo:TicketId:" + ticketId)
	fmt.Println(string(key))
	params := types.LocalDBGet{Keys: [][]byte{key}}
	var res types.TicketMinerInfo
	err = rpc.Call("Chain33.QueryTicketInfo", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var resp GetTicketMinerInfoResult
	resp.TicketId = res.TicketId
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
	cmd.MarkFlagRequired("count")
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
	ticketId, _ := cmd.Flags().GetString("ticket_id")

	if count <= 0 {
		fmt.Fprintln(os.Stderr, errors.New(fmt.Sprintf("input err, count:%v", count)))
		return
	}

	rpc, err := jsonrpc.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var key []byte
	prefix := []byte("Statistics:TicketInfoOrder:Addr:" + addr)
	if ticketId != "" && createTime != "" {
		key = []byte("Statistics:TicketInfoOrder:Addr:" + addr + ":CreateTime:" + createTime + ":TicketId:" + ticketId)
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

	var resp []GetTicketMinerInfoResult
	for _,v := range res {
		var ticket GetTicketMinerInfoResult
		ticket.TicketId = v.TicketId

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
		ticket.MinerTime = time.Unix(v.MinerTime, 0).Format("20060102150405")
		ticket.CloseTime = time.Unix(v.CloseTime, 0).Format("20060102150405")
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

