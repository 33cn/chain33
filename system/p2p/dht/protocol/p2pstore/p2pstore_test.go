package p2pstore

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/net"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func TestUpdateChunkWhiteList(t *testing.T) {
	p := &Protocol{}

	p.chunkWhiteList.Store("test1", time.Now())
	p.chunkWhiteList.Store("test2", time.Now().Add(-time.Minute*11))
	p.updateChunkWhiteList()
	_, ok := p.chunkWhiteList.Load("test1")
	assert.True(t, ok)
	_, ok = p.chunkWhiteList.Load("test2")
	assert.False(t, ok)
}

//HOST1 ID: Qma91H212PWtAFcioW7h9eKiosJtwHsb9x3RmjqRWTwciZ
//HOST2 ID: QmbazrBU4HthhnQWcUTiJLnj5ihbFHXsAkGAG6QfmrqJDs
func TestInit(t *testing.T) {
	protocol.ClearEventHandler()
	var err error
	q := queue.New("test")
	p2 := initEnv(t, q)
	time.Sleep(time.Second * 1)
	_ = p2
	msgCh := initMockBlockchain(q)
	client := q.Client()
	//client用来模拟blockchain模块向p2p模块发消息，host1接收topic为p2p的消息，host2接收topic为p2p2的消息
	//向host1请求数据
	msg := testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       999,
	})
	assert.False(t, msg.Data.(*types.Reply).IsOk, msg)
	//向host2请求数据
	msg = testGetBody(t, client, "p2p2", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     500,
		End:       999,
	})
	assert.False(t, msg.Data.(*types.Reply).IsOk, msg)

	// 通知host2保存数据
	testStoreChunk(t, client, "p2p2", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       999,
	})
	time.Sleep(time.Second / 2)
	p2.republish()
	//数据保存之后应该可以查到数据了

	//向host1请求BlockBody
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       99,
	})
	assert.Equal(t, 100, len(msg.Data.(*types.BlockBodys).Items))
	//向host2请求BlockBody
	msg = testGetBody(t, client, "p2p2", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     666,
		End:       888,
	})
	assert.Equal(t, 223, len(msg.Data.(*types.BlockBodys).Items))

	//向host1请求数据
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test1"),
		Start:     1000,
		End:       1999,
	})
	assert.False(t, msg.Data.(*types.Reply).IsOk, msg)
	//向host2请求数据
	msg = testGetBody(t, client, "p2p2", &types.ChunkInfoMsg{
		ChunkHash: []byte("test1"),
		Start:     1500,
		End:       1999,
	})
	assert.False(t, msg.Data.(*types.Reply).IsOk, msg)

	//向host1请求Block
	testGetBlock(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       499,
	})
	msg = <-msgCh
	assert.Equal(t, 500, len(msg.Data.(*types.Blocks).Items))

	//向host2请求Block
	testGetBlock(t, client, "p2p2", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     111,
		End:       666,
	})
	msg = <-msgCh
	assert.Equal(t, 556, len(msg.Data.(*types.Blocks).Items))

	//向host1请求Records
	testGetRecord(t, client, "p2p", &types.ReqChunkRecords{
		Start: 1,
		End:   100,
	})
	msg = <-msgCh
	assert.Equal(t, 100, len(msg.Data.(*types.ChunkRecords).Infos))

	//向host2请求Records
	testGetRecord(t, client, "p2p", &types.ReqChunkRecords{
		Start: 50,
		End:   60,
	})
	msg = <-msgCh
	assert.Equal(t, 11, len(msg.Data.(*types.ChunkRecords).Infos))

	//保存4000~4999的block,会检查0~3999的blockchunk是否能查到
	// 通知host1保存数据
	testStoreChunk(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test4"),
		Start:     4000,
		End:       4999,
	})
	time.Sleep(time.Second / 2)
	//数据保存之后应该可以查到数据了

	//向host1请求BlockBody
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test2"),
		Start:     2000,
		End:       2999,
	})
	assert.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test1"),
		Start:     1000,
		End:       1999,
	})
	assert.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test2"),
		Start:     2000,
		End:       2999,
	})
	assert.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test3"),
		Start:     3000,
		End:       3999,
	})
	assert.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test4"),
		Start:     4000,
		End:       4999,
	})
	assert.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))
	//向host2请求BlockBody
	msg = testGetBody(t, client, "p2p2", &types.ChunkInfoMsg{
		ChunkHash: []byte("test1"),
		Start:     1666,
		End:       1888,
	})
	assert.Equal(t, 223, len(msg.Data.(*types.BlockBodys).Items))

	err = p2.deleteChunkBlock([]byte("test1"))
	if err != nil {
		t.Fatal(err)
	}
	//向host2请求BlockBody
	msg = testGetBody(t, client, "p2p2", &types.ChunkInfoMsg{
		ChunkHash: []byte("test1"),
		Start:     1000,
		End:       1999,
	})
	assert.Equal(t, 1000, len(msg.Data.(*types.BlockBodys).Items))
}

