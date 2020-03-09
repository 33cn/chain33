// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txindex

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
	tx2.Sign(types.SECP256K1, priv)
	txs = append(txs, tx, tx2)
	var stateHash [32]byte
	stateHash[0] = 30

	cfg := types.NewChain33ConfigNoInit("")
	cfg.S("dbversion", 1)
	api := new(apimock.QueueProtocolAPI)
	api.On("GetConfig").Return(cfg)

	p := newTxindex()
	p.SetEnv(2, 20, 300)
	p.SetLocalDB(kvdb)
	p.SetAPI(api)

	detail := &types.BlockDetail{
		Block:    &types.Block{Txs: txs, StateHash: stateHash[:]},
		Receipts: []*types.ReceiptData{{}, {}},
	}

	_, _, err := p.CheckEnable(true)
	assert.NoError(t, err)
	kvs, err := p.ExecLocal(detail)
	assert.NoError(t, err)
	for _, kv := range kvs {
		assert.Contains(t, string(kv.Key), "LODBP-"+name+"-")
	}
	for _, kv := range kvs {
		err = kvdb.Set(kv.Key, kv.Value)
		assert.NoError(t, err)
	}

	// found
	var req types.ReqKey
	req.Key = tx2.Hash()
	found, err := p.Query("HasTx", types.Encode(&req))
	assert.Nil(t, err)
	f := found.(*types.Int32)
	assert.Equal(t, int32(1), f.GetData())

	result, err := p.Query("GetTx", types.Encode(&req))
	assert.Nil(t, err)
	r := result.(*types.TxResult)
	assert.Equal(t, int64(2), r.Height)

	// not found
	req.Key = []byte("not exist tx hash")
	found, err = p.Query("HasTx", types.Encode(&req))
	assert.Nil(t, err)
	f = found.(*types.Int32)
	assert.Equal(t, int32(0), f.GetData())

	_, err = p.Query("GetTx", types.Encode(&req))
	assert.Equal(t, types.ErrNotFound, err)

	kvs, err = p.ExecDelLocal(detail)
	assert.NoError(t, err)
	for _, kv := range kvs {
		assert.Contains(t, string(kv.Key), "LODBP-"+name+"-")
	}
}
