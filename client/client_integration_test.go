package client_test

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestGetMempool(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	txs, err := api.GetMempool(&types.ReqGetMempool{})
	assert.Nil(t, err)
	assert.NotNil(t, txs)
}

func TestGetPeerInfo(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	peers, err := api.PeerInfo(&types.P2PGetPeerReq{})
	assert.Nil(t, err)
	assert.NotNil(t, peers)
}

func TestGetLastBlockSequence(t *testing.T) {
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

func TestIsSync(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	reply, err := api.IsSync()
	assert.Nil(t, err)
	assert.NotNil(t, reply)
}

func TestIsNtpClockSync(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	_, err := api.IsNtpClockSync()
	assert.Nil(t, err)
}

func TestQueryTx(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	// Query non-existent transaction
	_, err = api.QueryTx(&types.ReqHash{Hash: make([]byte, 32)})
	assert.NotNil(t, err)
}
