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
