// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package headers

import (
	"testing"

	"github.com/33cn/chain33/client"
	p2pmgr "github.com/33cn/chain33/p2p/manage"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
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

func newTestProtocolWithQueue(q queue.Queue) *headerInfoProtol {
	env := newTestEnv(q)
	protocol := &headerInfoProtol{}
	protocol.BaseProtocol = new(prototypes.BaseProtocol)
	protocol.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)

	return protocol
}

func newTestProtocol() *headerInfoProtol {

	q := queue.New("test")
	return newTestProtocolWithQueue(q)
}

func TestHeaderInfoProtol_InitProtocol(t *testing.T) {

	protocol := newTestProtocol()
	assert.NotNil(t, protocol)

}

func testHandleEvent(protocol *headerInfoProtol, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleEvent(msg)
}

func TestFetchHeaderEvent(t *testing.T) {
	protocol := newTestProtocol()
	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventFetchBlockHeaders, &types.ReqBlocks{
		Pid:      []string{}, //[]string{"16Uiu2HAmHffWU9fXzNUG3hiCCgpdj8Y9q1BwbbK7ZBsxSsnaDXk3"},
		Start:    1,
		End:      1,
		IsDetail: false,
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventFetchBlockHeaders, &types.ReqBlocks{
		Pid:      []string{"16Uiu2HAmHffWU9fXzNUG3hiCCgpdj8Y9q1BwbbK7ZBsxSsnaDXk3"},
		Start:    1,
		End:      1,
		IsDetail: false,
	}))

	for _, msg := range msgs {
		testHandleEvent(protocol, msg)
	}

}

func Test_util(t *testing.T) {
	proto := newTestProtocol()
	handler := &headerInfoHander{}
	handler.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	handler.SetProtocol(proto)
	p2pgetheaders := &types.P2PGetHeaders{StartHeight: 1, EndHeight: 1,
		Version: 0}
	headerReq := &types.MessageHeaderReq{MessageData: proto.NewMessageCommon("uid122222", "16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg", []byte("322222222222222"), false),
		Message: p2pgetheaders}
	ok := handler.VerifyRequest(headerReq, headerReq.MessageData)
	assert.False(t, ok)

}
