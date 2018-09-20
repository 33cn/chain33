package executor

import (
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (g *Game) Exec_Create(payload *gt.GameCreate, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(g, tx, index)
	return action.GameCreate(payload)
}

func (g *Game) Exec_Cancel(payload *gt.GameCancel, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(g, tx, index)
	return action.GameCancel(payload)
}

func (g *Game) Exec_Close(payload *gt.GameClose, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(g, tx, index)
	return action.GameClose(payload)
}

func (g *Game) Exec_Match(payload *gt.GameMatch, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(g, tx, index)
	return action.GameMatch(payload)
}
