package dht

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/mock"

	clientMocks "github.com/33cn/chain33/client/mocks"

	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "github.com/libp2p/go-libp2p/core"

	"github.com/stretchr/testify/assert"

	l "github.com/33cn/chain33/common/log"
	p2p2 "github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	cprotocol "github.com/33cn/chain33/system/p2p/dht/protocol"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
)

var qapi *clientMocks.QueueProtocolAPI

func init() {
	l.SetLogLevel("err")
	qapi = &clientMocks.QueueProtocolAPI{}
}

func processMsg(q queue.Queue) {

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
			default:
				fmt.Println("no support eventid", msg.Ty)
				msg.Reply(client.NewMessage("p2p", msg.Ty, &types.Reply{IsOk: false}))
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

func NewP2p(cfg *types.Chain33Config, cli queue.Client) p2p2.IP2P {
	p2pmgr := p2p2.NewP2PMgr(cfg)
	//p2pmgr.SysAPI, _ = client.New(cli, nil)
	p2pmgr.Client = cli
	p2pmgr.SysAPI = qapi
	subCfg := p2pmgr.ChainCfg.GetSubConfig().P2P
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(subCfg[p2pty.DHTTypeName], mcfg)
	//NewAddrBook(cfg.GetModuleConfig().P2P).initKey()
	dhtcfg, _ := json.Marshal(mcfg)
	p2p := New(p2pmgr, dhtcfg)
	p2p.StartP2P()
	return p2p
}

func testP2PEvent(t *testing.T, qcli queue.Client) {

	msg := qcli.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{})
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{})
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{})
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventGetMempool, nil)
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventPeerInfo, &types.P2PGetPeerReq{P2PType: "DHT"})
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventGetNetInfo, nil)
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventFetchBlockHeaders, &types.ReqBlocks{})
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventAddBlacklist, &types.BlackPeer{PeerName: "16Uiu2HAm7vDB7XDuEv8XNPcoPqumVngsjWoogGXENNDXVYMiCJHM", Lifetime: "1second"})
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventShowBlacklist, &types.ReqNil{})
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventDelBlacklist, &types.BlackPeer{PeerName: "16Uiu2HAm7vDB7XDuEv8XNPcoPqumVngsjWoogGXENNDXVYMiCJHM"})
	qcli.Send(msg, false)
	require.True(t, qcli.Send(msg, false) == nil)

}

func newHost(subcfg *p2pty.P2PSubConfig, priv crypto.PrivKey, bandwidthTracker metrics.Reporter, maddr multiaddr.Multiaddr) host.Host {
	p := &P2P{ctx: context.Background(), subCfg: subcfg}
	options := p.buildHostOptions(priv, bandwidthTracker, maddr, nil)
	h, err := libp2p.New(options...)
	if err != nil {
		return nil
	}
	return h
}
func TestPrivateNetwork(t *testing.T) {
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
	var subcfg, subcfg2 p2pty.P2PSubConfig
	subcfg.Port = 22345
	subcfg2.Port = 22346
	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", subcfg.Port))
	if err != nil {
		panic(err)
	}

	maddr2, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", subcfg2.Port))
	if err != nil {
		panic(err)
	}
	testPSK := make([]byte, 32)
	rand.Reader.Read(testPSK)
	subcfg.Psk = hex.EncodeToString(testPSK)
	subcfg.MaxConnectNum = 1000
	subcfg2.Psk = subcfg.Psk
	h1 := newHost(&subcfg, prvKey1, nil, maddr)
	h2 := newHost(&subcfg2, prvKey2, nil, maddr2)
	h2info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	err = h1.Connect(context.Background(), h2info)
	//must be connect
	assert.Nil(t, err)
	t.Log("same privatenetwork test success")
	h2.Close()
	//断开与h2的网络连接
	h1.Network().ClosePeer(h2.ID())
	//h2 采用另外一种privatekey
	var testPsk2 [32]byte
	copy(testPsk2[:], testPSK)
	testPsk2[0] = 0x33
	testPsk2[31] = 0x34
	subcfg2.Psk = hex.EncodeToString(testPsk2[:])
	h2 = newHost(&subcfg2, prvKey2, nil, maddr2)
	h2info = peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = h1.Connect(ctx, h2info)
	assert.NotNil(t, err)
	//t.Log("err:",err)
	t.Log("different privatenetwork test success")

	//测试没有启用privatenetwork相连接
	h1.Close()
	h2.Close()
	subcfg2.Psk = ""

	h1 = newHost(&subcfg, prvKey1, nil, maddr)
	h2 = newHost(&subcfg2, prvKey2, nil, maddr2)
	h2info = peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	err = h1.Connect(ctx, h2info)
	assert.NotNil(t, err)

	h1info := peer.AddrInfo{
		ID:    h1.ID(),
		Addrs: h1.Addrs(),
	}

	err = h2.Connect(ctx, h1info)
	assert.NotNil(t, err)
	t.Log("disable privatenetwork test success")

}

