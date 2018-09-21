package executor

/*
token执行器支持token的创建，

主要提供操作有以下几种：
1）预创建token；
2）完成创建token
3）撤销预创建
*/

import (
	"reflect"
	"strings"

	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/address"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
	pty "gitlab.33.cn/chain33/chain33/types/executor/token"
)

var tokenlog = log.New("module", "execs.token")

const (
	finisherKey       = "token-finisher"
	tokenAssetsPrefix = "token-assets:"
	blacklist         = "token-blacklist"
)

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = pty.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&token{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	drivers.Register(GetName(), newToken, types.ForkV2AddToken)
	setReciptPrefix()
}

func GetName() string {
	return newToken().GetName()
}

type token struct {
	drivers.DriverBase
}

func newToken() drivers.Driver {
	t := &token{}
	t.SetChild(t)
	t.SetExecutorType(executorType)
	return t
}

func (t *token) GetDriverName() string {
	return "token"
}

func (c *token) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func (t *token) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var tokenAction types.TokenAction
	err := types.Decode(tx.Payload, &tokenAction)
	if err != nil {
		return nil, err
	}
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
		if tokenAction.GetTransfer() == nil {
			return nil, types.ErrInputPara
		}
		token := tokenAction.GetTransfer().GetCointoken()
		db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
		if err != nil {
			return nil, err
		}
		return t.ExecTransWithdraw(db, tx, &tokenAction, index)

	case types.ActionWithdraw:
		if tokenAction.GetWithdraw() == nil {
			return nil, types.ErrInputPara
		}
		token := tokenAction.GetWithdraw().GetCointoken()
		db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
		if err != nil {
			return nil, err
		}
		return t.ExecTransWithdraw(db, tx, &tokenAction, index)

	case types.TokenActionTransferToExec:
		if tokenAction.GetTransferToExec() == nil {
			return nil, types.ErrInputPara
		}
		token := tokenAction.GetTransferToExec().GetCointoken()
		db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
		if err != nil {
			return nil, err
		}
		return t.ExecTransWithdraw(db, tx, &tokenAction, index)
	}

	return nil, types.ErrActionNotSupport
}

func (t *token) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var action types.TokenAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var set *types.LocalDBSet
	if action.Ty == types.ActionTransfer || action.Ty == types.ActionWithdraw {
		set, err = t.ExecLocalTransWithdraw(tx, receipt, index)
		if err != nil {
			return nil, err
		}

		if action.Ty == types.ActionTransfer && action.GetTransfer() != nil {
			transfer := action.GetTransfer()
			// 添加个人资产列表
			//tokenlog.Info("ExecLocalTransWithdraw", "addr", tx.GetRealToAddr(), "asset", transfer.Cointoken)
			kv := AddTokenToAssets(tx.GetRealToAddr(), t.GetLocalDB(), transfer.Cointoken)
			if kv != nil {
				set.KV = append(set.KV, kv...)
			}
		}
		if types.GetSaveTokenTxList() {
			kvs, err := t.makeTokenTxKvs(tx, &action, receipt, index, false)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kvs...)
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
				if item.Ty == types.TyLogFinishCreateToken && action.GetTokenfinishcreate() != nil {
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
		if err != nil {
			return nil, err
		}
		if types.GetSaveTokenTxList() {
			kvs, err := t.makeTokenTxKvs(tx, &action, receipt, index, true)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kvs...)
		}
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
		//tokenlog.Info("token Query", "function name", funcName, "query tokens", reqtokens)
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
	case "GetTxByToken":
		if !types.GetSaveTokenTxList() {
			return nil, types.ErrActionNotSupport
		}
		var req types.ReqTokenTx
		err := types.Decode(params, &req) // TODO_x to test show err log
		if err != nil {
			return nil, errors.Wrap(err, "GetTxByToken Decode request failed")
		}
		tokenlog.Debug("query debug", "func", funcName, "req", req)
		return t.GetTxByToken(&req)
	}
	return nil, types.ErrActionNotSupport
}

func (t *token) QueryTokenAssetsKey(addr string) (*types.ReplyStrings, error) {
	key := CalcTokenAssetsKey(addr)
	value, err := t.GetLocalDB().Get(key)
	if value == nil || err != nil {
		tokenlog.Error("tokendb", "GetTokenAssetsKey", types.ErrNotFound)
		return nil, types.ErrNotFound
	}
	var assets types.ReplyStrings
	err = types.Decode(value, &assets)
	if err != nil {
		tokenlog.Error("tokendb", "GetTokenAssetsKey", err)
		return nil, err
	}
	return &assets, nil
}

