package token

/*
token执行器支持token的创建，

主要提供操作有以下几种：
1）预创建token；
2）完成创建token
3）撤销预创建
*/

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var tokenlog = log.New("module", "execs.token")

const (
	finisherKey       = "token-finisher"
	tokenAssetsPrefix = "token-assets:"
	blacklist         = "token-blacklist"
)

func init() {
	t := newToken()
	drivers.Register(t.GetName(), t, types.ForkV2_add_token)
}

type token struct {
	drivers.DriverBase
}

func newToken() *token {
	t := &token{}
	t.SetChild(t)
	return t
}

func (t *token) GetName() string {
	return "token"
}

func (t *token) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var tokenAction types.TokenAction
	err := types.Decode(tx.Payload, &tokenAction)
	if err != nil {
		return nil, err
	}
	tokenlog.Info("exec token action", "txhash", common.Bytes2Hex(tx.Hash()), "tokenAction.GetTy()", tokenAction.GetTy())

	switch tokenAction.GetTy() {
	case types.TokenActionPreCreate:
		action := newTokenAction(t, "", tx)
		return action.preCreate(tokenAction.GetTokenprecreate())

	case types.TokenActionFinishCreate:
		action := newTokenAction(t, types.FundKeyAddr, tx)
		return action.finishCreate(tokenAction.GetTokenfinishcreate())

	case types.TokenActionRevokeCreate:
		action := newTokenAction(t, "", tx)
		return action.revokeCreate(tokenAction.GetTokenrevokecreate())

	case types.ActionTransfer:
		token := tokenAction.GetTransfer().GetCointoken()
		return t.ExecTransWithdraw(account.NewTokenAccount(token, t.GetDB()), tx, &tokenAction, index)

	case types.ActionWithdraw:
		token := tokenAction.GetWithdraw().GetCointoken()
		return t.ExecTransWithdraw(account.NewTokenAccount(token, t.GetDB()), tx, &tokenAction, index)
	}

	return nil, types.ErrActionNotSupport
}

func (t *token) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var action types.TokenAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	tokenlog.Info("exec token ExecLocal tx=", "tx=", action)
	var set *types.LocalDBSet
	if action.Ty == types.ActionTransfer || action.Ty == types.ActionWithdraw {
		set, err = t.ExecLocalTransWithdraw(tx, receipt, index)

		if action.Ty == types.ActionTransfer {
			transfer := action.GetTransfer()
			// 添加个人资产列表
			tokenlog.Info("ExecLocalTransWithdraw", "addr", tx.To, "asset", transfer.Cointoken)
			kv := AddTokenToAssets(tx.To, t.GetLocalDB(), transfer.Cointoken)
			if kv != nil {
				set.KV = append(set.KV, kv...)
			}
		}
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

				receiptKV := t.saveLogs(&receipt)
				set.KV = append(set.KV, receiptKV...)

				// 添加个人资产列表
				if item.Ty == types.TyLogFinishCreateToken {
					kv := AddTokenToAssets(action.GetTokenfinishcreate().Owner, t.GetLocalDB(), action.GetTokenfinishcreate().Symbol)
					if kv != nil {
						set.KV = append(set.KV, kv...)
					}
				}
			}
		}
	}

	return set, nil
}

func (t *token) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var action types.TokenAction
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

func (t *token) Query(funcName string, params []byte) (types.Message, error) {
	switch funcName {
	//GetTokens,支持所有状态下的单个token，多个token，以及所有token的信息的查询
	case "GetTokens":
		var reqtokens types.ReqTokens
		err := types.Decode(params, &reqtokens)
		if err != nil {
			return nil, err
		}
		tokenlog.Info("token Query", "function name", funcName, "query tokens", reqtokens)
		return t.GetTokens(&reqtokens)
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
	case "GetAccountTokenAssets":
		var req types.ReqAccountTokenAssets
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return t.GetAccountTokenAssets(&req)
	}
	return nil, types.ErrActionNotSupport
}