func testStreamEOFReSet(t *testing.T) {
	hosts := getNetHosts(context.Background(), 3, t)
	msgID := "/streamTest/1.0"
	h1 := hosts[0]
	h2 := hosts[1]
	h3 := hosts[2]
	t.Log("h1", h1.ID(), "h1.addr", h1.Addrs())
	t.Log("h2:", h2.ID(), "h2.addr", h2.Addrs())
	t.Log("h3:", h3.ID(), "h3.addr", h3.Addrs())

	h1.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		t.Log("Meow! It worked!")
		s.Close()
	})

	h2.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		t.Log("H2 Stream! It worked!")
		s.Close()
	})

	h3.SetStreamHandler(protocol.ID(msgID), func(s core.Stream) {
		t.Log("H3 Stream! It worked!")
		var req types.ReqNil
		err := cprotocol.ReadStream(&req, s)
		if err != nil {
			t.Log("readstream:", err)
		}
		require.NoError(t, err)
		s.Conn().Close()
	})

	h2info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	err := h1.Connect(context.Background(), h2info)
	require.NoError(t, err)

	s, err := h1.NewStream(context.Background(), h2.ID(), protocol.ID(msgID))
	require.NoError(t, err)

	var resp types.ReqNil
	err = cprotocol.ReadStream(&resp, s)
	require.True(t, err != nil)
	if err != nil {
		//在stream关闭的时候返回EOF
		t.Log("readStream from H2 Err", err)
		require.Equal(t, err.Error(), "EOF")
		s.Reset()
	}

	err = h1.Connect(context.Background(), peer.AddrInfo{
		ID:    h3.ID(),
		Addrs: h3.Addrs(),
	})
	require.NoError(t, err)

	s, err = h1.NewStream(context.Background(), h3.ID(), protocol.ID(msgID))
	if err != nil {
		t.Log("newStream err:", err.Error())
	}
	require.NoError(t, err)

	err = cprotocol.WriteStream(&types.ReqNil{}, s)
	if err != nil {
		t.Log("newStream Write err:", err.Error())
	}
	require.NoError(t, err)

	err = cprotocol.ReadStream(&resp, s)
	require.True(t, err != nil)
	if err != nil {
		//服务端close connect 之后，客户端会触发：Application error 0x0 或者stream reset
		t.Log("readStream from H3 Err:", err)
		if !strings.Contains(err.Error(), "Application error 0x0") {
			require.Equal(t, err.Error(), "stream reset")
		}
	}

	h1.Close()
	h2.Close()
	h3.Close()
}

func Test_pubkey(t *testing.T) {
	priv, pub, err := GenPrivPubkey()
	require.Nil(t, err)
	require.NotNil(t, priv, pub)
	pubstr, err := GenPubkey(hex.EncodeToString(priv))
	require.Nil(t, err)
	require.Equal(t, pubstr, hex.EncodeToString(pub))
}

func testHost(t *testing.T) {
	mcfg := &p2pty.P2PSubConfig{}

	_, err := GenPubkey("123456")
	require.NotNil(t, err)

	priv, pub, err := GenPrivPubkey()
	require.Nil(t, err)
	t.Log("priv size", len(priv))
	cpriv, err := crypto.UnmarshalSecp256k1PrivateKey(priv)
	require.Nil(t, err)

	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 26666))

	if err != nil {
		return
	}
	mcfg.RelayHop = true
	mcfg.MaxConnectNum = 10000
	host := newHost(mcfg, cpriv, nil, maddr)
	hpub := host.Peerstore().PubKey(host.ID())
	hpb, err := hpub.Raw()
	require.Nil(t, err)
	require.Equal(t, hpb, pub)
	host.Close()
}

func testAddrbook(t *testing.T, cfg *types.P2P) *AddrBook {
	cfg.DbPath = cfg.DbPath + "test/"
	addrbook := NewAddrBook(cfg)
	priv, pub := addrbook.GetPrivPubKey()
	require.Equal(t, priv, "")
	require.Equal(t, pub, "")
	var paddrinfos []peer.AddrInfo
	paddrinfos = append(paddrinfos, peer.AddrInfo{})
	addrbook.SaveAddr(paddrinfos)
	addrbook.Randkey()
	priv, pub = addrbook.GetPrivPubKey()
	addrbook.saveKey(priv, pub)
	ok := addrbook.loadDb()
	require.True(t, ok)
	return addrbook
}

