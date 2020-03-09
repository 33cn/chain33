// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addrindex

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
	tx2 := &types.Transaction{Fee: 1000000, Execer: []byte("none"), To: "15xMtKs3Ca89X1KNoPZibChBtRPa5BYth1"}
	tx2.Sign(types.SECP256K1, priv)
	txs = append(txs, tx, tx2)
	var stateHash [32]byte
	stateHash[0] = 30

	cfg := types.NewChain33ConfigNoInit("")
	cfg.S("dbversion", int64(1))
	api := new(apimock.QueueProtocolAPI)
	api.On("GetConfig").Return(cfg)
	ver := cfg.GInt("dbversion")
	assert.Equal(t, int64(1), ver)

	p := newAddrindex()
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
	assert.NoError(t, err)
	for _, kv := range kvs {
		err = kvdb.Set(kv.Key, kv.Value)
		assert.NoError(t, err)
	}

	// found
	var req types.ReqKey
	req.Key = []byte(tx2.From())
	counts, err := p.Query("GetAddrTxsCount", types.Encode(&req))
	assert.Nil(t, err)
	c := counts.(*types.Int64)
	assert.Equal(t, int64(2), c.GetData())

	req2 := types.ReqAddr{
		Addr:   tx2.From(),
		Count:  1,
		Flag:   0,
		Height: -1,
	}

	r2, err := p.Query("GetTxsByAddr", types.Encode(&req2))
	assert.Nil(t, err)
	result2 := r2.(*types.ReplyTxInfos)
	assert.Equal(t, 1, len(result2.GetTxInfos()))

	kvs, err = p.ExecDelLocal(detail)
	assert.NoError(t, err)
	for _, kv := range kvs {
		assert.Contains(t, string(kv.Key), "LODBP-"+name+"-")
	}
}
