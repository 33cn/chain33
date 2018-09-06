package executor

import (
	"fmt"
	"math/rand"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.blackwhite")

var blackwhiteAddr = address.ExecAddress(gt.BlackwhiteX)

type Blackwhite struct {
	drivers.DriverBase
}

func NewBlackwhite() drivers.Driver {
	c := &Blackwhite{}
	c.SetChild(c)
	return c
}

func GetName() string {
	return types.ExecName(gt.BlackwhiteX)
}

func (c *Blackwhite) GetName() string {
	return GetName()
}

func (c *Blackwhite) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := c.DriverBase.Exec(tx, index)
	if err != nil {
		return nil, err
	}
	var payload gt.BlackwhiteAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}
	action := newAction(c, tx, int32(index))
	if payload.Ty == gt.BlackwhiteActionCreate && payload.GetCreate() != nil {
		return action.Create(payload.GetCreate())
	} else if payload.Ty == gt.BlackwhiteActionPlay && payload.GetPlay() != nil {
		return action.Play(payload.GetPlay())
	} else if payload.Ty == gt.BlackwhiteActionShow && payload.GetShow() != nil {
		return action.Show(payload.GetShow())
	} else if payload.Ty == gt.BlackwhiteActionTimeoutDone && payload.GetTimeoutDone() != nil {
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
	var payload gt.BlackwhiteAction
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
				var receipt gt.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				kv := c.saveHeightIndex(&receipt)
				set.KV = append(set.KV, kv...)
				break
			}
		case types.TyLogBlackwhiteLoopInfo:
			{
				var res gt.ReplyLoopResults
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
	var payload gt.BlackwhiteAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		switch log.Ty {
		case types.TyLogBlackwhiteCreate:
			{
				var receipt gt.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				kv := c.delHeightIndex(&receipt)
				set.KV = append(set.KV, kv...)
				break
			}
		case types.TyLogBlackwhitePlay:
		case types.TyLogBlackwhiteShow:
		case types.TyLogBlackwhiteTimeout:
		case types.TyLogBlackwhiteDone:
			{
				var receipt gt.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				//状态数据库由于默克尔树特性，之前生成的索引无效，故不需要回滚，只回滚localDB
				kv := c.delHeightIndex(&receipt)
				set.KV = append(set.KV, kv...)

				kv = c.saveRollHeightIndex(&receipt)
				set.KV = append(set.KV, kv...)
				break
			}
		case types.TyLogBlackwhiteLoopInfo:
			{
				var res gt.ReplyLoopResults
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

func (c *Blackwhite) saveLoopResult(res *gt.ReplyLoopResults) (kvs []*types.KeyValue) {
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4LoopResult(res.GetGameID())
	kv.Value = types.Encode(res)
	kvs = append(kvs, kv)
	return kvs
}

func (c *Blackwhite) delLoopResult(res *gt.ReplyLoopResults) (kvs []*types.KeyValue) {
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4LoopResult(res.GetGameID())
	kv.Value = nil
	kvs = append(kvs, kv)
	return kvs
}

func (c *Blackwhite) saveHeightIndex(res *gt.ReceiptBlackwhiteStatus) (kvs []*types.KeyValue) {
	heightstr := genHeightIndexStr(res.GetIndex())
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4AddrHeight(res.GetAddr(), heightstr)
	kv.Value = []byte(res.GetGameID())
	kvs = append(kvs, kv)

	kv1 := &types.KeyValue{}
	kv1.Key = calcRoundKey4StatusAddrHeight(res.GetStatus(), res.GetAddr(), heightstr)
	kv1.Value = []byte(res.GetGameID())
	kvs = append(kvs, kv1)

	if res.GetStatus() >= 1 {
		kv := &types.KeyValue{}
		kv.Key = calcRoundKey4StatusAddrHeight(res.GetPrevStatus(), res.GetAddr(), heightstr)
		kv.Value = nil
		kvs = append(kvs, kv)
	}
	return kvs
}

func (c *Blackwhite) saveRollHeightIndex(res *gt.ReceiptBlackwhiteStatus) (kvs []*types.KeyValue) {
	heightstr := genHeightIndexStr(res.GetIndex())
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4AddrHeight(res.GetAddr(), heightstr)
	kv.Value = []byte(res.GetGameID())
	kvs = append(kvs, kv)

	kv1 := &types.KeyValue{}
	kv1.Key = calcRoundKey4StatusAddrHeight(res.GetPrevStatus(), res.GetAddr(), heightstr)
	kv1.Value = []byte(res.GetGameID())
	kvs = append(kvs, kv1)

	return kvs
}

func (c *Blackwhite) delHeightIndex(res *gt.ReceiptBlackwhiteStatus) (kvs []*types.KeyValue) {
	heightstr := genHeightIndexStr(res.GetIndex())
	kv := &types.KeyValue{}
	kv.Key = calcRoundKey4AddrHeight(res.GetAddr(), heightstr)
	kv.Value = nil
	kvs = append(kvs, kv)

	kv1 := &types.KeyValue{}
	kv1.Key = calcRoundKey4StatusAddrHeight(res.GetStatus(), res.GetAddr(), heightstr)
	kv1.Value = nil
	kvs = append(kvs, kv1)
	return kvs
}

func (c *Blackwhite) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == gt.GetBlackwhiteRoundInfo {
		var in gt.ReqBlackwhiteRoundInfo
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetBlackwhiteRoundInfo(&in)
	} else if funcName == gt.GetBlackwhiteByStatusAndAddr {
		var in gt.ReqBlackwhiteRoundList
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetBwRoundListInfo(&in)
	} else if funcName == gt.GetBlackwhiteloopResult {
		var in gt.ReqLoopResult
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetBwRoundLoopResult(&in)
	} else if funcName == gt.BlackwhiteCreateTx {
		in := &gt.BlackwhiteCreateTxReq{}
		err := types.Decode(params, in)
		if err != nil {
			return nil, err
		}
		return c.createTx(in)
	} else if funcName == gt.BlackwhitePlayTx {
		in := &gt.BlackwhitePlayTxReq{}
		err := types.Decode(params, in)
		if err != nil {
			return nil, err
		}
		return c.playTx(in)
	} else if funcName == gt.BlackwhiteShowTx {
		in := &gt.BlackwhiteShowTxReq{}
		err := types.Decode(params, in)
		if err != nil {
			return nil, err
		}
		return c.showTx(in)
	} else if funcName == gt.BlackwhiteTimeoutDoneTx {
		in := &gt.BlackwhiteTimeoutDoneTxReq{}
		err := types.Decode(params, in)
		if err != nil {
			return nil, err
		}
		return c.timeoutDoneTx(in)
	}
	return nil, types.ErrActionNotSupport
}

func (c *Blackwhite) timeoutDoneTx(parm *gt.BlackwhiteTimeoutDoneTxReq) (types.Message, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	head := &gt.BlackwhiteTimeoutDone{
		GameID: parm.GameID,
	}

	val := &gt.BlackwhiteAction{
		Ty:    gt.BlackwhiteActionTimeoutDone,
		Value: &gt.BlackwhiteAction_TimeoutDone{head},
	}
	tx := &types.Transaction{
		Execer:  gt.ExecerBlackwhite,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(gt.ExecerBlackwhite)),
	}

	tx.SetRealFee(types.MinFee)
	return tx, nil
}

func (c *Blackwhite) showTx(parm *gt.BlackwhiteShowTxReq) (types.Message, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	head := &gt.BlackwhiteShow{
		GameID: parm.GameID,
		Secret: parm.Secret,
	}

	val := &gt.BlackwhiteAction{
		Ty:    gt.BlackwhiteActionShow,
		Value: &gt.BlackwhiteAction_Show{head},
	}
	tx := &types.Transaction{
		Execer:  gt.ExecerBlackwhite,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(gt.ExecerBlackwhite)),
	}

	tx.SetRealFee(types.MinFee)
	return tx, nil
}

