package p2p

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.aliyun.com/chain33/chain33/queue"
	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func (m *P2pCli) Close() {

	close(m.loopdone)

}

type P2pCli struct {
	network  *P2p
	mtx      sync.Mutex
	taskinfo map[int64]bool
	loopdone chan struct{}
}
type intervalInfo struct {
	start int
	end   int
}

func NewP2pCli(network *P2p) *P2pCli {
	pcli := &P2pCli{
		network:  network,
		loopdone: make(chan struct{}, 1),
	}

	return pcli
}

func (m *P2pCli) BroadCastTx(msg queue.Message) {
	defer func() {
		<-m.network.txFactory
		atomic.AddInt32(&m.network.txCapcity, 1)
	}()
	pub.FIFOPub(&pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)}, "tx")
	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{true, []byte("ok")}))

}

func (m *P2pCli) GetMemPool(msg queue.Message) {
	defer func() {
		<-m.network.otherFactory
	}()
	var Txs = make([]*pb.Transaction, 0)
	var ableInv = make([]*pb.Inventory, 0)
	peers, _ := m.network.node.GetActivePeers()

	for _, peer := range peers {
		//获取远程 peer invs

		resp, err := peer.mconn.conn.GetMemPool(context.Background(), &pb.P2PGetMempool{Version: m.network.node.nodeInfo.cfg.GetVersion()})
		P2pComm.CollectPeerStat(err, peer)
		if err != nil {
			continue
		}

		invs := resp.GetInvs()
		//与本地mempool 对比 tx数组
		msg := m.network.c.NewMessage("mempool", pb.EventGetMempool, nil)
		txresp, err := m.network.c.Wait(msg)
		if err != nil {
			continue
		}
		txlist := txresp.GetData().(*pb.ReplyTxList)
		txs := txlist.GetTxs()
		var txmap = make(map[string]*pb.Transaction)
		for _, tx := range txs {
			txmap[hex.EncodeToString(tx.Hash())] = tx
		}

		//去重过滤
		for _, inv := range invs {
			if _, ok := txmap[hex.EncodeToString(inv.Hash)]; !ok {
				ableInv = append(ableInv, inv)
			}
		}
		//获取真正的交易Tx call GetData
		datacli, dataerr := peer.mconn.conn.GetData(context.Background(), &pb.P2PGetData{Invs: ableInv, Version: m.network.node.nodeInfo.cfg.GetVersion()})
		P2pComm.CollectPeerStat(dataerr, peer)
		if dataerr != nil {
			continue
		}

		invdatas, recerr := datacli.Recv()
		if recerr != nil && recerr != io.EOF {
			log.Error("GetMemPool", "err", recerr.Error())
			datacli.CloseSend()
			continue
		}

		for _, invdata := range invdatas.Items {
			Txs = append(Txs, invdata.GetTx())
		}
		datacli.CloseSend()
		break
	}
	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReplyTxList, &pb.ReplyTxList{Txs: Txs}))

}

func (m *P2pCli) GetAddr(peer *peer) ([]string, error) {

	resp, err := peer.mconn.conn.GetAddr(context.Background(), &pb.P2PGetAddr{Nonce: int64(rand.Int31n(102040))})
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return nil, err
	}

	log.Debug("GetAddr Resp", "Resp", resp, "addrlist", resp.Addrlist)
	return resp.Addrlist, nil
}

