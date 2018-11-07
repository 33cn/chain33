package executor

import (
	"fmt"

	"strings"

	"gitlab.33.cn/chain33/chain33/account"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

type tokenDB struct {
	token tokenty.Token
}

func newTokenDB(preCreate *tokenty.TokenPreCreate, creator string) *tokenDB {
	t := &tokenDB{}
	t.token.Name = preCreate.GetName()
	t.token.Symbol = preCreate.GetSymbol()
	t.token.Introduction = preCreate.GetIntroduction()
	t.token.Total = preCreate.GetTotal()
	t.token.Price = preCreate.GetPrice()
	//token可以由自己进行创建，也可以通过委托给其他地址进行创建
	t.token.Owner = preCreate.GetOwner()
	t.token.Creator = creator
	t.token.Status = tokenty.TokenStatusPreCreated
	return t
}

func (t *tokenDB) save(db dbm.KV, key []byte) {
	set := t.getKVSet(key)
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func (t *tokenDB) getLogs(ty int32, status int32) []*types.ReceiptLog {
	var log []*types.ReceiptLog
	value := types.Encode(&tokenty.ReceiptToken{t.token.Symbol, t.token.Owner, t.token.Status})
	log = append(log, &types.ReceiptLog{ty, value})

	return log
}

//key:mavl-create-token-addr-xxx or mavl-token-xxx <-----> value:token
func (t *tokenDB) getKVSet(key []byte) (kvset []*types.KeyValue) {
	value := types.Encode(&t.token)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
}

func getTokenFromDB(db dbm.KV, symbol string, owner string) (*tokenty.Token, error) {
	key := calcTokenAddrKeyS(symbol, owner)
	value, err := db.Get(key)
	if err != nil {
		// not found old key
		key = calcTokenAddrNewKeyS(symbol, owner)
		value, err = db.Get(key)
		if err != nil {
			return nil, err
		}
	}

	var token tokenty.Token
	if err = types.Decode(value, &token); err != nil {
		tokenlog.Error("getTokenFromDB", "Fail to decode types.token for key", string(key), "err info is", err)
		return nil, err
	}
	return &token, nil
}

type tokenAction struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	toaddr       string
	blocktime    int64
	height       int64
	execaddr     string
}

func newTokenAction(t *token, toaddr string, tx *types.Transaction) *tokenAction {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &tokenAction{t.GetCoinsAccount(), t.GetStateDB(), hash, fromaddr, toaddr,
		t.GetBlockTime(), t.GetHeight(), dapp.ExecAddress(string(tx.Execer))}
}

func (action *tokenAction) preCreate(token *tokenty.TokenPreCreate) (*types.Receipt, error) {
	tokenlog.Debug("preCreate")
	if token == nil {
		return nil, types.ErrInvalidParam
	}
	if len(token.GetName()) > types.TokenNameLenLimit {
		return nil, tokenty.ErrTokenNameLen
	} else if len(token.GetIntroduction()) > types.TokenIntroLenLimit {
		return nil, tokenty.ErrTokenIntroLen
	} else if len(token.GetSymbol()) > types.TokenSymbolLenLimit {
		return nil, tokenty.ErrTokenSymbolLen
	} else if token.GetTotal() > types.MaxTokenBalance || token.GetTotal() <= 0 {
		return nil, tokenty.ErrTokenTotalOverflow
	}

	if !ValidSymbolWithHeight([]byte(token.GetSymbol()), action.height) {
		tokenlog.Error("token precreate ", "symbol need be upper", token.GetSymbol())
		return nil, tokenty.ErrTokenSymbolUpper
	}

	if CheckTokenExist(token.GetSymbol(), action.db) {
		return nil, tokenty.ErrTokenExist
	}

	if checkTokenHasPrecreate(token.GetSymbol(), token.GetOwner(), tokenty.TokenStatusPreCreated, action.db) {
		return nil, tokenty.ErrTokenHavePrecreated
	}

	if types.IsDappFork(action.height, tokenty.TokenX, "ForkTokenBlackList") {
		found, err := inBlacklist(token.GetSymbol(), blacklist, action.db)
		if err != nil {
			return nil, err
		}
		if found {
			return nil, tokenty.ErrTokenBlacklist
		}
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	if types.IsDappFork(action.height, tokenty.TokenX, "ForkTokenPrice") && token.GetPrice() == 0 {
		// pay for create token offline
	} else {
		receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, token.GetPrice())
		if err != nil {
			tokenlog.Error("token precreate ", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", token.GetTotal())
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}

	tokendb := newTokenDB(token, action.fromaddr)
	var statuskey []byte
	var key []byte
	if types.IsFork(action.height, "ForkExecKey") {
		statuskey = calcTokenStatusNewKeyS(tokendb.token.Symbol, tokendb.token.Owner, tokenty.TokenStatusPreCreated)
		key = calcTokenAddrNewKeyS(tokendb.token.Symbol, tokendb.token.Owner)
	} else {
		statuskey = calcTokenStatusKey(tokendb.token.Symbol, tokendb.token.Owner, tokenty.TokenStatusPreCreated)
		key = calcTokenAddrKeyS(tokendb.token.Symbol, tokendb.token.Owner)
	}

	tokendb.save(action.db, statuskey)
	tokendb.save(action.db, key)

	logs = append(logs, tokendb.getLogs(tokenty.TyLogPreCreateToken, tokenty.TokenStatusPreCreated)...)
	kv = append(kv, tokendb.getKVSet(key)...)
	kv = append(kv, tokendb.getKVSet(statuskey)...)
	//tokenlog.Info("func token preCreate", "token:", tokendb.token.Symbol, "owner:", tokendb.token.Owner,
	//	"key:", key, "key string", string(key), "value:", tokendb.getKVSet(key)[0].Value)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *tokenAction) finishCreate(tokenFinish *tokenty.TokenFinishCreate) (*types.Receipt, error) {
	tokenlog.Debug("finishCreate")
	if tokenFinish == nil {
		return nil, types.ErrInvalidParam
	}
	token, err := getTokenFromDB(action.db, tokenFinish.GetSymbol(), tokenFinish.GetOwner())
	if err != nil || token.Status != tokenty.TokenStatusPreCreated {
		return nil, tokenty.ErrTokenNotPrecreated
	}

	approverValid := false

	for _, approver := range conf.GStrList("tokenApprs") {
		if approver == action.fromaddr {
			approverValid = true
			break
		}
	}

	hasPriv, ok := validFinisher(action.fromaddr, action.db)
	if (ok != nil || !hasPriv) && !approverValid {
		return nil, tokenty.ErrTokenCreatedApprover
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	if types.IsDappFork(action.height, tokenty.TokenX, "ForkTokenPrice") && token.GetPrice() == 0 {
		// pay for create token offline
	} else {
		//将之前冻结的资金转账到fund合约账户中
		receiptForCoin, err := action.coinsAccount.ExecTransferFrozen(token.Creator, action.toaddr, action.execaddr, token.Price)
		if err != nil {
			tokenlog.Error("token finishcreate ", "addr", action.fromaddr, "execaddr", action.execaddr, "token", token.Symbol)
			return nil, err
		}
		logs = append(logs, receiptForCoin.Logs...)
		kv = append(kv, receiptForCoin.KV...)
	}

	//创建token类型的账户，同时需要创建的额度存入

	tokenAccount, err := account.NewAccountDB("token", tokenFinish.GetSymbol(), action.db)
	if err != nil {
		return nil, err
	}
	tokenlog.Debug("finishCreate", "token.Owner", token.Owner, "token.GetTotal()", token.GetTotal())
	receiptForToken, err := tokenAccount.GenesisInit(token.Owner, token.GetTotal())
	if err != nil {
		return nil, err
	}
	//更新token的状态为已经创建
	token.Status = tokenty.TokenStatusCreated
	tokendb := &tokenDB{*token}
	var key []byte
	if types.IsFork(action.height, "ForkExecKey") {
		key = calcTokenAddrNewKeyS(tokendb.token.Symbol, tokendb.token.Owner)
	} else {
		key = calcTokenAddrKeyS(tokendb.token.Symbol, tokendb.token.Owner)
	}
	tokendb.save(action.db, key)

	logs = append(logs, receiptForToken.Logs...)
	logs = append(logs, tokendb.getLogs(tokenty.TyLogFinishCreateToken, tokenty.TokenStatusCreated)...)
	kv = append(kv, receiptForToken.KV...)
	kv = append(kv, tokendb.getKVSet(key)...)

	key = calcTokenKey(tokendb.token.Symbol)
	//因为该token已经被创建，需要保存一个全局的token，防止其他用户再次创建
	tokendb.save(action.db, key)
	kv = append(kv, tokendb.getKVSet(key)...)
	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *tokenAction) revokeCreate(tokenRevoke *tokenty.TokenRevokeCreate) (*types.Receipt, error) {
	if tokenRevoke == nil {
		return nil, types.ErrInvalidParam
	}
	token, err := getTokenFromDB(action.db, tokenRevoke.GetSymbol(), tokenRevoke.GetOwner())
	if err != nil {
		tokenlog.Error("token revokeCreate ", "Can't get token form db for token", tokenRevoke.GetSymbol())
		return nil, tokenty.ErrTokenNotPrecreated
	}

	if token.Status != tokenty.TokenStatusPreCreated {
		tokenlog.Error("token revokeCreate ", "token's status should be precreated to be revoked for token", tokenRevoke.GetSymbol())
		return nil, tokenty.ErrTokenCanotRevoked
	}

	//确认交易发起者的身份，token的发起人可以撤销该项token的创建
	//token的owner允许撤销交易
	if action.fromaddr != token.Owner && action.fromaddr != token.Creator {
		tokenlog.Error("tprocTokenRevokeCreate, different creator/owner vs actor of this revoke",
			"action.fromaddr", action.fromaddr, "creator", token.Creator, "owner", token.Owner)
		return nil, tokenty.ErrTokenRevoker
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	if types.IsDappFork(action.height, tokenty.TokenX, "ForkTokenPrice") && token.GetPrice() == 0 {
		// pay for create token offline
	} else {
		//解锁之前冻结的资金
		receipt, err := action.coinsAccount.ExecActive(token.Creator, action.execaddr, token.Price)
		if err != nil {
			tokenlog.Error("token revokeCreate error ", "error info", err, "creator addr", token.Creator, "execaddr", action.execaddr, "token", token.Symbol)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}

	token.Status = tokenty.TokenStatusCreateRevoked
	tokendb := &tokenDB{*token}
	var key []byte
	if types.IsFork(action.height, "ForkExecKey") {
		key = calcTokenAddrNewKeyS(tokendb.token.Symbol, tokendb.token.Owner)
	} else {
		key = calcTokenAddrKeyS(tokendb.token.Symbol, tokendb.token.Owner)
	}
	tokendb.save(action.db, key)

	logs = append(logs, tokendb.getLogs(tokenty.TyLogRevokeCreateToken, tokenty.TokenStatusCreateRevoked)...)
	kv = append(kv, tokendb.getKVSet(key)...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func CheckTokenExist(token string, db dbm.KV) bool {
	_, err := db.Get(calcTokenKey(token))
	return err == nil
}

func checkTokenHasPrecreate(token, owner string, status int32, db dbm.KV) bool {
	_, err := db.Get(calcTokenAddrKeyS(token, owner))
	if err == nil {
		return true
	}
	_, err = db.Get(calcTokenAddrNewKeyS(token, owner))
	return err == nil
}

func validFinisher(addr string, db dbm.KV) (bool, error) {
	return validOperator(addr, finisherKey, db)
}

func getManageKey(key string, db dbm.KV) ([]byte, error) {
	manageKey := types.ManageKey(key)
	value, err := db.Get([]byte(manageKey))
	if err != nil {
		tokenlog.Info("tokendb", "get db key", "not found manageKey", "key", manageKey)
		return getConfigKey(key, db)
	}
	return value, nil
}

func getConfigKey(key string, db dbm.KV) ([]byte, error) {
	configKey := types.ConfigKey(key)
	value, err := db.Get([]byte(configKey))
	if err != nil {
		tokenlog.Info("tokendb", "get db key", "not found configKey", "key", configKey)
		return nil, err
	}
	return value, nil
}

func validOperator(addr, key string, db dbm.KV) (bool, error) {
	value, err := getManageKey(key, db)
	if err != nil {
		tokenlog.Info("tokendb", "get db key", "not found", "key", key)
		return false, err
	}
	if value == nil {
		tokenlog.Info("tokendb", "get db key", "  found nil value", "key", key)
		return false, nil
	}

	var item types.ConfigItem
	err = types.Decode(value, &item)
	if err != nil {
		tokenlog.Error("tokendb", "get db key", err)
		return false, err // types.ErrBadConfigValue
	}

	for _, op := range item.GetArr().Value {
		if op == addr {
			return true, nil
		}
	}

	return false, nil
}

func CalcTokenAssetsKey(addr string) []byte {
	return []byte(fmt.Sprintf(tokenAssetsPrefix+"%s", addr))
}

func GetTokenAssetsKey(addr string, db dbm.KVDB) (*types.ReplyStrings, error) {
	key := CalcTokenAssetsKey(addr)
	value, err := db.Get(key)
	if err != nil && err != types.ErrNotFound {
		tokenlog.Error("tokendb", "GetTokenAssetsKey", err)
		return nil, err
	}
	var assets types.ReplyStrings
	if err == types.ErrNotFound {
		return &assets, nil
	}
	err = types.Decode(value, &assets)
	if err != nil {
		tokenlog.Error("tokendb", "GetTokenAssetsKey", err)
		return nil, err
	}
	return &assets, nil
}

func AddTokenToAssets(addr string, db dbm.KVDB, symbol string) []*types.KeyValue {
	tokenAssets, err := GetTokenAssetsKey(addr, db)
	if err != nil {
		return nil
	}
	if tokenAssets == nil {
		tokenAssets = &types.ReplyStrings{}
	}

	var found = false
	for _, sym := range tokenAssets.Datas {
		if sym == symbol {
			found = true
			break
		}
	}
	if !found {
		tokenAssets.Datas = append(tokenAssets.Datas, symbol)
	}
	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{CalcTokenAssetsKey(addr), types.Encode(tokenAssets)})
	return kv
}

func inBlacklist(symbol, key string, db dbm.KV) (bool, error) {
	found, err := validOperator(symbol, key, db)
	return found, err
}

func IsUpperChar(a byte) bool {
	res := (a <= 'Z' && a >= 'A')
	return res
}

func ValidSymbol(cs []byte) bool {
	for _, c := range cs {
		if !IsUpperChar(c) {
			return false
		}
	}
	return true
}

func ValidSymbolWithHeight(cs []byte, height int64) bool {
	if !types.IsDappFork(height, tokenty.TokenX, "ForkBadTokenSymbol") {
		symbol := string(cs)
		upSymbol := strings.ToUpper(symbol)
		return upSymbol == symbol
	}
	return ValidSymbol(cs)
}