func (t *token) GetAccountTokenAssets(req *types.ReqAccountTokenAssets) (types.Message, error) {
	var reply = &types.ReplyAccountTokenAssets{}
	assets, err := t.QueryTokenAssetsKey(req.Address)
	if err != nil {
		return nil, err
	}
	for _, asset := range assets.Datas {
		acc, err := account.NewAccountDB(t.GetName(), asset, t.GetStateDB())
		if err != nil {
			return nil, err
		}
		var acc1 *types.Account
		if req.Execer == "trade" {
			execaddress := address.ExecAddress(req.Execer)
			acc1 = acc.LoadExecAccount(req.Address, execaddress)
		} else if req.Execer == t.GetName() {
			acc1 = acc.LoadAccount(req.Address)
		}
		if acc1 == nil {
			continue
		}
		tokenAsset := &types.TokenAsset{asset, acc1}
		reply.TokenAssets = append(reply.TokenAssets, tokenAsset)
	}
	return reply, nil
}

func (t *token) GetAddrReceiverforTokens(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	var reply = &types.ReplyAddrRecvForTokens{}
	db := t.GetLocalDB()
	reciver := types.Int64{}
	for _, token := range addrTokens.Token {
		addrRecv, err := db.Get(calcAddrKey(token, addrTokens.Addr))
		if addrRecv == nil || err != nil {
			continue
		}
		err = types.Decode(addrRecv, &reciver)
		if err != nil {
			continue
		}

		recv := &types.TokenRecv{token, reciver.Data}
		reply.TokenRecvs = append(reply.TokenRecvs, recv)
	}

	return reply, nil
}

func (t *token) GetTokenInfo(symbol string) (types.Message, error) {
	db := t.GetStateDB()
	token, err := db.Get(calcTokenKey(symbol))
	if err != nil {
		return nil, types.ErrEmpty
	}
	var tokenInfo types.Token
	err = types.Decode(token, &tokenInfo)
	if err != nil {
		return &tokenInfo, err
	}
	return &tokenInfo, nil
}

func (t *token) GetTokens(reqTokens *types.ReqTokens) (types.Message, error) {
	replyTokens := &types.ReplyTokens{}
	keys, err := t.listTokenKeys(reqTokens)
	if err != nil {
		return nil, err
	}
	tokenlog.Debug("token Query GetTokens", "get count", len(keys), "KEY", string(keys[0]))
	if reqTokens.SymbolOnly {
		for _, key := range keys {
			idx := strings.LastIndex(string(key), "-")
			if idx < 0 || idx >= len(key) {
				continue
			}
			symbol := key[idx+1:]
			token := types.Token{Symbol: string(symbol)}
			replyTokens.Tokens = append(replyTokens.Tokens, &token)
		}
		return replyTokens, nil
	}

	db := t.GetStateDB()
	for _, key := range keys {
		if tokenValue, err := db.Get(key); err == nil {
			var token types.Token
			err = types.Decode(tokenValue, &token)
			if err == nil {
				replyTokens.Tokens = append(replyTokens.Tokens, &token)
			}
		}
	}

	//tokenlog.Info("token Query", "replyTokens", replyTokens)
	return replyTokens, nil
}

func (t *token) listTokenKeys(reqTokens *types.ReqTokens) ([][]byte, error) {
	querydb := t.GetLocalDB()
	if reqTokens.QueryAll {
		//list := dbm.NewListHelper(querydb)
		keys, err := querydb.List(calcTokenStatusKeyNewPrefix(reqTokens.Status), nil, 0, 0)
		if err != nil && err != types.ErrNotFound {
			return nil, err
		}
		keys2, err := querydb.List(calcTokenStatusKeyPrefix(reqTokens.Status), nil, 0, 0)
		if err != nil && err != types.ErrNotFound {
			return nil, err
		}
		keys = append(keys, keys2...)
		if len(keys) == 0 {
			return nil, types.ErrNotFound
		}
		tokenlog.Debug("token Query GetTokens", "get count", len(keys))
		return keys, nil
	} else {
		var keys [][]byte
		for _, token := range reqTokens.Tokens {
			//list := dbm.NewListHelper(querydb)
			keys1, err := querydb.List(calcTokenStatusSymbolNewPrefix(reqTokens.Status, token), nil, 0, 0)
			if err != nil && err != types.ErrNotFound {
				return nil, err
			}
			keys = append(keys, keys1...)

			keys2, err := querydb.List(calcTokenStatusSymbolPrefix(reqTokens.Status, token), nil, 0, 0)
			if err != nil && err != types.ErrNotFound {
				return nil, err
			}
			keys = append(keys, keys2...)
			tokenlog.Debug("token Query GetTokens", "get count", len(keys))
		}
		if len(keys) == 0 {
			return nil, types.ErrNotFound
		}
		return keys, nil
	}
}

