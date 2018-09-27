package types

import (
	"reflect"

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
	// log for blackwhite game
	TyLogBlackwhiteCreate   = 750
	TyLogBlackwhitePlay     = 751
	TyLogBlackwhiteShow     = 752
	TyLogBlackwhiteTimeout  = 753
	TyLogBlackwhiteDone     = 754
	TyLogBlackwhiteLoopInfo = 755
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

	logInfo = map[int64]*types.LogInfo{
		TyLogBlackwhiteCreate:   &types.LogInfo{reflect.TypeOf(ReceiptBlackwhite{}), "LogBlackwhiteCreate"},
		TyLogBlackwhitePlay:     &types.LogInfo{reflect.TypeOf(ReceiptBlackwhite{}), "LogBlackwhitePlay"},
		TyLogBlackwhiteShow:     &types.LogInfo{reflect.TypeOf(ReceiptBlackwhite{}), "LogBlackwhiteShow"},
		TyLogBlackwhiteTimeout:  &types.LogInfo{reflect.TypeOf(ReceiptBlackwhite{}), "LogBlackwhiteTimeout"},
		TyLogBlackwhiteDone:     &types.LogInfo{reflect.TypeOf(ReceiptBlackwhite{}), "LogBlackwhiteDone"},
		TyLogBlackwhiteLoopInfo: &types.LogInfo{reflect.TypeOf(ReplyLoopResults{}), "LogBlackwhiteLoopInfo"},
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerBlackwhite)
}
