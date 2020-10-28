// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package peer

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/p2p"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"

	"crypto/rand"

	"github.com/33cn/chain33/system/p2p/dht/manage"

	"context"
	"fmt"
	snet "net"

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

	ctx := context.Background()
	pubsub, err := net.NewPubSub(ctx, env.Host)
	if err != nil {
		return nil
	}
	env.Ctx = ctx
	env.Pubsub = pubsub
	env.Discovery = net.InitDhtDiscovery(env.Ctx, host, nil, cfg, subCfg)
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
	assert.Equal(t, false, protocol.checkDone())

	time.Sleep(time.Second * 11)
	testMempoolReq(q)
	testBlockReq(q)
	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventPeerInfo, &types.P2PGetPeerReq{P2PType: "DHT"}))
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventGetNetInfo, nil))
	testHandleEvent(protocol, msgs[0])
	testNetInfoHandleEvent(protocol, msgs[1])
	var req types.MessageP2PVersionReq
	req.Message = &types.P2PVersion{Version: 123}
	_, err := protocol.processVerReq(&req, "192.168.1.1")
	assert.NotNil(t, err)
	req.Message = &types.P2PVersion{Version: 0}
	_, err = protocol.processVerReq(&req, "192.168.1.1")
	assert.Nil(t, err)
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
		Message: &types.P2PVersion{Version: 0, AddrRecv: "/ip4/127.0.0.1/tcp/13802", AddrFrom: "/ip4/192.168.0.1/tcp/13802"}}
	resp, _ := proto.processVerReq(p2pVerReq, "/ip4/192.168.0.1/tcp/13802")
	assert.NotNil(t, resp)

	//-----------------
	proto.getPeerInfo()
	proto.setExternalAddr("150.109.6.160")
	assert.NotEmpty(t, proto.getExternalAddr())
	proto.setExternalAddr("/ip4/150.109.6.160/tcp/13802")
	assert.Equal(t, "150.109.6.160", proto.getExternalAddr())
	assert.True(t, !isPublicIP(snet.ParseIP("127.0.0.1")))
	ips, err := localIPv4s()
	assert.Nil(t, err)
	assert.True(t, !isPublicIP(snet.ParseIP(ips[0])))
	assert.True(t, isPublicIP(snet.ParseIP("112.74.59.221")))
	assert.False(t, isPublicIP(snet.ParseIP("10.74.59.221")))
	assert.False(t, isPublicIP(snet.ParseIP("172.16.59.221")))
	assert.False(t, isPublicIP(snet.ParseIP("192.168.59.221")))
	//增加测试用例
	_, err = multiaddr.NewMultiaddr("/ip4/122.224.166.26/tcp/13802")
	assert.Nil(t, err)
	ok = proto.checkRemotePeerExternalAddr("/ip4/192.168.1.1/tcp/13802")
	assert.False(t, ok)

	ok = proto.checkRemotePeerExternalAddr("/ip4/122.224.166.26/tcp/13802")
	assert.True(t, ok)
	pid, err := peer.Decode("16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg")
	assert.Nil(t, err)
	assert.Equal(t, pid.String(), "16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg")

	proto.setAddrToPeerStore(pid, "/ip4/122.224.166.26/tcp/13802", "/ip4/192.168.1.1/tcp/12345")
	pid, err = peer.IDFromString("16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg")
	assert.NotNil(t, err)
}
