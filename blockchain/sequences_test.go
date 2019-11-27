package blockchain

import (
	"io/ioutil"
	"os"
	"testing"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func Test_recommendSeqs(t *testing.T) {
	seqs := recommendSeqs(2, 10)
	assert.Equal(t, 3, len(seqs))

	seqs = recommendSeqs(20, 10)
	assert.Equal(t, 6, len(seqs))
	assert.Equal(t, []int64{20, 19, 18, 17, 16, 0}, seqs)

	seqs = recommendSeqs(10000, 6)
	assert.Equal(t, 6, len(seqs))
	assert.Equal(t, []int64{10000, 9999, 9998, 9898, 9798, 9698}, seqs)

	seqs = recommendSeqs(200, 6)
	assert.Equal(t, 5, len(seqs))
	assert.Equal(t, []int64{200, 199, 198, 98, 0}, seqs)
}

func Test_loadSequanceForAddCallback(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	store := &BlockStore{
		db:           dbm.NewDB("blockchain", "leveldb", dir, 100),
		saveSequence: true,
	}
	cb := &types.BlockSeqCB{
		Name:          "D7",
		URL:           "http://192.1.1.1:101",
		Encode:        "json",
		LastSequence:  11,
		LastHeight:    11,
		LastBlockHash: "0xae42f51d3e21ef6169ab2c40892b88359d069008178040cde07e87fc8f37fdd0",
	}
	loadSequanceForAddCallback(store, cb)
}
