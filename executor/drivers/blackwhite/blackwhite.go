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
	action := newAction(c, tx)
	if payload.Ty == types.BlackwhiteActionCreate && payload.GetCreate() != nil {
		return action.Create(payload.GetCreate())
	} else if payload.Ty == types.BlackwhiteActionPlay && payload.GetPlay() != nil {
		return action.Play(payload.GetPlay())
	} else if payload.Ty == types.BlackwhiteActionShow && payload.GetShow() != nil {
		return action.Show(payload.GetShow())
	} else if payload.Ty == types.BlackwhiteActionTimeoutDone && payload.GetTimeoutDone() != nil {
		return action.TimeoutDone(payload.GetTimeoutDone())
	}
	return nil, types.ErrActionNotSupport
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
			types.TyLogBlackwhitePlay,
			types.TyLogBlackwhiteShow,
			types.TyLogBlackwhiteTimeout,
			types.TyLogBlackwhiteDone:
			{
				var receipt types.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				kv := c.saveGame(&receipt)
				set.KV = append(set.KV, kv...)
				break
			}
		case types.TyLogBlackwhiteLoopInfo:
			{
				var res types.ReplyLoopResults
				err := types.Decode(log.Log, &res)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				kv := c.saveLoopResult(&res)
				set.KV = append(set.KV, kv...)
				break
			}
		default:
			break
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
		case types.TyLogBlackwhiteTimeout:
		case types.TyLogBlackwhiteDone:
			{
				var receipt types.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				//这几种Action才有上个prevStatus,回滚即可
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
		case types.TyLogBlackwhiteLoopInfo:
			{
				var res types.ReplyLoopResults
				err := types.Decode(log.Log, &res)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				kv := c.delLoopResult(&res)
				set.KV = append(set.KV, kv...)
				break
			}
		default:
			break
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

func (c *Blackwhite) saveLoopResult(res *types.ReplyLoopResults) (kvs []*types.KeyValue) {
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4LoopResult(res.GetGameID())
	kv.Value = types.Encode(res)
	kvs = append(kvs, kv)
	return kvs
}

func (c *Blackwhite) delLoopResult(res *types.ReplyLoopResults) (kvs []*types.KeyValue) {
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4LoopResult(res.GetGameID())
	kv.Value = nil
	kvs = append(kvs, kv)
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
	} else if funcName == bw.GetBlackwhiteByStatusAndAddr {
		var in types.ReqBlackwhiteRoundList
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetBwRoundListInfo(&in)
	} else if funcName == bw.GetBlackwhiteloopResult {
		var in types.ReqLoopResult
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetBwRoundLoopResult(&in)
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

	var round types.BlackwhiteRound
	err = types.Decode(values, &round)
	if err != nil {
		return nil, err
	}
	var receipt types.ReceiptBlackwhite
	receipt.Round = &round
	return &receipt, nil
}

func (c *Blackwhite) GetBwRoundListInfo(req *types.ReqBlackwhiteRoundList) (types.Message, error) {
	localDb := c.GetLocalDB()
	values, err := localDb.List(calcRoundKey4StatusAddrPrefix(req.GetStatus(), req.GetAddress()), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, types.ErrNotFound
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

func (c *Blackwhite) GetBwRoundLoopResult(req *types.ReqLoopResult) (types.Message, error) {
	localDb := c.GetLocalDB()
	values, err := localDb.Get(calcRoundKey4LoopResult(req.GameID))
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, types.ErrNotFound
	}

	var result types.ReplyLoopResults
	err = types.Decode(values, &result)
	if err != nil {
		return nil, err
	}

	if req.LoopSeq > 0 { //取出具体一轮
		if len(result.Results) < int(req.LoopSeq) {
			return nil, types.ErrNoLoopSeq
		}
		res := &types.ReplyLoopResults{
			GameID: result.GameID,
		}
		index := int(req.LoopSeq)
		perRes := &types.PerLoopResult{}
		perRes.Winers = append(perRes.Winers, res.Results[index-1].Winers...)
		perRes.Losers = append(perRes.Losers, res.Results[index-1].Losers...)
		res.Results = append(res.Results, perRes)
		return res, nil
	}
	return &result, nil //将所有轮数取出
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
