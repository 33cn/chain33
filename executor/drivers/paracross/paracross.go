package paracross


import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
	pt "gitlab.33.cn/chain33/chain33/types/executor/paracross"
)

var clog = log.New("module", "execs.paracross")

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
	}

	return nil, types.ErrActionNotSupport
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
		} else if log.Ty == types.TyLogParacrossDode {
			var g types.ReceiptParacrossDone
			types.Decode(log.Log, &g)

			key := calcLocalTitleKey(g.Title)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			key = calcTitleHeightKey(g.Title, g.Height)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})
		} else if log.Ty == types.TyLogParacrossRecord {
			var g types.ReceiptParacrossRecord
			types.Decode(log.Log, &g)

			var r types.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), types.Encode(&r)})
		}
	}
	return set, nil
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
		if log.Ty == types.TyLogParacrossCommit { //} || log.Ty == types.TyLogParacrossRecord {
			var g types.ReceiptParacrossCommit
			types.Decode(log.Log, &g)

			var r types.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), nil})
		} else if log.Ty == types.TyLogParacrossDode {
			var g types.ReceiptParacrossDone
			types.Decode(log.Log, &g)
			g.Height = g.Height - 1

			key := calcLocalTitleKey(g.Title)
			set.KV = append(set.KV, &types.KeyValue{key, types.Encode(&g)})

			key = calcTitleHeightKey(g.Title, g.Height)
			set.KV = append(set.KV, &types.KeyValue{key, nil})
		} else if log.Ty == types.TyLogParacrossRecord {
			var g types.ReceiptParacrossRecord
			types.Decode(log.Log, &g)

			var r types.ParacrossTx
			r.TxHash = string(tx.Hash())
			set.KV = append(set.KV, &types.KeyValue{calcLocalTxKey(g.Status.Title, g.Status.Height, tx.From()), nil})
		}
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
	}
	return nil, types.ErrActionNotSupport
}
