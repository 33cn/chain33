package db

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGoLevelDBStatsPerLevelFileCounts 守住 Stats() 能读到真实的各层文件数。
//
// 原实现把 "leveldb.num-files-at-level{n}" 当字面量传给 goleveldb，而 goleveldb
// 只认具体的 level0..level6 ⇒ 该 key 永远取不到值，各层文件数在 Stats() 里一直是缺的。
func TestGoLevelDBStatsPerLevelFileCounts(t *testing.T) {
	dir := t.TempDir()
	d, err := NewGoLevelDB("statsprobe", dir, 64)
	require.NoError(t, err)
	defer d.Close()

	for i := 0; i < 200; i++ {
		require.NoError(t, d.Set([]byte(fmt.Sprintf("k%03d", i)), []byte("v")))
	}

	st := d.Stats()
	v, ok := st["leveldb.num-files-at-level0"]
	require.True(t, ok, "Stats() 未返回 level0 文件数：key 可能退回了字面量 {n}")
	_, err = strconv.Atoi(v)
	require.NoError(t, err)
}
