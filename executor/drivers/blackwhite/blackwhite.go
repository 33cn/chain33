package blackwhite

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
	bw "gitlab.33.cn/chain33/chain33/types/executor/blackwhite"
)

var clog = log.New("module", "execs.blackwhite")

var blackwhiteAddr = address.ExecAddress(types.BlackwhiteX)

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
	} else if payload.Ty == types.BlackwhiteActionPlay && payload.GetPlay() != nil {
		action := newAction(c, tx)
		return action.Play(payload.GetPlay())
	} else if payload.Ty == types.BlackwhiteActionShow && payload.GetShow() != nil {
		action := newAction(c, tx)
		return action.Show(payload.GetShow())
	} else if payload.Ty == types.BlackwhiteActionTimeoutDone && payload.GetTimeoutDone() != nil {
		action := newAction(c, tx)
		return action.TimeoutDone(payload.GetTimeoutDone())
	}

	return nil, types.ErrActionNotSupport
}

//func (c *Blackwhite) updateInfo(receipt *types.ReceiptBlackwhite, isDel bool) []*types.KeyValue {
//	gameId := receipt.Round.GameID
//	addr := receipt.Round.CreateAddr
//
//	key := calcRoundKey(gameId)
//	value := types.Encode(receipt)
//
//	key1 := calcRoundKey4StatusAndAddr(addr, gameId)
//	value1 := key
//
//	var kv []*types.KeyValue
//	if isDel { //回退删除
//		clog.Info("****************test delete key", "gameId is ", gameId)
//		kv = append(kv, &types.KeyValue{key, nil})
//		kv = append(kv, &types.KeyValue{key1, nil})
//	} else {
//		clog.Info("****************test add key", "gameId is ", gameId)
//		kv = append(kv, &types.KeyValue{key, value})
//		kv = append(kv, &types.KeyValue{key1, value1})
//	}
//
//	return kv
//}

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
			types.TyLogBlackwhitePlay,
			types.TyLogBlackwhiteShow,
			types.TyLogBlackwhiteTimeoutDone,
			types.TyLogBlackwhiteDone:

			var receipt types.ReceiptBlackwhiteStatus
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := c.saveGame(&receipt)
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
		case types.TyLogBlackwhiteCreate:
			{
				var receipt types.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				kv := c.delGame(&receipt)
				set.KV = append(set.KV, kv...)
				break
			}
		case types.TyLogBlackwhitePlay:
		case types.TyLogBlackwhiteShow:
		case types.TyLogBlackwhiteTimeoutDone:
		case types.TyLogBlackwhiteDone:
			{
				var receipt types.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				//这三种Action才有上个prevStatus,回滚即可
				prevLog := &types.ReceiptBlackwhiteStatus{
					GameID: receipt.GetGameID(),
					Status: receipt.GetPrevStatus(),
					Addr:   receipt.GetAddr(),
				}
				//状态数据库由于默克尔树特性，之前生成的索引无效，故不需要回滚，只回滚localDB
				kv := c.saveGame(prevLog)
				set.KV = append(set.KV, kv...)

				kv = c.delGame(&receipt)
				set.KV = append(set.KV, kv...)
				break
			}
		default:

		}
	}
	return set, nil
}

func (c *Blackwhite) saveGame(gamelog *types.ReceiptBlackwhiteStatus) (kvs []*types.KeyValue) {
	kvs = append(kvs, addGame(gamelog.Status, gamelog.Addr, gamelog.GameID))
	if gamelog.GetPrevStatus() >= 0 {
		kvs = append(kvs, delGame(gamelog.GetPrevStatus(), gamelog.GetAddr(), gamelog.GameID))
	}
	return kvs
}

func (c *Blackwhite) delGame(gamelog *types.ReceiptBlackwhiteStatus) (kvs []*types.KeyValue) {
	kvs = append(kvs, delGame(gamelog.Status, gamelog.Addr, gamelog.GameID))
	return kvs
}


func (c *Blackwhite) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == bw.GetBlackwhiteRoundInfo {
		var in types.ReqBlackwhiteRoundInfo
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetBlackwhiteRoundInfo(&in)
	}else if funcName == bw.GetBlackwhiteByStatusAndAddr {
		var q types.ReqBlackwhiteRoundList
		err := types.Decode(params, &q)
		if err != nil {
			return nil, err
		}
		return c.GetBwRoundListInfo(&q)
	}
	return nil, types.ErrActionNotSupport
}

func (c *Blackwhite) GetBlackwhiteRoundInfo(req *types.ReqBlackwhiteRoundInfo) (types.Message, error) {
	gameId := req.GameID
	key := calcRoundKey(gameId)
	values, err := c.GetStateDB().Get(key)
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

func (c *Blackwhite) GetBwRoundListInfo(req *types.ReqBlackwhiteRoundList) (types.Message, error) {
	localDb := c.GetLocalDB()
	values, err := localDb.List(calcRoundKey4StatusAddrPrefix(req.GetStatus(), req.GetAddress()), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return &types.ReplyGameList{}, nil
	}

	storeDb := c.GetStateDB()
	var rep types.ReplyBlackwhiteRoundList
	for _, value := range values {
		v, err := storeDb.Get(calcRoundKey(string(value)))
		if nil != err {
			return nil, err
		}

		var round types.BlackwhiteRound
		err = types.Decode(v, &round)
		if err != nil {
			return nil, err
		}
		rep.Round = append(rep.Round, &round)
	}
	return &rep, nil
}

func addGame(status int32, addr, gameId string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4StatusAddr(status, addr, gameId)
	kv.Value = []byte(gameId)
	return kv
}

func delGame(status int32, addr, gameId string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4StatusAddr(status, addr, gameId)
	kv.Value = nil
	return kv
}