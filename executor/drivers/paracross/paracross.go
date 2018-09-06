package paracross

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
	pt "gitlab.33.cn/chain33/chain33/types/executor/paracross"
	"gitlab.33.cn/chain33/chain33/util"
)

var (
	clog                    = log.New("module", "execs.paracross")
	enableParacrossTransfer = true
)

type Paracross struct {
	drivers.DriverBase
}

func Init() {
	drivers.Register(newParacross().GetName(), newParacross, 0)
	setPrefix()
}

func newParacross() drivers.Driver {
	c := &Paracross{}
	c.SetChild(c)
	return c
}

func (c *Paracross) GetName() string {
	return types.ExecName("paracross")
}

func (c *Paracross) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := c.DriverBase.Exec(tx, index)
	if err != nil {
		return nil, err
	}
	var payload types.ParacrossAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}

	if payload.Ty == pt.ParacrossActionCommit && payload.GetCommit() != nil {
		commit := payload.GetCommit()
		a := newAction(c, tx)
		return a.Commit(commit)
	} else if payload.Ty == pt.ParacrossActionTransfer && payload.GetAssetTransfer() != nil {
		_, err := c.checkTxGroup(tx, index)
		if err != nil {
			clog.Error("ParacrossActionTransfer", "get tx group failed", err, "hash", common.Bytes2Hex(tx.Hash()))
			return nil, err
		}
		a := newAction(c, tx)
		return a.AssetTransfer(payload.GetAssetTransfer())
	} else if payload.Ty == pt.ParacrossActionWithdraw && payload.GetAssetWithdraw() != nil {
		_, err := c.checkTxGroup(tx, index)
		if err != nil {
			clog.Error("ParacrossActionWithdraw", "get tx group failed", err, "hash", common.Bytes2Hex(tx.Hash()))
			return nil, err
		}
		a := newAction(c, tx)
		return a.AssetWithdraw(payload.GetAssetWithdraw())
	}

	return nil, types.ErrActionNotSupport
}

func (c *Paracross) checkTxGroup(tx *types.Transaction, index int) ([]*types.Transaction, error) {
	if tx.GroupCount >= 2 {
		txs, err := c.GetTxGroup(index)
		if err != nil {
			clog.Error("ParacrossActionTransfer", "get tx group failed", err, "hash", common.Bytes2Hex(tx.Hash()))
			return nil, err
		}
		return txs, nil
	}
	return nil, nil
}

func (c *Paracross) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var payload types.ParacrossAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		if log.Ty == types.TyLogParacrossCommit {
			var g types.ReceiptParacrossCommit
			types.Decode(log.Log, &g)

			var r types.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), types.Encode(&r)})
		} else if log.Ty == types.TyLogParacrossCommitDone {
			var g types.ReceiptParacrossDone
			types.Decode(log.Log, &g)

			key := calcLocalTitleKey(g.Title)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			key = calcTitleHeightKey(g.Title, g.Height)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			r, err := c.saveLocalParaTxs(tx, false)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, r.KV...)
		} else if log.Ty == types.TyLogParacrossCommitRecord {
			var g types.ReceiptParacrossRecord
			types.Decode(log.Log, &g)

			var r types.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), types.Encode(&r)})
		} else if log.Ty == types.TyLogParaAssetWithdraw {
			// TODO
		} else if log.Ty == types.TyLogParaAssetTransfer {
			// TODO
		} else if log.Ty == types.TyLogExecTransfer {
			//  主链转出记录，
			//  转入在 commit done 时几率， 因为没有日志里没有当时tx信息
			if tx.ActionName() == pt.ParacrossActionTransferStr {
				c.initLocalAssetTransfer(tx, true, false)

			} // else if tx.ActionName() == pt.ParacrossActionCommitStr {
			//
			//}

		} /* Token 暂时不支持 */
	}
	return set, nil
}

func (c *Paracross) saveLocalParaTxs(tx *types.Transaction, isDel bool) (*types.LocalDBSet, error) {
	var set types.LocalDBSet

	var payload types.ParacrossAction
	err := types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}
	if payload.Ty != pt.ParacrossActionCommit || payload.GetCommit() == nil {
		return nil, nil
	}

	commit := payload.GetCommit()
	for i := 0; i < len(commit.Status.CrossTxHashs); i++ {
		success := util.BitMapBit(commit.Status.CrossTxResult, uint32(i))

		paraTx, err := GetTx(c.GetApi(), commit.Status.CrossTxHashs[i])
		if err != nil {
			clog.Crit("paracross.Commit Load Tx failed", "para title", commit.Status.Title,
				"para height", commit.Status.Height, "para tx index", i, "error", err, "txHash",
				common.Bytes2Hex(commit.Status.CrossTxHashs[i]))
			return nil, err
		}

		var payload types.ParacrossAction
		err = types.Decode(paraTx.Tx.Payload, &payload)
		if err != nil {
			clog.Crit("paracross.Commit Decode Tx failed", "para title", commit.Status.Title,
				"para height", commit.Status.Height, "para tx index", i, "error", err, "txHash",
				common.Bytes2Hex(commit.Status.CrossTxHashs[i]))
			return nil, err
		}
		if payload.Ty == pt.ParacrossActionTransfer {
			kv, err := c.updateLocalAssetTransfer(tx, paraTx.Tx, success, isDel)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kv)
		} else if payload.Ty == pt.ParacrossActionWithdraw {
			kv, err := c.initLocalAssetWithdraw(tx, paraTx.Tx, true, success, isDel)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kv)
		}
	}

	return &set, nil
}

