// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"errors"
	"testing"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockedTestAddr 是 TestPrivkeyList[1] 派生的地址，测试里注入黑名单
const blockedTestAddr = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

func newBlockedExecutor(t *testing.T, blockedAddrs []string) (*executor, *types.Chain33Config) {
	t.Helper()
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	// local 标题下 SetAllFork(0)，ForkAccountBlacklist 从高度 0 启用
	restore := types.SetBlockedAccountsForTest(blockedAddrs)
	t.Cleanup(restore)
	ctx := &executorCtx{
		height:     1,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
	}
	return newExecutor(ctx, exec, nil, nil, nil), cfg
}

func TestCheckTxBlockedAccount(t *testing.T) {
	execute, cfg := newBlockedExecutor(t, []string{blockedTestAddr})
	normalPriv := util.TestPrivkeyList[0]
	blockedPriv := util.TestPrivkeyList[1]
	normalAddr, _ := util.Genaddress()

	t.Run("hit from", func(t *testing.T) {
		tx := util.CreateCoinsTx(cfg, blockedPriv, normalAddr, 1000)
		err := execute.checkTx(tx, 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, types.ErrBlockedAccount))
	})

	t.Run("hit to", func(t *testing.T) {
		tx := util.CreateCoinsTx(cfg, normalPriv, blockedTestAddr, 1000)
		err := execute.checkTx(tx, 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, types.ErrBlockedAccount))
	})

	t.Run("normal pass", func(t *testing.T) {
		tx := util.CreateCoinsTx(cfg, normalPriv, normalAddr, 1000)
		assert.NoError(t, execute.checkTx(tx, 0))
	})
}

func TestCheckTxGroupBlockedAccount(t *testing.T) {
	execute, cfg := newBlockedExecutor(t, []string{blockedTestAddr})
	normalPriv := util.TestPrivkeyList[0]
	addr2, priv2 := util.Genaddress()
	addr3, priv3 := util.Genaddress()

	// 组内第二笔 to 命中黑名单 -> 整组 checkTxGroup 返回 ErrBlockedAccount
	txs := []*types.Transaction{
		util.CreateCoinsTx(cfg, normalPriv, addr2, 1000),
		util.CreateCoinsTx(cfg, priv2, blockedTestAddr, 1000),
		util.CreateCoinsTx(cfg, priv3, addr3, 1000),
	}
	txgroup, err := types.CreateTxGroup(txs, cfg.GetMinTxFeeRate())
	require.NoError(t, err)
	err = execute.checkTxGroup(txgroup, 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, types.ErrBlockedAccount))

	// 全部正常 -> 通过
	txs2 := []*types.Transaction{
		util.CreateCoinsTx(cfg, normalPriv, addr2, 1000),
		util.CreateCoinsTx(cfg, priv2, addr3, 1000),
	}
	txgroup2, err := types.CreateTxGroup(txs2, cfg.GetMinTxFeeRate())
	require.NoError(t, err)
	assert.NoError(t, execute.checkTxGroup(txgroup2, 0))
}
