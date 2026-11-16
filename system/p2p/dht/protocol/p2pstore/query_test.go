package p2pstore

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestFetchShardPeersReturnsRemoteError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1 := newFetchShardTestHost(t)
	h2 := newFetchShardTestHost(t)
	err := h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
	require.Nil(t, err)

	// 对端返回业务错误时，客户端不能把错误响应当成空 peer 列表。
	protocol.RegisterStreamHandler(h1, fetchShardPeer, protocol.HandlerWithRW(
		func(req *types.P2PRequest, res *types.P2PResponse) error {
			return types.ErrInvalidParam
		}))

	p := &Protocol{P2PEnv: &protocol.P2PEnv{Ctx: ctx, Host: h2}}
	peers, err := p.fetchShardPeers(nil, 0, h1.ID())
	require.Error(t, err)
	require.Nil(t, peers)
}

func newFetchShardTestHost(t *testing.T) host.Host {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.Nil(t, err)
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), libp2p.Identity(priv))
	require.Nil(t, err)
	t.Cleanup(func() { _ = h.Close() })
	return h
}
