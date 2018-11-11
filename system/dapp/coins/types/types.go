// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"github.com/33cn/chain33/types"
)

const (
	CoinsActionTransfer       = 1
	CoinsActionGenesis        = 2
	CoinsActionWithdraw       = 3
	CoinsActionTransferToExec = 10
)

var (
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

type CoinsType struct {
	types.ExecTypeBase
}

func NewType() *CoinsType {
	c := &CoinsType{}
	c.SetChild(c)
	return c
}

func (coins *CoinsType) GetPayload() types.Message {
	return &CoinsAction{}
}

func (coins *CoinsType) GetName() string {
	return CoinsX
}

func (coins *CoinsType) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

func (c *CoinsType) GetTypeMap() map[string]int32 {
	return actionName
}

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
