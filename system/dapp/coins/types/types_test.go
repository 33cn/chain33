// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestTypeReflact(t *testing.T) {
	ty := NewType()
	assert.NotNil(t, ty)
	//创建一个json字符串
	data, err := types.PBToJSON(&types.AssetsTransfer{Amount: 10})
	assert.Nil(t, err)
	raw := json.RawMessage(data)
	tx, err := ty.CreateTx("Transfer", raw)
	assert.Nil(t, err)
	name, val, err := ty.DecodePayloadValue(tx)
	assert.Nil(t, err)
	assert.Equal(t, "Transfer", name)
	assert.Equal(t, !types.IsNil(val) && val.CanInterface(), true)
	if !types.IsNil(val) && val.CanInterface() {
		assert.Equal(t, int64(10), val.Interface().(*types.AssetsTransfer).GetAmount())
	}
}

func TestCoinsType(t *testing.T) {
	ty := NewType()
	payload := ty.GetPayload()
	assert.Equal(t, &CoinsAction{}, payload.(*CoinsAction))

	assert.Equal(t, "coins", ty.GetName())
	assert.Equal(t, logmap, ty.GetLogMap())
	assert.Equal(t, actionName, ty.GetTypeMap())

	create := &types.CreateTx{}
	tx, err := ty.RPC_Default_Process("transfer", create)
	assert.NoError(t, err)

	_, err = ty.GetAssets(tx)
	assert.NoError(t, err)
}

func TestCoinsPb(t *testing.T) {
	b := &proto.Buffer{}
	ca := &CoinsAction{Value: &CoinsAction_Transfer{&types.AssetsTransfer{}}}
	var err error

	transfer := ca.GetTransfer()
	assert.NotNil(t, transfer)
	err = _CoinsAction_OneofMarshaler(ca, b)
	assert.NoError(t, err)

	ca.Value = &CoinsAction_Genesis{&types.AssetsGenesis{}}
	genesis := ca.GetGenesis()
	assert.NotNil(t, genesis)
	err = _CoinsAction_OneofMarshaler(ca, b)
	assert.NoError(t, err)
}