func (m *P2pCli) SendVersion(peer *peer, nodeinfo *NodeInfo) error {

	client := nodeinfo.qclient
	msg := client.NewMessage("blockchain", pb.EventGetBlockHeight, nil)
	err := client.Send(msg, true)
	if err != nil {
		log.Error("SendVesion", "Error", err.Error())
		return err
	}
	rsp, err := client.Wait(msg)
	if err != nil {
		log.Error("GetHeight", "Error", err.Error())
		return err
	}

	blockheight := rsp.GetData().(*pb.ReplyBlockHeight).GetHeight()
	randNonce := rand.Int31n(102040)
	in, err := P2pComm.Signature(peer.mconn.key, &pb.P2PPing{Nonce: int64(randNonce), Addr: nodeinfo.GetExternalAddr().IP.String(), Port: int32(nodeinfo.GetExternalAddr().Port)})
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return err
	}
	addrfrom := nodeinfo.GetExternalAddr().String()

	nodeinfo.blacklist.Add(addrfrom)
	resp, err := peer.mconn.conn.Version2(context.Background(), &pb.P2PVersion{Version: nodeinfo.cfg.GetVersion(), Service: SERVICE, Timestamp: time.Now().Unix(),
		AddrRecv: peer.Addr(), AddrFrom: addrfrom, Nonce: int64(rand.Int31n(102040)),
		UserAgent: hex.EncodeToString(in.Sign.GetPubkey()), StartHeight: blockheight})

	log.Debug("SendVersion", "resp", resp, "addrfrom", addrfrom, "sendto", peer.Addr())
	if err != nil {
		log.Info("SendVersion", "Verson", err.Error(), "peer", peer.Addr())
		if strings.Contains(err.Error(), VersionNotSupport) {
			peer.version.SetSupport(false)
			P2pComm.CollectPeerStat(err, peer)
		}
		return err
	}
	P2pComm.CollectPeerStat(err, peer)
	port, err := strconv.Atoi(strings.Split(resp.GetAddrRecv(), ":")[1])
	if err != nil {
		return err
	}
	if peer.IsPersistent() == false {
		return nil //如果不是种子节点，则直接返回，不用校验自身的外网地址
	}
	if strings.Split(resp.GetAddrRecv(), ":")[0] != nodeinfo.GetExternalAddr().IP.String() {
		externalIp := strings.Split(resp.GetAddrRecv(), ":")[0]
		log.Debug("sendVersion", "externalip", externalIp)
		if exaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", externalIp, port)); err == nil {
			nodeinfo.SetExternalAddr(exaddr)
		}

	}

	log.Debug("SHOW VERSION BACK", "VersionBack", resp)
	return nil
}

func (m *P2pCli) SendPing(peer *peer, nodeinfo *NodeInfo) error {
	randNonce := rand.Int31n(102040)
	ping := &pb.P2PPing{Nonce: int64(randNonce), Addr: nodeinfo.GetExternalAddr().IP.String(), Port: int32(nodeinfo.GetExternalAddr().Port)}
	_, err := P2pComm.Signature(peer.mconn.key, ping)
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return err
	}
	log.Debug("SendPing", "Peer", peer.Addr(), "nonce", randNonce)
	r, err := peer.mconn.conn.Ping(context.Background(), ping)
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return err
	}

	log.Debug("SendPing", "recv pone", r.Nonce, "Ping nonce:", randNonce)
	return nil
}
func (m *P2pCli) GetBlockHeight(nodeinfo *NodeInfo) (int64, error) {
	client := nodeinfo.qclient
	msg := client.NewMessage("blockchain", pb.EventGetLastHeader, nil)
	err := client.Send(msg, true)
	if err != nil {
		log.Error("GetBlockHeight", "Error", err.Error())
		return 0, err
	}
	resp, err := client.Wait(msg)
	if err != nil {
		return 0, err
	}

	header := resp.GetData().(*pb.Header)
	return header.GetHeight(), nil
}
func (m *P2pCli) GetPeerInfo(msg queue.Message) {

	tempServer := NewP2pServer()

	tempServer.node = m.network.node
	peerinfo, err := tempServer.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: m.network.node.nodeInfo.cfg.GetVersion()})
	if err != nil {
		log.Error("GetPeerInfo", "Err", err.Error())
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventPeerList, &pb.PeerList{Peers: m.PeerInfos()}))
		return
	}

	var peers = m.PeerInfos()
	var peer pb.Peer
	peer.Addr = peerinfo.GetAddr()
	peer.Port = peerinfo.GetPort()
	peer.Name = peerinfo.GetName()
	peer.MempoolSize = peerinfo.GetMempoolSize()
	peer.Self = true
	peer.Header = peerinfo.GetHeader()
	peers = append(peers, &peer)
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventPeerList, &pb.PeerList{Peers: peers}))
	return
}

