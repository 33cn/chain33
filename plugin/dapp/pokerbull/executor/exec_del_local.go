package executor

import (
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (g *PokerBull) rollbackIndex(log *pkt.ReceiptPBGame) (kvs []*types.KeyValue) {
	kvs = append(kvs, delPBGameStatus(log.Status, log.PlayerNum, log.Value, log.Index))

	if log.Status == pkt.PBGameActionContinue {
		kvs = append(kvs, addPBGameStatus(log.Status, log.PlayerNum, log.PrevIndex, log.Value, log.GameId))
	}

	if log.Status == pkt.PBGameActionQuit {
		kvs = append(kvs, addPBGameStatus(pkt.PBGameActionContinue, log.PlayerNum, log.PrevIndex, log.Value, log.GameId))
	}
	return kvs
}

func (g *PokerBull) execDelLocal(receiptData *types.ReceiptData) (*types.LocalDBSet, error) {
	dbSet := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		return dbSet, nil
	}
	for _, log := range receiptData.Logs {
		switch log.GetTy() {
		case pkt.TyLogPBGameStart, pkt.TyLogPBGameContinue, pkt.TyLogPBGameQuit:
			receiptGame := &pkt.ReceiptPBGame{}
			if err := types.Decode(log.Log, receiptGame); err != nil {
				return nil, err
			}
			kv := g.rollbackIndex(receiptGame)
			dbSet.KV = append(dbSet.KV, kv...)
		}
	}
	return dbSet, nil
}

func (g *PokerBull) ExecDelLocal_Start(payload *pkt.PBGameStart, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return g.execDelLocal(receiptData)
}

func (g *PokerBull) ExecDelLocal_Continue(payload *pkt.PBGameContinue, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return g.execDelLocal(receiptData)
}

func (g *PokerBull) ExecDelLocal_Quit(payload *pkt.PBGameQuit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return g.execDelLocal(receiptData)
}
