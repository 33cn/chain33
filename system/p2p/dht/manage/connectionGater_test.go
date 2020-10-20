package manage

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func Test_MaxLimit(t *testing.T) {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 12345))
	if err != nil {
		return
	}

	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	var host1 host.Host
	//设置0，意味着拒绝所有的连接
	CacheLimit = 0
	gater := NewConnGater(context.Background(), &host1, 0, 0)
	host1, err = libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
		libp2p.ConnectionGater(gater),
	)
	if err != nil {
		return
	}
	m2, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 12346))
	if err != nil {
		return
	}
	priv2, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	host2, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m2),
		libp2p.Identity(priv2),
	)
	if err != nil {
		return
	}
	h1info := peer.AddrInfo{
		ID:    host1.ID(),
		Addrs: host1.Addrs(),
	}
	err = host2.Connect(context.Background(), h1info)
	assert.NotNil(t, err)

}

func Test_InterceptAccept(t *testing.T) {
	var host1 host.Host
	gater := NewConnGater(context.Background(), &host1, 0, 0)

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
	gater := NewConnGater(context.Background(), &host1, 0, 0)
	var ip = "47.97.223.101"
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 3000))
	assert.NoError(t, err)
	assert.True(t, gater.InterceptAddrDial("", multiAddress))
}

func Test_InterceptPeerDial(t *testing.T) {
	var host1 host.Host
	ctx := context.Background()
	defer ctx.Done()
	gater := NewConnGater(context.Background(), &host1, 1, time.Second)
	var pid = "16Uiu2HAmCyJhBvE1vn62MQWhhaPph1cxeU9nNZJoZQ1Pe1xASZUg"

	gater.blacklist.Add(pid, 0)
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
	gater := NewConnGater(ctx, &host1, 1, time.Second)
	allow, _ := gater.InterceptUpgraded(nil)
	assert.True(t, allow)
	assert.True(t, gater.InterceptSecured(network.DirInbound, "", nil))

}

func Test_timecache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewTimeCache(ctx, time.Second)
	cache.Add("one", 0)
	cache.Add("two", time.Second*3)
	cache.Add("three", time.Second*5)
	time.Sleep(time.Second * 2)
	assert.False(t, cache.Has("one"))
	assert.True(t, cache.Has("two"))
	assert.True(t, cache.Has("three"))
	time.Sleep(time.Second * 2)
	assert.False(t, cache.Has("two"))
	assert.True(t, cache.Has("three"))
	cancel()
	time.Sleep(time.Second * 2)
	assert.True(t, cache.Has("three"))

}
