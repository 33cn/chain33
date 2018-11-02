package executor

import (
	"bytes"

	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

func (e *Paracross) ExecLocal_Commit(payload *pt.ParacrossCommitAction, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	for _, log := range receiptData.Logs {
		if log.Ty == pt.TyLogParacrossCommit {
			var g pt.ReceiptParacrossCommit
			types.Decode(log.Log, &g)

			var r pt.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), types.Encode(&r)})
		} else if log.Ty == pt.TyLogParacrossCommitDone {
			var g pt.ReceiptParacrossDone
			types.Decode(log.Log, &g)

			key := calcLocalTitleKey(g.Title)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			key = calcLocalHeightKey(g.Title, g.Height)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			r, err := e.saveLocalParaTxs(tx, false)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, r.KV...)
		} else if log.Ty == pt.TyLogParacrossCommitRecord {
			var g pt.ReceiptParacrossRecord
			types.Decode(log.Log, &g)

			var r pt.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), types.Encode(&r)})
		}
	}
	return &set, nil
}

func (e *Paracross) ExecLocal_AssetTransfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet

	//  主链转出记录，
	//  转入在 commit done 时记录， 因为没有日志里没有当时tx信息
	r, err := e.initLocalAssetTransfer(tx, true, false)
	if err != nil {
		return nil, err
	}
	set.KV = append(set.KV, r)

	return &set, nil
}

func (e *Paracross) ExecLocal_AssetWithdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}

func (e *Paracross) ExecLocal_Miner(payload *pt.ParacrossMinerAction, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	if index != 0 {
		return nil, pt.ErrParaMinerBaseIndex
	}

	var set types.LocalDBSet

	var mixTxHashs, paraTxHashs, crossTxHashs [][]byte
	txs := e.GetTxs()
	//remove the 0 vote tx
	for i := 1; i < len(txs); i++ {
		tx := txs[i]
		hash := tx.Hash()
		mixTxHashs = append(mixTxHashs, hash)
		//跨链交易包含了主链交易，需要过滤出来
		if types.IsParaExecName(string(tx.Execer)) {
			paraTxHashs = append(paraTxHashs, hash)
		}
	}
	for i := 1; i < len(txs); i++ {
		tx := txs[i]
		if tx.GroupCount >= 2 {
			crossTxs, end := crossTxGroupProc(txs, i)
			for _, crossTx := range crossTxs {
				crossTxHashs = append(crossTxHashs, crossTx.Hash())
			}
			i = int(end) - 1
			continue
		}
		if types.IsParaExecName(string(tx.Execer)) &&
			bytes.HasSuffix(tx.Execer, []byte(pt.ParaX)) {
			crossTxHashs = append(crossTxHashs, tx.Hash())
		}
	}

	payload.Status.TxHashs = paraTxHashs
	payload.Status.TxResult = util.CalcSubBitMap(mixTxHashs, paraTxHashs, e.GetReceipt()[1:])
	payload.Status.CrossTxHashs = crossTxHashs
	payload.Status.CrossTxResult = util.CalcSubBitMap(mixTxHashs, crossTxHashs, e.GetReceipt()[1:])

	set.KV = append(set.KV, &types.KeyValue{
		pt.CalcMinerHeightKey(payload.Status.Title, payload.Status.Height),
		types.Encode(payload.Status)})

	return &set, nil
}

func (e *Paracross) ExecLocal_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}

func (e *Paracross) ExecLocal_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}

func (e *Paracross) ExecLocal_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}
