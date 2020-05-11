package dht

import (
	"bufio"
	"context"

	"os"

	"github.com/33cn/chain33/util"

	"crypto/rand"
	"fmt"
	l "github.com/33cn/chain33/common/log"
	p2p2 "github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	circuit "github.com/libp2p/go-libp2p-circuit"
	core "github.com/libp2p/go-libp2p-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/multiformats/go-multiaddr"
	"testing"
	"time"

	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/stretchr/testify/assert"
)

func init() {
	l.SetLogLevel("err")
}

func processMsg(q queue.Queue) {
	go func() {
		cfg := q.GetConfig()
		wcli := wallet.New(cfg)
		client := q.Client()
		wcli.SetQueueClient(client)
		//导入种子，解锁钱包
		password := "a12345678"
		seed := "cushion canal bitter result harvest sentence ability time steel basket useful ask depth sorry area course purpose search exile chapter mountain project ranch buffalo"
		saveSeedByPw := &types.SaveSeedByPw{Seed: seed, Passwd: password}
		_, err := wcli.GetAPI().ExecWalletFunc("wallet", "SaveSeed", saveSeedByPw)
		if err != nil {
			return
		}
		walletUnLock := &types.WalletUnLock{
			Passwd:         password,
			Timeout:        0,
			WalletOrTicket: false,
		}

		_, err = wcli.GetAPI().ExecWalletFunc("wallet", "WalletUnLock", walletUnLock)
		if err != nil {
			return
		}
	}()

	go func() {
		blockchainKey := "blockchain"
		client := q.Client()
		client.Sub(blockchainKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetBlocks:
				if req, ok := msg.GetData().(*types.ReqBlocks); ok {
					if req.Start == 1 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.BlockDetails{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}

			case types.EventGetHeaders:
				if req, ok := msg.GetData().(*types.ReqBlocks); ok {
					if req.Start == 10 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventHeaders, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventHeaders, &types.Headers{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}

			case types.EventGetLastHeader:
				msg.Reply(client.NewMessage("p2p", types.EventHeader, &types.Header{Height: 2019}))
			case types.EventGetBlockHeight:

				msg.Reply(client.NewMessage("p2p", types.EventReplyBlockHeight, &types.ReplyBlockHeight{Height: 2019}))

			}

		}

	}()

	go func() {
		mempoolKey := "mempool"
		client := q.Client()
		client.Sub(mempoolKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetMempoolSize:
				msg.Reply(client.NewMessage("p2p", types.EventMempoolSize, &types.MempoolSize{Size: 0}))
			}
		}
	}()
}

func NewP2p(cfg *types.Chain33Config) p2p2.IP2P {
	p2pmgr := p2p2.NewP2PMgr(cfg)
	subCfg := p2pmgr.ChainCfg.GetSubConfig().P2P
	p2p := New(p2pmgr, subCfg[p2pty.DHTTypeName])
	p2p.StartP2P()
	return p2p
}

func testP2PEvent(t *testing.T, qcli queue.Client) {

	msg := qcli.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{})
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{})
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{})
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventGetMempool, nil)
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventPeerInfo, &types.P2PGetPeerReq{P2PType: "DHT"})
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventGetNetInfo, nil)
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventFetchBlockHeaders, &types.ReqBlocks{})
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

}

func testP2PClose(t *testing.T, p2p p2p2.IP2P) {
	p2p.CloseP2P()

}

func testStreamEOFReSet(t *testing.T) {
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

	r = rand.Reader
	prvKey3, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	msgID := "/streamTest"

	var subcfg p2pty.P2PSubConfig
	subcfg.Port = 12345
	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg.Port))
	if err != nil {
		panic(err)
	}
	h1 := newHost(&subcfg, prvKey1, nil, maddr)
	//-------------------------
	var subcfg2 p2pty.P2PSubConfig
	subcfg2.Port = 12346
	maddr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg2.Port))
	if err != nil {
		panic(err)
	}
	h2 := newHost(&subcfg2, prvKey2, nil, maddr)

	//-------------------------------------
	var subcfg3 p2pty.P2PSubConfig
	subcfg3.Port = 12347

	maddr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg3.Port))
	if err != nil {
		panic(err)
	}
	h3 := newHost(&subcfg3, prvKey3, nil, maddr)
	h1.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		t.Log("Meow! It worked!")
		var buf []byte
		_, err := s.Read(buf)
		if err != nil {
			t.Log("readStreamErr", err)
		}
		s.Close()
	})

	h2.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		t.Log("H2 Stream! It worked!")
		s.Close()
	})

	h3.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		t.Log("H3 Stream! It worked!")
		s.Conn().Close()
	})

	h2info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	err = h1.Connect(context.Background(), h2info)
	assert.NoError(t, err)

	s, err := h1.NewStream(context.Background(), h2.ID(), protocol.ID(msgID))
	assert.NoError(t, err)

	s.Write([]byte("hello"))
	var buf = make([]byte, 128)
	_, err = s.Read(buf)
	assert.True(t, err != nil)
	if err != nil {
		//在stream关闭的时候返回EOF
		t.Log("readStream from H2 Err", err)
		assert.Equal(t, err.Error(), "EOF")
	}

	h3info := peer.AddrInfo{
		ID:    h3.ID(),
		Addrs: h3.Addrs(),
	}

	err = h1.Connect(context.Background(), h3info)
	assert.NoError(t, err)
	s, err = h1.NewStream(context.Background(), h3.ID(), protocol.ID(msgID))
	assert.NoError(t, err)

	s.Write([]byte("hello"))
	_, err = s.Read(buf)
	assert.True(t, err != nil)
	if err != nil {
		//在连接断开的时候，返回 stream reset
		t.Log("readStream from H3 Err", err)
		assert.Equal(t, err.Error(), "stream reset")
	}

}

