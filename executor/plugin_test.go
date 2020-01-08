package executor

import (
	"testing"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestPlugin(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	ctx := &executorCtx{
		stateHash:  nil,
		height:     1,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	var txs []*types.Transaction
	addr, priv := util.Genaddress()
	tx := util.CreateCoinsTx(cfg, priv, addr, types.Coin)
	tx.Sign(types.SECP256K1, priv)
	txs = append(txs, tx)
	var stateHash [32]byte
	stateHash[0] = 30
	for _, plugin := range globalPlugins {
		detail := &types.BlockDetail{
			Block:    &types.Block{Txs: txs, StateHash: stateHash[:]},
			Receipts: []*types.ReceiptData{{}},
		}
		executor := newExecutor(ctx, exec, kvdb, txs, nil)
		_, _, err := plugin.CheckEnable(executor, false)
		assert.NoError(t, err)
		kvs, err := plugin.ExecLocal(executor, detail)
		assert.NoError(t, err)
		for _, kv := range kvs {
			err = kvdb.Set(kv.Key, kv.Value)
			assert.NoError(t, err)
		}
		_, err = plugin.ExecDelLocal(executor, detail)
		assert.NoError(t, err)
	}
}

func TestPluginBase(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	base := new(pluginBase)
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	ctx := &executorCtx{
		stateHash:  nil,
		height:     0,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	executor := newExecutor(ctx, exec, kvdb, nil, nil)
	_, _, err := base.checkFlag(executor, nil, true)
	assert.NoError(t, err)

	k := []byte("test")
	v := types.Encode(&types.Int64{})
	err = kvdb.Set(k, v)
	assert.NoError(t, err)
	_, _, err = base.checkFlag(executor, k, true)
	assert.NoError(t, err)
}
