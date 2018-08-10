package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
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
		BlackwhiteCancelRawTxCmd(),
		BlackwhitePlayRawTxCmd(),
		BlackwhiteTimeoutDoneTxCmd(),
		ShowBlackwhiteRoundCmd(),
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
	cmd.Flags().Uint64P("playAmount", "m", 0, "max frozen amount")
	cmd.MarkFlagRequired("playAmount")

	cmd.Flags().Uint32P("playerCount", "p", 0, "player count")
	cmd.MarkFlagRequired("playerCount")

	cmd.Flags().Float64P("timeout", "t", 0, "timeout(s),default:120s")

	cmd.Flags().StringP("gameName", "g", "", "game name")
}

func blackwhiteCreate(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount, _ := cmd.Flags().GetUint64("playAmount")
	playerCount, _ := cmd.Flags().GetUint32("playerCount")
	timeout, _ := cmd.Flags().GetFloat64("timeout")
	gameName, _ := cmd.Flags().GetString("gameName")

	amountInt64 := int64(amount)

	if timeout == 0 {
		timeout = 120
	}

	params := &types.BlackwhiteCreate{
		PlayAmount:  amountInt64 * types.Coin,
		PlayerCount: int32(playerCount),
		Timeout:     int64(timeout),
		GameName:    gameName,
	}

	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhiteCreateTx", params, &res)
	ctx.RunWithoutMarshal()
}

func BlackwhiteCancelRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a no start blackwhite game",
		Run:   blackwhiteCancel,
	}
	addBlackwhiteCancelFlags(cmd)
	return cmd
}

func addBlackwhiteCancelFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.MarkFlagRequired("gameID")
}

func blackwhiteCancel(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")

	params := &types.BlackwhiteCancel{
		GameID: gameID,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhiteCancelTx", params, &res)
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

	cmd.Flags().Uint64P("amount", "m", 0, "frozen amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("isBlackStr", "i", "", "[0-1-1-1-1-1-0-0-1-1] (0:black,1:white,once round need 10 time)")
	cmd.MarkFlagRequired("isBlackStr")

}

func blackwhitePlay(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")
	amount, _ := cmd.Flags().GetUint64("amount")
	isBlackStr, _ := cmd.Flags().GetString("isBlackStr")

	blacks := strings.Split(isBlackStr, "-")
	var isBlack []bool
	for _, black := range blacks {
		if black == "0" {
			isBlack = append(isBlack, false)
		} else {
			isBlack = append(isBlack, true)
		}
	}

	amountInt64 := int64(amount)

	params := &types.BlackwhitePlay{
		GameID:  gameID,
		Amount:  amountInt64 * types.Coin,
		IsBlack: isBlack,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhitePlayTx", params, &res)
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

	params := &types.BlackwhiteCancel{
		GameID: gameID,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.BlackwhiteTimeoutDoneTx", params, &res)
	ctx.RunWithoutMarshal()
}

func ShowBlackwhiteRoundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showInfo",
		Short: "show black white round info",
		Run:   showBlackwhiteRound,
	}
	addshowBlackwhiteRoundlags(cmd)
	return cmd
}

func addshowBlackwhiteRoundlags(cmd *cobra.Command) {
	cmd.Flags().StringP("gameID", "g", "", "game ID")
	cmd.MarkFlagRequired("gameID")
}

func showBlackwhiteRound(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	gameID, _ := cmd.Flags().GetString("gameID")

	var reqRoundInfo types.ReqBlackwhiteRoundInfo
	reqRoundInfo.GameID = gameID

	params := jsonrpc.Query4Cli{
		Execer:   types.BlackwhiteX,
		FuncName: bw.GetBlackwhiteRoundInfo,
		Payload:  reqRoundInfo,
	}

	var res types.ReplyBlackwhiteRoundInfo
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	//ctx.SetResultCb(parseBlackwhiteRound)
	ctx.Run()
}
