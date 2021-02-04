package manage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func newTestHost(port int) (core.Host, error) {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	return libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
	)

}

func Test_MaxLimit(t *testing.T) {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 12345))
	require.Nil(t, err)

	var host1 host.Host
	CacheLimit = 0
	gater := NewConnGater(&host1, 1, nil, nil)
	host1, err = libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.ConnectionGater(gater),
	)
	require.Nil(t, err)

	host2, err := newTestHost(12346)
	require.Nil(t, err)
	h1info := peer.AddrInfo{
		ID:    host1.ID(),
		Addrs: host1.Addrs(),
	}
	err = host2.Connect(context.Background(), h1info)
	require.Nil(t, err)

	host3, err := newTestHost(12347)
	require.Nil(t, err)
	//超过上限，会拒绝连接，所以host3连接host1会被拒绝，连接失败
	err = host3.Connect(context.Background(), h1info)
	assert.NotNil(t, err)
}

func Test_InterceptAccept(t *testing.T) {
	var host1 host.Host
	gater := NewConnGater(&host1, 0, nil, nil)

	var ip = "47.97.223.101"
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 3000))
	require.NoError(t, err)
	for i := 0; i < ipBurst; i++ {
		valid := gater.validateDial(multiAddress)
		require.True(t, valid)
	}
	valid := gater.validateDial(multiAddress)
	require.False(t, valid)
	//test whiteList
	whitePeer1, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%v", "192.168.102.123", 3001, "16Uiu2HAmK9PAPYoTzHnobzB5nQFnY7p9ZVcJYQ1BgzKCr7izAhbJ"))
	peerInfo, err := peer.AddrInfoFromP2pAddr(whitePeer1)
	assert.Nil(t, err)
	gater2 := NewConnGater(&host1, 0, nil, []*peer.AddrInfo{peerInfo})
	whitePeer2, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", "192.168.102.123", 3002))
	//在白名单内部，校验通过
	assert.True(t, gater2.checkWhitAddr(whitePeer2))
	//通过peerID 校验白名单
	assert.True(t, gater2.checkWhitePeerList(peerInfo.ID))

}

func Test_InterceptAddrDial(t *testing.T) {
	var host1 host.Host
	gater := NewConnGater(&host1, 0, nil, nil)
	var ip = "47.97.223.101"
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 3000))
	require.NoError(t, err)
	require.True(t, gater.InterceptAddrDial("", multiAddress))
}

func Test_InterceptPeerDial(t *testing.T) {
	var host1 host.Host
	ctx := context.Background()
	defer ctx.Done()
	whitePeer1, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%v", "192.168.105.123", 3001, "16Uiu2HAmK9PAPYoTzHnobzB5nQFnY7p9ZVcJYQ1BgzKCr7izAhbJ"))
	peerInfo, err := peer.AddrInfoFromP2pAddr(whitePeer1)
	require.Nil(t, err)
	gater := NewConnGater(&host1, 1, NewTimeCache(context.Background(), time.Second), nil)
	var pid = "16Uiu2HAmCyJhBvE1vn62MQWhhaPph1cxeU9nNZJoZQ1Pe1xASZUg"

	gater.blacklist.Add(pid, 0)
	id, err := peer.Decode(pid)
	require.NoError(t, err)
	ok := gater.InterceptPeerDial(id)
	require.False(t, ok)
	time.Sleep(time.Second * 2)
	ok = gater.InterceptPeerDial(id)
	require.True(t, ok)
	//白名单校验
	gater = NewConnGater(&host1, 1, NewTimeCache(context.Background(), time.Second), []*peer.AddrInfo{peerInfo})
	ok = gater.InterceptPeerDial(id)
	//因为ID不在白名单内部，所有会被拦截
	require.False(t, ok)

}

func Test_otherInterface(t *testing.T) {
	var host1 host.Host
	ctx := context.Background()
	defer ctx.Done()
	gater := NewConnGater(&host1, 1, NewTimeCache(context.Background(), time.Second), nil)
	allow, _ := gater.InterceptUpgraded(nil)
	require.True(t, allow)
	require.True(t, gater.InterceptSecured(network.DirInbound, "", nil))

}

func Test_timecache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewTimeCache(ctx, time.Second)
	cache.Add("one", 0)
	cache.Add("two", time.Second*3)
	cache.Add("three", time.Second*5)
	time.Sleep(time.Second * 2)
	require.False(t, cache.Has("one"))
	require.True(t, cache.Has("two"))
	require.True(t, cache.Has("three"))
	time.Sleep(time.Second * 2)
	require.False(t, cache.Has("two"))
	require.True(t, cache.Has("three"))
	cancel()
	time.Sleep(time.Second * 2)
	require.True(t, cache.Has("three"))

}
