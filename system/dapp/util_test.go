package dapp

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestKVCreator(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	creator := NewKVCreator(kvdb)
	creator.AddKV([]byte("a"), []byte("b"))
	_, err := kvdb.Get([]byte("a"))
	assert.Equal(t, err, types.ErrNotFound)
	creator.Add([]byte("a"), []byte("b"))
	value, err := kvdb.Get([]byte("a"))
	assert.Equal(t, err, nil)
	assert.Equal(t, value, []byte("b"))
}

func TestHeightIndexStr(t *testing.T) {
	assert.Equal(t, "000000000000100001", HeightIndexStr(1, 1))
}
