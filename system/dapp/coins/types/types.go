// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/33cn/chain33/types"
)

const (
	// CoinsActionTransfer defines const number
	CoinsActionTransfer = 1
	// CoinsActionGenesis  defines const coinsactiongenesis number
	CoinsActionGenesis = 2
	// CoinsActionWithdraw defines const number coinsactionwithdraw
	CoinsActionWithdraw = 3
	// CoinsActionTransferToExec defines const number coinsactiontransfertoExec
	CoinsActionTransferToExec = 10
)

var (
	// CoinsX defines a global string
	CoinsX = "coins"
	// ExecerCoins execer coins
	ExecerCoins = []byte(CoinsX)
	actionName  = map[string]int32{
		"Transfer":       CoinsActionTransfer,
		"TransferToExec": CoinsActionTransferToExec,
		"Withdraw":       CoinsActionWithdraw,
		"Genesis":        CoinsActionGenesis,
	}
	logmap = make(map[int64]*types.LogInfo)
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerCoins)
	types.RegistorExecutor("coins", NewType())

	types.RegisterDappFork(CoinsX, "Enable", 0)
}

// CoinsType defines exec type
type CoinsType struct {
	types.ExecTypeBase
}

// NewType new coinstype
func NewType() *CoinsType {
	c := &CoinsType{}
	c.SetChild(c)
	return c
}

// GetPayload  return payload
func (c *CoinsType) GetPayload() types.Message {
	return &CoinsAction{}
}

// GetName  return coins string
func (c *CoinsType) GetName() string {
	return CoinsX
}

// GetLogMap return log for map
func (c *CoinsType) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

// GetTypeMap return actionname for map
func (c *CoinsType) GetTypeMap() map[string]int32 {
	return actionName
}

//DecodePayloadValue 为了性能考虑，coins 是最常用的合约，我们这里不用反射吗，做了特殊化的优化
func (c *CoinsType) DecodePayloadValue(tx *types.Transaction) (string, reflect.Value, error) {
	name, value, err := c.decodePayloadValue(tx)
	return name, value, err
}

func (c *CoinsType) decodePayloadValue(tx *types.Transaction) (string, reflect.Value, error) {
	var action CoinsAction
	if tx.GetPayload() == nil {
		return "", reflect.ValueOf(nil), types.ErrActionNotSupport
	}
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "", reflect.ValueOf(nil), err
	}
	var name string
	var value types.Message

	switch action.Ty {
	case CoinsActionTransfer:
		name = "Transfer"
		value = action.GetTransfer()
	case CoinsActionTransferToExec:
		name = "TransferToExec"
		value = action.GetTransferToExec()
	case CoinsActionWithdraw:
		name = "Withdraw"
		value = action.GetWithdraw()
	case CoinsActionGenesis:
		name = "Genesis"
		value = action.GetGenesis()
	}
	if value == nil {
		return "", reflect.ValueOf(nil), types.ErrActionNotSupport
	}
	return name, reflect.ValueOf(value), nil
}

// RPC_Default_Process default process fo rpc
func (c *CoinsType) RPC_Default_Process(action string, msg interface{}) (*types.Transaction, error) {
	var create *types.CreateTx
	if _, ok := msg.(*types.CreateTx); !ok {
		return nil, types.ErrInvalidParam
	}
	create = msg.(*types.CreateTx)
	if create.IsToken {
		return nil, types.ErrNotSupport
	}
	tx, err := c.AssertCreate(create)
	if err != nil {
		return nil, err
	}
	//to地址的问题,如果是主链交易，to地址就是直接是设置to
	if !types.IsPara() {
		tx.To = create.To
	}
	return tx, err
}

// GetAssets return asset list
func (c *CoinsType) GetAssets(tx *types.Transaction) ([]*types.Asset, error) {
	assets, err := c.ExecTypeBase.GetAssets(tx)
	if err != nil || len(assets) == 0 {
		return nil, err
	}
	if assets[0].Symbol == "" {
		assets[0].Symbol = types.GetCoinSymbol()
	}
	if assets[0].Symbol == "bty" {
		assets[0].Symbol = "BTY"
	}
	return assets, nil
}
