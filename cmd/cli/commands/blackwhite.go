package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common"
	bw "gitlab.33.cn/chain33/chain33/types/executor/blackwhite"
	"strings"
)

func BlackwhiteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blackwhite",
		Short: "blackwhite game management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		BlackwhiteCreateRawTxCmd(),
		BlackwhitePlayRawTxCmd(),
		BlackwhiteShowRawTxCmd(),
		BlackwhiteTimeoutDoneTxCmd(),
		ShowBlackwhiteInfoCmd(),
	)

	return cmd
}

func BlackwhiteCreateRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new round blackwhite game",
		Run:   blackwhiteCreate,
	}
	addBlackwhiteCreateFlags(cmd)
	return cmd
}

func addBlackwhiteCreateFlags(cmd *cobra.Command) {
	cmd.Flags().Uint64P("amount", "a", 0, "amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().Uint32P("playerCount", "p", 0, "player count")
	cmd.MarkFlagRequired("playerCount")

	cmd.Flags().Int64P("timeout", "t", 0, "timeout(min),default:10min")

	cmd.Flags().StringP("gameName", "g", "", "game name")
}

func blackwhiteCreate(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount, _ := cmd.Flags().GetUint64("amount")
	playerCount, _ := cmd.Flags().GetUint32("playerCount")
	timeout, _ := cmd.Flags().GetInt64("timeout")
	gameName, _ := cmd.Flags().GetString("gameName")

	amountInt64 := int64(amount)

	if timeout == 0 {
		timeout = 10
	}
	timeout = 60 * timeout

	params := &types.BlackwhiteCreate{
		PlayAmount:  amountInt64 * types.Coin,
		PlayerCount: int32(playerCount),
		Timeout:     timeout,
		GameName:    gameName,
	}

	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhiteCreateTx", params, &res)
	ctx.RunWithoutMarshal()
}

func BlackwhitePlayRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "play",
		Short: "play a blackwhite game",
		Run:   blackwhitePlay,
	}
	addBlackwhitePlayFlags(cmd)
	return cmd
}

func addBlackwhitePlayFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.MarkFlagRequired("gameID")

	cmd.Flags().Uint64P("amount", "a", 0, "amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("isBlackStr", "i", "", "[0-1-1-1-1-1-0-0-1-1] (1:black,0:white)")
	cmd.MarkFlagRequired("isBlackStr")

	cmd.Flags().StringP("secret", "s", "", "secret key")
	cmd.MarkFlagRequired("secret")

}

func blackwhitePlay(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")
	amount, _ := cmd.Flags().GetUint64("amount")
	isBlackStr, _ := cmd.Flags().GetString("isBlackStr")
	secret, _ := cmd.Flags().GetString("secret")

	blacks := strings.Split(isBlackStr, "-")

	var hashValues [][]byte
	for _, black := range blacks {
		if black == "1" {
			hashValues =append(hashValues, common.Sha256([]byte(secret+black)))
		} else {
			white := "0"
			hashValues =append(hashValues, common.Sha256([]byte(secret+white)))
		}
	}


	amountInt64 := int64(amount)
	params := &types.BlackwhitePlay{
		GameID:  gameID,
		Amount:  amountInt64 * types.Coin,
		HashValues: hashValues,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhitePlayTx", params, &res)
	ctx.RunWithoutMarshal()
}

func BlackwhiteShowRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "show secret key",
		Run:   blackwhiteShow,
	}
	addBlackwhiteShowFlags(cmd)
	return cmd
}

func addBlackwhiteShowFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.MarkFlagRequired("gameID")

	cmd.Flags().StringP("secret", "s", "", "secret key")
	cmd.MarkFlagRequired("secret")

}

func blackwhiteShow(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")
	secret, _ := cmd.Flags().GetString("secret")

	params := &types.BlackwhiteShow{
		GameID: gameID,
		Secret: secret,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhiteShowTx", params, &res)
	ctx.RunWithoutMarshal()
}

func BlackwhiteTimeoutDoneTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeoutDone",
		Short: "timeout done the game result",
		Run:   blackwhiteTimeoutDone,
	}
	addBlackwhiteTimeoutDonelags(cmd)
	return cmd
}

func addBlackwhiteTimeoutDonelags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.MarkFlagRequired("gameID")
}

func blackwhiteTimeoutDone(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")

	params := &types.BlackwhiteTimeoutDone{
		GameID: gameID,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhiteTimeoutDoneTx", params, &res)
	ctx.RunWithoutMarshal()
}

func ShowBlackwhiteInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showInfo",
		Short: "show black white round info",
		Run:   showBlackwhiteInfo,
	}
	addshowBlackwhiteInfoflags(cmd)
	return cmd
}

func addshowBlackwhiteInfoflags(cmd *cobra.Command) {
	cmd.Flags().Uint32P("type", "t", 0, "type")
	cmd.MarkFlagRequired("type")

	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.Flags().Uint32P("status", "s", 0, "status")
	cmd.Flags().StringP("addr", "a", "", "addr")
	cmd.Flags().Uint32P("loopSeq", "l", 0, "loopSeq")


}

func showBlackwhiteInfo(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")
	typ, _ := cmd.Flags().GetUint32("type")
	status, _ := cmd.Flags().GetUint32("status")
	addr, _ := cmd.Flags().GetString("addr")
	loopSeq, _ := cmd.Flags().GetUint32("loopSeq")

	var params jsonrpc.Query4Cli

	var  rep interface{}

	params.Execer = types.BlackwhiteX
	if 0 == typ {
		req := types.ReqBlackwhiteRoundInfo{
			GameID: gameID,
		}
		params.FuncName = bw.GetBlackwhiteRoundInfo
		params.Payload  = req
		rep = &types.ReplyBlackwhiteRoundInfo {}
	}else if 1 == typ {
		req := types.ReqBlackwhiteRoundList{
			Status: int32(status),
			Address: addr,
		}
		params.FuncName = bw.GetBlackwhiteByStatusAndAddr
		params.Payload  = req
		rep = &types.ReplyBlackwhiteRoundList {}
	}else if 2 == typ {
		req := types.ReqLoopResult{
			GameID: gameID,
			LoopSeq: int32(loopSeq),
		}
		params.FuncName = bw.GetBlackwhiteloopResult
		params.Payload  = req
		rep = &types.ReplyLoopResults {}
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, rep)
	//ctx.SetResultCb(parseBlackwhiteRound)
	ctx.Run()
}
