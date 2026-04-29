package dapp

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestLoadKVExpiredChecker(t *testing.T) {
	f, ok := LoadKVExpiredChecker("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, f)
}

func TestKVExpiredCheckerList(t *testing.T) {
	list := KVExpiredCheckerList()
	// Returns nil or empty slice when no checkers registered
	assert.Len(t, list, 0)
}

func TestLoadDriverNotFound(t *testing.T) {
	_, err := LoadDriver("nonexistent_driver_xyz", 0)
	assert.Equal(t, types.ErrUnRegistedDriver, err)
}