func TestFullNode(t *testing.T) {
	protocol.ClearEventHandler()
	q := queue.New("test")
	p2 := initFullNode(t, q)
	time.Sleep(time.Second * 1)
	_ = p2
	msgCh := initMockBlockchain(q)
	client := q.Client()
	//client用来模拟blockchain模块向p2p模块发消息，host1接收topic为p2p的消息，host2接收topic为p2p2的消息
	//向host1请求数据
	msg := testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       999,
	})
	assert.False(t, msg.Data.(*types.Reply).IsOk, msg)
	//向host2请求数据
	msg = testGetBody(t, client, "p2p3", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     500,
		End:       999,
	})
	assert.False(t, msg.Data.(*types.Reply).IsOk, msg)

	// 通知host3保存数据
	testStoreChunk(t, client, "p2p3", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       999,
	})
	testStoreChunk(t, client, "p2p3", &types.ChunkInfoMsg{
		ChunkHash: []byte("test1"),
		Start:     1000,
		End:       1999,
	})
	testStoreChunk(t, client, "p2p3", &types.ChunkInfoMsg{
		ChunkHash: []byte("test2"),
		Start:     2000,
		End:       2999,
	})
	time.Sleep(time.Second / 2)
	//向host1请求BlockBody
	msg = testGetBody(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       99,
	})
	assert.Equal(t, 100, len(msg.Data.(*types.BlockBodys).Items))
	//向host2请求BlockBody
	msg = testGetBody(t, client, "p2p3", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     666,
		End:       888,
	})
	assert.Equal(t, 223, len(msg.Data.(*types.BlockBodys).Items))

	//向host1请求Block
	testGetBlock(t, client, "p2p", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     0,
		End:       499,
	})
	msg = <-msgCh
	assert.Equal(t, 500, len(msg.Data.(*types.Blocks).Items))

	//向host2请求Block
	testGetBlock(t, client, "p2p3", &types.ChunkInfoMsg{
		ChunkHash: []byte("test0"),
		Start:     111,
		End:       666,
	})
	msg = <-msgCh
	assert.Equal(t, 556, len(msg.Data.(*types.Blocks).Items))
}

