package executor

import (
	"gitlab.33.cn/chain33/chain33/common/db"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *token) ExecLocal_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.ExecLocalTransWithdraw(tx, receiptData, index)
	if err != nil {
		return nil, err
	}
	// 添加个人资产列表
	//tokenlog.Info("ExecLocalTransWithdraw", "addr", tx.GetRealToAddr(), "asset", transfer.Cointoken)
	kv := AddTokenToAssets(tx.GetRealToAddr(), t.GetLocalDB(), payload.Cointoken)
	if kv != nil {
		set.KV = append(set.KV, kv...)
	}
	if cfg.SaveTokenTxList {
		tokenAction := tokenty.TokenAction{
			Ty: tokenty.ActionTransfer,
			Value: &tokenty.TokenAction_Transfer{
				payload,
			},
		}
		kvs, err := t.makeTokenTxKvs(tx, &tokenAction, receiptData, index, false)
		if err != nil {
			return nil, err
		}
		set.KV = append(set.KV, kvs...)
	}
	return set, nil
}

func (t *token) ExecLocal_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.ExecLocalTransWithdraw(tx, receiptData, index)
	if err != nil {
		return nil, err
	}
	// 添加个人资产列表
	kv := AddTokenToAssets(tx.From(), t.GetLocalDB(), payload.Cointoken)
	if kv != nil {
		set.KV = append(set.KV, kv...)
	}
	if cfg.SaveTokenTxList {
		tokenAction := tokenty.TokenAction{
			Ty: tokenty.ActionWithdraw,
			Value: &tokenty.TokenAction_Withdraw{
				payload,
			},
		}
		kvs, err := t.makeTokenTxKvs(tx, &tokenAction, receiptData, index, false)
		if err != nil {
			return nil, err
		}
		set.KV = append(set.KV, kvs...)
	}
	return set, nil
}

func (t *token) ExecLocal_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.ExecLocalTransWithdraw(tx, receiptData, index)
	if err != nil {
		return nil, err
	}
	if cfg.SaveTokenTxList {
		tokenAction := tokenty.TokenAction{
			Ty: tokenty.TokenActionTransferToExec,
			Value: &tokenty.TokenAction_TransferToExec{
				payload,
			},
		}
		kvs, err := t.makeTokenTxKvs(tx, &tokenAction, receiptData, index, false)
		if err != nil {
			return nil, err
		}
		set.KV = append(set.KV, kvs...)
	}
	return set, nil
}

func (t *token) ExecLocal_TokenPreCreate(payload *tokenty.TokenPreCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	localToken := newLocalToken(payload)
	localToken = setPrepare(localToken, tx.From(), t.GetHeight(), t.GetBlockTime())
	key := calcTokenStatusKeyLocal(payload.Symbol, payload.Owner, tokenty.TokenStatusPreCreated)

	var set []*types.KeyValue
	set = append(set, &types.KeyValue{Key: key, Value: types.Encode(localToken)})
	return &types.LocalDBSet{KV: set}, nil
}

func (t *token) ExecLocal_TokenFinishCreate(payload *tokenty.TokenFinishCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	prepareKey := calcTokenStatusKeyLocal(payload.Symbol, payload.Owner, tokenty.TokenStatusPreCreated)
	localToken, err := loadLocalToken(payload.Symbol, payload.Owner, tokenty.TokenStatusPreCreated, t.GetLocalDB())
	if err != nil {
		return nil, err
	}
	localToken = setCreated(localToken, t.GetHeight(), t.GetBlockTime())
	key := calcTokenStatusKeyLocal(payload.Symbol, payload.Owner, tokenty.TokenStatusCreated)
	var set []*types.KeyValue
	set = append(set, &types.KeyValue{Key: prepareKey, Value: nil})
	set = append(set, &types.KeyValue{Key: key, Value: types.Encode(localToken)})
	kv := AddTokenToAssets(payload.Owner, t.GetLocalDB(), payload.Symbol)
	set = append(set, kv...)
	return &types.LocalDBSet{KV: set}, nil
}

func (t *token) ExecLocal_TokenRevokeCreate(payload *tokenty.TokenRevokeCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	prepareKey := calcTokenStatusKeyLocal(payload.Symbol, payload.Owner, tokenty.TokenStatusPreCreated)
	localToken, err := loadLocalToken(payload.Symbol, payload.Owner, tokenty.TokenStatusPreCreated, t.GetLocalDB())
	if err != nil {
		return nil, err
	}
	localToken = setRevoked(localToken, t.GetHeight(), t.GetBlockTime())
	key := calcTokenStatusKeyLocal(payload.Symbol, payload.Owner, tokenty.TokenStatusCreateRevoked)
	var set []*types.KeyValue
	set = append(set, &types.KeyValue{Key: prepareKey, Value: nil})
	set = append(set, &types.KeyValue{Key: key, Value: types.Encode(localToken)})
	return &types.LocalDBSet{KV: set}, nil
}

func newLocalToken(payload *tokenty.TokenPreCreate) *tokenty.LocalToken {
	localToken := tokenty.LocalToken{
		Name:                payload.Name,
		Symbol:              payload.Symbol,
		Introduction:        payload.Introduction,
		Total:               payload.Total,
		Price:               payload.Price,
		Owner:               payload.Owner,
		Creator:             "",
		Status:              tokenty.TokenStatusPreCreated,
		CreatedHeight:       0,
		CreatedTime:         0,
		PrepareCreateHeight: 0,
		PrepareCreateTime:   0,
		Precision:           8,
		TotalTransferTimes:  0,
		RevokedHeight:       0,
		RevokedTime:         0,
	}
	return &localToken
}

func setPrepare(t *tokenty.LocalToken, creator string, height, time int64) *tokenty.LocalToken {
	t.Creator = creator
	t.PrepareCreateHeight = height
	t.PrepareCreateTime = time
	return t
}

func loadLocalToken(symbol, owner string, status int32, db db.KVDB) (*tokenty.LocalToken, error) {
	key := calcTokenStatusKeyLocal(symbol, owner, status)
	v, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	var localToken tokenty.LocalToken
	err = types.Decode(v, &localToken)
	if err != nil {
		return nil, err
	}
	return &localToken, nil
}

func setCreated(t *tokenty.LocalToken, height, time int64) *tokenty.LocalToken {
	t.CreatedTime = time
	t.CreatedHeight = height
	t.Status = tokenty.TokenStatusCreated
	return t
}

func setRevoked(t *tokenty.LocalToken, height, time int64) *tokenty.LocalToken {
	t.RevokedTime = time
	t.RevokedHeight = height
	t.Status = tokenty.TokenStatusCreateRevoked
	return t
}

func resetCreated(t *tokenty.LocalToken) *tokenty.LocalToken {
	t.CreatedTime = 0
	t.CreatedHeight = 0
	t.Status = tokenty.TokenStatusPreCreated
	return t
}

func resetRevoked(t *tokenty.LocalToken) *tokenty.LocalToken {
	t.RevokedTime = 0
	t.RevokedHeight = 0
	t.Status = tokenty.TokenStatusPreCreated
	return t
}
