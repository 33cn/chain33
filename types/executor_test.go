// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadExecutorType(t *testing.T) {
	exec := LoadExecutorType("manage")
	assert.NotEqual(t, exec, nil)
	assert.Equal(t, exec.GetName(), "manage")

	exec = LoadExecutorType("coins")
	assert.NotEqual(t, exec, nil)
	assert.Equal(t, exec.GetName(), "coins")

	exec = LoadExecutorType("xxxx")
	assert.Equal(t, exec, nil)

}

func TestFormatTx(t *testing.T) {
	tx := &Transaction{
		Payload: []byte("this is  a test."),
	}
	tx, err := FormatTx("user.p.none", tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("user.p.none"))
	fee, _ := tx.GetRealFee(GInt("MinFee"))
	assert.Equal(t, tx.Fee, fee)
}

func TestFormatTxEncode(t *testing.T) {
	data, err := FormatTxEncode("coins", &Transaction{
		Payload: []byte("this is  a test."),
	})
	assert.Equal(t, err, nil)
	var tx Transaction
	err = Decode(data, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("coins"))
}

func TestCallCreateTxJSON(t *testing.T) {
	modify := &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "add",
		Addr:  "",
	}
	data, err := json.Marshal(modify)
	assert.Equal(t, err, nil)

	result, err := CallCreateTxJSON("manage", "Modify", data)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	var tx Transaction
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ := tx.GetRealFee(GInt("MinFee"))
	assert.Equal(t, tx.Fee, fee)

	_, err = CallCreateTxJSON("coins", "Modify", data)
	assert.NotEqual(t, err, nil)

	_, err = CallCreateTxJSON("xxxx", "xxx", data)
	assert.NotEqual(t, err, nil)

	modify = &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "delete",
		Addr:  "",
	}
	data, err = json.Marshal(modify)
	assert.Equal(t, err, nil)

	result, err = CallCreateTxJSON("manage", "Modify", data)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ = tx.GetRealFee(GInt("MinFee"))
	assert.Equal(t, tx.Fee, fee)

}

func TestCallCreateTx(t *testing.T) {
	modify := &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "add",
		Addr:  "",
	}

	result, err := CallCreateTx("manage", "Modify", modify)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	var tx Transaction
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ := tx.GetRealFee(GInt("MinFee"))
	assert.Equal(t, tx.Fee, fee)

	_, err = CallCreateTx("coins", "Modify", modify)
	assert.NotEqual(t, err, nil)

	_, err = CallCreateTx("xxxx", "xxx", modify)
	assert.NotEqual(t, err, nil)

	modify = &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "delete",
		Addr:  "",
	}

	result, err = CallCreateTx("manage", "Modify", modify)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ = tx.GetRealFee(GInt("MinFee"))
	assert.Equal(t, tx.Fee, fee)
}

func TestIsAssetsTransfer(t *testing.T) {
	assert.Equal(t, true, IsAssetsTransfer(&AssetsTransfer{}))
	assert.Equal(t, true, IsAssetsTransfer(&AssetsWithdraw{}))
	assert.Equal(t, true, IsAssetsTransfer(&AssetsTransferToExec{}))
	assert.Equal(t, false, IsAssetsTransfer(&Transaction{}))
}
