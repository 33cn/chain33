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

const (
	BlackwhiteX     = "blackwhite"
)

var (
	ExecerBlackwhite = []byte(BlackwhiteX)
)