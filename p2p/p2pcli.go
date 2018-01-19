package p2p

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/queue"
	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type P2pCli struct {
	network  *P2p
	mtx      sync.Mutex
	taskinfo map[int64]bool
	done     chan struct{}
}
type intervalInfo struct {
	start int
	end   int
}

func (m *P2pCli) addTask(taskid int64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.taskinfo[taskid] = true
}

func (m *P2pCli) queryTask(taskid int64) bool {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if _, ok := m.taskinfo[taskid]; ok {
		return true
	}
	return false
}

func (m *P2pCli) removeTask(taskid int64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if _, ok := m.taskinfo[taskid]; ok {
		delete(m.taskinfo, taskid)
	}
	return
}
func NewP2pCli(network *P2p) *P2pCli {
	pcli := &P2pCli{
		network:  network,
		taskinfo: make(map[int64]bool),
		done:     make(chan struct{}, 1),
	}

	return pcli
}

func (m *P2pCli) CollectPeerStat(err error, peer *peer) {
	if err != nil {
		peer.peerStat.NotOk()
	} else {
		peer.setRunning(true)
		peer.peerStat.Ok()
	}
	m.deletePeer(peer)
}
func (m *P2pCli) BroadCastTx(msg queue.Message) {
	defer func() {
		<-m.network.txFactory
		atomic.AddInt32(&m.network.txCapcity, 1)
	}()

	//m.broadcastByStream(&pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)})
	//	if m.network.node.Size() == 0 {
	//		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
	//		return
	//	}

	peers := m.network.node.GetRegisterPeers()
	for _, pr := range peers {
		pr.SendData(&pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)}) //异步处理
	}

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
		m.CollectPeerStat(err, peer)
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
		m.CollectPeerStat(dataerr, peer)
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
	m.CollectPeerStat(err, peer)
	if err != nil {
		return nil, err
	}

	log.Debug("GetAddr Resp", "Resp", resp, "addrlist", resp.Addrlist)
	return resp.Addrlist, nil
}

func (m *P2pCli) SendVersion(peer *peer, nodeinfo *NodeInfo) error {

	client := nodeinfo.qclient
	msg := client.NewMessage("blockchain", pb.EventGetBlockHeight, nil)
	client.Send(msg, true)
	rsp, err := client.Wait(msg)
	if err != nil {
		log.Error("GetHeight", "Error", err.Error())
		return err
	}

	blockheight := rsp.GetData().(*pb.ReplyBlockHeight).GetHeight()
	randNonce := rand.Int31n(102040)
	in, err := m.signature(peer.mconn.key, &pb.P2PPing{Nonce: int64(randNonce), Addr: ExternalIp, Port: int32(nodeinfo.externalAddr.Port)})
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return err
	}
	addrfrom := fmt.Sprintf("%v:%v", ExternalIp, nodeinfo.externalAddr.Port)
	nodeinfo.blacklist.Add(addrfrom)
	resp, err := peer.mconn.conn.Version2(context.Background(), &pb.P2PVersion{Version: nodeinfo.cfg.GetVersion(), Service: SERVICE, Timestamp: time.Now().Unix(),
		AddrRecv: peer.Addr(), AddrFrom: addrfrom, Nonce: int64(rand.Int31n(102040)),
		UserAgent: hex.EncodeToString(in.Sign.GetPubkey()), StartHeight: blockheight})
	m.CollectPeerStat(err, peer)
	if err != nil {
		log.Debug("SendVersion", "Verson", err.Error())
		peer.version.SetSupport(false)
		if strings.Compare(err.Error(), VersionNotSupport) == 0 {
			m.CollectPeerStat(err, peer)

		}
		return err
	}

	log.Debug("SHOW VERSION BACK", "VersionBack", resp)
	return nil
}

