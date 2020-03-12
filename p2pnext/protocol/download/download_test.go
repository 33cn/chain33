// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package download

import (
	"github.com/33cn/chain33/p2p"
	"testing"

	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/33cn/chain33/p2pnext/dht"
	"github.com/33cn/chain33/p2pnext/manage"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"

	"github.com/33cn/chain33/client"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func newTestEnv(q queue.Queue) *prototypes.P2PEnv {

	cfg := types.NewChain33Config(types.ReadFile("../../../cmd/chain33/chain33.test.toml"))
	q.SetConfig(cfg)
	go q.Start()

	mgr := p2p.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysAPI, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], subCfg)
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
		PeerInfoManager: manage.NewPeerInfoManager(mgr.Client),
		Discovery:       nil,
		P2PManager:      mgr,
		SubConfig:       subCfg,
	}

	env.Discovery = dht.InitDhtDiscovery(host, nil, subCfg, true)
	env.ConnManager = manage.NewConnManager(host, env.Discovery, nil)
	return env
}

func newTestProtocolWithQueue(q queue.Queue) *downloadProtol {
	env := newTestEnv(q)
	protocol := &downloadProtol{}
	protocol.BaseProtocol = new(prototypes.BaseProtocol)
	protocol.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)

	return protocol
}

func newTestProtocol(q queue.Queue) *downloadProtol {

	return newTestProtocolWithQueue(q)
}

func TestHeaderInfoProtol_InitProtocol(t *testing.T) {
	q := queue.New("test")

	protocol := newTestProtocol(q)
	assert.NotNil(t, protocol)

}

func testHandleEvent(protocol *downloadProtol, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleEvent(msg)
}

func TestFetchBlockEvent(t *testing.T) {
	q := queue.New("test")

	protocol := newTestProtocol(q)
	testBlockReq(q)

	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{
		Pid:      []string{}, //[]string{"16Uiu2HAmHffWU9fXzNUG3hiCCgpdj8Y9q1BwbbK7ZBsxSsnaDXk3"},
		Start:    1,
		End:      257,
		IsDetail: false,
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{
		Pid:      []string{"16Uiu2HAmHffWU9fXzNUG3hiCCgpdj8Y9q1BwbbK7ZBsxSsnaDXk4"},
		Start:    1,
		End:      10,
		IsDetail: false,
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{
		Pid:      []string{"16Uiu2HAmHffWU9fXzNUG3hiCCgpdj8Y9q1BwbbK7ZBsxSsnaDXk4"},
		Start:    100000,
		End:      100000,
		IsDetail: false,
	}))
	protocol.GetPeerInfoManager().Add("16Uiu2HAmHffWU9fXzNUG3hiCCgpdj8Y9q1BwbbK7ZBsxSsnaDXk4", &types.Peer{Header: &types.Header{Height: 10000}})

	for _, msg := range msgs {
		testHandleEvent(protocol, msg)
	}

	time.Sleep(time.Second * 4)

}

func Test_util(t *testing.T) {

	q := queue.New("test")
	protocol := newTestProtocol(q)
	handler := &downloadHander{}
	handler.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	handler.SetProtocol(protocol)

	testBlockReq(q)

	p2pgetblocks := &types.P2PGetBlocks{StartHeight: 1, EndHeight: 1,
		Version: 0}
	blockReq := &types.MessageGetBlocksReq{MessageData: protocol.NewMessageCommon("uid122222", "16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg", []byte("322222222222222"), false),
		Message: p2pgetblocks}
	ok := handler.VerifyRequest(blockReq, blockReq.MessageData)
	assert.False(t, ok)

	protocol.processReq("uid122222", p2pgetblocks)

	resp, _ := protocol.SendToBlockChain(types.EventGetBlocks, blockReq)
	assert.NotNil(t, resp)
}

func testBlockReq(q queue.Queue) {
	client := q.Client()
	client.Sub("blockchain")
	go func() {
		for msg := range client.Recv() {
			msg.Reply(client.NewMessage("p2p", types.EventFetchBlocks, &types.BlockDetails{Items: []*types.BlockDetail{}}))
		}
	}()

}
