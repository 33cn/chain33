// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"
	"time"

	"strings"

	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestLoadDriverFork(t *testing.T) {
	str := types.GetDefaultCfgstring()
	new := strings.Replace(str, "Title=\"local\"", "Title=\"chain33\"", 1)
	exec, _ := initEnv(new)
	cfg := exec.client.GetConfig()
	execInit(cfg)

	var txs []*types.Transaction
	addr, _ := util.Genaddress()
	genkey := util.TestPrivkeyList[0]
	tx := util.CreateCoinsTx(cfg, genkey, addr, types.Coin)
	txs = append(txs, tx)

	// local fork值 为0, 测试不出fork前的情况
	//types.SetTitleOnlyForTest("chain33")
	t.Log("get fork value", cfg.GetFork("ForkCacheDriver"), cfg.GetTitle())
	cases := []struct {
		height    int64
		cacheSize int
	}{
		{cfg.GetFork("ForkCacheDriver") - 1, 1},
		{cfg.GetFork("ForkCacheDriver"), 0},
	}
	for _, c := range cases {
		ctx := &executorCtx{
			stateHash:  nil,
			height:     c.height,
			blocktime:  time.Now().Unix(),
			difficulty: 1,
			mainHash:   nil,
			parentHash: nil,
		}
		execute := newExecutor(ctx, exec, nil, txs, nil)
		_ = execute.loadDriver(tx, 0)
		assert.Equal(t, c.cacheSize, len(execute.execCache))
	}
}
