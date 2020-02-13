// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fee

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"

	apimock "github.com/33cn/chain33/client/mocks"
)

func TestPlugin(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)

	var txs []*types.Transaction
	_, priv := util.Genaddress()
	tx := &types.Transaction{Fee: 1000000, Execer: []byte("none"), To: "15xMtKs3Ca89X1KNoPZibChBtRPa5BYth0"}
	tx.Sign(types.SECP256K1, priv)
	tx2 := &types.Transaction{Fee: 2000000, Execer: []byte("none"), To: "15xMtKs3Ca89X1KNoPZibChBtRPa5BYth1"}
	tx.Sign(types.SECP256K1, priv)
	txs = append(txs, tx, tx2)
	var stateHash [32]byte
	stateHash[0] = 30

	cfg := types.NewChain33ConfigNoInit("")
	cfg.S("dbversion", 1)
	api := new(apimock.QueueProtocolAPI)
	api.On("GetConfig").Return(cfg)

	p := newFee()
	p.SetEnv(2, 20, 300)
	p.SetLocalDB(kvdb)
	p.SetAPI(api)

	detail := &types.BlockDetail{
		Block:    &types.Block{Txs: txs, StateHash: stateHash[:]},
		Receipts: []*types.ReceiptData{{}, {}},
	}

	_, _, err := p.CheckEnable(false)
	assert.NoError(t, err)
	kvs, err := p.ExecLocal(detail)
	for _, kv := range kvs {
		assert.Contains(t, string(kv.Key), "LODBP-"+name+"-")
	}
	var totalFee types.TotalFee
	err = types.Decode(kvs[0].Value, &totalFee)
	assert.Nil(t, err)
	assert.Equal(t, int64(3000000), totalFee.GetFee())
	assert.NoError(t, err)
	for _, kv := range kvs {
		err = kvdb.Set(kv.Key, kv.Value)
		assert.NoError(t, err)
	}
	kvs, err = p.ExecDelLocal(detail)
	assert.NoError(t, err)
	for _, kv := range kvs {
		assert.Contains(t, string(kv.Key), "LODBP-"+name+"-")
	}
}
