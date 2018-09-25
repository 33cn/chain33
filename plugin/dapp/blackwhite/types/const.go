package types

import (
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

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

var (
	BlackwhiteX = "blackwhite"
	glog        = log15.New("module", BlackwhiteX)
	//GRPCName         = "chain33.blackwhite"
	JRPCName         = "Blackwhite"
	ExecerBlackwhite = []byte(BlackwhiteX)

	actionName = map[string]int32{
		"Create":      BlackwhiteActionCreate,
		"Play":        BlackwhiteActionPlay,
		"Show":        BlackwhiteActionShow,
		"TimeoutDone": BlackwhiteActionTimeoutDone,
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerBlackwhite)
}
