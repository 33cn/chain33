package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCount(t *testing.T) {
	dir, leveldb, kvdb := getdb()
	defer dbclose(dir, leveldb)
	count := NewCount("prefix", "name#hello", kvdb)
	count.Inc()
	count.Dec()
	count.Inc()
	i, err := count.Get()
	assert.Nil(t, err)
	assert.Equal(t, i, int64(1))
	kvs, err := count.Save()
	assert.Nil(t, err)
	setKV(leveldb, kvs)

	count = NewCount("prefix", "name#hello", kvdb)
	i, err = count.Get()
	assert.Nil(t, err)
	assert.Equal(t, i, int64(1))

	count.Set(2)
	i, err = count.Get()
	assert.Nil(t, err)
	assert.Equal(t, i, int64(2))
}
