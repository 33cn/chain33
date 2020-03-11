// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package peer

import (
	"testing"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"

	"crypto/rand"

	"github.com/33cn/chain33/p2pnext/manage"

	"context"
	"fmt"

	"github.com/33cn/chain33/client"
	p2pmgr "github.com/33cn/chain33/p2p/manage"
	"github.com/33cn/chain33/p2pnext/dht"
	multiaddr "github.com/multiformats/go-multiaddr"

	//"github.com/33cn/chain33/p2pnext/manage"
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
	env.Discovery = dht.InitDhtDiscovery(host, nil, subCfg, true)
	env.ConnManager = manage.NewConnManager(host, env.Discovery, nil)

	return env
}

func newTestProtocolWithQueue(q queue.Queue) *peerInfoProtol {
	env := newTestEnv(q)
	protocol := &peerInfoProtol{}
	protocol.BaseProtocol = new(prototypes.BaseProtocol)
	protocol.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)

	return protocol
}

func newTestProtocol() *peerInfoProtol {

	q := queue.New("test")
	return newTestProtocolWithQueue(q)
}

func TestPeerInfoProtol_InitProtocol(t *testing.T) {

	protocol := newTestProtocol()
	assert.NotNil(t, protocol)

}

func testHandleEvent(protocol *peerInfoProtol, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleEvent(msg)
}
func testNetInfoHandleEvent(protocol *peerInfoProtol, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.netinfoHandleEvent(msg)
}

func TestPeerInfoEvent(t *testing.T) {
	protocol := newTestProtocol()
	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventPeerInfo, &types.P2PGetPeerReq{P2PType: "DHT"}))
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventGetNetInfo, nil))
	testHandleEvent(protocol, msgs[0])
	testNetInfoHandleEvent(protocol, msgs[1])
}

func Test_util(t *testing.T) {

	proto := newTestProtocol()
	handler := &peerInfoHandler{}
	handler.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	handler.SetProtocol(proto)
	//peerReq := &types.P2PGetPeerReq{P2PType: "DHT"}
	msgReq := &types.MessagePeerInfoReq{MessageData: proto.NewMessageCommon("uid122222", "16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg", []byte("322222222222222"), false)}

	ok := handler.VerifyRequest(msgReq, msgReq.MessageData)
	assert.False(t, ok)
	decodeChannelVersion(255)

	proto.SetExternalAddr("192.168.1.1")
	assert.Empty(t, proto.GetExternalAddr())
	proto.SetExternalAddr("/ip4/192.168.1.1/13802")
	assert.Equal(t, "192.168.1.1", proto.GetExternalAddr())
}
