package manage

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	core "github.com/libp2p/go-libp2p-core"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func newTestHost(port int) (core.Host, error) {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	return libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
	)

}
func Test_MaxLimit(t *testing.T) {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 12345))
	if err != nil {
		return
	}

	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	var host1 host.Host
	CacheLimit = 0
	gater := NewConnGater(&host1, &p2pty.P2PSubConfig{MaxConnectNum: 1}, nil)
	host1, err = libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
		libp2p.ConnectionGater(gater),
	)

	if err != nil {
		return
	}
	host2, err := newTestHost(12346)
	if err != nil {
		return
	}
	h1info := peer.AddrInfo{
		ID:    host1.ID(),
		Addrs: host1.Addrs(),
	}
	err = host2.Connect(context.Background(), h1info)
	assert.Nil(t, err)

	host3, err := newTestHost(12347)
	if err != nil {
		return
	}
	//超过上限，会拒绝连接，所以host3连接host1会被拒绝，连接失败
	err = host3.Connect(context.Background(), h1info)
	assert.NotNil(t, err)
}

func Test_InterceptAccept(t *testing.T) {
	var host1 host.Host
	gater := NewConnGater(&host1, &p2pty.P2PSubConfig{MaxConnectNum: 0}, nil)

	var ip = "47.97.223.101"
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 3000))
	assert.NoError(t, err)
	for i := 0; i < ipBurst; i++ {
		valid := gater.validateDial(multiAddress)
		assert.True(t, valid)
	}
	valid := gater.validateDial(multiAddress)
	assert.False(t, valid)

}

func Test_InterceptAddrDial(t *testing.T) {
	var host1 host.Host
	gater := NewConnGater(&host1, &p2pty.P2PSubConfig{}, nil)
	var ip = "47.97.223.101"
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 3000))
	assert.NoError(t, err)
	assert.True(t, gater.InterceptAddrDial("", multiAddress))
}

func Test_InterceptPeerDial(t *testing.T) {
	var host1 host.Host
	ctx := context.Background()
	defer ctx.Done()
	gater := NewConnGater(&host1, &p2pty.P2PSubConfig{MaxConnectNum: 1}, NewTimeCache(ctx, time.Second))
	var pid = "16Uiu2HAmCyJhBvE1vn62MQWhhaPph1cxeU9nNZJoZQ1Pe1xASZUg"

	gater.blackCache.Add(pid, 0)
	id, err := peer.Decode(pid)
	assert.NoError(t, err)
	ok := gater.InterceptPeerDial(id)
	assert.False(t, ok)
	time.Sleep(time.Second * 2)
	ok = gater.InterceptPeerDial(id)
	assert.True(t, ok)
}

func Test_otherInterface(t *testing.T) {
	var host1 host.Host
	ctx := context.Background()
	defer ctx.Done()
	gater := NewConnGater(&host1, &p2pty.P2PSubConfig{MaxConnectNum: 1}, NewTimeCache(ctx, time.Second))
	allow, _ := gater.InterceptUpgraded(nil)
	assert.True(t, allow)
	assert.True(t, gater.InterceptSecured(network.DirInbound, "", nil))

}

func Test_timecache(t *testing.T) {
	cache := NewTimeCache(context.Background(), time.Second)
	cache.Add("one", 0)
	cache.Add("two", time.Second*3)
	time.Sleep(time.Second * 2)
	assert.False(t, cache.Has("one"))
	assert.True(t, cache.Has("two"))
	time.Sleep(time.Second * 2)
	assert.False(t, cache.Has("two"))
}