func testStoreChunk(t *testing.T, client queue.Client, topic string, req *types.ChunkInfoMsg) *queue.Message {
	msg := client.NewMessage(topic, types.EventNotifyStoreChunk, req)
	err := client.Send(msg, false)
	if err != nil {
		t.Fatal(err)
	}
	return msg
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

func initMockBlockchain(q queue.Queue) <-chan *queue.Message {
	client := q.Client()
	client.Sub("blockchain")
	ch := make(chan *queue.Message)
	go func() {
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetLastHeader:
				msg.Reply(queue.NewMessage(0, "", 0, &types.Header{Height: 14000}))
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

func initEnv(t *testing.T, q queue.Queue) *Protocol {
	privkey1 := "080012a709308204a30201000282010100a28d698a090b02222f97c928b45e78821a87b6382b5057ec9cf12331da3fd8a1c6c71731a8075ae41383460908b483585676f4312249de6929423c2c5d7865bb28d50c5a57e7cad3cc7ca2ddcbc486ac0260fe68e4cdff7f86e46ac65403baf6a5ef50ce7cbb9d0f5f23b02fcc6d5211e2df2bf24fc84565ba5d0777458ad82b46579cba0a16c88ff946812e7f17ad85a2b35dc1bae732a74f83262358fefcc985a632aee8129a73d1d17aaceebd5bae9ffbeab6c5505e8eafd8af8448a6dd74d76885bc71c7d85bad761680bc7cdd04a99cb90d8c27467769c500e603677469a73cec7983a7dba6d7656ab241b4446355a89a267eeb72f0fd7c89c470d93a6302030100010282010002db797f73a93de05bf5cf136818410608715a42a280470b61b6db6784ee9a603d9e424a1d2a03eefe68d0525854d3fa398addbfff5a4d0e8c2b1de3a9c0f408d62ee888ae02e50dd40a5cd289426b1b9aef1989be7be081dd5d268355f6bad29b1819d3875dc4e500472051b6c6352b1b51d0f3f17313c536016ca02c18c4b3f6dba52c616f93bf831589d0dd2fc190f875e37a4e9654bd9e63e04fc5d9cea45664cd6d26c17659ee4b8c6837c6dfe86e4e6b8e17af332a736267ee5a68ac0b0c60ced47f1aaf7ec65547f664a9f1409e7d116ca325c29b1058e5892dc04c79337a15b875e7139bca7ddfb6c5c7f822adff8cd65f1dfa84d1b0f87166604c0102818100c5694c5a55465a068075e5274ca926615632ef710917f4a2ece4b4108041ea6dc99ee244d97d1a687c5f6879a97df6685346d7fff315bb3be008c787f67ad9934563127b07511f57ac72be2f7771a9e29b67a022e12567be3591c033a0202e44742429e3266709f17e79c1caa4618f0e5c37a6d3f238f92f33539be7aa5beee502818100d2cba3ec75b664129ecdbe29324b3fde83ddc7291fe3d6073ebb2db508632f370f54affae7c7ebbc143a5c07ac8f7734eb2f537d3662e4bc05d80eed942a94d5084683dac388cfcd601c9cd59330ff021cf18fa618b25e8a5351f2036f65007a8b4162058f2242d953379d349d9a484c800e8ae539f3e3cd4c6dc9c7b455a7a70281806d790a2d61f2a483cc831473a9b077a72cbd0c493bc8bc12099a7e3c5453b963ee961c561fe19f5e67f224a6ab163e29f65c67f5f8e0893717f2e66b8084f9d91076734e246d991aee77a6fdfd97dba4dd9726979111442997dd5e9f8261b626a1dd58192e379facfafd1c397ad4db17148e8c0626e1ef557c7a160fef4a11fd028180555c679e3ab0c8678ded4d034bbd93389d77b2cde17f16cdca466c24f227901820da2f855054f20e30b6cd4bc2423a88b07072c3b2c16b55049cd0b6be985bbac4e62140f68bb172be67f7ceb9134f40e0cda559228920a5ad45f2d61746f461ab80a79c0eb15616c18f34d6f8b7606db231b167500786893d58fc2c25c7c5e302818100ad587bed92aef2dedb19766c72e5caeadf1f7226d2c3ed0c6ffd1e885b665f55df63d54f91d2fb3f2c4e608bc16bc70eec6300ec5fe61cd31dd48c19544058d1fbb3e39e09117b6e8ab0cc832c481b1c364fbce5b07bf681a0af8e554bef3017dfd53197b87bcebf080fbaef42df5f51c900148499fa7be9e05640dc79d04ad8"
	b1, _ := hex.DecodeString(privkey1)
	sk1, _ := crypto.UnmarshalPrivateKey(b1)
	privkey2 := "080012aa09308204a60201000282010100edc5cda732934aee4df8926b9520f3bc4e6b7e3232bb2b496ce591704b86684cbddf645b4b8dda506e2676801b9c43006607b5bad1835de65fe7e4b6f470210eec172eff4ebb6a88d930e9656d30b01fa01f57ccb9d280b284f0786ca0d3ebbe47346c2ab7c815067fe089cf7b2ce0968e50b0892533f7a3d0c8b7ca4b8884efd8731b455762b4298a89393a468ac3b0ace8ea8d89a683ba09b3312f608c4bc439aeef282150c32a2e92b4f80ab6153471900e3ca1a694ade8e589a5e89aa9274dd7df033c9b7c5f2b61b2dcc2e740f2709ed17908626fe55d59dd93433e623eff5e576949608e7f772eddf0bf5bd6044969f9018e6ad91a1b91a07192decff9020301000102820101008f8a76589583ae1ca71d84e745a41b007727158c206c35f9a1b00559117f16c01d701b19b246f4a0d19e8eb34ff7c9cb17cd57bc6c772ddcc1d13095f2832eb1df7d2f761985b30ee26f50b7566faa23ad7abe7a6d43d345f253699fca87a52dbdb6bc061de4c02ca84e5963d42c8778dc7981d9898811dbe75305012f103f8f91d52803513bbc294fbb86fc8852398a8e358513de1935202d38bc55ddfa05e592851e278309b2df6240f8abb4f411997baefc8f4652ac75a29c9faf9b39d53c3f19dcd8843e311344ca0ac4dea77da719972c025dbb114e6e5c8f690a4db8ccf27493db3977a6a8c0db968307c16ab9f8e8671793e18382d08744505687100102818100f09ef4e9249f4d640d9dbf29d6308c1eab87056564f0d5c820b8b15ea8112278be1b61a7492298d503872e493e1da0a5c99f422035cb203575c0b14e636ae54b3a4c707370b061196dc5f7169fa753e79092aa30fc40d08acbad4a28f35fb55b505ef0e917e8d3e1edb2bfcbaba119e28d1ceb3383e99b4fcf5e1428a9a5717902818100fcf83e52eca79ef469b5ba5ae15c9cafcc1dd46d908712202c0f1e040e47f100fdc59ab998a1c4b66663f245372df5b2ef3788140450f11744b9015d10ea30e337b6d62704ca2d0b42ca0a19ad0b33bc53eed19367a426f764138c9fb219b7df3c96be79b0319d0e2ddc24fc95305a61050a4dcfbae2682199082a03524f328102818100e9cf6be808400b7187919b29ca098e7e56ea72a1ddfdef9df1bdc60c567f9fe177c91f90f00e00382c9f74a89305330f25e5ecd963ac27760b1fdcaa710c74162f660b77012f428af5120251277dee97faf1a912c46b2eb94fc4e964f56830cfb43f2d1532b878faf68054c251d9cf4f4713acb07823cd59360512cd985b3cf102818100c79822ac9116ec5f122d15bd7104fe87e27842ccb3f52ec2fda06be16d572bfbc93f298678bc62963c116ded58cd45880a20f998399397b5f13e3baa2f97683d4f0f4ec6f88b80a0daf0c8a95b94741c8ae8eaa8f0645f6e60a2e0187c90b83845f8f68ed30b424d16b814e2c9df9ddfe0f7314fceb7a6cba390027e1e6a688102818100a9f015ca2223e7af4cddc53811efb583af65128ad9e0da28166634c41c36ada04cd5ae25e48bf53b5b8a596f3d714516a59b86ba9b37ff22d67200a75a4b7dae13406083ce74c9478474e36e73066e524d0ccd991719af60066a0e15001b8eaf7561458eb8d2982222da3d10eb7d23df3a9f3ef3c52921d26ad44c8780bfd379"
	b2, _ := hex.DecodeString(privkey2)
	sk2, _ := crypto.UnmarshalPrivateKey(b2)
	client1, client2 := q.Client(), q.Client()
	m1, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13806))
	if err != nil {
		t.Fatal(err)
	}
	host1, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m1), libp2p.Identity(sk1))
	if err != nil {
		t.Fatal(err)
	}
	m2, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13807))
	if err != nil {
		t.Fatal(err)
	}
	host2, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m2), libp2p.Identity(sk2))
	if err != nil {
		t.Fatal(err)
	}

	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &types2.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[types2.DHTTypeName], mcfg)
	mcfg.DisableFindLANPeers = true
	discovery1 := net.InitDhtDiscovery(context.Background(), host1, nil, cfg, &types2.P2PSubConfig{Channel: 888})
	env1 := prototypes.P2PEnv{
		ChainCfg:         cfg,
		QueueClient:      client1,
		Host:             host1,
		SubConfig:        mcfg,
		RoutingDiscovery: discovery1.RoutingDiscovery,
		RoutingTable:     discovery1.RoutingTable(),
		DB:               newTestDB(),
		Ctx:              context.Background(),
	}
	InitProtocol(&env1)
	host1.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(handleStreamIsHealthy))

	discovery2 := net.InitDhtDiscovery(context.Background(), host2, nil, cfg, &types2.P2PSubConfig{
		Seeds:   []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/13806/p2p/%s", host1.ID().Pretty())},
		Channel: 888,
	})
	env2 := prototypes.P2PEnv{
		ChainCfg:         cfg,
		QueueClient:      client2,
		Host:             host2,
		SubConfig:        mcfg,
		RoutingDiscovery: discovery2.RoutingDiscovery,
		RoutingTable:     discovery2.RoutingTable(),
		DB:               newTestDB(),
		Ctx:              context.Background(),
	}
	p2 := &Protocol{
		P2PEnv:              &env2,
		healthyRoutingTable: kb.NewRoutingTable(dht.KValue, kb.ConvertPeerID(env2.Host.ID()), time.Minute, env2.Host.Peerstore()),
		localChunkInfo:      make(map[string]LocalChunkInfo),
	}
	//注册p2p通信协议，用于处理节点之间请求
	p2.Host.SetStreamHandler(protocol.StoreChunk, protocol.HandlerWithAuth(p2.handleStreamStoreChunk))
	p2.Host.SetStreamHandler(protocol.FetchChunk, protocol.HandlerWithAuth(p2.handleStreamFetchChunk))
	p2.Host.SetStreamHandler(protocol.GetHeader, protocol.HandlerWithAuthAndSign(p2.handleStreamGetHeader))
	p2.Host.SetStreamHandler(protocol.GetChunkRecord, protocol.HandlerWithAuthAndSign(p2.handleStreamGetChunkRecord))
	p2.Host.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(handleStreamIsHealthy))
	go func() {
		for i := 0; i < 3; i++ { //节点启动后充分初始化 healthy routing table
			p2.updateHealthyRoutingTable()
			time.Sleep(time.Second * 1)
		}
	}()
	client1.Sub("p2p")
	client2.Sub("p2p2")
	go func() {
		for msg := range client1.Recv() {
			protocol.HandleEvent(msg)
		}
	}()
	go func() {
		for msg := range client2.Recv() {
			switch msg.Ty {
			case types.EventNotifyStoreChunk:
				protocol.EventHandlerWithRecover(p2.handleEventNotifyStoreChunk)(msg)
			case types.EventGetChunkBlock:
				protocol.EventHandlerWithRecover(p2.handleEventGetChunkBlock)(msg)
			case types.EventGetChunkBlockBody:
				protocol.EventHandlerWithRecover(p2.handleEventGetChunkBlockBody)(msg)
			case types.EventGetChunkRecord:
				protocol.EventHandlerWithRecover(p2.handleEventGetChunkRecord)(msg)
			}
		}
	}()

	return p2
}

