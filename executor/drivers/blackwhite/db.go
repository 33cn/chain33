package blackwhite

import (
	"fmt"
	"gitlab.33.cn/chain33/chain33/types"
	gt "gitlab.33.cn/chain33/chain33/types/executor/blackwhite"
)

var (
	roundPrefix      string
	loopResultPrefix string
)

func setReciptPrefix() {
	roundPrefix = "mavl-" + types.ExecName("blackwhite") + "-"
	loopResultPrefix = types.ExecName("blackwhite") + "-loop-"
}

func calcRoundKey(ID string) []byte {
	return []byte(fmt.Sprintf(roundPrefix+"%s", ID))
}

func calcRoundKey4StatusAddr(status int32, addr, ID string) []byte {
	return []byte(fmt.Sprintf(roundPrefix+"%d-","%s-"+"%s", status, addr, ID))
}

func calcRoundKey4StatusAddrPrefix(status int32, addr string) []byte {
	return []byte(fmt.Sprintf(roundPrefix+"%d-","%s-", status, addr))
}

func calcRoundKey4LoopResult(ID string) []byte {
	return []byte(fmt.Sprintf(loopResultPrefix+"%s", ID))
}

func newRound(create *types.BlackwhiteCreate, creator string) *types.BlackwhiteRound {
	t := &types.BlackwhiteRound{}

	t.Status = gt.BlackwhiteStatusCreate
	t.PlayAmount = create.PlayAmount
	t.PlayerCount = create.PlayerCount
	t.Timeout = create.Timeout
	t.Loop = calcloopNumByPlayer(create.PlayerCount)
	t.CreateAddr = creator
	t.GameName = create.GameName
	return t
}
