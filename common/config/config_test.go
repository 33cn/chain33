package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