func initFullNode(t *testing.T, q queue.Queue) *Protocol {
	privkey1 := "080012a709308204a30201000282010100a28d698a090b02222f97c928b45e78821a87b6382b5057ec9cf12331da3fd8a1c6c71731a8075ae41383460908b483585676f4312249de6929423c2c5d7865bb28d50c5a57e7cad3cc7ca2ddcbc486ac0260fe68e4cdff7f86e46ac65403baf6a5ef50ce7cbb9d0f5f23b02fcc6d5211e2df2bf24fc84565ba5d0777458ad82b46579cba0a16c88ff946812e7f17ad85a2b35dc1bae732a74f83262358fefcc985a632aee8129a73d1d17aaceebd5bae9ffbeab6c5505e8eafd8af8448a6dd74d76885bc71c7d85bad761680bc7cdd04a99cb90d8c27467769c500e603677469a73cec7983a7dba6d7656ab241b4446355a89a267eeb72f0fd7c89c470d93a6302030100010282010002db797f73a93de05bf5cf136818410608715a42a280470b61b6db6784ee9a603d9e424a1d2a03eefe68d0525854d3fa398addbfff5a4d0e8c2b1de3a9c0f408d62ee888ae02e50dd40a5cd289426b1b9aef1989be7be081dd5d268355f6bad29b1819d3875dc4e500472051b6c6352b1b51d0f3f17313c536016ca02c18c4b3f6dba52c616f93bf831589d0dd2fc190f875e37a4e9654bd9e63e04fc5d9cea45664cd6d26c17659ee4b8c6837c6dfe86e4e6b8e17af332a736267ee5a68ac0b0c60ced47f1aaf7ec65547f664a9f1409e7d116ca325c29b1058e5892dc04c79337a15b875e7139bca7ddfb6c5c7f822adff8cd65f1dfa84d1b0f87166604c0102818100c5694c5a55465a068075e5274ca926615632ef710917f4a2ece4b4108041ea6dc99ee244d97d1a687c5f6879a97df6685346d7fff315bb3be008c787f67ad9934563127b07511f57ac72be2f7771a9e29b67a022e12567be3591c033a0202e44742429e3266709f17e79c1caa4618f0e5c37a6d3f238f92f33539be7aa5beee502818100d2cba3ec75b664129ecdbe29324b3fde83ddc7291fe3d6073ebb2db508632f370f54affae7c7ebbc143a5c07ac8f7734eb2f537d3662e4bc05d80eed942a94d5084683dac388cfcd601c9cd59330ff021cf18fa618b25e8a5351f2036f65007a8b4162058f2242d953379d349d9a484c800e8ae539f3e3cd4c6dc9c7b455a7a70281806d790a2d61f2a483cc831473a9b077a72cbd0c493bc8bc12099a7e3c5453b963ee961c561fe19f5e67f224a6ab163e29f65c67f5f8e0893717f2e66b8084f9d91076734e246d991aee77a6fdfd97dba4dd9726979111442997dd5e9f8261b626a1dd58192e379facfafd1c397ad4db17148e8c0626e1ef557c7a160fef4a11fd028180555c679e3ab0c8678ded4d034bbd93389d77b2cde17f16cdca466c24f227901820da2f855054f20e30b6cd4bc2423a88b07072c3b2c16b55049cd0b6be985bbac4e62140f68bb172be67f7ceb9134f40e0cda559228920a5ad45f2d61746f461ab80a79c0eb15616c18f34d6f8b7606db231b167500786893d58fc2c25c7c5e302818100ad587bed92aef2dedb19766c72e5caeadf1f7226d2c3ed0c6ffd1e885b665f55df63d54f91d2fb3f2c4e608bc16bc70eec6300ec5fe61cd31dd48c19544058d1fbb3e39e09117b6e8ab0cc832c481b1c364fbce5b07bf681a0af8e554bef3017dfd53197b87bcebf080fbaef42df5f51c900148499fa7be9e05640dc79d04ad8"
	b1, _ := hex.DecodeString(privkey1)
	sk1, _ := crypto.UnmarshalPrivateKey(b1)
	privkey3 := "080012a809308204a40201000282010100cdcc5ff051a4a30405620f971038997817e5f899db5ebb4f03ca744537ba29f0149dd71a47b8ac3523052809e14accb565507f2d70ae64686dd065950f3fbc8a84a8d692cda207e923363da1a5d4a7d6a8eb420a0d87d383649633c2feee147382083cc8e451b90f217b0920d9381ace0a769fdcf1b5a4c88e5aebe7213a18207f68f737d323235c0a8e61130230c5fe272d84ffa8ef01857e327f391d62e88d72bc8b9969dc40451b211962339c950ebab4d5fdaa407fc6905720b1cfedb4d7f6fc6ab66c6f7fb43c004c2ba22033e24c878d3e4472eb36e81e4beef9eb2b9b7a0746d69e75468d00e0c4a17bc2fb8eb1142538db67b512cb88464869f2f52b0203010001028201010083ba46ca8fa7bf448aa18af319c9f0ca031a0bb787c82a42d85d55711ccb879e89c3c274aae5d52ca9fed9f301071ce31b379c401cb933c1f8508545151ea9f34c18ba47fb61b48891265deac337cc3ac5a2d88190c99924a854d04b075ca3309051ef7e734eb012b44e89b841f1fc8e57fa3837776bda4f1977af3a21758b0cd2cac32e6db0aec3a72d4f2b076cdd42cfd806a5f73b26a9e70ef009aaf83f2e5cf31000055fa59a090aadfade3387b9315e4ff786436457a8d7bc70af3f17bd542b9faca50f3abab8cf392023c3b4e1b448040ae78610458f2a2df9f9d96037cb565eb6ab2f1407eebd3e538205bcd6519fe2e7419688d5cbf530c2a332449102818100cf82a445bef1625df4522a2ecd666f1bc5818f48cba6e3c6fdeb05d67a7aada71e5cf1424ea4aa9314b0388c29309b28ca18afd4c6b88ee7bd08333f2883d1f4ebf7ce17b1dd17bd30217031bb377e05d036afa75d844a4528dc8b304ee3a42554457c7efc380fc144fe6d112268739210b3a56f2cc9ffb0401db8a654909f2902818100fde35258ad69b1af5e80c4fbc241ac2899142297dea902324b6055f4aaa554f8fd9d9809ff402efbd2104aa0d632f0f73296b5a56c94260adf8aad4514cfa135260c03e7dd53c6a2bdb869b9edb873eb6e1482276a5a55e890e8a8d3db87f70f4271cd7c2582359342ec279acb6dcf714059bbf58540ea496a47675a2eb140330281804e566e779a16fc60a5cca2fa1a36b279547d8dbf188abf70af091ba21588dca7bb71b0eeac4bc3cd54c11607ebc0dac2725111880d213d69c4d624aa923bf97631e2d21de5daa68c986ff72fff127af3ecdfc83e31b2b06b1d7aecdce6db4f6b7c3de33af9329cd80498dc49dca87c00c7675a6bf707a70c3d983ace281c94c902818100ea64e98c772542672eaf61ad30ede29c649f6344a4cb91fc8efc74befaa0c32f512e22c4f003f89c829689dfad81c057e83b9d9e08fd4995f64598ac53875144b948947e8726a6176f628731a1980e6547eee52eb0909009b367291ed6e9d31d2271e08d02301178506ba830d02924406171b706f82c3360ee1ed7fb396a6963028180667b478c2e5759fd73000e13874295611812b689f96f32a2b6e876eea4b66e0b90123d86fb7320548c30ce3d5f45f9f00b9faa23228fe70e79f4215a8328062ba474a753bfed99c5bb0efc120856f25141b00181757e47f84ec4e63d54e81595585b22651a01d1c956fc7d5c3ce9d1f7f1355b5dba7d79ceacf9e3c7e9f6be5a"
	b3, _ := hex.DecodeString(privkey3)
	sk3, _ := crypto.UnmarshalPrivateKey(b3)
	client1, client3 := q.Client(), q.Client()
	m1, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13808))
	if err != nil {
		t.Fatal(err)
	}
	host1, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m1), libp2p.Identity(sk1))
	if err != nil {
		t.Fatal(err)
	}

	m3, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13809))
	if err != nil {
		t.Fatal(err)
	}
	host3, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m3), libp2p.Identity(sk3))
	if err != nil {
		t.Fatal(err)
	}
	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &types2.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[types2.DHTTypeName], mcfg)
	mcfg.DisableFindLANPeers = true
	discovery1 := net.InitDhtDiscovery(context.Background(), host1, nil, cfg, &types2.P2PSubConfig{Channel: 888})
	env1 := prototypes.P2PEnv{
		ChainCfg:         cfg,
		QueueClient:      client1,
		Host:             host1,
		SubConfig:        mcfg,
		RoutingDiscovery: discovery1.RoutingDiscovery,
		RoutingTable:     discovery1.RoutingTable(),
		DB:               newTestDB(),
	}
	InitProtocol(&env1)
	host1.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(handleStreamIsHealthy))

	mcfg3 := &types2.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[types2.DHTTypeName], mcfg3)
	mcfg3.IsFullNode = true
	mcfg3.DisableFindLANPeers = true
	discovery3 := net.InitDhtDiscovery(context.Background(), host3, nil, cfg, &types2.P2PSubConfig{
		Seeds:   []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/13808/p2p/%s", host1.ID().Pretty())},
		Channel: 888,
	})
	env3 := prototypes.P2PEnv{
		ChainCfg:         cfg,
		QueueClient:      client3,
		Host:             host3,
		SubConfig:        mcfg3,
		RoutingDiscovery: discovery3.RoutingDiscovery,
		RoutingTable:     discovery3.RoutingTable(),
		DB:               newTestDB(),
	}
	p3 := &Protocol{
		P2PEnv:              &env3,
		healthyRoutingTable: kb.NewRoutingTable(dht.KValue, kb.ConvertPeerID(env3.Host.ID()), time.Minute, env3.Host.Peerstore()),
		localChunkInfo:      make(map[string]LocalChunkInfo),
	}
	//注册p2p通信协议，用于处理节点之间请求
	p3.Host.SetStreamHandler(protocol.StoreChunk, protocol.HandlerWithAuth(p3.handleStreamStoreChunk))
	p3.Host.SetStreamHandler(protocol.FetchChunk, protocol.HandlerWithAuth(p3.handleStreamFetchChunk))
	p3.Host.SetStreamHandler(protocol.GetHeader, protocol.HandlerWithAuthAndSign(p3.handleStreamGetHeader))
	p3.Host.SetStreamHandler(protocol.GetChunkRecord, protocol.HandlerWithAuthAndSign(p3.handleStreamGetChunkRecord))
	p3.Host.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(handleStreamIsHealthy2))
	go func() {
		for i := 0; i < 3; i++ {
			p3.updateHealthyRoutingTable()
			time.Sleep(time.Second * 1)
		}
	}()
	discovery.Advertise(context.Background(), p3.RoutingDiscovery, protocol.BroadcastFullNode)

	client1.Sub("p2p")
	client3.Sub("p2p3")
	go func() {
		for msg := range client1.Recv() {
			protocol.HandleEvent(msg)
		}
	}()

	go func() {
		for msg := range client3.Recv() {
			switch msg.Ty {
			case types.EventNotifyStoreChunk:
				protocol.EventHandlerWithRecover(p3.handleEventNotifyStoreChunk)(msg)
			case types.EventGetChunkBlock:
				protocol.EventHandlerWithRecover(p3.handleEventGetChunkBlock)(msg)
			case types.EventGetChunkBlockBody:
				protocol.EventHandlerWithRecover(p3.handleEventGetChunkBlockBody)(msg)
			case types.EventGetChunkRecord:
				protocol.EventHandlerWithRecover(p3.handleEventGetChunkRecord)(msg)
			}
		}
	}()

	return p3
}

