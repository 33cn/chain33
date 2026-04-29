package blockchain_test

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestChainStartAndBlocks(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	bc := mocker.GetBlockChain()
	assert.NotNil(t, bc)

	// Test get last header
	api := mocker.GetAPI()
	lastHeader, err := api.GetLastHeader()
	assert.Nil(t, err)
	assert.NotNil(t, lastHeader)
	assert.GreaterOrEqual(t, lastHeader.Height, int64(0))
}

func TestBlockHeightGrowth(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	// Wait for at least one block
	err = mocker.WaitHeight(1)
	assert.Nil(t, err)

	// Check height grew
	api := mocker.GetAPI()
	header, err := api.GetLastHeader()
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, header.Height, int64(1))
}

func TestGetBlocksDetails(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(1)
	assert.Nil(t, err)

	api := mocker.GetAPI()
	blocks, err := api.GetBlocks(&types.ReqBlocks{Start: 0, End: 1, IsDetail: true})
	assert.Nil(t, err)
	assert.NotNil(t, blocks)
	assert.GreaterOrEqual(t, len(blocks.Items), 1)
}

func TestGetBlockHash(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(1)
	assert.Nil(t, err)

	api := mocker.GetAPI()
	blocks, err := api.GetBlocks(&types.ReqBlocks{Start: 1, End: 1, IsDetail: false})
	assert.Nil(t, err)
	if len(blocks.Items) > 0 {
		assert.NotNil(t, blocks.Items[0].Block.Hash(cfg))
	}
}

func TestGetHeadersIntegration(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(1)
	assert.Nil(t, err)

	api := mocker.GetAPI()
	headers, err := api.GetHeaders(&types.ReqBlocks{Start: 0, End: 0, IsDetail: false})
	assert.Nil(t, err)
	assert.NotNil(t, headers)
	assert.GreaterOrEqual(t, len(headers.Items), 1)
}

func TestGetChainConfig(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	// Verify chain config is accessible
	assert.NotEmpty(t, cfg.GetTitle())
	assert.NotZero(t, cfg.GetChainID())
	assert.NotZero(t, cfg.GetMinTxFeeRate())
}

func TestGetBlockOverview(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(1)
	assert.Nil(t, err)

	api := mocker.GetAPI()
	overview, err := api.GetBlockOverview(&types.ReqHash{Hash: make([]byte, 32)})
	assert.NotNil(t, err) // random hash should not be found
	_ = overview
}
