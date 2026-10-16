// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dapp

import (
	"testing"

	"github.com/33cn/chain33/client/mocks"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 本文件验证 DriverBase 的框架层创世高度守卫:
// Genesis 动作只允许在高度 0 执行, 执行器自身漏写 height 校验时由框架统一兜底。
// 守卫覆盖 Exec 与 ExecLocal/ExecDelLocal 派发, 且只针对 action 名为 Genesis 的交易。
// 用例统一以 TestGenesisGuard 开头, 便于 -run TestGenesisGuard 一次跑全。

// genesisGuardDriver 只提供 Genesis/Transfer 两个动作, 且其 Genesis 处理函数无条件成功,
// 因此用例结果只可能来自框架层守卫; localCalled/delLocalCalled 用于确认本地派发是否发生。
type genesisGuardDriver struct {
	*DriverBase
	genesisCalled  bool
	transferCalled bool
	localCalled    bool
	delLocalCalled bool
}

func newGenesisGuardDriver(cfg *types.Chain33Config) *genesisGuardDriver {
	d := &genesisGuardDriver{DriverBase: &DriverBase{}}
	d.SetChild(d)
	// 复用 coins 的 ExecutorType: 它把 genesis 动作解析为 "Genesis",
	// 与 coins/coinsx/token/ticket 的 action 命名一致。
	ety := cty.NewType(cfg)
	ety.InitFuncList(types.ListMethod(d))
	d.SetExecutorType(ety)
	return d
}

func (d *genesisGuardDriver) GetDriverName() string { return "genesisguard" }

func (d *genesisGuardDriver) Exec_Genesis(_ *types.AssetsGenesis, _ *types.Transaction, _ int) (*types.Receipt, error) {
	d.genesisCalled = true
	return &types.Receipt{Ty: types.ExecOk}, nil
}

func (d *genesisGuardDriver) Exec_Transfer(_ *types.AssetsTransfer, _ *types.Transaction, _ int) (*types.Receipt, error) {
	d.transferCalled = true
	return &types.Receipt{Ty: types.ExecOk}, nil
}

func (d *genesisGuardDriver) ExecLocal_Genesis(_ *types.AssetsGenesis, _ *types.Transaction, _ *types.ReceiptData, _ int) (*types.LocalDBSet, error) {
	d.localCalled = true
	return &types.LocalDBSet{KV: []*types.KeyValue{{Key: []byte("genesis-local"), Value: []byte("1")}}}, nil
}

func (d *genesisGuardDriver) ExecDelLocal_Genesis(_ *types.AssetsGenesis, _ *types.Transaction, _ *types.ReceiptData, _ int) (*types.LocalDBSet, error) {
	d.delLocalCalled = true
	return &types.LocalDBSet{KV: []*types.KeyValue{{Key: []byte("genesis-local"), Value: nil}}}, nil
}

func newGenesisGuardGenesisTx() *types.Transaction {
	payload := &cty.CoinsAction{
		Ty: cty.CoinsActionGenesis,
		Value: &cty.CoinsAction_Genesis{
			Genesis: &types.AssetsGenesis{Amount: 100, ReturnAddress: "1HUiTRFvp6HvW6eacgV9EoBSgroRDiUsMs"},
		},
	}
	return &types.Transaction{Execer: []byte(cty.CoinsX), Payload: types.Encode(payload)}
}

func newGenesisGuardTransferTx() *types.Transaction {
	payload := &cty.CoinsAction{
		Ty: cty.CoinsActionTransfer,
		Value: &cty.CoinsAction_Transfer{
			Transfer: &types.AssetsTransfer{Amount: 100, To: "1HUiTRFvp6HvW6eacgV9EoBSgroRDiUsMs"},
		},
	}
	return &types.Transaction{Execer: []byte(cty.CoinsX), Payload: types.Encode(payload)}
}

func newGenesisGuardAPI(cfg *types.Chain33Config) *mocks.QueueProtocolAPI {
	api := &mocks.QueueProtocolAPI{}
	api.On("GetConfig", mock.Anything).Return(cfg)
	return api
}

func newGenesisGuardDriverAt(cfg *types.Chain33Config, height int64) *genesisGuardDriver {
	d := newGenesisGuardDriver(cfg)
	d.SetAPI(newGenesisGuardAPI(cfg))
	d.SetEnv(height, 1, 1)
	return d
}

func newGenesisGuardCfg() *types.Chain33Config {
	// 默认配置是 local, 所有 fork 归零, 因此这里显式锚定守卫所依赖的分叉高度。
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.SetFork(types.ForkGenesisOnlyAtHeightZero, 0)
	return cfg
}

// 高度 0 时创世动作照常执行(创世块本身不受影响)。
func TestGenesisGuardAllowedAtHeightZero(t *testing.T) {
	d := newGenesisGuardDriverAt(newGenesisGuardCfg(), 0)
	receipt, err := d.Exec(newGenesisGuardGenesisTx(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.True(t, d.genesisCalled)
}

// 高度 != 0 且分叉生效时, 即使执行器的 Exec_Genesis 不做任何校验也会被框架拦截。
func TestGenesisGuardRejectedAtNonZeroHeight(t *testing.T) {
	d := newGenesisGuardDriverAt(newGenesisGuardCfg(), 1)
	receipt, err := d.Exec(newGenesisGuardGenesisTx(), 0)
	assert.Equal(t, types.ErrReRunGenesis, err)
	assert.Nil(t, receipt)
	assert.False(t, d.genesisCalled, "被拦截时不应调用执行器的 Exec_Genesis")
}

// 分叉关闭(-1 -> MaxHeight)时保持历史行为, 旧区块可以原样重放。
func TestGenesisGuardNonZeroHeightWithoutFork(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.SetFork(types.ForkGenesisOnlyAtHeightZero, types.MaxHeight)

	d := newGenesisGuardDriverAt(cfg, 1)
	receipt, err := d.Exec(newGenesisGuardGenesisTx(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.True(t, d.genesisCalled)
}

// 分叉高度按 IsFork 语义生效: 高度 < 分叉高度放行, 达到分叉高度拦截。
func TestGenesisGuardRejectedFromForkHeight(t *testing.T) {
	const forkHeight = int64(100)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.SetFork(types.ForkGenesisOnlyAtHeightZero, forkHeight)

	d := newGenesisGuardDriverAt(cfg, forkHeight-1)
	receipt, err := d.Exec(newGenesisGuardGenesisTx(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.True(t, d.genesisCalled)

	d = newGenesisGuardDriverAt(cfg, forkHeight)
	receipt, err = d.Exec(newGenesisGuardGenesisTx(), 0)
	assert.Equal(t, types.ErrReRunGenesis, err)
	assert.Nil(t, receipt)
	assert.False(t, d.genesisCalled)
}

// 守卫只匹配 Genesis 动作, 非创世动作在高度 != 0 不受影响。
func TestGenesisGuardNonGenesisActionAtNonZeroHeight(t *testing.T) {
	d := newGenesisGuardDriverAt(newGenesisGuardCfg(), 1)
	receipt, err := d.Exec(newGenesisGuardTransferTx(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.True(t, d.transferCalled)
	assert.False(t, d.genesisCalled)
}

// 执行器未注入 API 时不 panic(真实执行路径一定已 SetAPI), 行为退化为不拦截。
func TestGenesisGuardWithoutAPI(t *testing.T) {
	cfg := newGenesisGuardCfg()
	d := newGenesisGuardDriver(cfg)
	d.SetEnv(1, 1, 1)
	receipt, err := d.Exec(newGenesisGuardGenesisTx(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.True(t, d.genesisCalled)
}

// 本地派发与 Exec 共用同一守卫: 被拒绝的创世交易不得写本地索引, 高度 0 的创世块不受影响。
func TestGenesisGuardLocalDispatchAtNonZeroHeight(t *testing.T) {
	cfg := newGenesisGuardCfg()
	tx := newGenesisGuardGenesisTx()
	receipt := &types.ReceiptData{Ty: types.ExecErr}

	d := newGenesisGuardDriverAt(cfg, 1)
	lset, err := d.ExecLocal(tx, receipt, 0)
	assert.NoError(t, err)
	assert.False(t, d.localCalled, "被拒绝的创世交易不应派发 ExecLocal_Genesis")
	assert.Empty(t, lset.GetKV())

	lset, err = d.ExecDelLocal(tx, receipt, 0)
	assert.NoError(t, err)
	assert.False(t, d.delLocalCalled, "被拒绝的创世交易不应派发 ExecDelLocal_Genesis")
	assert.Empty(t, lset.GetKV())

	// 高度 0: 本地派发照常
	d = newGenesisGuardDriverAt(cfg, 0)
	lset, err = d.ExecLocal(tx, &types.ReceiptData{Ty: types.ExecOk}, 0)
	assert.NoError(t, err)
	assert.True(t, d.localCalled)
	assert.Len(t, lset.GetKV(), 1)
}

// 分叉关闭时本地派发同样保持历史行为。
func TestGenesisGuardLocalDispatchWithoutFork(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.SetFork(types.ForkGenesisOnlyAtHeightZero, types.MaxHeight)

	d := newGenesisGuardDriverAt(cfg, 1)
	lset, err := d.ExecLocal(newGenesisGuardGenesisTx(), &types.ReceiptData{Ty: types.ExecOk}, 0)
	assert.NoError(t, err)
	assert.True(t, d.localCalled)
	assert.Len(t, lset.GetKV(), 1)
}