func getCommitHeight(payload []byte) (int64, error) {
	var a types.ParacrossAction
	err := types.Decode(payload, &a)
	if err != nil {
		return 0, err
	}
	if a.GetCommit() == nil {
		return 0, types.ErrInputPara
	}

	return a.GetCommit().Status.Height, nil
}

func (c *Paracross) initLocalAssetTransfer(tx *types.Transaction, success, isDel bool) (*types.KeyValue, error) {
	key := calcLocalAssetKey(tx.Hash())
	if isDel {
		c.GetLocalDB().Set(key, nil)
		return &types.KeyValue{key, nil}, nil
	}

	var asset types.ParacrossAsset

	amount, err := tx.Amount()
	if err != nil {
		return nil, err
	}

	asset = types.ParacrossAsset{
		From:       tx.From(),
		To:         tx.To,
		Amount:     amount,
		IsWithdraw: false,
		TxHash:     tx.Hash(),
		Height:     c.GetHeight(),
	}

	c.GetLocalDB().Set(key, types.Encode(&asset))
	return &types.KeyValue{key, types.Encode(&asset)}, nil
}

func (c *Paracross) initLocalAssetWithdraw(txCommit, tx *types.Transaction, isWithdraw, success, isDel bool) (*types.KeyValue, error) {
	key := calcLocalAssetKey(tx.Hash())
	if isDel {
		c.GetLocalDB().Set(key, nil)
		return &types.KeyValue{key, nil}, nil
	}

	var asset types.ParacrossAsset

	amount, err := tx.Amount()
	if err != nil {
		return nil, err
	}
	asset.ParaHeight, err = getCommitHeight(txCommit.Payload)
	if err != nil {
		return nil, err
	}
	asset = types.ParacrossAsset{
		From:       tx.From(),
		To:         tx.To,
		Amount:     amount,
		IsWithdraw: isWithdraw,
		TxHash:     tx.Hash(),
		Height:     c.GetHeight(),
	}

	asset.CommitDoneHeight = c.GetHeight()
	asset.Success = success

	c.GetLocalDB().Set(key, types.Encode(&asset))
	return &types.KeyValue{key, types.Encode(&asset)}, nil
}

func (c *Paracross) updateLocalAssetTransfer(txCommit, tx *types.Transaction, success, isDel bool) (*types.KeyValue, error) {
	key := calcLocalAssetKey(tx.Hash())

	var asset types.ParacrossAsset
	v, err := c.GetLocalDB().Get(key)
	if err != nil {
		return nil, err
	}
	err = types.Decode(v, &asset)
	if err != nil {
		panic(err)
	}
	if !isDel {
		asset.ParaHeight, err = getCommitHeight(txCommit.Payload)
		if err != nil {
			return nil, err
		}
		asset.CommitDoneHeight = c.GetHeight()
		asset.Success = success
	} else {
		asset.ParaHeight = 0
		asset.CommitDoneHeight = 0
		asset.Success = false
	}
	c.GetLocalDB().Set(key, types.Encode(&asset))
	return &types.KeyValue{key, types.Encode(&asset)}, nil
}

func (c *Paracross) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	//执行成功
	var payload types.ParacrossAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		if log.Ty == types.TyLogParacrossCommit { //} || log.Ty == types.TyLogParacrossCommitRecord {
			var g types.ReceiptParacrossCommit
			types.Decode(log.Log, &g)

			var r types.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), nil})
		} else if log.Ty == types.TyLogParacrossCommitDone {
			var g types.ReceiptParacrossDone
			types.Decode(log.Log, &g)
			g.Height = g.Height - 1

			key := calcLocalTitleKey(g.Title)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			key = calcTitleHeightKey(g.Title, g.Height)
			set.KV = append(set.KV, &types.KeyValue{key, nil})

			r, err := c.saveLocalParaTxs(tx, true)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, r.KV...)
		} else if log.Ty == types.TyLogParacrossCommitRecord {
			var g types.ReceiptParacrossRecord
			types.Decode(log.Log, &g)

			var r types.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), nil})
		} else if log.Ty == types.TyLogParaAssetWithdraw {
			// TODO
		} else if log.Ty == types.TyLogParaAssetTransfer {
			// TODO
		} else if log.Ty == types.TyLogExecTransfer {
			//  主链转出记录，
			//  转入在 commit done 时几率， 因为没有日志里没有当时tx信息
			if tx.ActionName() == pt.ParacrossActionTransferStr {
				c.initLocalAssetTransfer(tx, true, true)

			} // else if tx.ActionName() == pt.ParacrossActionCommitStr {
			//
			//}

		} /* Token 暂时不支持 */
	}
	return set, nil
}

func (c *Paracross) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "ParacrossGetTitle" {
		var in types.ReqStr
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.ParacrossGetHeight(in.ReqStr)
	} else if funcName == "ParacrossListTitles" {
		return c.ParacrossListTitles()
	} else if funcName == "ParacrossGetTitleHeight" {
		var in types.ReqParacrossTitleHeight
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.ParacrossGetTitleHeight(in.Title, in.Height)
	} else if funcName == "ParacrossGetAssetTxResult" {
		var in types.ReqHash
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.ParacrossGetAssetTxResult(in.Hash)
	}

	return nil, types.ErrActionNotSupport
}