func handleStreamIsHealthy(_ *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: true,
		},
	}
	return nil
}

func handleStreamIsHealthy2(_ *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: false,
		},
	}
	return nil
}

type TestDB struct {
	lock sync.Mutex
	m    map[string][]byte
}

func newTestDB() *TestDB {
	m := make(map[string][]byte, 100)
	return &TestDB{m: m}
}

func (db *TestDB) Get(key datastore.Key) (value []byte, err error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	v, ok := db.m[key.String()]
	if !ok {
		return nil, datastore.ErrNotFound
	}
	return v, nil
}

func (db *TestDB) Has(key datastore.Key) (exists bool, err error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	_, ok := db.m[key.String()]
	return ok, nil
}

func (db *TestDB) GetSize(key datastore.Key) (size int, err error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	v := db.m[key.String()]
	return len(v), nil
}

func (db *TestDB) Query(q query.Query) (query.Results, error) {
	return nil, nil
}

func (db *TestDB) Put(key datastore.Key, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.m[key.String()] = value
	return nil
}

func (db *TestDB) Delete(key datastore.Key) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	delete(db.m, key.String())
	return nil
}

func (db *TestDB) Close() error {
	return nil
}

func (db *TestDB) Batch() (datastore.Batch, error) {
	return nil, nil
}
