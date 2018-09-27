package commands

import (
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
		Value:       amountInt64 * types.Coin,
		PlayerNum:   int32(playerCount),
		Fee:         feeInt64,
	}

	var res string
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "PokerBull.PokerBullStartTx", params, &res)
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
		GameId:      gameID,
		Fee:         feeInt64,
	}

	var res string
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "PokerBull.PokerBullContinueTx", params, &res)
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
		GameId:      gameID,
		Fee:         feeInt64,
	}

	var res string
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "PokerBull.PokerBullQuitTx", params, &res)
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
	cmd.MarkFlagRequired("gameID")
}

func pokerbullQuery(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")

	var params types.Query4Cli
	params.Execer = pkt.PokerBullX

	req := &pkt.QueryPBGameInfo{
		GameId: gameID,
	}

	params.Payload = req
	params.FuncName = pkt.FuncName_QueryGameById
	var res pkt.ReplyPBGame
	ctx := jsonrpc.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.Run()
}