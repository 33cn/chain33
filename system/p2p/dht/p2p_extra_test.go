package dht

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestIsRestartFalse(t *testing.T) {
	p := &P2P{restart: 0}
	assert.False(t, p.isRestart())
}

func TestIsRestartTrue(t *testing.T) {
	p := &P2P{restart: 1}
	assert.True(t, p.isRestart())
}

func TestNewDBDefaults(t *testing.T) {
	dir := t.TempDir()
	db := newDB("", "", dir, 0)
	assert.NotNil(t, db)
	db.Close()
}

func TestAirDropIndexInRange(t *testing.T) {
	seen := map[int32]struct{}{}
	for seed := int64(0); seed < 64; seed++ {
		idx := airDropIndex(seed)
		assert.GreaterOrEqual(t, idx, int32(types.AirDropMinIndex))
		assert.LessOrEqual(t, idx, int32(types.AirDropMaxIndex))
		assert.Equal(t, idx, airDropIndex(seed))
		seen[idx] = struct{}{}
	}
	assert.Greater(t, len(seen), 1)
}

func TestNewDBCustomName(t *testing.T) {
	dir := t.TempDir()
	db := newDB("testdb", "leveldb", dir, 128)
	assert.NotNil(t, db)
	db.Close()
}
