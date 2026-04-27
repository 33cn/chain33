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
	NewChain33Config(GetDefaultCfgstring())
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
	cfg := NewChain33Config(GetDefaultCfgstring())
	tx := &Transaction{
		Payload: []byte("this is  a test."),
	}
	tx, err := FormatTx(cfg, "user.p.none", tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("user.p.none"))
	fee, _ := tx.GetRealFee(cfg.GetMinTxFeeRate())
	assert.Equal(t, tx.Fee, fee)
}

func TestFormatTxEncode(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	data, err := FormatTxEncode(cfg, "coins", &Transaction{
		Payload: []byte("this is  a test."),
	})
	assert.Equal(t, err, nil)
	var tx Transaction
	err = Decode(data, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("coins"))
}

func TestCallCreateTxJSON(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	modify := &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "add",
		Addr:  "",
	}
	data, err := json.Marshal(modify)
	assert.Equal(t, err, nil)

	result, err := CallCreateTxJSON(cfg, "manage", "Modify", data)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	var tx Transaction
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ := tx.GetRealFee(cfg.GetMinTxFeeRate())
	assert.Equal(t, tx.Fee, fee)

	_, err = CallCreateTxJSON(cfg, "coins", "Modify", data)
	assert.NotEqual(t, err, nil)

	_, err = CallCreateTxJSON(cfg, "xxxx", "xxx", data)
	assert.NotEqual(t, err, nil)

	modify = &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "delete",
		Addr:  "",
	}
	data, err = json.Marshal(modify)
	assert.Equal(t, err, nil)

	result, err = CallCreateTxJSON(cfg, "manage", "Modify", data)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ = tx.GetRealFee(cfg.GetMinTxFeeRate())
	assert.Equal(t, tx.Fee, fee)

}

func TestCallCreateTx(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	modify := &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "add",
		Addr:  "",
	}

	result, err := CallCreateTx(cfg, "manage", "Modify", modify)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	var tx Transaction
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ := tx.GetRealFee(cfg.GetMinTxFeeRate())
	assert.Equal(t, tx.Fee, fee)

	_, err = CallCreateTx(cfg, "coins", "Modify", modify)
	assert.NotEqual(t, err, nil)

	_, err = CallCreateTx(cfg, "xxxx", "xxx", modify)
	assert.NotEqual(t, err, nil)

	modify = &ModifyConfig{
		Key:   "token-finisher",
		Value: "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		Op:    "delete",
		Addr:  "",
	}

	result, err = CallCreateTx(cfg, "manage", "Modify", modify)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, result, nil)
	err = Decode(result, &tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.Execer, []byte("manage"))
	fee, _ = tx.GetRealFee(cfg.GetMinTxFeeRate())
	assert.Equal(t, tx.Fee, fee)
}

func TestFormatTxExt(t *testing.T) {
	tx := &Transaction{
		Payload: []byte("test payload"),
		Fee:     100,
	}
	tx, err := FormatTxExt(0, false, 100000, "coins", tx)
	assert.Nil(t, err)
	assert.Equal(t, int32(0), tx.ChainID)
	assert.Equal(t, "coins", string(tx.Execer))
	assert.Equal(t, int64(100), tx.Fee)
	assert.Greater(t, tx.Nonce, int64(0))
	assert.NotEmpty(t, tx.To)
}

func TestFormatTxExtEmptyTo(t *testing.T) {
	tx := &Transaction{Payload: []byte("test")}
	tx, err := FormatTxExt(0, false, 100000, "coins", tx)
	assert.Nil(t, err)
	assert.NotEmpty(t, tx.To)
}

func TestCreateFormatTx(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	tx, err := CreateFormatTx(cfg, "coins", []byte("payload"))
	assert.Nil(t, err)
	assert.Equal(t, "coins", string(tx.Execer))
	assert.Equal(t, "payload", string(tx.Payload))
}

func TestCallCreateTransactionNonExistent(t *testing.T) {
	modify := &ModifyConfig{
		Key:   "test-key",
		Value: "test-value",
		Op:    "add",
		Addr:  "",
	}
	_, err := CallCreateTransaction("nonexistent", "action", modify)
	assert.Equal(t, ErrNotSupport, err)
}

func TestCallCreateTxJSONWithUserExec(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	result, err := CallCreateTxJSON(cfg, "user.p.test", "TestAction", json.RawMessage(`{"key":"value"}`))
	assert.Nil(t, err)
	assert.NotNil(t, result)

	var tx Transaction
	err = Decode(result, &tx)
	assert.Nil(t, err)
	assert.Equal(t, "user.p.test", string(tx.Execer))
}

func TestLogTypeName(t *testing.T) {
	lt := newLogType([]byte("none"), 0)
	assert.Equal(t, "LogReserved", lt.Name())
}

func TestLoadLogReturnsNilForReserved(t *testing.T) {
	NewChain33Config(GetDefaultCfgstring())
	log := LoadLog([]byte("none"), 0)
	assert.Nil(t, log)
}

func TestRegistorExecutor(t *testing.T) {
	assert.Panics(t, func() {
		RegistorExecutor("coins", LoadExecutorType("coins"))
	})
}

func TestCallExecNewTx(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	modify := &ModifyConfig{
		Key:   "test-key",
		Value: "test-value",
		Op:    "add",
		Addr:  "",
	}
	result, err := CallExecNewTx(cfg, "manage", "Modify", modify)
	assert.Nil(t, err)
	assert.NotNil(t, result)

	_, err = CallExecNewTx(cfg, "nonexistent", "action", modify)
	assert.NotNil(t, err)
}