func (c *Blackwhite) playTx(parm *gt.BlackwhitePlayTxReq) (types.Message, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	head := &gt.BlackwhitePlay{
		GameID:     parm.GameID,
		Amount:     parm.Amount,
		HashValues: parm.HashValues,
	}

	val := &gt.BlackwhiteAction{
		Ty:    gt.BlackwhiteActionPlay,
		Value: &gt.BlackwhiteAction_Play{head},
	}
	tx := &types.Transaction{
		Execer:  gt.ExecerBlackwhite,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(gt.ExecerBlackwhite)),
	}

	tx.SetRealFee(types.MinFee)
	return tx, nil
}

func (c *Blackwhite) createTx(parm *gt.BlackwhiteCreateTxReq) (types.Message, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	head := &gt.BlackwhiteCreate{
		PlayAmount:  parm.PlayAmount,
		PlayerCount: parm.PlayerCount,
		Timeout:     parm.Timeout,
		GameName:    parm.GameName,
	}

	val := &gt.BlackwhiteAction{
		Ty:    gt.BlackwhiteActionCreate,
		Value: &gt.BlackwhiteAction_Create{head},
	}
	tx := &types.Transaction{
		Execer:  gt.ExecerBlackwhite,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(gt.ExecerBlackwhite)),
	}

	tx.SetRealFee(types.MinFee)
	return tx, nil
}

