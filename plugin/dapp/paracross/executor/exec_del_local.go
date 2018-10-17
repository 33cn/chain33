package executor

import (
	"gitlab.33.cn/chain33/chain33/types"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
)

func (e *Paracross) ExecDelLocal_Commit(payload *pt.ParacrossCommitAction, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.Receipt, error) {
	var set types.Receipt
	for _, log := range receiptData.Logs {
		if log.Ty == pt.TyLogParacrossCommit { //} || log.Ty == types.TyLogParacrossCommitRecord {
			var g pt.ReceiptParacrossCommit
			types.Decode(log.Log, &g)

			var r pt.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), nil})
		} else if log.Ty == pt.TyLogParacrossCommitDone {
			var g pt.ReceiptParacrossDone
			types.Decode(log.Log, &g)
			g.Height = g.Height - 1

			key := calcLocalTitleKey(g.Title)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			key = calcTitleHeightKey(g.Title, g.Height)
			set.KV = append(set.KV, &types.KeyValue{key, nil})

			r, err := e.saveLocalParaTxs(tx, true)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, r.KV...)
		} else if log.Ty == pt.TyLogParacrossCommitRecord {
			var g pt.ReceiptParacrossRecord
			types.Decode(log.Log, &g)

			var r pt.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), nil})
		}
	}
	return &set, nil
}

func (e *Paracross) ExecDelLocal_AssetTransfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.Receipt, error) {
	var set types.Receipt

	//  主链转出记录，
	//  转入在 commit done 时记录， 因为没有日志里没有当时tx信息
	r, err := e.initLocalAssetTransfer(tx, true, true)
	if err != nil {
		return nil, err
	}
	set.KV = append(set.KV, r)

	return &set, nil
}

func (e *Paracross) ExecDelLocal_AssetWithdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.Receipt, error) {
	return nil, nil
}

func (e *Paracross) ExecDelLocal_Miner(payload *pt.ParacrossMinerAction, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.Receipt, error) {
	if index != 0 {
		return nil, types.ErrParaMinerBaseIndex
	}

	var set types.Receipt
	set.KV = append(set.KV, &types.KeyValue{pt.CalcMinerHeightKey(payload.Status.Title, payload.Status.Height), nil})

	return nil, nil
}

func (e *Paracross) ExecDelLocal_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.Receipt, error) {
	return nil, nil
}

func (e *Paracross) ExecDelLocal_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.Receipt, error) {
	return nil, nil
}

func (e *Paracross) ExecDelLocal_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.Receipt, error) {
	return nil, nil
}
