package p2pnext

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/p2p/p2p-next/protos/broadcastTx"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-host"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("Testp2p")

type P2p struct {
	Host       host.Host
	discovery  *Discovery
	txServ     *tx.TxService
	streamMang *streamMange
	api        client.QueueProtocolAPI
	client     queue.Client
	Done       chan struct{}
}

func New(cfg *types.Chain33Config) *P2p {

	m, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/13803")
	if err != nil {
		return nil
	}
	var addrlist []multiaddr.Multiaddr
	addrlist = append(addrlist, m)
	keystr, _ := NewAddrBook(cfg.GetModuleConfig().P2P).GetPrivPubKey()
	//key string convert to crpyto.Privkey
	key, _ := hex.DecodeString(keystr)
	priv, err := crypto.UnmarshalSecp256k1PrivateKey(key)
	if err != nil {
		panic(err)
	}
	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(addrlist...),
		//用于生成对应得peerid,入参是私钥
		libp2p.Identity(priv),
	)

	p2p := &P2p{}
	p2p.streamMang = NewStreamManage(host)
	p2p.txServ = tx.NewService(host, p2p.streamMang.StreamStore)
	return p2p

}

func (p *P2p) managePeers() {

	peerChan, err := p.discovery.FindPeers(context.Background(), p.Host)
	if err != nil {
		panic("PeerFind Err")
	}
	for {
		select {
		case peer := <-peerChan:
			if peer.ID == p.Host.ID() {
				break
			}
			p.streamMang.newStream(context.Background(), peer)
		Recheck:
			if p.streamMang.Size() >= 25 {
				//达到连接节点数最大要求
				time.Sleep(time.Second * 10)
				goto Recheck
			}

		case <-p.Done:
			return

		}
	}

}

func (p *P2p) Close() {
	close(p.Done)
}

func (p *P2p) SetQueueClient(cli queue.Client) {
	var err error
	p.api, err = client.New(cli, nil)
	if err != nil {
		panic("SetQueueClient client.New err")
	}
	if p.client == nil {
		p.client = cli
	}
	go p.managePeers()
	go p.subP2pMsg()

}

func (p *P2p) subP2pMsg() {
	if p.client == nil {
		return
	}

	for msg := range p.client.Recv() {
		switch msg.Ty {

		case types.EventTxBroadcast: //广播tx
			p.txServ.BroadCastTx(msg)
		}
	}
}
