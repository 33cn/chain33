// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package headers

import (
	"testing"

	"github.com/33cn/chain33/p2p"

	"context"
	"crypto/rand"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func newTestEnv(q queue.Queue) *prototypes.P2PEnv {

	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	q.SetConfig(cfg)
	go q.Start()

	mgr := p2p.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysAPI, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], subCfg)
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 12345))
	if err != nil {
		return nil
	}

	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
	)

	if err != nil {
		panic(err)
	}

	env := &prototypes.P2PEnv{
		ChainCfg:        cfg,
		QueueClient:     q.Client(),
		Host:            host,
		ConnManager:     nil,
		PeerInfoManager: nil,
		Discovery:       nil,
		P2PManager:      mgr,
		SubConfig:       subCfg,
	}

	ctx, cancel := context.WithCancel(context.Background())
	env.Ctx = ctx
	env.Cancel = cancel
	return env
}

func newTestProtocolWithQueue(q queue.Queue) *headerInfoProtol {
	env := newTestEnv(q)
	protocol := &headerInfoProtol{}
	protocol.BaseProtocol = new(prototypes.BaseProtocol)
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)

	return protocol
}

func newTestProtocol(q queue.Queue) *headerInfoProtol {

	return newTestProtocolWithQueue(q)
}

func TestHeaderInfoProtol_InitProtocol(t *testing.T) {
	q := queue.New("test")

	protocol := newTestProtocol(q)
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
	q := queue.New("test")

	protocol := newTestProtocol(q)
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

	q := queue.New("test")
	protocol := newTestProtocol(q)
	handler := &headerInfoHander{}
	handler.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	handler.SetProtocol(protocol)
	p2pgetheaders := &types.P2PGetHeaders{StartHeight: 1, EndHeight: 1,
		Version: 0}
	headerReq := &types.MessageHeaderReq{MessageData: protocol.NewMessageCommon("uid122222", "16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg", []byte("322222222222222"), false),
		Message: p2pgetheaders}
	ok := handler.VerifyRequest(headerReq, headerReq.MessageData)
	assert.False(t, ok)
	testBlockReq(q)
	testHeaderReq(t, protocol)

}

func testBlockReq(q queue.Queue) {
	client := q.Client()
	client.Sub("blockchain")
	go func() {
		for msg := range client.Recv() {
			msg.Reply(client.NewMessage("p2p", types.EventGetHeaders, &types.Headers{Items: []*types.Header{}}))
		}
	}()

}

func testHeaderReq(t *testing.T, protocol *headerInfoProtol) {

	_, err := protocol.processReq("1231212", &types.P2PGetHeaders{StartHeight: 1, EndHeight: 3000})
	assert.NotNil(t, err)

	_, err = protocol.processReq("1231212", &types.P2PGetHeaders{StartHeight: 1, EndHeight: 10})
	assert.Nil(t, err)

}
