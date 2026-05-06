package client_test

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestClientSendTx(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	// Send a transaction through the API
	tx := types.NewTx()
	tx.Execer = []byte("none")
	tx.Payload = []byte("test")
	tx.Fee = 1e6
	tx.ChainID = cfg.GetChainID()
	reply, err := api.SendTx(tx)
	assert.Nil(t, err)
	assert.NotNil(t, reply)
}

func TestClientGetBlocks(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(2)
	assert.Nil(t, err)

	api := mocker.GetAPI()
	blocks, err := api.GetBlocks(&types.ReqBlocks{Start: 0, End: 1, IsDetail: true})
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, len(blocks.Items), 1)

	if blocks.Items[0].Block != nil {
		assert.NotNil(t, blocks.Items[0].Block.Hash(cfg))
	}
}

func TestClientQueryChain(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	seq, err := api.GetSequence(&types.ReqNil{})
	assert.Nil(t, err)
	assert.NotNil(t, seq)

	totalCoins, err := api.GetTotalCoins(&types.ReqGetTotalCoins{})
	assert.Nil(t, err)
	assert.NotNil(t, totalCoins)
}

func TestClientSyncStatus(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	reply, err := api.IsSync()
	assert.Nil(t, err)
	assert.NotNil(t, reply)

	reply2, err := api.IsNtpClockSync()
	assert.Nil(t, err)
	assert.NotNil(t, reply2)
}

func TestClientGetPeerList(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	peers, err := api.PeerInfo(&types.P2PGetPeerReq{})
	assert.Nil(t, err)
	assert.NotNil(t, peers)
}

func TestClientVersion(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	ver := api.Version()
	assert.NotNil(t, ver)
}
