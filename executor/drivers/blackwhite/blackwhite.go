package blackwhite

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
	bw "gitlab.33.cn/chain33/chain33/types/executor/blackwhite"
)

var clog = log.New("module", "execs.blackwhite")

type Blackwhite struct {
	drivers.DriverBase
}

func Init() {
	drivers.Register(newBlackwhite().GetName(), newBlackwhite, types.ForkV23Blackwhite)
	setReciptPrefix()
}

func newBlackwhite() drivers.Driver {
	c := &Blackwhite{}
	c.SetChild(c)
	return c
}

func (c *Blackwhite) GetName() string {
	return types.ExecName(types.BlackwhiteX)
}

func (c *Blackwhite) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := c.DriverBase.Exec(tx, index)
	if err != nil {
		return nil, err
	}
	var payload types.BlackwhiteAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}

	if payload.Ty == types.BlackwhiteActionCreate && payload.GetCreate() != nil {
		create := payload.GetCreate()
		action := newAction(c, tx)
		return action.Create(create)
	} else if payload.Ty == types.BlackwhiteActionCancel && payload.GetCancel() != nil {
		action := newAction(c, tx)
		cancel := payload.GetCancel()
		return action.Cancel(cancel)
	} else if payload.Ty == types.BlackwhiteActionPlay && payload.GetPlay() != nil {
		action := newAction(c, tx)
		return action.Play(payload.GetPlay())
	} else if payload.Ty == types.BlackwhiteActionTimeoutDone && payload.GetTimeoutDone() != nil {
		action := newAction(c, tx)
		return action.TimeoutDone(payload.GetTimeoutDone())
	}

	return nil, types.ErrActionNotSupport
}

func (c *Blackwhite) updateInfo(receipt *types.ReceiptBlackwhite, isDel bool) []*types.KeyValue {
	gameId := receipt.Round.GameID
	addr := receipt.Round.CreateAddr

	key := calcRoundKey(gameId)
	value := types.Encode(receipt)

	key1 := calcRoundKey4Addr(addr, gameId)
	value1 := key

	var kv []*types.KeyValue
	if isDel { //回退删除
		clog.Info("****************test delete key", "gameId is ", gameId)
		kv = append(kv, &types.KeyValue{key, nil})
		kv = append(kv, &types.KeyValue{key1, nil})
	} else {
		clog.Info("****************test add key", "gameId is ", gameId)
		kv = append(kv, &types.KeyValue{key, value})
		kv = append(kv, &types.KeyValue{key1, value1})
	}

	return kv
}

func (c *Blackwhite) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	//执行成功
	var payload types.BlackwhiteAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		switch log.Ty {
		case types.TyLogBlackwhiteCreate,
			types.TyLogBlackwhiteCancel,
			types.TyLogBlackwhitePlay,
			types.TyLogBlackwhiteTimeoutDone,
			types.TyLogBlackwhiteDone:
			var receipt types.ReceiptBlackwhite
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				return nil, err
			}
			kv := c.updateInfo(&receipt, false)
			set.KV = append(set.KV, kv...)
		default:

		}
	}
	return set, nil
}

func (c *Blackwhite) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var payload types.BlackwhiteAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		switch log.Ty {
		case types.TyLogBlackwhiteCreate,
			types.TyLogBlackwhiteCancel,
			types.TyLogBlackwhitePlay,
			types.TyLogBlackwhiteTimeoutDone,
			types.TyLogBlackwhiteDone:
			var receipt types.ReceiptBlackwhite
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				return nil, err
			}
			kv := c.updateInfo(&receipt, true)
			set.KV = append(set.KV, kv...)
		default:

		}
	}
	return set, nil
}

func (c *Blackwhite) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == bw.GetBlackwhiteRoundInfo {
		var in types.ReqBlackwhiteRoundInfo
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetBlackwhiteRoundInfo(&in)
	}
	return nil, types.ErrActionNotSupport
}

func (c *Blackwhite) GetBlackwhiteRoundInfo(req *types.ReqBlackwhiteRoundInfo) (types.Message, error) {
	gameId := req.GameID
	key := calcRoundKey(gameId)
	values, err := c.GetLocalDB().Get(key)
	if err != nil {
		return nil, err
	}

	var receipt types.ReceiptBlackwhite
	err = types.Decode(values, &receipt)
	if err != nil {
		return nil, err
	}

	return &receipt, nil
}
