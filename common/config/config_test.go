package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

func TestSubConfig(t *testing.T) {
	cfg, err := InitSubModule("testdata/chain33.toml")
	assert.Equal(t, 0, len(cfg.Consensus))
	assert.Equal(t, 2, len(cfg.Store))
	assert.Equal(t, 1, len(cfg.Exec))
	assert.Equal(t, 1, len(cfg.Wallet))
	assert.Nil(t, err)
}

func TestConfig(t *testing.T) {
	cfg, err := Init("testdata/chain33.toml")
	assert.Equal(t, cfg.Fork.System["ForkV16Withdraw"], int64(480000))
	assert.Equal(t, cfg.Fork.Sub["token"]["Enable"], int64(100899))
	assert.Nil(t, err)
}

func TestBityuanInit(t *testing.T) {
	cfg, err := Init("../../cmd/chain33/bityuan.toml")
	assert.Equal(t, int64(200000), cfg.Fork.System["ForkWithdraw"])
	assert.Equal(t, int64(0), cfg.Fork.Sub["token"]["Enable"])
	assert.Nil(t, err)
	types.InitForkConfig(cfg.Title, cfg.Fork)
}
