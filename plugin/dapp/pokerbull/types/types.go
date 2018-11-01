package types

import (
	"reflect"

	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	// init executor type
	types.RegistorExecutor(PokerBullX, NewType())
	types.AllowUserExec = append(types.AllowUserExec, ExecerPokerBull)
	types.RegisterDappFork(PokerBullX, "Enable", 0)
}

// exec
type PokerBullType struct {
	types.ExecTypeBase
}

func NewType() *PokerBullType {
	c := &PokerBullType{}
	c.SetChild(c)
	return c
}

func (t *PokerBullType) GetPayload() types.Message {
	return &PBGameAction{}
}

func (t *PokerBullType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Start":    PBGameActionStart,
		"Continue": PBGameActionContinue,
		"Quit":     PBGameActionQuit,
		"Query":    PBGameActionQuery,
	}
}

func (t *PokerBullType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{
		TyLogPBGameStart:    {reflect.TypeOf(ReceiptPBGame{}), "TyLogPBGameStart"},
		TyLogPBGameContinue: {reflect.TypeOf(ReceiptPBGame{}), "TyLogPBGameContinue"},
		TyLogPBGameQuit:     {reflect.TypeOf(ReceiptPBGame{}), "TyLogPBGameQuit"},
		TyLogPBGameQuery:    {reflect.TypeOf(ReceiptPBGame{}), "TyLogPBGameQuery"},
	}
}
