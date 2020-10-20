// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"testing"

	"github.com/33cn/chain33/client"
	commlog "github.com/33cn/chain33/common/log"
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

func newTestEnv(q queue.Queue) *protocol.P2PEnv {
	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	q.SetConfig(cfg)
	go q.Start()

	mgr := p2p.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysAPI, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], subCfg)
	h := newHost(13902)
	kademliaDHT, err := dht.New(context.Background(), h)
	if err != nil {
		panic(err)
	}
	env := &protocol.P2PEnv{
		Ctx:          context.Background(),
		ChainCfg:     cfg,
		QueueClient:  q.Client(),
		Host:         h,
		P2PManager:   mgr,
		SubConfig:    subCfg,
		RoutingTable: kademliaDHT.RoutingTable(),
	}
	env.Pubsub, _ = extension.NewPubSub(env.Ctx, env.Host)
	return env
}

func newTestProtocol() *Protocol {

	q := queue.New("test")
	protocol.ClearEventHandler()
	env := newTestEnv(q)
	InitProtocol(env)
	return defaultProtocol
}

func TestBroadCastEvent(t *testing.T) {
	p := newTestProtocol()
	var msgs []*queue.Message
	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventTxBroadcast, tx))
	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventBlockBroadcast, testBlock))
	//msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventTx, &types.LightTx{}))

	for _, msg := range msgs {
		protocol.GetEventHandler(msg.Ty)(msg)
	}
	_, ok := p.txFilter.Get(hex.EncodeToString(tx.Hash()))
	assert.True(t, ok)
	_, ok = p.blockFilter.Get(hex.EncodeToString(testBlock.Hash(p.ChainCfg)))
	assert.True(t, ok)
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
