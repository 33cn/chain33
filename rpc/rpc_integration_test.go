package rpc_test

import (
	"testing"

	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestRPCHealth(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithRPC(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	// Verify RPC is reachable
	jsonc := mocker.GetJSONC()
	assert.NotNil(t, jsonc)

	// Test get last header via JSON-RPC
	var result struct {
		Version   int32  `json:"version"`
		Title     string `json:"title"`
		BlockTime int64  `json:"blockTime"`
		Height    int64  `json:"height"`
	}
	err = jsonc.Call("Chain33.GetLastHeader", nil, &result)
	assert.Nil(t, err)
}

func TestRPCGetBlocks(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithRPC(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(2)
	assert.Nil(t, err)

	jsonc := mocker.GetJSONC()
	var blockResult interface{}
	err = jsonc.Call("Chain33.GetBlocks", map[string]interface{}{
		"start":    0,
		"end":      1,
		"isDetail": false,
	}, &blockResult)
	assert.Nil(t, err)
}

func TestRPCGetPeerInfo(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithRPC(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	jsonc := mocker.GetJSONC()
	var peerResult interface{}
	err := jsonc.Call("Chain33.GetPeerInfo", nil, &peerResult)
	assert.Nil(t, err)
}

func TestRPCVersion(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithRPC(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	jsonc := mocker.GetJSONC()
	var versionResult interface{}
	err := jsonc.Call("Chain33.Version", nil, &versionResult)
	assert.Nil(t, err)
}

func TestRPCGetMempool(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithRPC(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	jsonc := mocker.GetJSONC()
	var mempoolResult interface{}
	err = jsonc.Call("Chain33.GetMempool", nil, &mempoolResult)
	assert.Nil(t, err)
}
