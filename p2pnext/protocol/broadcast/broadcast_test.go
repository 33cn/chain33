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

var (
	payload = []byte("testpayload")
	minerTx = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 14600, Expire: 200}
	tx      = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 4600, Expire: 2}
	tx1     = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 460000000, Expire: 0}
	tx2     = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 100, Expire: 1}
	//txGroup, _ = types.CreateTxGroup([]*types.Transaction{tx1, tx2}, cfg.GetMinTxFeeRate())
	//gtx := txGroup.Tx()
	txList = append([]*types.Transaction{}, minerTx, tx, tx1, tx2)
	//memTxList := append([]*types.Transaction{}, tx, gtx)

	testBlock = &types.Block{
		TxHash: []byte("test"),
		Height: 10,
		Txs:    txList,
	}
	testAddr = "testPeerAddr"
	testPid  = "testPeerID"
)

func newTestEnv(q queue.Queue) *prototypes.P2PEnv {

	cfg := types.NewChain33Config(types.ReadFile("../../../cmd/chain33/chain33.test.toml"))
	q.SetConfig(cfg)
	go q.Start()

	mgr := p2pmgr.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysAPI, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pmgr.DHTTypeName], subCfg)
	subCfg.MinLtBlockTxNum = 1

	env := &prototypes.P2PEnv{
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

func newTestProtocolWithQueue(q queue.Queue) *broadCastProtocol {
	env := newTestEnv(q)
	protocol := &broadCastProtocol{}
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)
	return protocol
}

func newTestProtocol() *broadCastProtocol {

	q := queue.New("test")
	return newTestProtocolWithQueue(q)
}

func TestBroadCastProtocol_InitProtocol(t *testing.T) {

	protocol := newTestProtocol()
	assert.Equal(t, int32(1), protocol.p2pCfg.MinLtBlockTxNum)
	assert.Equal(t, defaultLtTxBroadCastTTL, int(protocol.p2pCfg.LightTxTTL))
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

func Test_util(t *testing.T) {
	proto := newTestProtocol()
	handler := &broadCastHandler{}
	handler.SetProtocol(proto)
	ok := handler.VerifyRequest(nil, nil)
	assert.True(t, ok)

	exist := addIgnoreSendPeerAtomic(proto.txSendFilter, "hash", "pid1")
	assert.False(t, exist)
	exist = addIgnoreSendPeerAtomic(proto.txSendFilter, "hash", "pid2")
	assert.False(t, exist)
	exist = addIgnoreSendPeerAtomic(proto.txSendFilter, "hash", "pid1")
	assert.True(t, exist)
}
