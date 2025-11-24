// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package download

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

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
	host1, err := libp2p.New(libp2p.ListenAddrs(m1), libp2p.Identity(sk1))
	if err != nil {
		t.Fatal(err)
	}
	m2, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13807))
	if err != nil {
		t.Fatal(err)
	}
	host2, err := libp2p.New(libp2p.ListenAddrs(m2), libp2p.Identity(sk2))
	if err != nil {
		t.Fatal(err)
	}

	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], mcfg)
	kademliaDHT1, err := dht.New(context.Background(), host1)
	if err != nil {
		panic(err)
	}

	env1 := protocol.P2PEnv{
		Ctx:             context.Background(),
		ChainCfg:        cfg,
		QueueClient:     client1,
		Host:            host1,
		SubConfig:       mcfg,
		RoutingTable:    kademliaDHT1.RoutingTable(),
		PeerInfoManager: &peerInfoManager{},
	}
	InitProtocol(&env1)

	kademliaDHT2, err := dht.New(context.Background(), host2)
	if err != nil {
		panic(err)
	}
	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/13806/p2p/%s", host1.ID().Pretty()))
	peerinfo, _ := peer.AddrInfoFromP2pAddr(addr)
	err = host2.Connect(context.Background(), *peerinfo)
	if err != nil {
		t.Log("connect error", err)
	}

	env2 := protocol.P2PEnv{
		Ctx:             context.Background(),
		ChainCfg:        cfg,
		QueueClient:     client2,
		Host:            host2,
		SubConfig:       mcfg,
		RoutingTable:    kademliaDHT2.RoutingTable(),
		PeerInfoManager: &peerInfoManager{},
	}
	p2 := &Protocol{
		P2PEnv:  &env2,
		counter: NewCounter(),
	}

	client1.Sub("p2p")
	go func() {
		for msg := range client1.Recv() {
			protocol.GetEventHandler(msg.Ty).CallBack(msg)
		}
	}()

	return p2
}

func TestFetchBlockEvent(t *testing.T) {
	q := queue.New("test")

	p := initEnv(t, q)
	testBlockReq(q)

	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{
		Pid:   []string{}, //[]string{"Qma91H212PWtAFcioW7h9eKiosJtwHsb9x3RmjqRWTwciZ"},
		Start: 1,
		End:   257,
	}))

	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{
		Pid:   []string{"Qma91H212PWtAFcioW7h9eKiosJtwHsb9x3RmjqRWTwciZ"},
		Start: 100,
		End:   157,
	}))

	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{
		Pid:   []string{"Qma91H212PWtAFcioW7h9eKiosJtwHsb9x3RmjqRWTwciZ"},
		Start: 100000,
		End:   100000,
	}))
	fmt.Println(p.RoutingTable.ListPeers())
	for _, msg := range msgs {
		p.handleEventDownloadBlock(msg)
	}

	//time.Sleep(time.Second * 4)
}

func testBlockReq(q queue.Queue) {
	client := q.Client()
	client.Sub("blockchain")
	go func() {
		for msg := range client.Recv() {
			msg.Reply(client.NewMessage("p2p", types.EventFetchBlocks, &types.BlockDetails{Items: []*types.BlockDetail{{Block: &types.Block{}}}}))
		}
	}()
}

type peerInfoManager struct{}

func (p *peerInfoManager) Refresh(info *types.Peer)      {}
func (p *peerInfoManager) Fetch(pid peer.ID) *types.Peer { return nil }
func (p *peerInfoManager) FetchAll() []*types.Peer       { return nil }
func (p *peerInfoManager) PeerHeight(pid peer.ID) int64 {
	return p.PeerMaxHeight()
}

func (p *peerInfoManager) PeerMaxHeight() int64 {
	return 100000
}