func (m *P2pCli) GetHeaders(msg queue.Message) {
	defer func() {
		<-m.network.otherFactory

	}()
	if m.network.node.Size() == 0 {
		log.Debug("GetHeaders", "boundNum", 0)
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	req := msg.GetData().(*pb.ReqBlocks)
	pid := req.GetPid()
	if len(pid) == 0 {
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no pid")}))
		return
	}
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{true, []byte("ok")}))
	peers, infos := m.network.node.GetActivePeers()
	for paddr, info := range infos {
		if info.GetName() == pid[0] { //匹配成功
			peer := m.network.node.GetRegisterPeer(paddr)
			peer, ok := peers[paddr]
			if ok && peer != nil {
				var err error

				headers, err := peer.mconn.conn.GetHeaders(context.Background(), &pb.P2PGetHeaders{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
					Version: m.network.node.nodeInfo.cfg.GetVersion()})
				P2pComm.CollectPeerStat(err, peer)
				if err != nil {
					log.Error("GetBlocks", "Err", err.Error())
					return
				}
				client := m.network.node.nodeInfo.qclient
				msg := client.NewMessage("blockchain", pb.EventAddBlockHeaders, &pb.Headers{Items: headers.GetHeaders()})
				client.Send(msg, false)
			}
		}
	}
}
func (m *P2pCli) GetBlocks(msg queue.Message) {
	defer func() {
		<-m.network.otherFactory

	}()
	if m.network.node.Size() == 0 {
		log.Debug("GetBlocks", "boundNum", 0)
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{true, []byte("downloading...")}))

	req := msg.GetData().(*pb.ReqBlocks)

	pids := req.GetPid()
	var MaxInvs = new(pb.P2PInv)
	var downloadPeers []*peer
	peers, infos := m.network.node.GetActivePeers()
	if len(pids) > 0 && pids[0] != "" { //指定Pid 下载数据
		log.Info("fetch from peer in pids")
		var pidmap = make(map[string]bool)
		for _, pid := range pids {
			pidmap[pid] = true
		}
		for paddr, info := range infos {
			if _, ok := pidmap[info.GetName()]; ok { //匹配成功

				peer, ok := peers[paddr]
				if ok && peer != nil {
					downloadPeers = append(downloadPeers, peer)
					//获取Invs
					if len(MaxInvs.GetInvs()) != int(req.GetEnd()-req.GetStart())+1 {
						var err error
						MaxInvs, err = peer.mconn.conn.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
							Version: m.network.node.nodeInfo.cfg.GetVersion()})
						P2pComm.CollectPeerStat(err, peer)
						if err != nil {
							log.Error("GetBlocks", "Err", err.Error())
							continue
						}
					}

				}
			}
		}

	} else {
		log.Info("fetch from all peers in pids")
		for _, peer := range peers { //限制对peer 的高频次调用
			log.Info("peer", "addr", peer.Addr(), "start", req.GetStart(), "end", req.GetEnd())
			peerinfo, ok := infos[peer.Addr()]
			if !ok {
				continue
			}
			var pr pb.Peer
			pr.Addr = peerinfo.GetAddr()
			pr.Port = peerinfo.GetPort()
			pr.Name = peerinfo.GetName()
			pr.MempoolSize = peerinfo.GetMempoolSize()
			pr.Header = peerinfo.GetHeader()

			m.network.node.nodeInfo.peerInfos.SetPeerInfo(&pr)
			if peerinfo.GetHeader().GetHeight() < req.GetEnd() {
				continue
			}
			invs, err := peer.mconn.conn.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
				Version: m.network.node.nodeInfo.cfg.GetVersion()})
			P2pComm.CollectPeerStat(err, peer)
			if err != nil {
				log.Error("GetBlocks", "Err", err.Error())
				continue
			}
			if len(invs.Invs) > len(MaxInvs.Invs) {
				MaxInvs = invs
				if len(MaxInvs.GetInvs()) == int(req.GetEnd()-req.GetStart())+1 {
					break
				}
			}

		}
		for _, peer := range peers {
			downloadPeers = append(downloadPeers, peer)
		}
	}

	log.Debug("Invs", "Invs show", MaxInvs.GetInvs())
	if len(MaxInvs.GetInvs()) == 0 {
		log.Error("GetBlocks", "getInvs", 0)
		return
	}
	var intervals = make(map[int]*intervalInfo)

	intervals = m.caculateInterval(len(downloadPeers), len(MaxInvs.GetInvs()))
	var bChan = make(chan *pb.Block, 256)
	log.Debug("downloadblock", "intervals", intervals)
	var gcount int
	for index, interval := range intervals {
		gcount++
		go m.downloadBlock(index, interval, MaxInvs, bChan, downloadPeers, infos)
	}
	i := 0
	for {
		timeout := time.NewTimer(time.Minute)
		select {
		case <-timeout.C:
			return
		case block := <-bChan:
			newmsg := m.network.node.nodeInfo.qclient.NewMessage("blockchain", pb.EventAddBlock, block)
			m.network.node.nodeInfo.qclient.SendAsyn(newmsg, false)
			i++
			if i == len(MaxInvs.GetInvs()) {
				return
			}
		}
		if !timeout.Stop() {
			<-timeout.C
		}
	}
}

