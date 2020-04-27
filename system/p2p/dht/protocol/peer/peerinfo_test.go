// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package peer

import (
	"testing"

	"github.com/33cn/chain33/p2p"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"

	"crypto/rand"

	"github.com/33cn/chain33/system/p2p/dht/manage"

	"context"
	"fmt"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/system/p2p/dht/net"
	multiaddr "github.com/multiformats/go-multiaddr"

	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
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
	env.Discovery = net.InitDhtDiscovery(host, nil, cfg, subCfg)
	env.ConnManager = manage.NewConnManager(host, env.Discovery, nil, subCfg)

	return env
}

func newTestProtocolWithQueue(q queue.Queue) *peerInfoProtol {
	env := newTestEnv(q)
	protocol := &peerInfoProtol{}
	protocol.BaseProtocol = new(prototypes.BaseProtocol)
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)

	return protocol
}

func newTestProtocol(q queue.Queue) *peerInfoProtol {

	return newTestProtocolWithQueue(q)
}

func testBlockReq(q queue.Queue) {
	client := q.Client()
	client.Sub("blockchain")
	go func() {
		for msg := range client.Recv() {
			msg.Reply(client.NewMessage("p2p", types.EventGetLastHeader, &types.Header{}))
		}
	}()

}

func testMempoolReq(q queue.Queue) {
	client := q.Client()
	client.Sub("mempool")
	go func() {
		for msg := range client.Recv() {
			msg.Reply(client.NewMessage("p2p", types.EventGetMempoolSize, &types.MempoolSize{Size: 10}))

		}
	}()

}

func TestPeerInfoProtol_InitProtocol(t *testing.T) {
	q := queue.New("test")
	protocol := newTestProtocol(q)
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
	q := queue.New("test")
	protocol := newTestProtocol(q)
	testMempoolReq(q)
	testBlockReq(q)
	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventPeerInfo, &types.P2PGetPeerReq{P2PType: "DHT"}))
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventGetNetInfo, nil))
	testHandleEvent(protocol, msgs[0])
	testNetInfoHandleEvent(protocol, msgs[1])
}

func Test_util(t *testing.T) {
	q := queue.New("test")
	proto := newTestProtocol(q)
	handler := &peerInfoHandler{}
	handler.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	handler.SetProtocol(proto)
	msgReq := &types.MessagePeerInfoReq{MessageData: proto.NewMessageCommon("uid122222", "16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg", []byte("322222222222222"), false)}
	ok := handler.VerifyRequest(msgReq, msgReq.MessageData)
	assert.False(t, ok)
	testMempoolReq(q)
	testBlockReq(q)
	peerinfo := proto.getLoacalPeerInfo()
	assert.NotNil(t, peerinfo)
	//----验证versionReq
	p2pVerReq := &types.MessageP2PVersionReq{MessageData: proto.NewMessageCommon("uid122222", "16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg", []byte("322222222222222"), false),
		Message: &types.P2PVersion{Version: 0, AddrRecv: "/ip4/127.0.0.1/13802", AddrFrom: "/ip4/192.168.0.1/13802"}}
	resp, _ := proto.processVerReq(p2pVerReq, "/ip4/192.168.0.1/13802")
	assert.NotNil(t, resp)

	//-----------------
	proto.getPeerInfo()
	proto.setExternalAddr("192.168.1.1")
	assert.NotEmpty(t, proto.getExternalAddr())
	proto.setExternalAddr("/ip4/192.168.1.1/13802")
	assert.Equal(t, "192.168.1.1", proto.getExternalAddr())
}
