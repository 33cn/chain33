// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package types

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func newHost(port int32, priv crypto.PrivKey) core.Host {
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

func newHostPair(port1, port2 int32) (h1, h2 core.Host) {

	r := rand.Reader
	prvKey1, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	r = rand.Reader
	prvKey2, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	msgID := "/streamTest1"
	msgID2 := "/streamTest2"
	h1 = newHost(port1, prvKey1)
	h2 = newHost(port2, prvKey2)
	h1.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		CloseStream(s)
	})
	h1.SetStreamHandler(protocol.ID(msgID2), func(s core.Stream) {
		CloseStream(s)
	})

	h2.SetStreamHandler(protocol.ID(msgID2), func(s core.Stream) {
		tx := &types.Transaction{}
		ReadStream(tx, s)
		WriteStream(tx, s)
		CloseStream(s)
	})
	h2.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		tx := &types.Transaction{}
		ReadStream(tx, s)
		CloseStream(s)
	})
	return
}

func TestStream(t *testing.T) {
	h1, h2 := newHostPair(13804, 13805)
	defer h1.Close()
	defer h2.Close()
	h2info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	msgID := "/streamTest1"
	msgID2 := "/streamTest2"
	err := h1.Connect(context.Background(), h2info)
	assert.Nil(t, err)
	proto := &testProtocol{}
	proto.P2PEnv = &P2PEnv{Host: h1}
	req := &StreamRequest{
		PeerID: h2.ID(),
		MsgID:  []core.ProtocolID{core.ProtocolID(msgID)},
		Data:   &types.Transaction{},
	}
	err = proto.SendPeer(req)
	assert.Nil(t, err)
	req.MsgID = []core.ProtocolID{core.ProtocolID(msgID2)}
	err = proto.SendRecvPeer(req, &types.Transaction{})
	assert.Nil(t, err)
	stream, err := NewStream(h1, h2.ID(), []core.ProtocolID{core.ProtocolID(msgID2)})
	assert.Equal(t, msgID2, string(stream.Protocol()))
	assert.Nil(t, err)
	err = WriteStream(&types.Transaction{}, stream)
	assert.Nil(t, err)
	err = ReadStream(&types.Transaction{}, stream)
	assert.Nil(t, err)
	CloseStream(stream)
}

// BenchmarkNewStream-8       50000             33760 ns/op
func BenchmarkNewStream(b *testing.B) {

	h1, h2 := newHostPair(13804, 13805)
	defer h1.Close()
	defer h2.Close()
	h2info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	msgID := "/streamTest1"
	err := h1.Connect(context.Background(), h2info)
	assert.Nil(b, err)
	proto := &testProtocol{}
	proto.P2PEnv = &P2PEnv{Host: h1}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			NewStream(h1, h2.ID(), []core.ProtocolID{core.ProtocolID((msgID))})
		}
	})
}
