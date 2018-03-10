package token

/*
token执行器支持token的创建，

主要提供操作有以下几种：
1）预创建token；
2）完成创建token
3）撤销预创建
*/

import (
	"code.aliyun.com/chain33/chain33/executor/drivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
	"code.aliyun.com/chain33/chain33/account"
)

var tokenlog = log.New("module", "execs.token")

func init() {
	t := NewToken()
	drivers.Register(t.GetName(), t)
	drivers.RegisterAddress(t.GetName())
}

type Token struct {
	drivers.DriverBase
}

func NewToken() *Token {
	t := &Token{}
	t.SetChild(t)
	return t
}

func (t *Token) GetName() string {
	return "token"
}

func (t *Token) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var tokenAction types.TokenAction
	err := types.Decode(tx.Payload, &tokenAction)
	if err != nil {
		return nil, err
	}
	tokenlog.Info("exec token create tx=", "tx=", tokenAction)

	switch tokenAction.GetTy() {
	case types.TokenActionPreCreate:
		action := NewTokenAction(t, "", tx)
		return action.preCreate(tokenAction.GetTokenprecreate())

	case types.TokenActionFinishCreate:
		action := NewTokenAction(t, types.FundKeyAddr, tx)
		return action.finishCreate(tokenAction.GetTokenfinishcreate(), []string{types.FundKeyAddr})

	case types.TokenActionRevokeCreate:
		action := NewTokenAction(t, "", tx)
		return action.revokeCreate(tokenAction.GetTokenrevokecreate())

	case types.ActionTransfer:
	case types.ActionWithdraw:
		token := tokenAction.GetTransfer().GetCointoken()
		return t.ExecTransWithdraw(account.NewTokenAccount(token, t.GetDB()), tx, &tokenAction)
	}

	return nil, types.ErrActionNotSupport
}

func (t *Token) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var action types.CoinsAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
    var set *types.LocalDBSet
	if action.Ty == types.ActionTransfer || action.Ty == types.ActionWithdraw {
		set, err = t.DriverBase.ExecLocalTransWithdraw(tx, receipt, index)
	} else {
		set, err = t.DriverBase.ExecLocal(tx, receipt, index)
		if err != nil {
			return nil, err
		}
		if receipt.GetTy() != types.ExecOk {
			return set, nil
		}

		for i := 0; i < len(receipt.Logs); i++ {
			item := receipt.Logs[i]
			if item.Ty == types.TyLogPreCreateToken || item.Ty == types.TyLogFinishCreateToken || item.Ty == types.TyLogRevokeCreateToken {
				var receipt types.ReceiptToken
				err := types.Decode(item.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}
				set.KV = append(set.KV, t.saveLogs(&receipt)...)
			}
		}
	}

	return set, nil
}

func (t *Token) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var action types.CoinsAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var set *types.LocalDBSet
	if action.Ty == types.ActionTransfer || action.Ty == types.ActionWithdraw {
		set, err = t.ExecDelLocalLocalTransWithdraw(tx, receipt, index)
	} else {
		set, err = t.DriverBase.ExecDelLocal(tx, receipt, index)
		if err != nil {
			return nil, err
		}
		if receipt.GetTy() != types.ExecOk {
			return set, nil
		}

		for i := 0; i < len(receipt.Logs); i++ {
			item := receipt.Logs[i]
			if item.Ty == types.TyLogPreCreateToken || item.Ty == types.TyLogFinishCreateToken || item.Ty == types.TyLogRevokeCreateToken {
				var receipt types.ReceiptToken
				err := types.Decode(item.Log, &receipt)
				if err != nil {
					tokenlog.Error("Failed to decode ReceiptToken in ExecDelLocal")
					continue
				}
				set.KV = append(set.KV, t.deleteLogs(&receipt)...)
			}
		}
	}

	return set, nil
}

func (t *Token) Query(funcName string, params []byte) (types.Message, error) {
	switch funcName {
	//GetTokens,支持所有状态下的单个token，多个token，以及所有token的信息的查询
	case "GetTokens":
		var reqtokens types.ReqTokens
		err := types.Decode(params, &reqtokens)
		if err != nil {
			return nil, err
		}
		return t.GetTokens(&reqtokens)

		//case "GetPrecreatedTokens":
		//case "GetAllCreatedTokenSymbols":
		//case "GetAllCreatedTokenDetails":
		//case "GetSomeCreatedTokenDetails":
	case "GetTokenInfo":
		var symbol types.ReqString
		err := types.Decode(params, &symbol)
		if err != nil {
			return nil, err
		}
		return t.GetTokenInfo(symbol.GetData())
	case "GetAddrReceiverforTokens":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetAddrReceiverforTokens(&addrTokens)
	}
	return nil, types.ErrActionNotSupport
}

func (t *Token) GetAddrReceiverforTokens(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	var reply = &types.ReplyAddrRecvForTokens{}
	db := t.GetQueryDB()
	reciver := types.Int64{}
	for _, token := range addrTokens.Token {
		addrRecv := db.Get(drivers.CalcAddrKey(token, addrTokens.Addr))
		if addrRecv == nil {
			continue
		}
		err := types.Decode(addrRecv, &reciver)
		if err != nil {
			continue
		}

		recv := &types.TokenRecv{token, reciver.Data}
		reply.Tokenrecv = append(reply.Tokenrecv, recv)
	}

	return reply, nil
}

func (t *Token) GetTokenInfo(symbol string) (types.Message, error) {
	db := t.GetDB()
	token, err := db.Get(tokenKey(symbol))
	if err != nil {
		return nil, types.ErrEmpty
	}
	var token_info types.Token
	err = types.Decode(token, &token_info)
	if err != nil {
		return &token_info, err
	}
	return &token_info, nil
}

func (t *Token) GetTokens(reqTokens *types.ReqTokens) (types.Message, error) {
	querydb := t.GetQueryDB()
	localdb := t.GetLocalDB()
	db := t.GetDB()

	replyTokens := &types.ReplyTokens{}
	if reqTokens.Queryall {
		keys := querydb.List(tokenStatusKeyPrefix(reqTokens.Status), nil, 0, 0)
		if len(keys) != 0 {
			for _, key := range keys {
				if value, err := localdb.Get(key); err != nil {
					continue
				} else {
					if tokenValue, err := db.Get(value); err == nil {
						var token types.Token
						err = types.Decode(tokenValue, &token)
						if err == nil {
							replyTokens.Tokens = append(replyTokens.Tokens, &token)
						}

					}

				}
			}
		}

	} else {

	}
	return replyTokens, nil
}

func (t *Token) saveLogs(receipt *types.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := tokenStatusKey(receipt.Symbol, receipt.Owner, receipt.Status)
	value := tokenAddrKey(receipt.Symbol, receipt.Owner)
	kv = append(kv, &types.KeyValue{key, value})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被删除
	if receipt.Status != types.TokenStatusPreCreated {
		key = tokenStatusKey(receipt.Symbol, receipt.Owner, types.TokenStatusPreCreated)
		kv = append(kv, &types.KeyValue{key, nil})
	}
	return kv
}

func (t *Token) deleteLogs(receipt *types.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := tokenStatusKey(receipt.Symbol, receipt.Owner, receipt.Status)
	kv = append(kv, &types.KeyValue{key, nil})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被恢复
	if receipt.Status != types.TokenStatusPreCreated {
		key = tokenStatusKey(receipt.Symbol, receipt.Owner, types.TokenStatusPreCreated)
		value := tokenAddrKey(receipt.Symbol, receipt.Owner)
		kv = append(kv, &types.KeyValue{key, value})
	}
	return kv
}