func (t *token) GetAccountTokenAssets(req *types.ReqAccountTokenAssets) (types.Message, error) {
	var reply = &types.ReplyAccountTokenAssets{}
	assets, err := QueryTokenAssetsKey(req.Address, t.GetQueryDB())
	if err != nil {
		return nil, err
	}
	for _, asset := range assets.Datas {
		acc := account.NewTokenAccount(asset, t.GetDB())
		var acc1 *types.Account
		if req.Execer == "trade" {
			execaddress := account.ExecAddress(req.Execer)
			acc1 = acc.LoadExecAccount(req.Address, execaddress.String())
		} else if req.Execer == "token" {
			acc1 = acc.LoadAccount(req.Address)
		}
		if acc1 == nil {
			continue
		}
		tokenAsset := &types.TokenAsset{asset, acc1}
		tokenlog.Info("GetAccountTokenAssets", "token-asset-symbol", asset, "info", acc1)
		reply.TokenAssets = append(reply.TokenAssets, tokenAsset)
	}
	return reply, nil
}

func (t *token) GetAddrReceiverforTokens(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	var reply = &types.ReplyAddrRecvForTokens{}
	db := t.GetQueryDB()
	reciver := types.Int64{}
	for _, token := range addrTokens.Token {
		addrRecv := db.Get(calcAddrKey(token, addrTokens.Addr))
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

func (t *token) GetTokenInfo(symbol string) (types.Message, error) {
	db := t.GetDB()
	token, err := db.Get(calcTokenKey(symbol))
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

func (t *token) GetTokens(reqTokens *types.ReqTokens) (types.Message, error) {
	querydb := t.GetQueryDB()
	db := t.GetDB()

	replyTokens := &types.ReplyTokens{}
	if reqTokens.Queryall {
		list := dbm.NewListHelper(querydb)
		keys := list.List(calcTokenStatusKeyPrefix(reqTokens.Status), nil, 0, 0)
		tokenlog.Debug("token Query GetTokens", "get count", len(keys))
		if len(keys) != 0 {
			for _, key := range keys {
				tokenlog.Debug("token Query GetTokens", "key in string", string(key))
				if tokenValue, err := db.Get(key); err == nil {
					var token types.Token
					err = types.Decode(tokenValue, &token)
					if err == nil {
						replyTokens.Tokens = append(replyTokens.Tokens, &token)
					}
				}
			}
		}

	} else {
		for _, token := range reqTokens.Tokens {
			list := dbm.NewListHelper(querydb)
			keys := list.List(calcTokenStatusSymbolPrefix(reqTokens.Status, token), nil, 0, 0)
			tokenlog.Debug("token Query GetTokens", "get count", len(keys))
			if len(keys) != 0 {
				for _, key := range keys {
					tokenlog.Debug("token Query GetTokens", "key in string", string(key))
					if tokenValue, err := db.Get(key); err == nil {
						var token types.Token
						err = types.Decode(tokenValue, &token)
						if err == nil {
							replyTokens.Tokens = append(replyTokens.Tokens, &token)
						}
					}
				}
			}
		}
	}

	tokenlog.Info("token Query", "replyTokens", replyTokens)
	return replyTokens, nil
}

func (t *token) saveLogs(receipt *types.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := calcTokenStatusKey(receipt.Symbol, receipt.Owner, receipt.Status)
	value := calcTokenAddrKey(receipt.Symbol, receipt.Owner)
	kv = append(kv, &types.KeyValue{key, value})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被删除
	if receipt.Status != types.TokenStatusPreCreated {
		key = calcTokenStatusKey(receipt.Symbol, receipt.Owner, types.TokenStatusPreCreated)
		kv = append(kv, &types.KeyValue{key, nil})
	}
	return kv
}

func (t *token) deleteLogs(receipt *types.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := calcTokenStatusKey(receipt.Symbol, receipt.Owner, receipt.Status)
	kv = append(kv, &types.KeyValue{key, nil})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被恢复
	if receipt.Status != types.TokenStatusPreCreated {
		key = calcTokenStatusKey(receipt.Symbol, receipt.Owner, types.TokenStatusPreCreated)
		value := calcTokenAddrKey(receipt.Symbol, receipt.Owner)
		kv = append(kv, &types.KeyValue{key, value})
	}
	return kv
}
