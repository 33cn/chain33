package types

import "gitlab.33.cn/chain33/chain33/types"

// status
const (
	BlackwhiteStatusCreate = iota + 1
	BlackwhiteStatusPlay
	BlackwhiteStatusShow
	BlackwhiteStatusTimeout
	BlackwhiteStatusDone
)

const (
	BlackwhiteCreateTx           = "BlackwhiteCreateTx"
	BlackwhitePlayTx             = "BlackwhitePlayTx"
	BlackwhiteShowTx             = "BlackwhiteShowTx"
	BlackwhiteTimeoutDoneTx      = "BlackwhiteTimeoutDoneTx"
	GetBlackwhiteRoundInfo       = "GetBlackwhiteRoundInfo"
	GetBlackwhiteByStatusAndAddr = "GetBlackwhiteByStatusAndAddr"
	GetBlackwhiteloopResult      = "GetBlackwhiteloopResult"
)

// blackwhite action type
const (
	BlackwhiteActionCreate = iota
	BlackwhiteActionPlay
	BlackwhiteActionShow
	BlackwhiteActionTimeoutDone
)

var (
	BlackwhiteX      = "blackwhite"
	GRPCName         = "chain33.blackwhite"
	JRPCName         = "Chain33"
	ExecerBlackwhite = []byte(BlackwhiteX)
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerBlackwhite)
}
