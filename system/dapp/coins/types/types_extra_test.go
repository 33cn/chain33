package types

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestGetTy(t *testing.T) {
	c := &CoinsAction{Ty: 3}
	assert.Equal(t, int32(3), c.GetTy())
	assert.Equal(t, int32(0), (&CoinsAction{}).GetTy())
}

func TestGetWithdraw(t *testing.T) {
	withdraw := &types.AssetsWithdraw{Amount: 100}
	c := &CoinsAction{Value: &CoinsAction_Withdraw{Withdraw: withdraw}}
	assert.NotNil(t, c.GetWithdraw())
	assert.Equal(t, int64(100), c.GetWithdraw().Amount)
	assert.Nil(t, (&CoinsAction{}).GetWithdraw())
}

func TestGetTransferToExec(t *testing.T) {
	transfer := &types.AssetsTransferToExec{Amount: 200}
	c := &CoinsAction{Value: &CoinsAction_TransferToExec{TransferToExec: transfer}}
	assert.NotNil(t, c.GetTransferToExec())
	assert.Equal(t, int64(200), c.GetTransferToExec().Amount)
	assert.Nil(t, (&CoinsAction{}).GetTransferToExec())
}

func TestInitFork(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	assert.NotPanics(t, func() {
		InitFork(cfg)
	})
}
