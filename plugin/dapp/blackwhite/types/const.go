package types

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

const ()

var (
	BlackwhiteX      = "blackwhite"
	GRPCName         = "chain33.blackwhite"
	JRPCName         = "Chain33"
	ExecerBlackwhite = []byte(BlackwhiteX)
)
