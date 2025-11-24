package p2pstore

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	kbt "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

const dataDir = "./datadir"
const chunkNum int64 = 1000
const blockHeight int64 = 14000 - 1
const p2pA, p2pB = "p2pA", "p2pB"

// HOST1 ID: Qma91H212PWtAFcioW7h9eKiosJtwHsb9x3RmjqRWTwciZ
// HOST2 ID: QmbazrBU4HthhnQWcUTiJLnj5ihbFHXsAkGAG6QfmrqJDs
func TestInit(t *testing.T) {
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)
	protocol.ClearEventHandler()
	var err error
	q := queue.New("test")
	msgCh := initMockBlockchain(q)
	p1, p2 := initEnv(t, q)
	_ = p1
	client := q.Client()
	//client用来模拟blockchain模块向p2p模块发消息，host1接收topic为p2pA的消息，host2接收topic为p2pB的消息
	//向host1请求数据
	var ChunkCount int
	for i := 0; i < 14; i++ {
		info := &types.ChunkInfoMsg{
			ChunkHash: []byte(fmt.Sprintf("test%d", i)),
			Start:     int64(i * 1000),
			End:       int64(i*1000 + 999),
		}
		msg := testGetBody(t, client, p2pA, info)
		require.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))

		msg2 := testGetBody(t, client, p2pB, info)
		require.Equal(t, 1000, len(msg2.Data.(*types.BlockBodys).Items))

		_, err = p2.loadChunk(info)
		if err == nil {
			ChunkCount++
		}
	}

	require.Equal(t, 6, ChunkCount)

	//向host1请求BlockBody
	msg := testGetBody(t, client, p2pB, &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       99,
	})

	require.Equal(t, 100, len(msg.Data.(*types.BlockBodys).Items))
	//向host2请求BlockBody
	msg = testGetBody(t, client, p2pA, &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     666,
		End:       888,
	})
	blockBodys, ok := msg.Data.(*types.BlockBodys)
	require.Equal(t, true, ok)
	bodys := blockBodys.Items
	require.Equal(t, 223, len(bodys))
	require.Equal(t, int64(666), bodys[0].Height)
	require.Equal(t, int64(888), bodys[222].Height)

	//向host1请求Block
	testGetBlock(t, client, p2pA, &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       499,
	})
	msg = <-msgCh
	require.Equal(t, 500, len(msg.Data.(*types.Blocks).Items))

	//向host2请求Block
	testGetBlock(t, client, p2pB, &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     111,
		End:       666,
	})
	msg = <-msgCh
	require.Equal(t, 556, len(msg.Data.(*types.Blocks).Items))
	//向host1请求Records
	testGetRecord(t, client, p2pA, &types.ReqChunkRecords{
		Start: 0,
		End:   13,
	})
	msg = <-msgCh
	require.Equal(t, 14, len(msg.Data.(*types.ChunkRecords).Infos))

	//向host2请求BlockBody
	msg = testGetBody(t, client, p2pB, &types.ChunkInfoMsg{
		ChunkHash: []byte("test1"),
		Start:     1666,
		End:       1888,
	})
	require.Equal(t, 223, len(msg.Data.(*types.BlockBodys).Items))

	// 删除数据后应该找不到数据
	err = p2.deleteChunkBlock(&types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       999,
	})
	if err != nil {
		t.Fatal(err)
	}
	//向host2请求BlockBody
	msg = testGetBody(t, client, p2pB, &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       999,
	})
	require.False(t, msg.Data.(*types.Reply).IsOk)

	testGetHeaders(t, client, p2pA, &types.ReqBlocks{
		Start: 100,
		End:   199,
		Pid:   []string{p2.Host.ID().Pretty()},
	})
	msg = <-msgCh
	require.Equal(t, p2.Host.ID().Pretty(), msg.Data.(*types.HeadersPid).GetPid())
	require.Equal(t, 100, len(msg.Data.(*types.HeadersPid).GetHeaders().Items))

	//　更新数据后应该能查到数据
	p2.refreshLocalChunk()
	for i := 0; i < 14; i++ {
		info := &types.ChunkInfoMsg{
			ChunkHash: []byte(fmt.Sprintf("test%d", i)),
			Start:     int64(i * 1000),
			End:       int64(i*1000 + 999),
		}
		msg := testGetBody(t, client, p2pA, info)
		require.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))

		msg2 := testGetBody(t, client, p2pB, info)
		require.Equal(t, 1000, len(msg2.Data.(*types.BlockBodys).Items))
	}
}

