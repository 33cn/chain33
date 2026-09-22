package dht

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestAirDropIndexInRangeAndDistinct(t *testing.T) {
	seen := map[int32]struct{}{}
	for i := 0; i < 64; i++ {
		idx := airDropIndex(int64(i) + 1)
		assert.GreaterOrEqual(t, idx, int32(types.AirDropMinIndex))
		assert.LessOrEqual(t, idx, int32(types.AirDropMaxIndex))
		seen[idx] = struct{}{}
	}
	// 同一秒内不同进程不能稳定得到同一个索引
	sameSecond := int64(1_700_000_000) * int64(1e9)
	assert.NotEqual(t, airDropIndex(sameSecond+1), airDropIndex(sameSecond+2))
	assert.Greater(t, len(seen), 1)
}

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

func TestNewDBCustomName(t *testing.T) {
	dir := t.TempDir()
	db := newDB("testdb", "leveldb", dir, 128)
	assert.NotNil(t, db)
	db.Close()
}