func (c *Blackwhite) GetBlackwhiteRoundInfo(req *gt.ReqBlackwhiteRoundInfo) (types.Message, error) {
	gameId := req.GameID
	key := calcRoundKey(gameId)
	values, err := c.GetStateDB().Get(key)
	if err != nil {
		return nil, err
	}

	var round gt.BlackwhiteRound
	err = types.Decode(values, &round)
	if err != nil {
		return nil, err
	}
	//密钥不显示
	for _, addRes := range round.AddrResult {
		addRes.ShowSecret = ""
	}
	roundRes := &gt.BlackwhiteRoundResult{
		GameID:         round.GameID,
		Status:         round.Status,
		PlayAmount:     round.PlayAmount,
		PlayerCount:    round.PlayerCount,
		CurPlayerCount: round.CurPlayerCount,
		Loop:           round.Loop,
		CurShowCount:   round.CurShowCount,
		CreateTime:     round.CreateTime,
		ShowTime:       round.ShowTime,
		Timeout:        round.Timeout,
		CreateAddr:     round.CreateAddr,
		GameName:       round.GameName,
		AddrResult:     round.AddrResult,
		Winner:         round.Winner,
		Index:          round.Index,
	}
	var rep gt.ReplyBlackwhiteRoundInfo
	rep.Round = roundRes
	return &rep, nil
}

func (c *Blackwhite) GetBwRoundListInfo(req *gt.ReqBlackwhiteRoundList) (types.Message, error) {
	var key []byte
	var values [][]byte
	var err error
	var prefix []byte

	if 0 == req.Status {
		prefix = calcRoundKey4AddrHeight(req.Address, "")
	} else {
		prefix = calcRoundKey4StatusAddrHeight(req.Status, req.Address, "")
	}
	localDb := c.GetLocalDB()
	if req.GetIndex() == -1 {
		values, err = localDb.List(prefix, nil, req.Count, req.GetDirection())
		if err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, types.ErrNotFound
		}
	} else { //翻页查找指定的txhash列表
		heightstr := genHeightIndexStr(req.GetIndex())
		if 0 == req.Status {
			key = calcRoundKey4AddrHeight(req.Address, heightstr)
		} else {
			key = calcRoundKey4StatusAddrHeight(req.Status, req.Address, heightstr)
		}
		values, err = localDb.List(prefix, key, req.Count, req.Direction)
		if err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, types.ErrNotFound
		}
	}

	if len(values) == 0 {
		return nil, types.ErrNotFound
	}
	storeDb := c.GetStateDB()
	var rep gt.ReplyBlackwhiteRoundList
	for _, value := range values {
		v, err := storeDb.Get(calcRoundKey(string(value)))
		if nil != err {
			return nil, err
		}
		var round gt.BlackwhiteRound
		err = types.Decode(v, &round)
		if err != nil {
			return nil, err
		}
		//密钥不显示
		for _, addRes := range round.AddrResult {
			addRes.ShowSecret = ""
		}
		roundRes := &gt.BlackwhiteRoundResult{
			GameID:         round.GameID,
			Status:         round.Status,
			PlayAmount:     round.PlayAmount,
			PlayerCount:    round.PlayerCount,
			CurPlayerCount: round.CurPlayerCount,
			Loop:           round.Loop,
			CurShowCount:   round.CurShowCount,
			CreateTime:     round.CreateTime,
			ShowTime:       round.ShowTime,
			Timeout:        round.Timeout,
			CreateAddr:     round.CreateAddr,
			GameName:       round.GameName,
			AddrResult:     round.AddrResult,
			Winner:         round.Winner,
			Index:          round.Index,
		}
		rep.Round = append(rep.Round, roundRes)
	}
	return &rep, nil
}

func (c *Blackwhite) GetBwRoundLoopResult(req *gt.ReqLoopResult) (types.Message, error) {
	localDb := c.GetLocalDB()
	values, err := localDb.Get(calcRoundKey4LoopResult(req.GameID))
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, types.ErrNotFound
	}

	var result gt.ReplyLoopResults
	err = types.Decode(values, &result)
	if err != nil {
		return nil, err
	}

	if req.LoopSeq > 0 { //取出具体一轮
		if len(result.Results) < int(req.LoopSeq) {
			return nil, types.ErrNoLoopSeq
		}
		res := &gt.ReplyLoopResults{
			GameID: result.GameID,
		}
		index := int(req.LoopSeq)
		perRes := &gt.PerLoopResult{}
		perRes.Winers = append(perRes.Winers, res.Results[index-1].Winers...)
		perRes.Losers = append(perRes.Losers, res.Results[index-1].Losers...)
		res.Results = append(res.Results, perRes)
		return res, nil
	}
	return &result, nil //将所有轮数取出
}

func genHeightIndexStr(index int64) string {
	return fmt.Sprintf("%018d", index)
}

func heightIndexToIndex(height int64, index int32) int64 {
	return height*types.MaxTxsPerBlock + int64(index)
}
