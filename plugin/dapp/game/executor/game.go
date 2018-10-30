package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", "execs.game")

var driverName = gt.GameX

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Game{}))
}

func Init(name string, sub []byte) {
	drivers.Register(GetName(), newGame, types.GetDappFork(driverName, "Enable"))
}

type Game struct {
	drivers.DriverBase
}

func newGame() drivers.Driver {
	t := &Game{}
	t.SetChild(t)
	t.SetExecutorType(types.LoadExecutorType(driverName))
	return t
}

func GetName() string {
	return newGame().GetName()
}

func (g *Game) GetDriverName() string {
	return driverName
}

//更新索引
func (g *Game) updateIndex(log *gt.ReceiptGame) (kvs []*types.KeyValue) {
	//先保存本次Action产生的索引
	kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.Addr, log.Index))
	kvs = append(kvs, addGameStatusIndex(log.Status, log.GameId, log.Index))
	if log.Status == gt.GameActionMatch {
		kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.CreateAddr, log.Index))
		kvs = append(kvs, delGameAddrIndex(gt.GameActionCreate, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, delGameStatusIndex(gt.GameActionCreate, log.PrevIndex))
	}
	if log.Status == gt.GameActionCancel {
		kvs = append(kvs, delGameAddrIndex(gt.GameActionCreate, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, delGameStatusIndex(gt.GameActionCreate, log.PrevIndex))
	}

	if log.Status == gt.GameActionClose {
		kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.MatchAddr, log.Index))
		kvs = append(kvs, delGameAddrIndex(gt.GameActionMatch, log.MatchAddr, log.PrevIndex))
		kvs = append(kvs, delGameAddrIndex(gt.GameActionMatch, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, delGameStatusIndex(gt.GameActionMatch, log.PrevIndex))
	}
	return kvs
}

//回滚索引
func (g *Game) rollbackIndex(log *gt.ReceiptGame) (kvs []*types.KeyValue) {
	//先删除本次Action产生的索引
	kvs = append(kvs, delGameAddrIndex(log.Status, log.Addr, log.Index))
	kvs = append(kvs, delGameStatusIndex(log.Status, log.Index))

	if log.Status == gt.GameActionMatch {
		kvs = append(kvs, delGameAddrIndex(log.Status, log.CreateAddr, log.Index))
		kvs = append(kvs, addGameAddrIndex(gt.GameActionCreate, log.GameId, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, addGameStatusIndex(gt.GameActionCreate, log.GameId, log.PrevIndex))
	}

	if log.Status == gt.GameActionCancel {
		kvs = append(kvs, addGameAddrIndex(gt.GameActionCreate, log.GameId, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, addGameStatusIndex(gt.GameActionCreate, log.GameId, log.PrevIndex))
	}

	if log.Status == gt.GameActionClose {
		kvs = append(kvs, delGameAddrIndex(log.Status, log.MatchAddr, log.Index))
		kvs = append(kvs, addGameAddrIndex(gt.GameActionMatch, log.GameId, log.MatchAddr, log.PrevIndex))
		kvs = append(kvs, addGameAddrIndex(gt.GameActionMatch, log.GameId, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, addGameStatusIndex(gt.GameActionMatch, log.GameId, log.PrevIndex))
	}
	return kvs
}

func calcGameStatusIndexKey(status int32, index int64) []byte {
	key := fmt.Sprintf("LODB-game-status:%d:%018d", status, index)
	return []byte(key)
}

func calcGameStatusIndexPrefix(status int32) []byte {
	key := fmt.Sprintf("LODB-game-status:%d:", status)
	return []byte(key)
}
func calcGameAddrIndexKey(status int32, addr string, index int64) []byte {
	key := fmt.Sprintf("LODB-game-addr:%d:%s:%018d", status, addr, index)
	return []byte(key)
}
func calcGameAddrIndexPrefix(status int32, addr string) []byte {
	key := fmt.Sprintf("LODB-game-addr:%d:%s:", status, addr)
	return []byte(key)
}
func addGameStatusIndex(status int32, gameId string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameStatusIndexKey(status, index)
	record := &gt.GameRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}
func addGameAddrIndex(status int32, gameId, addr string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameAddrIndexKey(status, addr, index)
	record := &gt.GameRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}
func delGameStatusIndex(status int32, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameStatusIndexKey(status, index)
	kv.Value = nil
	return kv
}
func delGameAddrIndex(status int32, addr string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameAddrIndexKey(status, addr, index)
	//value置nil,提交时，会自动执行删除操作
	kv.Value = nil
	return kv
}

type ReplyGameList struct {
	Games []*Game `json:"games"`
}

type ReplyGame struct {
	Game *Game `json:"game"`
}

func (c *Game) GetPayloadValue() types.Message {
	return &gt.GameAction{}
}

func (c *Game) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Create": gt.GameActionCreate,
		"Match":  gt.GameActionMatch,
		"Cancel": gt.GameActionCancel,
		"Close":  gt.GameActionClose,
	}
}
