// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"errors"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/queue"
	nty "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const blockedMempoolAddr = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

// TestMempoolCheckTxBlockedAccount mempool 入口拦截（统一深度判定，无 fork 门控）
func TestMempoolCheckTxBlockedAccount(t *testing.T) {
	_, mem := initEnv(1)
	cfg := mem.client.GetConfig()
	restore := types.SetBlockedAccountsForTest([]string{blockedMempoolAddr})
	defer restore()

	normalPriv := util.TestPrivkeyList[0]
	blockedPriv := util.TestPrivkeyList[1]
	normalAddr, _ := util.Genaddress()

	// 需要 header 才能走完整 checkTx
	mem.setHeader(&types.Header{Height: 1, BlockTime: 1})

	t.Run("hit from", func(t *testing.T) {
		tx := createTx(cfg, blockedPriv, normalAddr, 1000)
		msg := mem.checkTx(&queue.Message{Data: tx})
		require.NotNil(t, msg.Data)
		err, ok := msg.Data.(error)
		require.True(t, ok, "expect error data, got %T", msg.Data)
		assert.True(t, errors.Is(err, types.ErrBlockedAccount))
	})

	t.Run("hit to", func(t *testing.T) {
		tx := createTx(cfg, normalPriv, blockedMempoolAddr, 1000)
		msg := mem.checkTx(&queue.Message{Data: tx})
		require.NotNil(t, msg.Data)
		err, ok := msg.Data.(error)
		require.True(t, ok, "expect error data, got %T", msg.Data)
		assert.True(t, errors.Is(err, types.ErrBlockedAccount))
	})

	t.Run("normal pass", func(t *testing.T) {
		tx := createTx(cfg, normalPriv, normalAddr, 1000)
		msg := mem.checkTx(&queue.Message{Data: tx})
		// 正常交易 Data 仍为原 tx
		assert.Equal(t, tx, msg.Data)
	})

	// 深度判定关键用例：EVM 纯转账真实收款方藏在 Para 里，tx.To 是执行器地址。
	// 旧的浅层 From/To 判断会漏拦，改用 CheckTxBlockedAccountImmediate 后才能覆盖。
	t.Run("hit evm para hidden receiver", func(t *testing.T) {
		raw, err := address.NewBtcAddress(blockedMempoolAddr)
		require.NoError(t, err)
		action := &types.EVMContractAction4Chain33{
			Amount:       1,
			GasLimit:     10000,
			GasPrice:     1,
			Para:         raw.Hash160[:],
			ContractAddr: address.ExecAddress("evm"),
		}
		tx := &types.Transaction{
			Execer:  []byte("evm"),
			To:      address.ExecAddress("evm"),
			Payload: types.Encode(action),
			Fee:     1e6,
			ChainID: cfg.GetChainID(),
		}
		tx.Sign(types.SECP256K1, normalPriv)

		// 浅层判断（旧实现）看不到 Para 里的地址
		assert.False(t, types.IsBlockedAccount(tx.From()))
		assert.False(t, types.IsBlockedAccount(tx.To))
		// 深度判定能拦住
		msg := mem.checkTx(&queue.Message{Data: tx})
		berr, ok := msg.Data.(error)
		require.True(t, ok, "expect error data, got %T", msg.Data)
		assert.True(t, errors.Is(berr, types.ErrBlockedAccount))
	})
}

// createCommitDelayBlock 构造携带 CommitDelayTx 的 none 区块，内嵌延时交易 innerTx
func createCommitDelayBlock(innerTx *types.Transaction, height int64) *types.Block {
	action := &nty.NoneAction{
		Ty: nty.TyCommitDelayTxAction,
		Value: &nty.NoneAction_CommitDelayTx{CommitDelayTx: &nty.CommitDelayTx{
			DelayTx:             common.ToHex(types.Encode(innerTx)),
			RelativeDelayHeight: 1,
		}},
	}
	blockTx := &types.Transaction{Execer: []byte(nty.NoneX), Payload: types.Encode(action)}
	return &types.Block{Height: height, BlockTime: height, Txs: []*types.Transaction{blockTx}}
}

// TestEventAddDelayTxBlockedAccount 延时交易提交入口拦截
func TestEventAddDelayTxBlockedAccount(t *testing.T) {
	// delayCache 容量为 poolCacheSize/2，取 10 保证正常交易可入 cache
	_, mem := initEnv(10)
	restore := types.SetBlockedAccountsForTest([]string{blockedMempoolAddr})
	defer restore()
	cfg := mem.client.GetConfig()

	normalPriv := util.TestPrivkeyList[0]
	blockedPriv := util.TestPrivkeyList[1]
	normalAddr, _ := util.Genaddress()

	sendDelayTx := func(delayTx *types.DelayTx) *types.Reply {
		msg := queue.NewMessage(0, "mempool", types.EventAddDelayTx, delayTx)
		mem.eventAddDelayTx(msg)
		resp, err := mem.client.Wait(msg)
		require.NoError(t, err)
		reply, ok := resp.GetData().(*types.Reply)
		require.True(t, ok, "expect *types.Reply, got %T", resp.GetData())
		return reply
	}

	// 命中黑名单：提交即拒，不进 delayCache
	reply := sendDelayTx(&types.DelayTx{Tx: createTx(cfg, blockedPriv, normalAddr, 1000), EndDelayTime: 100})
	require.False(t, reply.IsOk)
	assert.Contains(t, string(reply.Msg), types.ErrBlockedAccount.Error())
	assert.Equal(t, 0, len(mem.cache.delayCache.hashCache))

	// 正常延时交易仍可入 cache
	reply = sendDelayTx(&types.DelayTx{Tx: createTx(cfg, normalPriv, normalAddr, 1000), EndDelayTime: 100})
	require.True(t, reply.IsOk, string(reply.Msg))
	assert.Equal(t, 1, len(mem.cache.delayCache.hashCache))
}

// TestAddDelayTxBlockedAccount 区块 CommitDelayTx 入 cache 拦截
func TestAddDelayTxBlockedAccount(t *testing.T) {
	_, mem := initEnv(1)
	restore := types.SetBlockedAccountsForTest([]string{blockedMempoolAddr})
	defer restore()
	cfg := mem.client.GetConfig()

	blockedPriv := util.TestPrivkeyList[1]
	normalAddr, _ := util.Genaddress()

	cache := newDelayTxCache(10)

	// 命中黑名单的内嵌延时交易不入 cache
	mem.addDelayTx(cache, createCommitDelayBlock(createTx(cfg, blockedPriv, normalAddr, 1000), 10))
	assert.Equal(t, 0, len(cache.hashCache))

	// 正常内嵌延时交易仍可入 cache
	mem.addDelayTx(cache, createCommitDelayBlock(createTx(cfg, util.TestPrivkeyList[0], normalAddr, 1000), 11))
	assert.Equal(t, 1, len(cache.hashCache))
}
