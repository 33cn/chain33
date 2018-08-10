package blackwhite

import (
	"fmt"
	"gitlab.33.cn/chain33/chain33/types"
	gt "gitlab.33.cn/chain33/chain33/types/executor/blackwhite"
)

var (
	roundPrefix         string
	guessingTimesPrefix string
)

func setReciptPrefix() {
	roundPrefix = "mavl-" + types.ExecName("blackwhite") + "-"
	guessingTimesPrefix = types.ExecName("blackwhite") + "-times-"
}

func calcRoundKey(ID string) []byte {
	return []byte(fmt.Sprintf(roundPrefix+"%s", ID))
}

func calcRoundKey4Addr(addr, ID string) []byte {
	return []byte(fmt.Sprintf(roundPrefix+"%s-"+"%s", addr, ID))
}

func newRound(create *types.BlackwhiteCreate, creator string) *types.BlackwhiteRound {
	t := &types.BlackwhiteRound{}

	t.Status = gt.BlackwhiteStatusReady
	t.PlayAmount = create.PlayAmount
	t.PlayerCount = create.PlayerCount
	t.CreateAddr = creator
	t.Timeout = create.Timeout
	t.GameName = create.GameName
	return t
}

//func (o *order) save(db dbm.KV, key, value []byte) {
//	db.Set([]byte(key), value)
//}
