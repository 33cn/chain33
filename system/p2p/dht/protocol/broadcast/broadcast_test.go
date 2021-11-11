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

	"github.com/stretchr/testify/require"

	net "github.com/33cn/chain33/system/p2p/dht/extension"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"

	"github.com/33cn/chain33/client"
	commlog "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

func init() {
	commlog.SetLogLevel("error")
}

var (
	payload = []byte("testpayload")

	txList    = append([]*types.Transaction{}, minerTx, tx, tx1, tx2)
	testBlock = &types.Block{
		TxHash: []byte("test"),
		Height: 10,
		Txs:    txList,
	}
	testAddr   = "testPeerAddr"
	testPidStr = "16Uiu2HAm14hiGBFyFChPdG98RaNAMtcFJmgZjEQLuL87xsSkv72U"
	testPid, _ = peer.Decode("16Uiu2HAm14hiGBFyFChPdG98RaNAMtcFJmgZjEQLuL87xsSkv72U")
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

func newTestEnv(q queue.Queue) (*prototypes.P2PEnv, context.CancelFunc) {
	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	q.SetConfig(cfg)
	go q.Start()

	mgr := p2p.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysAPI, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], subCfg)
	env := &prototypes.P2PEnv{
		ChainCfg:        cfg,
		QueueClient:     q.Client(),
		Host:            newHost(13902),
		ConnManager:     nil,
		PeerInfoManager: nil,
		P2PManager:      mgr,
		SubConfig:       subCfg,
		Ctx:             context.Background(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	env.Ctx = ctx
	var err error
	env.Pubsub, err = net.NewPubSub(env.Ctx, env.Host, &p2pty.PubSubConfig{})
	if err != nil {
		panic("new pubsub err" + err.Error())
	}
	return env, cancel
}

func newTestProtocolWithQueue(q queue.Queue) *broadcastProtocol {
	env, cancel := newTestEnv(q)
	prototypes.ClearEventHandler()
	p := &broadcastProtocol{syncStatus: true}
	p.init(env)
	cancel()
	return p
}

func newTestProtocol() *broadcastProtocol {

	q := queue.New("test")
	return newTestProtocolWithQueue(q)
}

func TestBroadCastProtocol_InitProtocol(t *testing.T) {

	protocol := newTestProtocol()
	assert.Equal(t, defaultMinLtBlockSize, protocol.cfg.MinLtBlockSize)
	assert.Equal(t, defaultLtBlockTimeout, int(protocol.cfg.LtBlockPendTimeout))
}

func testHandleEvent(protocol *broadcastProtocol, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleBroadcastSend(msg)
}

func TestBroadCastSend(t *testing.T) {
	protocol := newTestProtocol()
	defer protocol.Ctx.Done()
	msgs := []*queue.Message{
		protocol.QueueClient.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{}),
		protocol.QueueClient.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{}),
		protocol.QueueClient.NewMessage("p2p", types.EventAddBlock, nil),
		protocol.QueueClient.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{Txs: []*types.Transaction{tx1, tx2}}),
	}
	protocol.cfg.MinLtBlockSize = 0
	for _, msg := range msgs {
		testHandleEvent(protocol, msg)
	}
	_, ok := protocol.txFilter.Get(hex.EncodeToString((&types.Transaction{}).Hash()))
	assert.True(t, ok)
	_, ok = protocol.blockFilter.Get(hex.EncodeToString((&types.Block{}).Hash(protocol.ChainCfg)))
	assert.True(t, ok)
	_, ok = protocol.blockFilter.Get(hex.EncodeToString((&types.Block{Txs: []*types.Transaction{tx1, tx2}}).Hash(protocol.ChainCfg)))
	assert.True(t, ok)
}

func TestBroadCastReceive(t *testing.T) {

	p := newTestProtocol()
	defer p.Ctx.Done()
	pid := p.Host.ID()
	peerTopic := p.getPeerTopic(pid)
	msgs := []subscribeMsg{
		{value: &types.Transaction{}, topic: psTxTopic},
		{value: &types.Block{}, topic: psBlockTopic},
		{value: &types.LightBlock{}, topic: psLtBlockTopic},
		{value: &types.PeerPubSubMsg{MsgID: blockReqMsgID, ProtoMsg: types.Encode(&types.ReqInt{})}, topic: peerTopic},
		{value: &types.PeerPubSubMsg{MsgID: blockRespMsgID}, topic: peerTopic},
		{value: &types.PeerPubSubMsg{}, topic: peerTopic},
	}
	for _, msg := range msgs {
		msg.receiveFrom = pid
		msg.publisher = pid
		p.handleBroadcastReceive(msg)
	}
	require.Equal(t, 0, p.ltB.pendBlockList.Len())
	require.Equal(t, 0, p.ltB.blockRequestList.Len())
}