func (m *P2pCli) downloadBlock(index int, interval *intervalInfo, invs *pb.P2PInv, bchan chan *pb.Block,
	peers []*peer, pinfos map[string]*pb.Peer) {
	if interval.end < interval.start {
		return
	}
	peersize := len(peers)
	log.Debug("downloadBlock", "download from index", index, "interval", interval, "peersize", peersize)
FOOR_LOOP:
	for i := 0; i < peersize; i++ {

		index = index % peersize
		log.Debug("downloadBlock", "index", index)
		var p2pdata pb.P2PGetData
		p2pdata.Version = m.network.node.nodeInfo.cfg.GetVersion()
		p2pdata.Invs = invs.Invs[interval.start:interval.end]
		log.Debug("downloadBlock", "interval invs", p2pdata.Invs, "start", interval.start, "end", interval.end)
		if index >= peersize {
			log.Error("download", "index", index, "peersise", peersize)
			continue
		}

		peer := peers[index]
		if peer == nil {
			index++
			log.Debug("download", "peer", "nil")
			continue
		}
		if pinfo, ok := pinfos[peer.Addr()]; ok {
			if pinfo.GetHeader().GetHeight() < int64(invs.Invs[interval.end-1].GetHeight()) {
				index++
				continue
			}
		} else {
			log.Debug("download", "pinfo", "no this addr", peer.Addr())
			index++
			continue
		}
		log.Debug("downloadBlock", "index", index, "peersize", peersize, "peeraddr", peer.Addr(), "p2pdata", p2pdata)
		resp, err := peer.mconn.conn.GetData(context.Background(), &p2pdata)
		P2pComm.CollectPeerStat(err, peer)
		if err != nil {
			log.Error("downloadBlock", "GetData err", err.Error())
			index++
			continue
		}
		var count int
		for {
			invdatas, err := resp.Recv()
			if err == io.EOF {
				log.Info("download", "recv", "IO.EOF", "count", count)
				resp.CloseSend()
				break FOOR_LOOP
			}
			if err != nil {
				log.Error("download", "resp,Recv err", err.Error(), "download from", peer.Addr())
				resp.CloseSend()
				break FOOR_LOOP
			}
			count++
			for _, item := range invdatas.Items {
				bchan <- item.GetBlock()
			}
		}
	}
	log.Info("download", "out of func", "ok")
}

