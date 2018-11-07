package executor

/*
token执行器支持token的创建，

主要提供操作有以下几种：
1）预创建token；
2）完成创建token
3）撤销预创建
*/

import (
	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/address"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/system/dapp"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var tokenlog = log.New("module", "execs.token")

const (
	finisherKey       = "token-finisher"
	tokenAssetsPrefix = "LODB-token-assets:"
	blacklist         = "token-blacklist"
)

var driverName = "token"
var conf = types.ConfSub(driverName)

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&token{}))
}

type subConfig struct {
	SaveTokenTxList bool `json:"saveTokenTxList"`
}

var cfg subConfig

func Init(name string, sub []byte) {
	if sub != nil {
		types.MustDecode(sub, &cfg)
	}
	drivers.Register(GetName(), newToken, types.GetDappFork(driverName, "Enable"))
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
	t.SetExecutorType(types.LoadExecutorType(driverName))
	return t
}

func (t *token) GetDriverName() string {
	return driverName
}

func (c *token) CheckTx(tx *types.Transaction, index int) error {
	return nil
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

func (t *token) GetAccountTokenAssets(req *tokenty.ReqAccountTokenAssets) (types.Message, error) {
	var reply = &tokenty.ReplyAccountTokenAssets{}
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
		tokenAsset := &tokenty.TokenAsset{asset, acc1}
		reply.TokenAssets = append(reply.TokenAssets, tokenAsset)
	}
	return reply, nil
}

func (t *token) GetAddrReceiverforTokens(addrTokens *tokenty.ReqAddrTokens) (types.Message, error) {
	var reply = &tokenty.ReplyAddrRecvForTokens{}
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

		recv := &tokenty.TokenRecv{token, reciver.Data}
		reply.TokenRecvs = append(reply.TokenRecvs, recv)
	}

	return reply, nil
}

