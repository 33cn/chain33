package executor

import (
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (g *PokerBull) rollbackIndex(log *pkt.ReceiptPBGame) (kvs []*types.KeyValue) {
	kvs = append(kvs, delPBGameStatusAndPlayer(log.Status, log.PlayerNum, log.Value, log.Index))
	kvs = append(kvs, addPBGameStatusAndPlayer(log.PreStatus, log.PlayerNum, log.PrevIndex, log.Value, log.GameId))
	kvs = append(kvs, delPBGameStatusIndexKey(log.Status, log.Index))
	kvs = append(kvs, addPBGameStatusIndexKey(log.PreStatus, log.GameId, log.PrevIndex))

	for _, v := range log.Players {
		kvs = append(kvs, delPBGameAddrIndexKey(v, log.Index))
		kvs = append(kvs, addPBGameAddrIndexKey(log.PreStatus, v, log.GameId, log.PrevIndex))
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