func Test_LocalAddr(t *testing.T) {
	seedip := "120.24.92.123:13803"
	t.Log("seedip", seedip)
	spliteIP := strings.Split(seedip, ":")
	conn, err := net.DialTimeout("tcp4", net.JoinHostPort(spliteIP[0], spliteIP[1]), time.Second)
	if err != nil {
		t.Log("Could not dial remote peer")
		return
	}

	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	t.Log("test localAddr", localAddr)
}
func Test_Id(t *testing.T) {
	encodeIDStr := "16Uiu2HAm7vDB7XDuEv8XNPcoPqumVngsjWoogGXENNDXVYMiCJHM"
	pubkey, err := PeerIDToPubkey(encodeIDStr)
	require.Nil(t, err)
	require.Equal(t, pubkey, "02b99bc73bfb522110634d5644d476b21b3171eefab517da0646ef2aba39dbf4a0")

}
func Test_genAddrInfos(t *testing.T) {
	peer := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%v", "192.168.105.123", 3001, "16Uiu2HAmK9PAPYoTzHnobzB5nQFnY7p9ZVcJYQ1BgzKCr7izAhbJ")
	addrinfos := genAddrInfos([]string{peer})
	require.NotNil(t, addrinfos)
	require.Equal(t, 1, len(addrinfos))
	peer = fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%v", "afasfase", 3001, "16Uiu2HAmK9PAPYoTzHnobzB5nQFnY7p9ZVcJYQ1BgzKCr7izAhbJ")
	addrinfos = genAddrInfos([]string{peer})
	require.Nil(t, addrinfos)
	peer2 := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%v", "192.168.105.124", 3001, "16Uiu2HAkxg1xyu3Ja2MERb3KyBev2CAKXvZwoe6EQgyg2tstg9wr")
	peer3 := fmt.Sprintf("%v:%d", "192.168.105.124", 3001)
	addrinfos = genAddrInfos([]string{peer, peer2, peer3})
	require.NotNil(t, addrinfos)
	require.Equal(t, 1, len(addrinfos))
	t.Log(addrinfos[0].Addrs[0].String())
}
func Test_p2p(t *testing.T) {
	cfg := types.NewChain33Config(types.ReadFile("../../../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	datadir := util.ResetDatadir(cfg.GetModuleConfig(), "$TEMP/")
	setLibp2pLog(cfg.GetModuleConfig().Log.LogFile, "")
	cfg.GetModuleConfig().Log.LogFile = ""
	cfg.GetModuleConfig().Address.DefaultDriver = "BTC"
	q.SetConfig(cfg)
	processMsg(q)

	defer func(path string) {

		if err := os.RemoveAll(path); err != nil {
			log.Error("removeTestDatadir", "err", err.Error())
		}
		t.Log("removed path", path)
	}(datadir)

	qapi.On("ExecWalletFunc", "wallet", "NewAccountByIndex", mock.Anything).Return(&types.ReplyString{Data: "87a229d16d035b033434274303cb3b9994a85cd66b2dd091f2a2efdd2e74a5e1"}, nil)
	qapi.On("ExecWalletFunc", "wallet", "GetWalletStatus", &types.ReqNil{}).Return(&types.WalletStatus{IsWalletLock: false, IsHasSeed: true, IsAutoMining: false}, nil)
	qapi.On("ExecWalletFunc", "wallet", "WalletImportPrivkey", mock.Anything).Return(&types.Reply{
		IsOk: true,
	}, nil)
	var mcfg p2pty.P2PSubConfig
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], &mcfg)
	jcfg, err := json.Marshal(mcfg)
	require.Nil(t, err)
	cfg.GetSubConfig().P2P[p2pty.DHTTypeName] = jcfg
	p2p := NewP2p(cfg, q.Client())
	dhtp2p := p2p.(*P2P)

	t.Log("listpeer", dhtp2p.discovery.ListPeers())
	err = dhtp2p.discovery.Update(dhtp2p.host.ID())
	t.Log("discovery update", err)
	pinfo := dhtp2p.discovery.FindLocalPeers([]peer.ID{dhtp2p.host.ID()})
	t.Log("findlocalPeers", pinfo)
	dhtp2p.discovery.Remove(dhtp2p.host.ID())
	testP2PEvent(t, q.Client())
	testStreamEOFReSet(t)
	testHost(t)

	var tcfg types.P2P
	tcfg.Driver = "leveldb"
	tcfg.DbCache = 4
	tcfg.DbPath = filepath.Join(datadir, "addrbook")
	testAddrbook(t, &tcfg)
	dhtp2p.reStart()
}
