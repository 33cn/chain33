package types

import (
	"encoding/json"
	"gitlab.33.cn/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var pblog = log.New("module", PokerBullX)

func init() {
	name := PokerBullX
	// init executor type
	types.RegistorExecutor(name, &PokerBullType{})
}

type PokerBullType struct {
	types.ExecTypeBase
}

func NewType() *PokerBullType {
	c := &PokerBullType{}
	c.SetChild(c)
	return c
}

func (b *PokerBullType) GetPayload() types.Message {
	return &PBGameAction{}
}

func (b *PokerBullType) GetName() string {
	return PokerBullX
}

func (b *PokerBullType) GetLogMap() map[int64]*types.LogInfo {
	return logInfo
}

func (b *PokerBullType) GetTypeMap() map[string]int32 {
	return actionName
}

func (m PokerBullType) ActionName(tx *types.Transaction) string {
	var g PBGameAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-Pokerbull-action-err"
	}
	if g.Ty == PBGameActionStart && g.GetStart() != nil {
		return "PokerbullStart"
	} else if g.Ty == PBGameActionContinue && g.GetContinue() != nil {
		return "PokerbullContinue"
	} else if g.Ty == PBGameActionQuit && g.GetQuit() != nil {
		return "PokerbullQuit"
	} else if g.Ty == PBGameActionQuery && g.GetQuery() != nil {
		return "PokerbullQuery"
	}
	return "unkown"
}

func (m PokerBullType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action PBGameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (m PokerBullType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (m PokerBullType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	pblog.Debug("Pokerbull.CreateTx", "action", action)
	var tx *types.Transaction
	return tx, nil
}
