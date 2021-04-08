package dht

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/33cn/chain33/client"
	l "github.com/33cn/chain33/common/log"
	p2p2 "github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/wallet"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
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

func NewP2p(cfg *types.Chain33Config, cli queue.Client) p2p2.IP2P {
	p2pmgr := p2p2.NewP2PMgr(cfg)
	p2pmgr.SysAPI, _ = client.New(cli, nil)
	p2pmgr.Client = cli
	subCfg := p2pmgr.ChainCfg.GetSubConfig().P2P
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(subCfg[p2pty.DHTTypeName], mcfg)
	mcfg.RelayEnable = true

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

}

func newHost(subcfg *p2pty.P2PSubConfig, priv crypto.PrivKey, bandwidthTracker metrics.Reporter, maddr multiaddr.Multiaddr) host.Host {
	p := &P2P{ctx: context.Background(), subCfg: subcfg}
	options := p.buildHostOptions(priv, bandwidthTracker, maddr, nil)
	h, err := libp2p.New(context.Background(), options...)
	if err != nil {
		return nil
	}
	return h
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

	var subcfg, subcfg2, subcfg3 p2pty.P2PSubConfig
	subcfg.Port = 22345
	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg.Port))
	if err != nil {
		panic(err)
	}
	h1 := newHost(&subcfg, prvKey1, nil, maddr)
	t.Log("h1", h1.ID())
	//-------------------------

	subcfg2.Port = 22346
	maddr2, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg2.Port))
	if err != nil {
		panic(err)
	}
	h2 := newHost(&subcfg2, prvKey2, nil, maddr2)
	t.Log("h2", h2.ID())
	//-------------------------------------

	subcfg3.Port = 22347
	maddr3, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", subcfg3.Port))
	if err != nil {
		panic(err)
	}
	h3 := newHost(&subcfg3, prvKey3, nil, maddr3)
	t.Log("h3", h3.ID())
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
	var retrycount int
	for {
		err = h1.Connect(context.Background(), h2info)
		if err != nil {
			retrycount++
			if retrycount > 3 {
				break
			}
			continue
		}

		break
	}

	require.NoError(t, err)

	s, err := h1.NewStream(context.Background(), h2.ID(), protocol.ID(msgID))
	require.NoError(t, err)

	s.Write([]byte("hello"))
	var buf = make([]byte, 128)
	_, err = s.Read(buf)
	require.True(t, err != nil)
	if err != nil {
		//在stream关闭的时候返回EOF
		t.Log("readStream from H2 Err", err)
		require.Equal(t, err.Error(), "EOF")
	}

	h3info := peer.AddrInfo{
		ID:    h3.ID(),
		Addrs: h3.Addrs(),
	}

	err = h1.Connect(context.Background(), h3info)
	require.NoError(t, err)
	s, err = h1.NewStream(context.Background(), h3.ID(), protocol.ID(msgID))
	require.NoError(t, err)

	s.Write([]byte("hello"))
	_, err = s.Read(buf)
	require.True(t, err != nil)
	if err != nil {
		//在连接断开的时候，返回 stream reset
		t.Log("readStream from H3 Err", err)
		require.Equal(t, err.Error(), "stream reset")
	}

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

func testAddrbook(t *testing.T, cfg *types.P2P) {
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
	q.SetConfig(cfg)
	processMsg(q)

	defer func(path string) {

		if err := os.RemoveAll(path); err != nil {
			log.Error("removeTestDatadir", "err", err.Error())
		}
		t.Log("removed path", path)
	}(datadir)

	var tcfg types.P2P
	tcfg.Driver = "leveldb"
	tcfg.DbCache = 4
	tcfg.DbPath = filepath.Join(datadir, "addrbook")
	testAddrbook(t, &tcfg)

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
	//netw := swarmt.GenSwarm(t, context.Background())
	//h2 := bhost.NewBlankHost(netw)
	//dhtp2p.pruePeers(h2.ID(), true)
	dhtp2p.discovery.Remove(dhtp2p.host.ID())
	testP2PEvent(t, q.Client())

	testStreamEOFReSet(t)
	testHost(t)
	p2p.CloseP2P()
	time.Sleep(time.Second)
}