func (t *token) GetTokenInfo(symbol string) (types.Message, error) {
	if symbol == "" {
		return nil, types.ErrInvalidParam
	}
	key := calcTokenStatusTokenKeyPrefixLocal(tokenty.TokenStatusCreated, symbol)
	values, err := t.GetLocalDB().List(key, nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 || values[0] == nil || len(values[0]) == 0 {
		return nil, types.ErrNotFound
	}
	var tokenInfo tokenty.LocalToken
	err = types.Decode(values[0], &tokenInfo)
	if err != nil {
		return &tokenInfo, err
	}
	return &tokenInfo, nil
}

func (t *token) GetTokens(reqTokens *tokenty.ReqTokens) (types.Message, error) {
	replyTokens := &tokenty.ReplyTokens{}
	tokens, err := t.listTokenKeys(reqTokens)
	if err != nil {
		return nil, err
	}
	tokenlog.Error("token Query GetTokens", "get count", len(tokens))
	if reqTokens.SymbolOnly {
		for _, t1 := range tokens {
			if len(t1) == 0 {
				continue
			}

			var tokenValue tokenty.LocalToken
			err = types.Decode(t1, &tokenValue)
			if err == nil {
				token := tokenty.LocalToken{Symbol: tokenValue.Symbol}
				replyTokens.Tokens = append(replyTokens.Tokens, &token)
			}
		}
		return replyTokens, nil
	}

	for _, t1 := range tokens {
		// delete impl by set nil
		if len(t1) == 0 {
			continue
		}

		var token tokenty.LocalToken
		err = types.Decode(t1, &token)
		if err == nil {
			replyTokens.Tokens = append(replyTokens.Tokens, &token)
		}
	}

	//tokenlog.Info("token Query", "replyTokens", replyTokens)
	return replyTokens, nil
}

func (t *token) listTokenKeys(reqTokens *tokenty.ReqTokens) ([][]byte, error) {
	querydb := t.GetLocalDB()
	if reqTokens.QueryAll {
		keys, err := querydb.List(calcTokenStatusKeyPrefixLocal(reqTokens.Status), nil, 0, 0)
		if err != nil && err != types.ErrNotFound {
			return nil, err
		}
		if len(keys) == 0 {
			return nil, types.ErrNotFound
		}
		tokenlog.Debug("token Query GetTokens", "get count", len(keys))
		return keys, nil
	} else {
		var keys [][]byte
		for _, token := range reqTokens.Tokens {
			keys1, err := querydb.List(calcTokenStatusTokenKeyPrefixLocal(reqTokens.Status, token), nil, 0, 0)
			if err != nil && err != types.ErrNotFound {
				return nil, err
			}
			keys = append(keys, keys1...)

			tokenlog.Debug("token Query GetTokens", "get count", len(keys))
		}
		if len(keys) == 0 {
			return nil, types.ErrNotFound
		}
		return keys, nil
	}
}

// value 对应 statedb 的key
func (t *token) saveLogs(receipt *tokenty.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := calcTokenStatusKeyLocal(receipt.Symbol, receipt.Owner, receipt.Status)
	var value []byte
	if types.IsFork(t.GetHeight(), "ForkExecKey") {
		value = calcTokenAddrNewKeyS(receipt.Symbol, receipt.Owner)
	} else {
		value = calcTokenAddrKeyS(receipt.Symbol, receipt.Owner)
	}
	kv = append(kv, &types.KeyValue{key, value})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被删除
	if receipt.Status != tokenty.TokenStatusPreCreated {
		key = calcTokenStatusKeyLocal(receipt.Symbol, receipt.Owner, tokenty.TokenStatusPreCreated)
		kv = append(kv, &types.KeyValue{key, nil})
	}
	return kv
}

func (t *token) deleteLogs(receipt *tokenty.ReceiptToken) []*types.KeyValue {
	var kv []*types.KeyValue

	key := calcTokenStatusKeyLocal(receipt.Symbol, receipt.Owner, receipt.Status)
	kv = append(kv, &types.KeyValue{key, nil})
	//如果当前需要被更新的状态不是Status_PreCreated，则认为之前的状态是precreate，且其对应的key需要被恢复
	if receipt.Status != tokenty.TokenStatusPreCreated {
		key = calcTokenStatusKeyLocal(receipt.Symbol, receipt.Owner, tokenty.TokenStatusPreCreated)
		var value []byte
		if types.IsFork(t.GetHeight(), "ForkExecKey") {
			value = calcTokenAddrNewKeyS(receipt.Symbol, receipt.Owner)
		} else {
			value = calcTokenAddrKeyS(receipt.Symbol, receipt.Owner)
		}
		kv = append(kv, &types.KeyValue{key, value})
	}
	return kv
}

func (t *token) makeTokenTxKvs(tx *types.Transaction, action *tokenty.TokenAction, receipt *types.ReceiptData, index int, isDel bool) ([]*types.KeyValue, error) {
	var kvs []*types.KeyValue
	var symbol string
	if action.Ty == tokenty.ActionTransfer {
		symbol = action.GetTransfer().Cointoken
	} else if action.Ty == tokenty.ActionWithdraw {
		symbol = action.GetWithdraw().Cointoken
	} else if action.Ty == tokenty.TokenActionTransferToExec {
		symbol = action.GetTransferToExec().Cointoken
	} else {
		return kvs, nil
	}

	kvs, err := TokenTxKvs(tx, symbol, t.GetHeight(), int64(index), isDel)
	return kvs, err
}

func findTokenTxListUtil(req *tokenty.ReqTokenTx) ([]byte, []byte) {
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

func (t *token) GetTxByToken(req *tokenty.ReqTokenTx) (types.Message, error) {
	if req.Flag != 0 && req.Flag != dapp.TxIndexFrom && req.Flag != dapp.TxIndexTo {
		err := types.ErrInvalidParam
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
		return nil, errors.Wrapf(types.ErrNotFound, "key=%s, prefix=%s", string(key), string(prefix))
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
