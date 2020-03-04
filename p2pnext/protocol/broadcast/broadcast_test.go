// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/client"
	p2pmgr "github.com/33cn/chain33/p2p/manage"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

type broadcastProtoMock struct {
	*broadCastProtocol
}

func (b *broadcastProtoMock) sendStream(pid string, data interface{}) error {
	return nil
}

func newTestEnv() *prototypes.GlobalData {

	cfg := types.NewChain33Config(types.ReadFile("../../../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	q.SetConfig(cfg)
	go q.Start()
	defer q.Close()

	mgr := p2pmgr.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysApi, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pmgr.DHTTypeName], subCfg)
	subCfg.MinLtBlockTxNum = 1

	env := &prototypes.GlobalData{
		ChainCfg:        cfg,
		QueueClient:     q.Client(),
		Host:            nil,
		ConnManager:     nil,
		PeerInfoManager: nil,
		Discovery:       nil,
		P2PManager:      mgr,
		SubConfig:       subCfg,
	}
	return env
}

func newTestProtocol() *broadCastProtocol {

	env := newTestEnv()
	protocol := &broadCastProtocol{}
	protocol.InitProtocol(env)
	return protocol
}

func newHandler() *broadCastHandler {
	handler := &broadCastHandler{}
	protocol := newTestProtocol()
	handler.SetProtocol(protocol)
	return handler
}

func TestBroadCastProtocol_InitProtocol(t *testing.T) {

	protocol := newTestProtocol()
	assert.Equal(t, int32(1), protocol.p2pCfg.MinLtBlockTxNum)
	assert.Equal(t, DefaultLtTxBroadCastTTL, int(protocol.p2pCfg.LightTxTTL))
}

func testHandleEvent(protocol *broadCastProtocol, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleEvent(msg)
}

func TestBroadCastEvent(t *testing.T) {
	protocol := newTestProtocol()

	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{}))
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{}))
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventTx, &types.LightTx{}))

	for _, msg := range msgs {
		testHandleEvent(protocol, msg)
	}
	val, ok := protocol.txFilter.Get(hex.EncodeToString((&types.Transaction{}).Hash()))
	assert.True(t, ok)
	assert.True(t, val.(bool))
	val, ok = protocol.blockFilter.Get(hex.EncodeToString((&types.Block{}).Hash(protocol.ChainCfg)))
	assert.True(t, ok)
	assert.True(t, val.(bool))
}