func testRelay(t *testing.T) {
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

	r = rand.Reader
	prvKey3, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	var subcfg p2pty.P2PSubConfig
	subcfg.Port = 12345
	subcfg.Relay_Discovery = true
	subcfg.Relay_Active = false
	subcfg.Relay_Hop = false
	h1 := newHost(&subcfg, prvKey1, nil, nil)

	//--------------------------------
	var subcfg2 p2pty.P2PSubConfig
	subcfg2.Port = 12346
	subcfg2.Relay_Hop = true //接收中继客户端的请求，proxy 作为中继节点必须配置此选项
	subcfg2.Relay_Discovery = false
	subcfg2.Relay_Active = false
	h2 := newHost(&subcfg2, prvKey2, nil, nil)

	//----------------------------------
	var subcfg3 p2pty.P2PSubConfig
	subcfg3.Port = 12347
	subcfg3.Relay_Active = false
	subcfg3.Relay_Discovery = false
	subcfg3.Relay_Hop = true
	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg3.Port))
	if err != nil {
		panic(err)
	}
	h3 := newHost(&subcfg3, prvKey3, nil, maddr)

	//-------------------------------
	h2info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	t.Log("h1 addrs", h1.Addrs())
	t.Log("h2 addrs", h2.Addrs())
	t.Log("h3 addrs", h3.Addrs())
	// Connect both h1 and h3 to h2, but not to each other
	if err := h1.Connect(context.Background(), h2info); err != nil {
		panic(err)
	}
	if err := h3.Connect(context.Background(), h2info); err != nil {
		panic(err)
	}

	// Now, to test things, let's set up a protocol handler on h3
	h3.SetStreamHandler("/relays", func(s network.Stream) {
		//fmt.Println("Meow! It worked!")
		t.Log("SteamHandler,ReadIn")
		s.Write([]byte("Meow! It worked!\n"))

	})
	//基于预期，h1一定不会连上h3
	_, err = h1.NewStream(context.Background(), h3.ID(), "/relays")
	assert.NotNil(t, err)

	t.Log("Okay, no connection from h1 to h3: ", err, "h3 id", h3.ID())

	// Creates a relay address
	relayaddr, err := multiaddr.NewMultiaddr("/p2p-circuit/ipfs/" + h3.ID().Pretty())
	assert.Nil(t, err)
	t.Log("h3 relayaddr", relayaddr.String())
	h1.Network().(*swarm.Swarm).Backoff().Clear(h3.ID())
	h3relayInfo := peer.AddrInfo{
		ID:    h3.ID(),
		Addrs: []multiaddr.Multiaddr{relayaddr},
	}
	//h1.Network().ClosePeer(h2.ID())
	err = h1.Connect(context.Background(), h3relayInfo)
	assert.Nil(t, err)
	// Woohoo! we're connected!
	s, err := h1.NewStream(context.Background(), h3.ID(), "/relays")
	if err != nil {
		t.Log("h1.Newstream h3", err)
	}
	assert.Nil(t, err)

	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	assert.Nil(t, err)
	t.Log("my read:", str)

}

