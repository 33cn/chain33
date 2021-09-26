package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-kad-dht/crawler"
	"github.com/multiformats/go-multiaddr"
	"os"
	"time"
)

//dhtprotoId="/chain33-0/kad/1.0.0"
var (
	//dht 协议ID，需要根据具体区块链网络进行配置
	dhtProtoId = flag.String("proto", "/chain33-0/kad/1.0.0", "dht protocol id")
	//扫描的引导节点
	startNodes = flag.String("node", "", "bootstrap nodes")
	//扫描数据输出
	out       = flag.String("out", "topology.txt", "topology graph")
	allpeers  = flag.String("all", "allpeers.txt", "show all peers")
	jsonpeers = flag.String("json", "topology.json", "json format")
	peerMap   = make(map[peer.ID]map[peer.ID]*peer.AddrInfo) //peer---store---> peers
	peerJson  = make(map[string][]string)                    //peer--->[p1,p2,p3,....,pn]
	allPeers  = make(map[peer.ID]*peer.AddrInfo)             //pid---->[ip:port,ip:port,ip:port]
)

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := libp2p.New(ctx)
	if err != nil {
		panic(err)
	}

	//初始化DHT的网络爬虫
	var opts []crawler.Option
	opts = append(opts, crawler.WithConnectTimeout(time.Second*10), crawler.WithMsgTimeout(time.Second*10),
		crawler.WithProtocols([]protocol.ID{protocol.ID(*dhtProtoId)}))
	cl, err := crawler.New(h, opts...)
	if err != nil {
		panic(err)
	}
	startingPeer, err := multiaddr.NewMultiaddr(*startNodes)
	if err != nil {
		panic(err)
	}

	startingPeerInfo, err := peer.AddrInfoFromP2pAddr(startingPeer)
	if err != nil {
		panic(err)
	}
	h.Connect(ctx, *startingPeerInfo)
	fmt.Println(` 
            dddddddd                                          
            d::::::dhhhhhhh                     tttt          
            d::::::dh:::::h                  ttt:::t          
            d::::::dh:::::h                  t:::::t          
            d:::::d h:::::h                  t:::::t          
    ddddddddd:::::d  h::::h hhhhh      ttttttt:::::ttttttt    
  dd::::::::::::::d  h::::hh:::::hhh   t:::::::::::::::::t    
 d::::::::::::::::d  h::::::::::::::hh t:::::::::::::::::t    
d:::::::ddddd:::::d  h:::::::hhh::::::htttttt:::::::tttttt    
d::::::d    d:::::d  h::::::h   h::::::h     t:::::t          
d:::::d     d:::::d  h:::::h     h:::::h     t:::::t          
d:::::d     d:::::d  h:::::h     h:::::h     t:::::t          
d:::::d     d:::::d  h:::::h     h:::::h     t:::::t    tttttt
d::::::ddddd::::::dd h:::::h     h:::::h     t::::::tttt:::::t
 d:::::::::::::::::d h:::::h     h:::::h     tt::::::::::::::t
  d:::::::::ddd::::d h:::::h     h:::::h       tt:::::::::::tt
   ddddddddd   ddddd hhhhhhh     hhhhhhh         ttttttttttt     crawler start working....,wait a moment`)

	cl.Run(ctx, []*peer.AddrInfo{startingPeerInfo}, handlerSuccess, nil)
	fmt.Println("peerMap", len(peerMap))
	OutputData(h, *out, *allpeers, *jsonpeers)

}

func handlerSuccess(p peer.ID, rtPeers []*peer.AddrInfo) {
	if _, ok := peerMap[p]; !ok {
		peers := make(map[peer.ID]*peer.AddrInfo, len(rtPeers))
		for _, pinfo := range rtPeers {
			peers[pinfo.ID] = pinfo
			peerJson[p.String()] = append(peerJson[p.String()], pinfo.ID.String())
		}

		peerMap[p] = peers
	}
}

func OutputData(host core.Host, filePath, allpeerfile, jsonfile string) {
	f, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	f.WriteString("chain33  peers topology{ \n")
	for p, rtPeers := range peerMap {
		pinfo := host.Peerstore().PeerInfo(p)
		allPeers[p] = &pinfo
		for rtp, pinfo := range rtPeers {
			f.WriteString(fmt.Sprintf("%v ---> %v;\n", p, rtp))
			if _, ok := allPeers[rtp]; !ok {
				allPeers[rtp] = pinfo
			}
		}
	}
	f.WriteString("\n}")
	f.Close()
	//write all peers
	f, err = os.Create(allpeerfile)
	if err != nil {
		panic(err)
	}
	f.WriteString(fmt.Sprintf("chain33 all peers:%d{ \n", len(allPeers)))

	for p, info := range allPeers {
		f.WriteString(fmt.Sprintf("%v ---> %v;\n", p, info.Addrs))
	}
	f.WriteString("\n}")
	f.Close()

	f, err = os.Create(jsonfile)
	if err != nil {
		panic(err)
	}

	//json format
	jsondata, err := json.MarshalIndent(peerJson, " ", "\t")
	if err != nil {
		panic(err)
	}

	f.WriteString(string(jsondata))
	f.Close()
	return
}
