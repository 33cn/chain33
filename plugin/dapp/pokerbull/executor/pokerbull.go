package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var logger = log.New("module", "execs.pokerbull")

func Init(name string, sub []byte) {
	drivers.Register(newPBGame().GetName(), newPBGame, types.GetDappFork(driverName, "Enable"))
}

var driverName = pkt.PokerBullX

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&PokerBull{}))
}

type PokerBull struct {
	drivers.DriverBase
}

func newPBGame() drivers.Driver {
	t := &PokerBull{}
	t.SetChild(t)
	t.SetExecutorType(types.LoadExecutorType(driverName))
	return t
}

func GetName() string {
	return newPBGame().GetName()
}

func (g *PokerBull) GetDriverName() string {
	return pkt.PokerBullX
}

func calcPBGameAddrPrefix(addr string) []byte {
	key := fmt.Sprintf("LODB-game-addr:%s:", addr)
	return []byte(key)
}

func calcPBGameAddrKey(addr string, index int64) []byte {
	key := fmt.Sprintf("LODB-game-addr:%s:%d", addr, index)
	return []byte(key)
}

func calcPBGameStatusKey(status, player int32, value, index int64) []byte {
	key := fmt.Sprintf("LODB-game-status:%d:%d:%d:%d", status, player, value, index)
	return []byte(key)
}

func calcPBGameStatusAndPlayerPrefix(status, player int32, value int64) []byte {
	var key string
	if value == 0 {
		key = fmt.Sprintf("LODB-game-status:%d:%d:", status, player)
	} else {
		key = fmt.Sprintf("LODB-game-status:%d:%d:%d", status, player, value)
	}

	return []byte(key)
}

func addPBGameAddrIndexKey(status int32, addr, gameID string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameAddrKey(addr, index)
	record := &pkt.PBGameRecord{
		GameId: gameID,
		Status: status,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}

func delPBGameAddrIndexKey(addr string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameAddrKey(addr, index)
	kv.Value = nil
	return kv
}

func addPBGameStatus(status int32, player int32, value, index int64, gameId string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, player, value, index)
	record := &pkt.PBGameRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}

func delPBGameStatus(status int32, player int32, value, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, player, value, index)
	kv.Value = nil
	return kv
}