func (m *P2pCli) SendPing(peer *peer, nodeinfo *NodeInfo) error {
	randNonce := rand.Int31n(102040)
	in, err := m.signature(peer.mconn.key, &pb.P2PPing{Nonce: int64(randNonce), Addr: ExternalIp, Port: int32(nodeinfo.externalAddr.Port)})
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return err
	}
	log.Debug("SEND PING", "Peer", peer.Addr(), "nonce", randNonce)
	r, err := peer.mconn.conn.Ping(context.Background(), in)
	m.CollectPeerStat(err, peer)
	if err != nil {
		log.Warn("SEND PING", "Err", err.Error(), "peer", peer.Addr())
		return err
	}

	log.Debug("RECV PONG", "resp:", r.Nonce, "Ping nonce:", randNonce)
	return nil
}

func (m *P2pCli) GetPeerInfo(msg queue.Message) {

	//log.Info("GetPeerInfo", "info", m.PeerInfos())
	//添加自身peerInfo
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
	var MaxInvs = new(pb.P2PInv)
	peers, pinfos := m.network.node.GetActivePeers()
	for _, peer := range peers { //限制对peer 的高频次调用

		log.Error("peer", "addr", peer.Addr(), "start", req.GetStart(), "end", req.GetEnd())
		peerinfo, err := peer.GetPeerInfo(m.network.node.nodeInfo.cfg.GetVersion())
		m.CollectPeerStat(err, peer)
		if err != nil {
			log.Error("GetPeers", "Err", err.Error())
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
		m.CollectPeerStat(err, peer)
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
	log.Info("Invs", "Invs show", MaxInvs.GetInvs())
	if len(MaxInvs.GetInvs()) == 0 {
		log.Error("GetBlocks", "getInvs", 0)
		return
	}

	intervals := m.caculateInterval(len(MaxInvs.GetInvs()))
	var bChan = make(chan *pb.Block, 256)
	log.Error("downloadblock", "intervals", intervals)
	var gcount int
	for index, interval := range intervals {
		gcount++
		go m.downloadBlock(index, interval, MaxInvs, bChan, peers, pinfos)
	}
	i := 0
	for {
		select {
		case <-time.After(time.Minute):
			return
		case block := <-bChan:
			newmsg := m.network.node.nodeInfo.qclient.NewMessage("blockchain", pb.EventAddBlock, block)
			m.network.node.nodeInfo.qclient.SendAsyn(newmsg, false)
			i++
			if i == len(MaxInvs.GetInvs()) {
				return
			}
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
		//判断请求的节点的高度是否在节点的实际范围内
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
				//log.Debug("download", "much height", pinfo.GetHeader().GetHeight(), "invs height", int64(invs.Invs[interval.end-1].GetHeight()))
				continue
			}
		} else {
			log.Debug("download", "pinfo", "no this addr", peer.Addr())
			index++
			continue
		}
		log.Debug("downloadBlock", "index", index, "peersize", peersize, "peeraddr", peer.Addr(), "p2pdata", p2pdata)
		resp, err := peer.mconn.conn.GetData(context.Background(), &p2pdata)
		m.CollectPeerStat(err, peer)
		if err != nil {
			log.Error("downloadBlock", "GetData err", err.Error())
			index++
			continue
		}
		var count int
		for {
			invdatas, err := resp.Recv()
			if err == io.EOF {
				log.Error("download", "recv", "IO.EOF", "count", count)
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
	log.Error("download", "out of func", "ok")
}

func (m *P2pCli) lastPeerInfo() map[string]*pb.Peer {
	var peerlist = make(map[string]*pb.Peer)
	peers := m.network.node.GetRegisterPeers()
	for _, peer := range peers {
		if peer.Addr() == fmt.Sprintf("%v:%v", ExternalIp, m.network.node.GetExterPort()) {
			continue
		}
		peerinfo, err := peer.GetPeerInfo(m.network.node.nodeInfo.cfg.GetVersion())
		peer.setRunning(false)
		m.CollectPeerStat(err, peer)
		if err != nil {
			continue
		}
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

func (m *P2pCli) wait(wg *sync.WaitGroup) {
	wg.Wait()
}

func (m *P2pCli) sortKeys(bchan chan *pb.Block) (map[int64]*pb.Block, []int) {

	var keys []int
	var blocks = make(map[int64]*pb.Block)
	for block := range bchan {
		keys = append(keys, int(block.GetHeight()))
		blocks[block.GetHeight()] = block
	}

	sort.Ints(keys)
	return blocks, keys

}

func (m *P2pCli) caculateInterval(invsNum int) map[int]*intervalInfo {
	log.Debug("caculateInterval", "invsNum", invsNum)
	var result = make(map[int]*intervalInfo)
	peerNum := m.network.node.Size()
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
		log.Info("caculateInterval", "createinfo", result[i])
		start = end
	}
	return result

}
func (m *P2pCli) broadcastByStream(data interface{}) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	select {
	case <-ticker.C:
		//log.Error("broadcastByStream", "timeout", "return")
		return
	case m.network.node.nodeInfo.p2pBroadcastChan <- data:
	}

}
func (m *P2pCli) BlockBroadcast(msg queue.Message) {
	defer func() {
		<-m.network.otherFactory
		//log.Error("BlockBroadcast", "Release Task", "ok")
	}()
	block := msg.GetData().(*pb.Block)
	peers, _ := m.network.node.GetActivePeers()
	m.broadcastByStream(&pb.P2PBlock{Block: block})

	log.Debug("BlockBroadcast", "SendTOP2P", msg.GetData())
	if m.network.node.Size() == 0 {
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}

	for _, peer := range peers {
		//比较peer 的高度是否低于广播的高度，如果高于，则不广播给对方
		peerinfo, err := peer.GetPeerInfo(m.network.node.nodeInfo.cfg.GetVersion())
		if err != nil {
			continue
		}
		if peerinfo.GetHeader().GetHeight() > block.GetHeight() {
			continue
		}
		resp, err := peer.mconn.conn.BroadCastBlock(context.Background(), &pb.P2PBlock{Block: block})
		m.CollectPeerStat(err, peer)
		if err != nil {
			log.Error("BlockBroadcast", "Error", err.Error())
			continue
		}
		log.Debug("BlockBroadcast", "Resp", resp)
	}
	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{true, []byte("ok")}))
}

func (m *P2pCli) GetExternIp(addr string) []string {
	var addrlist []string
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		log.Error("grpc DialCon", "did not connect: %v", err)
		return addrlist
	}
	defer conn.Close()
	gconn := pb.NewP2PgserviceClient(conn)
	resp, err := gconn.RemotePeerAddr(context.Background(), &pb.P2PGetAddr{Nonce: 12})
	if err != nil {
		return addrlist
	}

	return resp.Addrlist

}
func (m *P2pCli) Close() {

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	select {
	case m.done <- struct{}{}:
	case <-ticker.C:
		return
	}

}
func (m *P2pCli) deletePeer(peer *peer) {

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	select {
	case (*peer.nodeInfo).monitorChan <- peer:
	case <-ticker.C:
		return
	}
}
func (m *P2pCli) signature(key string, in *pb.P2PPing) (*pb.P2PPing, error) {
	data := pb.Encode(in)

	cr, err := crypto.New(pb.GetSignatureTypeName(pb.SECP256K1))
	if err != nil {
		log.Error("CryPto Error", "Error", err.Error())
		return nil, err
	}
	pribyts, err := hex.DecodeString(key)
	if err != nil {
		log.Error("DecodeString Error", "Error", err.Error())
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(pribyts)
	if err != nil {
		log.Error("Load PrivKey", "Error", err.Error())
		return nil, err
	}
	in.Sign = new(pb.Signature)
	in.Sign.Signature = priv.Sign(data).Bytes()
	in.Sign.Ty = pb.SECP256K1
	in.Sign.Pubkey = priv.PubKey().Bytes()
	return in, nil
}
func (m *P2pCli) flushPeerInfos(in []*pb.Peer) {
	m.network.node.nodeInfo.peerInfos.flushPeerInfos(in)

}

func (m *P2pCli) PeerInfos() []*pb.Peer {

	peerinfos := m.network.node.nodeInfo.peerInfos.GetPeerInfos()
	var peers []*pb.Peer
	for _, peer := range peerinfos {

		if peer.GetAddr() == ExternalIp && peer.GetPort() == int32(m.network.node.GetExterPort()) {
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

			case <-m.done:
				log.Error("monitorPeerInfo", "done", "close")
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
