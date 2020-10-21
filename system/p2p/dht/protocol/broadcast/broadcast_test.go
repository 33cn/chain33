// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/33cn/chain33/client"
	commlog "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/common/pubsub"
	"github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/p2p/utils"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/extension"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func init() {
	commlog.SetLogLevel("error")
}

var (
	payload = []byte("testpayload")
	minerTx = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 14600, Expire: 200}
	tx      = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 4600, Expire: 2}
	tx1     = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 460000000, Expire: 0}
	tx2     = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 100, Expire: 1}
	txList  = append([]*types.Transaction{}, minerTx, tx, tx1, tx2)

	testBlock = &types.Block{
		TxHash: []byte("test"),
		Height: 10,
		Txs:    txList,
	}
)

func newHost(port int32) core.Host {
	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil
	}

	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
	)

	if err != nil {
		panic(err)
	}

	return host
}

func newTestEnv(q queue.Queue, port int32) (*protocol.P2PEnv, context.CancelFunc) {
	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))

	mgr := p2p.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysAPI, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], subCfg)
	h := newHost(port)
	kademliaDHT, err := dht.New(context.Background(), h)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	env := &protocol.P2PEnv{
		Ctx:          ctx,
		ChainCfg:     cfg,
		QueueClient:  q.Client(),
		Host:         h,
		P2PManager:   mgr,
		SubConfig:    subCfg,
		RoutingTable: kademliaDHT.RoutingTable(),
	}
	env.Pubsub, _ = extension.NewPubSub(env.Ctx, env.Host)
	return env, cancel
}

func newTestProtocol(q queue.Queue, port int32) (*Protocol, context.CancelFunc) {
	protocol.ClearEventHandler()
	env, cancel := newTestEnv(q, port)
	InitProtocol(env)
	return defaultProtocol, cancel
}

func TestBroadCastEvent(t *testing.T) {
	q := queue.New("test")
	p, cancel := newTestProtocol(q, 13902)
	var msgs []*queue.Message
	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventTxBroadcast, tx))
	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventBlockBroadcast, testBlock))

	for _, msg := range msgs {
		protocol.GetEventHandler(msg.Ty)(msg)
	}
	_, ok := p.txFilter.Get(hex.EncodeToString(tx.Hash()))
	assert.True(t, ok)
	_, ok = p.blockFilter.Get(hex.EncodeToString(testBlock.Hash(p.ChainCfg)))
	assert.True(t, ok)
	time.Sleep(time.Second * 2)
	cancel()
	time.Sleep(time.Second)
}

func TestBroadCastEventNew(t *testing.T) {

	protocol.ClearEventHandler()
	q := queue.New("test")
	p1, cancel1 := newTestProtocol(q, 13902)
	env, cancel2 := newTestEnv(q, 13903)
	p2 := &Protocol{
		P2PEnv:          env,
		txFilter:        utils.NewFilter(txRecvFilterCacheNum),
		blockFilter:     utils.NewFilter(blockRecvFilterCacheNum),
		txSendFilter:    utils.NewFilter(txSendFilterCacheNum),
		blockSendFilter: utils.NewFilter(blockSendFilterCacheNum),
		ps:              pubsub.NewPubSub(10000),
		txQueue:         make(chan *types.Transaction, 10000),
		blockQueue:      make(chan *types.Block, 100),
	}

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/13902/p2p/%s", p1.Host.ID().Pretty()))
	peerInfo, _ := peer.AddrInfoFromP2pAddr(addr)
	err := p2.Host.Connect(context.Background(), *peerInfo)
	assert.Nil(t, err)
	_, err = p2.RoutingTable.Update(p1.Host.ID())
	assert.Equal(t, 1, p2.RoutingTable.Size())
	assert.Nil(t, err)
	p2.refreshPeers()

	msg1 := p1.QueueClient.NewMessage("p2p", types.EventTxBroadcast, tx)
	msg2 := p1.QueueClient.NewMessage("p2p", types.EventBlockBroadcast, testBlock)
	go p2.broadcastRoutine()
	go newPubSub(p2).broadcast()
	p2.handleEventBroadcastTx(msg1)
	p2.handleEventBroadcastBlock(msg2)

	blockchainCLI := q.Client()
	blockchainCLI.Sub("blockchain")
	mempoolCLI := q.Client()
	mempoolCLI.Sub("mempool")
	<-blockchainCLI.Recv()
	<-mempoolCLI.Recv()
	time.Sleep(time.Second * 2)
	cancel1()
	cancel2()
}

func TestFilter(t *testing.T) {
	filter := utils.NewFilter(100)
	exist := addIgnoreSendPeerAtomic(filter, "hash", "pid1")
	assert.False(t, exist)
	exist = addIgnoreSendPeerAtomic(filter, "hash", "pid2")
	assert.False(t, exist)
	exist = addIgnoreSendPeerAtomic(filter, "hash", "pid1")
	assert.True(t, exist)
}
