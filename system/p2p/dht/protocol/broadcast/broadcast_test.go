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

func newTestEnv(q queue.Queue) *prototypes.P2PEnv {
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
	env.Pubsub, _ = net.NewPubSub(env.Ctx, env.Host)
	return env
}

func newTestProtocolWithQueue(q queue.Queue) *broadcastProtocol {
	env := newTestEnv(q)
	prototypes.ClearEventHandler()
	p := &broadcastProtocol{}
	p.init(env)
	return p
}

func newTestProtocol() *broadcastProtocol {

	q := queue.New("test")
	return newTestProtocolWithQueue(q)
}

func TestBroadCastProtocol_InitProtocol(t *testing.T) {

	protocol := newTestProtocol()
	assert.Equal(t, defaultMinLtBlockSize, int(protocol.p2pCfg.MinLtBlockSize))
	assert.Equal(t, defaultLtTxBroadCastTTL, int(protocol.p2pCfg.LightTxTTL))
}

func testHandleEvent(protocol *broadcastProtocol, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleBroadCastEvent(msg)
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
	_, ok := protocol.txFilter.Get(hex.EncodeToString((&types.Transaction{}).Hash()))
	assert.True(t, ok)
	_, ok = protocol.blockFilter.Get(hex.EncodeToString((&types.Block{}).Hash(protocol.ChainCfg)))
	assert.True(t, ok)
}

func Test_util(t *testing.T) {
	proto := newTestProtocol()
	exist := addIgnoreSendPeerAtomic(proto.txSendFilter, "hash", "pid1")
	assert.False(t, exist)
	exist = addIgnoreSendPeerAtomic(proto.txSendFilter, "hash", "pid2")
	assert.False(t, exist)
	exist = addIgnoreSendPeerAtomic(proto.txSendFilter, "hash", "pid1")
	assert.True(t, exist)
}
