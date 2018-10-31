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
	key := fmt.Sprintf("LODB-pokerbull-addr:%s:", addr)
	return []byte(key)
}

func calcPBGameAddrKey(addr string, index int64) []byte {
	key := fmt.Sprintf("LODB-pokerbull-addr:%s:%018d", addr, index)
	return []byte(key)
}

func calcPBGameStatusPrefix(status int32) []byte {
	key := fmt.Sprintf("LODB-pokerbull-status-index:%d:", status)
	return []byte(key)
}

func calcPBGameStatusKey(status int32, index int64) []byte {
	key := fmt.Sprintf("LODB-pokerbull-status-index:%d:%018d", status, index)
	return []byte(key)
}

func calcPBGameStatusAndPlayerKey(status, player int32, value, index int64) []byte {
	key := fmt.Sprintf("LODB-pokerbull-status:%d:%d:%d:%018d", status, player, value, index)
	return []byte(key)
}

func calcPBGameStatusAndPlayerPrefix(status, player int32, value int64) []byte {
	var key string
	if value == 0 {
		key = fmt.Sprintf("LODB-pokerbull-status:%d:%d:", status, player)
	} else {
		key = fmt.Sprintf("LODB-pokerbull-status:%d:%d:%d", status, player, value)
	}

	return []byte(key)
}

func addPBGameStatusIndexKey(status int32, gameID string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, index)
	record := &pkt.PBGameIndexRecord{
		GameId: gameID,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}

func delPBGameStatusIndexKey(status int32, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, index)
	kv.Value = nil
	return kv
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

func addPBGameStatusAndPlayer(status int32, player int32, value, index int64, gameId string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusAndPlayerKey(status, player, value, index)
	record := &pkt.PBGameIndexRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}

func delPBGameStatusAndPlayer(status int32, player int32, value, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusAndPlayerKey(status, player, value, index)
	kv.Value = nil
	return kv
}
