package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
	"reflect"
)

var logger = log.New("module", "execs.pokerbull")

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = pkt.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&PokerBull{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	drivers.Register(newPBGame().GetName(), newPBGame, 0)
}

type PokerBull struct {
	drivers.DriverBase
}

func newPBGame() drivers.Driver {
	t := &PokerBull{}
	t.SetChild(t)
	t.SetExecutorType(executorType)
	return t
}

func GetName() string {
	return newPBGame().GetName()
}

func (g *PokerBull) GetDriverName() string {
	return pkt.PokerBullX
}

func (g *PokerBull) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func (g *PokerBull) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func calcPBGameStatusKey(status int32, player int32, index int64) []byte {
	key := fmt.Sprintf("PBgame-status:%d:%d:%d", status, player, index)
	return []byte(key)
}

func calcPBGameStatusAndPlayerPrefix(status int32, player int32) []byte {
	key := fmt.Sprintf("PBgame-status:%d:%d:", status, player)
	return []byte(key)
}

func addPBGameStatus(status int32, player int32, index int64, gameId string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, player, index)
	record := &pkt.PBGameRecord{
		GameId: gameId,
	}
	kv.Value = types.Encode(record)
	return kv
}

func delPBGameStatus(status int32, player int32, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, player, index)
	kv.Value = nil
	return kv
}