func testRelay_v2(t *testing.T) {
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

	r = rand.Reader
	prvKey3, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	prvKey4, _, _ := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	var subcfg p2pty.P2PSubConfig
	subcfg.Port = 12345
	subcfg.Relay_Discovery = true
	subcfg.Relay_Active = false
	subcfg.Relay_Hop = false
	h1 := newHost(&subcfg, prvKey1, nil, nil)

	//--------------------------------
	var subcfg2 p2pty.P2PSubConfig
	subcfg2.Port = 12346
	subcfg2.Relay_Hop = true //接收中继客户端的请求，proxy 作为中继节点必须配置此选项
	subcfg2.Relay_Discovery = false
	subcfg2.Relay_Active = false
	h2 := newHost(&subcfg2, prvKey2, nil, nil)

	//----------------------------------
	var subcfg3 p2pty.P2PSubConfig
	subcfg3.Port = 12347
	subcfg3.Relay_Active = false
	subcfg3.Relay_Discovery = false
	subcfg3.Relay_Hop = true
	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg3.Port))
	if err != nil {
		panic(err)
	}
	h3 := newHost(&subcfg3, prvKey3, nil, maddr)
	// Now, to test things, let's set up a protocol handler on h3
	h3.SetStreamHandler("/relays", func(s network.Stream) {
		//fmt.Println("Meow! It worked!")
		t.Log("SteamHandler,ReadIn")
		s.Write([]byte("Meow! It worked!\n"))
		s.Close()
	})
	//-------------------------------
	//----------------------------------
	var subcfg4 p2pty.P2PSubConfig
	subcfg4.Port = 12349

	h4 := newHost(&subcfg4, prvKey4, nil, nil)

	h4.SetStreamHandler("/relays", func(s network.Stream) {
		//fmt.Println("Meow! It worked!")
		t.Log("h4 handler,ReadIn")
		s.Write([]byte("h4 It worked!\n"))
		s.Close()
	})
	//-------------------------------
	var maddrs []multiaddr.Multiaddr
	for _, addr := range h2.Addrs() {
		astr := addr.String()
		relayAddr := fmt.Sprintf("%s/p2p/%s/p2p-circuit/p2p/%s", astr, h2.ID(), h3.ID())
		t.Log("relayAddr", relayAddr)
		maddr, _ := multiaddr.NewMultiaddr(relayAddr)
		maddrs = append(maddrs, maddr)
	}

	h2info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}

	t.Log("h1 addrs", h1.Addrs())
	t.Log("h2 id", h2.ID(), "h2 addrs", h2.Addrs())
	t.Log("h3 id", h3.ID(), "h3 addrs", h3.Addrs())
	t.Log("h4 id", h4.ID(), "h4 addrs", h4.Addrs())
	// Connect both h1 and h3 to h2, but not to each other

	if err := h3.Connect(context.Background(), h2info); err != nil {
		panic(err)
	}

	h1.Peerstore().AddAddrs(h3.ID(), maddrs, peerstore.TempAddrTTL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := h1.NewStream(ctx, h3.ID(), "/relays")
	if err != nil {
		t.Log("h1.Newstream h3", err)
		return
	}

	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	assert.Nil(t, err)
	t.Log("my read again:", str)

}
func newTestRelay(t *testing.T, ctx context.Context, host host.Host, opts ...circuit.RelayOpt) *circuit.Relay {
	r, err := circuit.NewRelay(ctx, host, swarmt.GenUpgrader(host.Network().(*swarm.Swarm)), opts...)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func getNetHosts(t *testing.T, ctx context.Context, n int) []host.Host {
	var out []host.Host

	for i := 0; i < n; i++ {
		netw := swarmt.GenSwarm(t, ctx)
		h := bhost.NewBlankHost(netw)
		out = append(out, h)
	}

	return out
}
func connect(t *testing.T, a, b host.Host) {
	pinfo := a.Peerstore().PeerInfo(a.ID())
	err := b.Connect(context.Background(), pinfo)
	if err != nil {
		t.Fatal(err)
	}
}

func testRelay_v3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := getNetHosts(t, ctx, 3)

	connect(t, hosts[0], hosts[1])
	r1 := newTestRelay(t, ctx, hosts[0], circuit.OptDiscovery)
	newTestRelay(t, ctx, hosts[1], circuit.OptHop)

	rinfo := hosts[1].Peerstore().PeerInfo(hosts[1].ID())
	dinfo := hosts[2].Peerstore().PeerInfo(hosts[2].ID())

	rctx, rcancel := context.WithTimeout(ctx, time.Second)
	defer rcancel()
	_, err := r1.DialPeer(rctx, rinfo, dinfo)
	assert.NotNil(t, err)

}

func Test_p2p(t *testing.T) {

	cfg := types.NewChain33Config(types.ReadFile("../../../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	datadir := util.ResetDatadir(cfg.GetModuleConfig(), "$TEMP/")
	q.SetConfig(cfg)
	processMsg(q)
	p2p := NewP2p(cfg)
	defer func(path string) {
		if err := os.RemoveAll(path); err != nil {
			log.Error("removeTestDatadir", "err", err)
		}
	}(datadir)
	testP2PEvent(t, q.Client())
	testP2PClose(t, p2p)
	testStreamEOFReSet(t)
	testRelay(t)
	testRelay_v2(t)
	testRelay_v3(t)
}
