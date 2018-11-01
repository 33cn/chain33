package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	"gitlab.33.cn/chain33/chain33/types"
)

func PokerBullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pokerbull",
		Short: "poker bull game management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		PokerBullStartRawTxCmd(),
		PokerBullContinueRawTxCmd(),
		PokerBullQuitRawTxCmd(),
		PokerBullQueryResultRawTxCmd(),
	)

	return cmd
}

func PokerBullStartRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new round poker bull game",
		Run:   pokerbullStart,
	}
	addPokerbullStartFlags(cmd)
	return cmd
}

func addPokerbullStartFlags(cmd *cobra.Command) {
	cmd.Flags().Uint64P("value", "a", 0, "value")
	cmd.MarkFlagRequired("value")

	cmd.Flags().Uint32P("playerCount", "p", 0, "player count")
	cmd.MarkFlagRequired("playerCount")
}

func pokerbullStart(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	value, _ := cmd.Flags().GetUint64("value")
	playerCount, _ := cmd.Flags().GetUint32("playerCount")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)
	amountInt64 := int64(value)

	params := &pkt.PBStartTxReq{
		Value:     amountInt64 * types.Coin,
		PlayerNum: int32(playerCount),
		Fee:       feeInt64,
	}

	var res string
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "pokerbull.PokerBullStartTx", params, &res)
	ctx.RunWithoutMarshal()
}

func PokerBullContinueRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "continue",
		Short: "Continue a new round poker bull game",
		Run:   pokerbullContinue,
	}
	addPokerbullContinueFlags(cmd)
	return cmd
}

func addPokerbullContinueFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.MarkFlagRequired("gameID")
}

func pokerbullContinue(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)

	params := &pkt.PBContinueTxReq{
		GameId: gameID,
		Fee:    feeInt64,
	}

	var res string
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "pokerbull.PokerBullContinueTx", params, &res)
	ctx.RunWithoutMarshal()
}

func PokerBullQuitRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quit",
		Short: "Quit game",
		Run:   pokerbullQuit,
	}
	addPokerbullQuitFlags(cmd)
	return cmd
}

func addPokerbullQuitFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.MarkFlagRequired("gameID")
}

func pokerbullQuit(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee * 1e4)

	params := &pkt.PBContinueTxReq{
		GameId: gameID,
		Fee:    feeInt64,
	}

	var res string
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "pokerbull.PokerBullQuitTx", params, &res)
	ctx.RunWithoutMarshal()
}

func PokerBullQueryResultRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query result",
		Run:   pokerbullQuery,
	}
	addPokerbullQueryFlags(cmd)
	return cmd
}

func addPokerbullQueryFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.Flags().StringP("address", "a", "", "address")
	cmd.Flags().StringP("index", "i", "", "index")
	cmd.Flags().StringP("status", "s", "", "status")
	cmd.Flags().StringP("gameIDs", "d", "", "gameIDs")
}

func pokerbullQuery(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")
	address, _ := cmd.Flags().GetString("address")
	statusStr, _ := cmd.Flags().GetString("status")
	status, _ := strconv.ParseInt(statusStr, 10, 32)
	indexstr, _ := cmd.Flags().GetString("index")
	index, _ := strconv.ParseInt(indexstr, 10, 64)
	gameIDs, _ := cmd.Flags().GetString("gameIDs")

	var params types.Query4Cli
	params.Execer = pkt.PokerBullX
	req := &pkt.QueryPBGameInfo{
		GameId: gameID,
		Addr:   address,
		Status: int32(status),
		Index:  index,
	}
	params.Payload = req
	if gameID != "" {
		params.FuncName = pkt.FuncName_QueryGameById
		var res pkt.ReplyPBGame
		ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
		ctx.Run()
	} else if address != "" {
		params.FuncName = pkt.FuncName_QueryGameByAddr
		var res pkt.PBGameRecords
		ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
		ctx.Run()
	} else if statusStr != "" {
		params.FuncName = pkt.FuncName_QueryGameByStatus
		var res pkt.PBGameRecords
		ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
		ctx.Run()
	} else if gameIDs != "" {
		params.FuncName = pkt.FuncName_QueryGameListByIds
		var gameIDsS []string
		gameIDsS = append(gameIDsS, gameIDs)
		gameIDsS = append(gameIDsS, gameIDs)
		req := &pkt.QueryPBGameInfos{gameIDsS}
		params.Payload = req
		var res pkt.ReplyPBGameList
		ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
		ctx.Run()
	} else {
		fmt.Println("Error: requeres at least one of gameID, address or status")
		cmd.Help()
	}
}
