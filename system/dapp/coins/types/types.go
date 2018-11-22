// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
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
	CoinsX      = "coins"
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
func (coins *CoinsType) GetPayload() types.Message {
	return &CoinsAction{}
}

// GetName  return coins string
func (coins *CoinsType) GetName() string {
	return CoinsX
}

// GetLogMap return log for map
func (coins *CoinsType) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

// GetTypeMap return actionname for map
func (c *CoinsType) GetTypeMap() map[string]int32 {
	return actionName
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
	assetlist, err := c.ExecTypeBase.GetAssets(tx)
	if err != nil || len(assetlist) == 0 {
		return nil, err
	}
	if assetlist[0].Symbol == "" {
		assetlist[0].Symbol = types.BTY
	}
	return assetlist, nil
}
