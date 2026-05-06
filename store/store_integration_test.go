package store_test

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestStoreGetSet(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	header, err := api.GetLastHeader()
	assert.Nil(t, err)
	assert.NotNil(t, header.GetStateHash())
}

func TestStoreBlockQuery(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	blocks, err := api.GetBlocks(&types.ReqBlocks{Start: 0, End: 0, IsDetail: false})
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, len(blocks.Items), 1)
}

func TestStoreHeadersQuery(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	headers, err := api.GetHeaders(&types.ReqBlocks{Start: 0, End: 0, IsDetail: false})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(headers.Items))
}

func TestStoreLocalDB(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	seq, err := api.GetLastBlockSequence()
	assert.Nil(t, err)
	assert.NotNil(t, seq)
}

func TestStoreQueryByHash(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	block := mocker.GetBlock(0)
	assert.NotNil(t, block)

	api := mocker.GetAPI()
	blocks, err := api.GetBlocks(&types.ReqBlocks{Start: 0, End: 0, IsDetail: false})
	assert.Nil(t, err)
	if len(blocks.Items) > 0 {
		assert.NotNil(t, blocks.Items[0].Block.Hash(cfg))
	}
}