// value 对应 statedb 的key
func (t *token) saveLogs(receipt *types.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := calcTokenStatusNewKey(receipt.Symbol, receipt.Owner, receipt.Status)
	var value []byte
	if t.GetHeight() >= types.ForkV13ExecKey {
		value = calcTokenAddrNewKey(receipt.Symbol, receipt.Owner)
	} else {
		value = calcTokenAddrKey(receipt.Symbol, receipt.Owner)
	}
	kv = append(kv, &types.KeyValue{key, value})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被删除
	if receipt.Status != types.TokenStatusPreCreated {
		key = calcTokenStatusNewKey(receipt.Symbol, receipt.Owner, types.TokenStatusPreCreated)
		kv = append(kv, &types.KeyValue{key, nil})
	}
	return kv
}

func (t *token) deleteLogs(receipt *types.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := calcTokenStatusNewKey(receipt.Symbol, receipt.Owner, receipt.Status)
	kv = append(kv, &types.KeyValue{key, nil})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被恢复
	if receipt.Status != types.TokenStatusPreCreated {
		key = calcTokenStatusNewKey(receipt.Symbol, receipt.Owner, types.TokenStatusPreCreated)
		var value []byte
		if t.GetHeight() >= types.ForkV13ExecKey {
			value = calcTokenAddrNewKey(receipt.Symbol, receipt.Owner)
		} else {
			value = calcTokenAddrKey(receipt.Symbol, receipt.Owner)
		}
		kv = append(kv, &types.KeyValue{key, value})
	}
	return kv
}

func (t *token) makeTokenTxKvs(tx *types.Transaction, action *types.TokenAction, receipt *types.ReceiptData, index int, isDel bool) ([]*types.KeyValue, error) {
	var kvs []*types.KeyValue
	var symbol string
	if action.Ty == types.ActionTransfer {
		symbol = action.GetTransfer().Cointoken
	} else if action.Ty != types.ActionWithdraw {
		symbol = action.GetWithdraw().Cointoken
	} else {
		return kvs, nil
	}

	kvs, err := TokenTxKvs(tx, symbol, t.GetHeight(), int64(index), isDel)
	return kvs, err
}

func findTokenTxListUtil(req *types.ReqTokenTx) ([]byte, []byte) {
	var key, prefix []byte
	if len(req.Addr) > 0 {
		if req.Flag == 0 {
			prefix = CalcTokenAddrTxKey(req.Symbol, req.Addr, -1, 0)
			key = CalcTokenAddrTxKey(req.Symbol, req.Addr, req.Height, req.Index)
		} else {
			prefix = CalcTokenAddrTxDirKey(req.Symbol, req.Addr, req.Flag, -1, 0)
			key = CalcTokenAddrTxDirKey(req.Symbol, req.Addr, req.Flag, req.Height, req.Index)
		}
	} else {
		prefix = CalcTokenTxKey(req.Symbol, -1, 0)
		key = CalcTokenTxKey(req.Symbol, req.Height, req.Index)
	}
	if req.Height == -1 {
		key = nil
	}
	return key, prefix
}

func (t *token) GetTxByToken(req *types.ReqTokenTx) (types.Message, error) {
	if req.Flag != 0 && req.Flag != dapp.TxIndexFrom && req.Flag != dapp.TxIndexTo {
		err := types.ErrInputPara
		return nil, errors.Wrap(err, "flag unknown")
	}
	key, prefix := findTokenTxListUtil(req)
	tokenlog.Debug("GetTxByToken", "key", string(key), "prefix", string(prefix))

	db := t.GetLocalDB()
	txinfos, err := db.List(prefix, key, req.Count, req.Direction)
	if err != nil {
		return nil, errors.Wrap(err, "db.List to find token tx list")
	}
	if len(txinfos) == 0 {
		return nil, errors.New("tx does not exist")
	}

	var replyTxInfos types.ReplyTxInfos
	replyTxInfos.TxInfos = make([]*types.ReplyTxInfo, len(txinfos))
	for index, txinfobyte := range txinfos {
		var replyTxInfo types.ReplyTxInfo
		err := types.Decode(txinfobyte, &replyTxInfo)
		if err != nil {
			return nil, err
		}
		replyTxInfos.TxInfos[index] = &replyTxInfo
	}
	return &replyTxInfos, nil
}

func (t *token) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func (t *token) GetPayloadValue() types.Message {
	return &types.TokenAction{}
}

func (t *token) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Tokenprecreate":    types.TokenActionPreCreate,
		"Tokenfinishcreate": types.TokenActionFinishCreate,
		"Tokenrevokecreate": types.TokenActionRevokeCreate,
		"Transfer":          types.ActionTransfer,
		"Withdraw":          types.ActionWithdraw,
		"Genesis":           types.ActionGenesis,
		"TransferToExec":    types.TokenActionTransferToExec,
	}
}