func testGetBody(t *testing.T, client queue.Client, topic string, req *types.ChunkInfoMsg) *queue.Message {
	msg := client.NewMessage(topic, types.EventGetChunkBlockBody, req)
	err := client.Send(msg, true)
	if err != nil {
		t.Fatal(err)
	}
	msg, err = client.Wait(msg)
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func testGetBlock(t *testing.T, client queue.Client, topic string, req *types.ChunkInfoMsg) *queue.Message {
	msg := client.NewMessage(topic, types.EventGetChunkBlock, req)
	err := client.Send(msg, false)
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func testGetRecord(t *testing.T, client queue.Client, topic string, req *types.ReqChunkRecords) *queue.Message {
	msg := client.NewMessage(topic, types.EventGetChunkRecord, req)
	err := client.Send(msg, false)
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func testGetHeaders(t *testing.T, client queue.Client, topic string, req *types.ReqBlocks) *queue.Message {
	msg := client.NewMessage(topic, types.EventFetchBlockHeaders, req)
	err := client.Send(msg, true)
	if err != nil {
		t.Fatal(err)
	}
	msg, err = client.Wait(msg)
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func initMockBlockchain(q queue.Queue) <-chan *queue.Message {
	client := q.Client()
	client.Sub("blockchain")
	ch := make(chan *queue.Message)
	go func() {
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventIsSync:
				msg.Reply(queue.NewMessage(0, "", 0, &types.IsCaughtUp{Iscaughtup: true}))
			case types.EventGetLastHeader:
				msg.Reply(queue.NewMessage(0, "", 0, &types.Header{Height: blockHeight}))
			case types.EventGetChunkBlockBody:
				req := msg.Data.(*types.ChunkInfoMsg)
				bodys := &types.BlockBodys{Items: make([]*types.BlockBody, 0, req.End-req.Start+1)}
				for i := req.Start; i <= req.End; i++ {
					bodys.Items = append(bodys.Items, &types.BlockBody{
						Height: i,
					})
				}
				msg.Reply(queue.NewMessage(0, "", 0, bodys))
			case types.EventGetHeaders:
				req := msg.Data.(*types.ReqBlocks)
				items := make([]*types.Header, 0, req.End-req.Start+1)
				for i := req.Start; i <= req.End; i++ {
					items = append(items, &types.Header{
						Height: i,
					})
				}
				headers := &types.Headers{
					Items: items,
				}
				msg.Reply(queue.NewMessage(0, "", 0, headers))
			case types.EventGetChunkRecord:
				req := msg.Data.(*types.ReqChunkRecords)
				if req.Start > req.End || req.End > blockHeight/chunkNum {
					msg.Reply(&queue.Message{Data: &types.Reply{}})
					break
				}
				records := types.ChunkRecords{}
				for i := req.Start; i <= req.End; i++ {
					records.Infos = append(records.Infos, &types.ChunkInfo{
						ChunkHash: []byte(fmt.Sprintf("test%d", i)),
						Start:     i * 1000,
						End:       i*1000 + 999,
					})
				}
				msg.Reply(&queue.Message{Data: &records})
			default:
				msg.Reply(&queue.Message{})
				ch <- msg
			}
		}
	}()

	return ch
}

func makeProtocol(name string, q queue.Queue, h host.Host) *Protocol {
	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &types2.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[types2.DHTTypeName], mcfg)
	mcfg.DisableFindLANPeers = true
	cli := q.Client()
	if q.GetConfig() == nil {
		q.SetConfig(cfg)
	}
	mockAPI, err := client.New(cli, nil)
	if err != nil {
		panic(err)
	}
	rt, _ := kbt.NewRoutingTable(20, kbt.ConvertPeerID(h.ID()), time.Minute, h.Peerstore(), time.Hour, nil)
	exRT, _ := kbt.NewRoutingTable(20, kbt.ConvertPeerID(h.ID()), time.Minute, h.Peerstore(), time.Hour, nil)
	env := protocol.P2PEnv{
		Ctx:             context.Background(),
		ChainCfg:        cfg,
		API:             mockAPI,
		QueueClient:     cli,
		Host:            h,
		SubConfig:       mcfg,
		RoutingTable:    rt,
		PeerInfoManager: &peerInfoManager{},
		DB:              dbm.NewDB(name, "leveldb", dataDir, 128),
		Discovery:       &defaultDiscovery{},
	}
	p := &Protocol{
		P2PEnv:               &env,
		localChunkInfo:       make(map[string]LocalChunkInfo),
		chunkToSync:          make(chan *types.ChunkInfoMsg, 100),
		chunkToDelete:        make(chan *types.ChunkInfoMsg, 100),
		chunkToDownload:      make(chan *types.ChunkInfoMsg, 1024),
		wakeup:               make(map[string]chan struct{}),
		chunkInfoCache:       make(map[string]*types.ChunkInfoMsg),
		peerAddrRequestTrace: make(map[peer.ID]map[peer.ID]time.Time),
		chunkRequestTrace:    make(map[string]map[peer.ID]time.Time),
		chunkProviderCache:   make(map[string]map[peer.ID]time.Time),
		extendRoutingTable:   exRT,
	}

	//注册p2p通信协议，用于处理节点之间请求
	protocol.RegisterStreamHandler(p.Host, requestPeerInfoForChunk, p.handleStreamRequestPeerInfoForChunk)
	protocol.RegisterStreamHandler(p.Host, responsePeerInfoForChunk, p.handleStreamResponsePeerInfoForChunk)
	protocol.RegisterStreamHandler(p.Host, requestPeerAddr, p.handleStreamRequestPeerAddr)
	protocol.RegisterStreamHandler(p.Host, responsePeerAddr, p.handleStreamResponsePeerAddr)
	protocol.RegisterStreamHandler(p.Host, fetchPeerAddr, protocol.HandlerWithRW(p.handleStreamPeerAddr))
	protocol.RegisterStreamHandler(p.Host, fetchActivePeer, protocol.HandlerWithWrite(p.handleStreamFetchActivePeer))
	protocol.RegisterStreamHandler(p.Host, getHeaderOld, p.handleStreamGetHeaderOld)
	protocol.RegisterStreamHandler(p.Host, fetchShardPeer, protocol.HandlerWithRW(p.handleStreamFetchShardPeers))
	protocol.RegisterStreamHandler(p.Host, fullNode, protocol.HandlerWithWrite(p.handleStreamIsFullNode))
	protocol.RegisterStreamHandler(p.Host, fetchChunk, p.handleStreamFetchChunk) //数据较大，采用特殊写入方式
	protocol.RegisterStreamHandler(p.Host, getHeader, protocol.HandlerWithAuthAndSign(p.Host, p.handleStreamGetHeader))
	protocol.RegisterStreamHandler(p.Host, getChunkRecord, protocol.HandlerWithAuthAndSign(p.Host, p.handleStreamGetChunkRecord))

	cli.Sub(name)

	go func() {
		for msg := range cli.Recv() {
			switch msg.Ty {
			case types.EventNotifyStoreChunk:
				protocol.EventHandlerWithRecover(p.handleEventNotifyStoreChunk)(msg)
			case types.EventGetChunkBlock:
				protocol.EventHandlerWithRecover(p.handleEventGetChunkBlock)(msg)
			case types.EventGetChunkBlockBody:
				protocol.EventHandlerWithRecover(p.handleEventGetChunkBlockBody)(msg)
			case types.EventGetChunkRecord:
				protocol.EventHandlerWithRecover(p.handleEventGetChunkRecord)(msg)
			case types.EventFetchBlockHeaders:
				protocol.EventHandlerWithRecover(p.handleEventGetHeaders)(msg)
			}
		}
	}()

	return p
}

func initEnv(t *testing.T, q queue.Queue) (*Protocol, *Protocol) {

	types2.RefreshInterval = time.Second
	privkey1 := "080012a709308204a30201000282010100a28d698a090b02222f97c928b45e78821a87b6382b5057ec9cf12331da3fd8a1c6c71731a8075ae41383460908b483585676f4312249de6929423c2c5d7865bb28d50c5a57e7cad3cc7ca2ddcbc486ac0260fe68e4cdff7f86e46ac65403baf6a5ef50ce7cbb9d0f5f23b02fcc6d5211e2df2bf24fc84565ba5d0777458ad82b46579cba0a16c88ff946812e7f17ad85a2b35dc1bae732a74f83262358fefcc985a632aee8129a73d1d17aaceebd5bae9ffbeab6c5505e8eafd8af8448a6dd74d76885bc71c7d85bad761680bc7cdd04a99cb90d8c27467769c500e603677469a73cec7983a7dba6d7656ab241b4446355a89a267eeb72f0fd7c89c470d93a6302030100010282010002db797f73a93de05bf5cf136818410608715a42a280470b61b6db6784ee9a603d9e424a1d2a03eefe68d0525854d3fa398addbfff5a4d0e8c2b1de3a9c0f408d62ee888ae02e50dd40a5cd289426b1b9aef1989be7be081dd5d268355f6bad29b1819d3875dc4e500472051b6c6352b1b51d0f3f17313c536016ca02c18c4b3f6dba52c616f93bf831589d0dd2fc190f875e37a4e9654bd9e63e04fc5d9cea45664cd6d26c17659ee4b8c6837c6dfe86e4e6b8e17af332a736267ee5a68ac0b0c60ced47f1aaf7ec65547f664a9f1409e7d116ca325c29b1058e5892dc04c79337a15b875e7139bca7ddfb6c5c7f822adff8cd65f1dfa84d1b0f87166604c0102818100c5694c5a55465a068075e5274ca926615632ef710917f4a2ece4b4108041ea6dc99ee244d97d1a687c5f6879a97df6685346d7fff315bb3be008c787f67ad9934563127b07511f57ac72be2f7771a9e29b67a022e12567be3591c033a0202e44742429e3266709f17e79c1caa4618f0e5c37a6d3f238f92f33539be7aa5beee502818100d2cba3ec75b664129ecdbe29324b3fde83ddc7291fe3d6073ebb2db508632f370f54affae7c7ebbc143a5c07ac8f7734eb2f537d3662e4bc05d80eed942a94d5084683dac388cfcd601c9cd59330ff021cf18fa618b25e8a5351f2036f65007a8b4162058f2242d953379d349d9a484c800e8ae539f3e3cd4c6dc9c7b455a7a70281806d790a2d61f2a483cc831473a9b077a72cbd0c493bc8bc12099a7e3c5453b963ee961c561fe19f5e67f224a6ab163e29f65c67f5f8e0893717f2e66b8084f9d91076734e246d991aee77a6fdfd97dba4dd9726979111442997dd5e9f8261b626a1dd58192e379facfafd1c397ad4db17148e8c0626e1ef557c7a160fef4a11fd028180555c679e3ab0c8678ded4d034bbd93389d77b2cde17f16cdca466c24f227901820da2f855054f20e30b6cd4bc2423a88b07072c3b2c16b55049cd0b6be985bbac4e62140f68bb172be67f7ceb9134f40e0cda559228920a5ad45f2d61746f461ab80a79c0eb15616c18f34d6f8b7606db231b167500786893d58fc2c25c7c5e302818100ad587bed92aef2dedb19766c72e5caeadf1f7226d2c3ed0c6ffd1e885b665f55df63d54f91d2fb3f2c4e608bc16bc70eec6300ec5fe61cd31dd48c19544058d1fbb3e39e09117b6e8ab0cc832c481b1c364fbce5b07bf681a0af8e554bef3017dfd53197b87bcebf080fbaef42df5f51c900148499fa7be9e05640dc79d04ad8"
	b1, _ := hex.DecodeString(privkey1)
	sk1, _ := crypto.UnmarshalPrivateKey(b1)
	privkey2 := "080012aa09308204a60201000282010100edc5cda732934aee4df8926b9520f3bc4e6b7e3232bb2b496ce591704b86684cbddf645b4b8dda506e2676801b9c43006607b5bad1835de65fe7e4b6f470210eec172eff4ebb6a88d930e9656d30b01fa01f57ccb9d280b284f0786ca0d3ebbe47346c2ab7c815067fe089cf7b2ce0968e50b0892533f7a3d0c8b7ca4b8884efd8731b455762b4298a89393a468ac3b0ace8ea8d89a683ba09b3312f608c4bc439aeef282150c32a2e92b4f80ab6153471900e3ca1a694ade8e589a5e89aa9274dd7df033c9b7c5f2b61b2dcc2e740f2709ed17908626fe55d59dd93433e623eff5e576949608e7f772eddf0bf5bd6044969f9018e6ad91a1b91a07192decff9020301000102820101008f8a76589583ae1ca71d84e745a41b007727158c206c35f9a1b00559117f16c01d701b19b246f4a0d19e8eb34ff7c9cb17cd57bc6c772ddcc1d13095f2832eb1df7d2f761985b30ee26f50b7566faa23ad7abe7a6d43d345f253699fca87a52dbdb6bc061de4c02ca84e5963d42c8778dc7981d9898811dbe75305012f103f8f91d52803513bbc294fbb86fc8852398a8e358513de1935202d38bc55ddfa05e592851e278309b2df6240f8abb4f411997baefc8f4652ac75a29c9faf9b39d53c3f19dcd8843e311344ca0ac4dea77da719972c025dbb114e6e5c8f690a4db8ccf27493db3977a6a8c0db968307c16ab9f8e8671793e18382d08744505687100102818100f09ef4e9249f4d640d9dbf29d6308c1eab87056564f0d5c820b8b15ea8112278be1b61a7492298d503872e493e1da0a5c99f422035cb203575c0b14e636ae54b3a4c707370b061196dc5f7169fa753e79092aa30fc40d08acbad4a28f35fb55b505ef0e917e8d3e1edb2bfcbaba119e28d1ceb3383e99b4fcf5e1428a9a5717902818100fcf83e52eca79ef469b5ba5ae15c9cafcc1dd46d908712202c0f1e040e47f100fdc59ab998a1c4b66663f245372df5b2ef3788140450f11744b9015d10ea30e337b6d62704ca2d0b42ca0a19ad0b33bc53eed19367a426f764138c9fb219b7df3c96be79b0319d0e2ddc24fc95305a61050a4dcfbae2682199082a03524f328102818100e9cf6be808400b7187919b29ca098e7e56ea72a1ddfdef9df1bdc60c567f9fe177c91f90f00e00382c9f74a89305330f25e5ecd963ac27760b1fdcaa710c74162f660b77012f428af5120251277dee97faf1a912c46b2eb94fc4e964f56830cfb43f2d1532b878faf68054c251d9cf4f4713acb07823cd59360512cd985b3cf102818100c79822ac9116ec5f122d15bd7104fe87e27842ccb3f52ec2fda06be16d572bfbc93f298678bc62963c116ded58cd45880a20f998399397b5f13e3baa2f97683d4f0f4ec6f88b80a0daf0c8a95b94741c8ae8eaa8f0645f6e60a2e0187c90b83845f8f68ed30b424d16b814e2c9df9ddfe0f7314fceb7a6cba390027e1e6a688102818100a9f015ca2223e7af4cddc53811efb583af65128ad9e0da28166634c41c36ada04cd5ae25e48bf53b5b8a596f3d714516a59b86ba9b37ff22d67200a75a4b7dae13406083ce74c9478474e36e73066e524d0ccd991719af60066a0e15001b8eaf7561458eb8d2982222da3d10eb7d23df3a9f3ef3c52921d26ad44c8780bfd379"
	b2, _ := hex.DecodeString(privkey2)
	sk2, _ := crypto.UnmarshalPrivateKey(b2)

	listenOption := libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0")
	host1, err := libp2p.New(listenOption, libp2p.Identity(sk1))
	if err != nil {
		t.Fatal(err)
	}

	host2, err := libp2p.New(listenOption, libp2p.Identity(sk2))
	if err != nil {
		t.Fatal(err)
	}

	addrInfo := peer.AddrInfo{
		ID:    host1.ID(),
		Addrs: host1.Addrs(),
	}
	err = host2.Connect(context.Background(), addrInfo)
	require.Nil(t, err)

	p1 := makeProtocol(p2pA, q, host1)
	p2 := makeProtocol(p2pB, q, host2)
	_, err = p1.RoutingTable.TryAddPeer(host2.ID(), true, false)
	require.Nil(t, err)
	_, err = p2.RoutingTable.TryAddPeer(host1.ID(), true, false)
	require.Nil(t, err)

	go p1.processLocalChunk()
	go p2.processLocalChunk()
	p1.refreshLocalChunk()
	p2.refreshLocalChunk()

	return p1, p2
}

type defaultDiscovery struct{}

func (dd *defaultDiscovery) Advertise(ctx context.Context, ns string, opts ...discovery.Option) (time.Duration, error) {
	return 0, nil
}

func (dd *defaultDiscovery) FindPeers(ctx context.Context, ns string, opts ...discovery.Option) (<-chan peer.AddrInfo, error) {
	return nil, types.ErrNotFound
}

type peerInfoManager struct{}

func (p *peerInfoManager) Refresh(info *types.Peer)      {}
func (p *peerInfoManager) Fetch(pid peer.ID) *types.Peer { return nil }
func (p *peerInfoManager) FetchAll() []*types.Peer       { return nil }
func (p *peerInfoManager) PeerHeight(pid peer.ID) int64 {
	return p.PeerMaxHeight()
}

func (p *peerInfoManager) PeerMaxHeight() int64 {
	return blockHeight
}
