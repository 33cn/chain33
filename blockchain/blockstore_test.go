package blockchain_test

import (
	"testing"

	"io/ioutil"
	"os"

	"github.com/33cn/chain33/blockchain"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStoreUpgradeMeta(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	blockStore := blockchain.NewBlockStore(nil, blockStoreDB, nil)
	require.NotNil(t, blockStore)

	meta, err := blockStore.GetStoreUpgradeMeta()
	require.NoError(t, err)
	require.Equal(t, meta.Version, "0.0.0")

	meta.Version = "1.0.0"
	err = blockStore.SetStoreUpgradeMeta(meta)
	require.NoError(t, err)
	meta, err = blockStore.GetStoreUpgradeMeta()
	require.NoError(t, err)
	require.Equal(t, meta.Version, "1.0.0")
}
