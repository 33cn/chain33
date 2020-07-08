// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"
	"time"

	drivers "github.com/33cn/chain33/system/dapp"

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
	drivers.Register(cfg, "notAllow", newAllowApp, 0)
	var txs []*types.Transaction
	addr, _ := util.Genaddress()
	genkey := util.TestPrivkeyList[0]
	tx := util.CreateCoinsTx(cfg, genkey, addr, types.Coin)
	txs = append(txs, tx)
	tx.Execer = []byte("notAllow")
	tx1 := *tx
	tx1.Execer = []byte("user.p.para.notAllow")
	tx2 := *tx
	tx2.Execer = []byte("user.p.test.notAllow")
	// local fork值 为0, 测试不出fork前的情况
	//types.SetTitleOnlyForTest("chain33")
	t.Log("get fork value", cfg.GetFork("ForkCacheDriver"), cfg.GetTitle())
	cases := []struct {
		height     int64
		driverName string
		tx         *types.Transaction
		index      int
	}{
		{cfg.GetFork("ForkCacheDriver") - 1, "notAllow", tx, 0},
		//相当于平行链交易 使用none执行器
		{cfg.GetFork("ForkCacheDriver") - 1, "none", &tx1, 0},
		// index > 0的allow, 但由于是fork之前，所以不会检查allow, 直接采用上一个缓存的，即none执行器
		{cfg.GetFork("ForkCacheDriver") - 1, "none", &tx1, 1},
		// 不能通过allow判定 加载none执行器
		{cfg.GetFork("ForkCacheDriver"), "none", &tx2, 0},
		// fork之后需要重新判定, index>0 通过allow判定
		{cfg.GetFork("ForkCacheDriver"), "notAllow", &tx2, 1},
	}
	ctx := &executorCtx{
		stateHash:  nil,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	execute := newExecutor(ctx, exec, nil, txs, nil)
	for idx, c := range cases {
		if execute.height != c.height {
			execute.driverCache = make(map[string]drivers.Driver)
			execute.height = c.height
		}
		driver := execute.loadDriver(c.tx, c.index)
		assert.Equal(t, c.driverName, driver.GetDriverName(), "case index=%d", idx)
	}
}

type notAllowApp struct {
	*drivers.DriverBase
}

func newAllowApp() drivers.Driver {
	app := &notAllowApp{DriverBase: &drivers.DriverBase{}}
	app.SetChild(app)
	return app
}

func (app *notAllowApp) GetDriverName() string {
	return "notAllow"
}

func (app *notAllowApp) Allow(tx *types.Transaction, index int) error {

	if app.GetHeight() == 0 {
		return types.ErrActionNotSupport
	}
	// 这里简单模拟 同比交易不同action 的allow限定，大于0 只是配合上述测试用例
	if index > 0 || string(tx.Execer) == "notAllow" {
		return nil
	}
	return types.ErrActionNotSupport
}