func (m *P2pCli) lastPeerInfo() map[string]*pb.Peer {
	var peerlist = make(map[string]*pb.Peer)
	peers := m.network.node.GetRegisterPeers()
	for _, peer := range peers {

		if peer.Addr() == m.network.node.nodeInfo.GetExternalAddr().String() { //fmt.Sprintf("%v:%v", ExternalIp, m.network.node.GetExterPort())
			continue
		}
		peerinfo, err := peer.GetPeerInfo(m.network.node.nodeInfo.cfg.GetVersion())
		if err != nil {
			if strings.Contains(err.Error(), VersionNotSupport) {
				peer.version.SetSupport(false)
				P2pComm.CollectPeerStat(err, peer)

			}
			continue
		}
		P2pComm.CollectPeerStat(err, peer)
		var pr pb.Peer
		pr.Addr = peerinfo.GetAddr()
		pr.Port = peerinfo.GetPort()
		pr.Name = peerinfo.GetName()
		pr.MempoolSize = peerinfo.GetMempoolSize()
		pr.Header = peerinfo.GetHeader()
		peerlist[fmt.Sprintf("%v:%v", peerinfo.Addr, peerinfo.Port)] = &pr
	}
	return peerlist
}

func (m *P2pCli) caculateInterval(peerNum, invsNum int) map[int]*intervalInfo {
	log.Debug("caculateInterval", "invsNum", invsNum, "peerNum", peerNum)
	var result = make(map[int]*intervalInfo)
	if peerNum == 0 {
		//如果没有peer,那么没有办法分割
		result[0] = &intervalInfo{0, invsNum}
		return result
	}
	var interval = invsNum / peerNum
	if interval == 0 {
		interval = 1
	}
	var start, end int

	for i := 0; i < peerNum; i++ {
		end += interval
		if end >= invsNum || i == peerNum-1 {
			end = invsNum
		}
		result[i] = &intervalInfo{start: start, end: end}
		log.Debug("caculateInterval", "createinfo", result[i])
		start = end
	}
	return result

}

func (m *P2pCli) BlockBroadcast(msg queue.Message) {
	defer func() {
		<-m.network.otherFactory
	}()
	pub.FIFOPub(&pb.P2PBlock{Block: msg.GetData().(*pb.Block)}, "block")
}

func (m *P2pCli) GetExternIp(addr string) (string, bool) {
	var addrlist string
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		log.Error("grpc DialCon", "did not connect: %v", err)
		return addrlist, false
	}
	defer conn.Close()
	gconn := pb.NewP2PgserviceClient(conn)
	resp, err := gconn.RemotePeerAddr(context.Background(), &pb.P2PGetAddr{Nonce: 12})
	if err != nil {
		return "", false
	}

	return resp.Addr, resp.Isoutside

}

func (m *P2pCli) flushPeerInfos(in []*pb.Peer) {
	m.network.node.nodeInfo.peerInfos.flushPeerInfos(in)
}

func (m *P2pCli) PeerInfos() []*pb.Peer {

	peerinfos := m.network.node.nodeInfo.peerInfos.GetPeerInfos()
	var peers []*pb.Peer
	for _, peer := range peerinfos {

		if peer.GetAddr() == m.network.node.nodeInfo.GetExternalAddr().IP.String() && peer.GetPort() == int32(m.network.node.nodeInfo.GetExternalAddr().Port) {
			continue
		}
		peers = append(peers, peer)
	}
	return peers
}

func (m *P2pCli) monitorPeerInfo() {

	go func(m *P2pCli) {
		m.fetchPeerInfo()
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
	FOR_LOOP:
		for {
			select {
			case <-ticker.C:
				m.fetchPeerInfo()

			case <-m.loopdone:
				log.Error("p2pcli monitorPeerInfo", "loop", "done")
				break FOR_LOOP
			}

		}
	}(m)
}

func (m *P2pCli) fetchPeerInfo() {
	var peerlist []*pb.Peer
	peerInfos := m.lastPeerInfo()
	for _, peerinfo := range peerInfos {
		peerlist = append(peerlist, peerinfo)
	}
	m.flushPeerInfos(peerlist)
}
