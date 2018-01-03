package p2p

import (
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/queue"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
)

type Msg struct {
	mtx       sync.Mutex
	pmtx      sync.Mutex
	wg        sync.WaitGroup
	network   *P2p
	tempdata  map[int64]*pb.Block
	peermtx   sync.Mutex
	peers     []*peer
	peerInfos map[string]*pb.Peer
	done      chan struct{}
}

func NewInTrans(network *P2p) *Msg {
	pmsg := &Msg{
		network:   network,
		peerInfos: make(map[string]*pb.Peer),
		done:      make(chan struct{}, 1),
	}

	return pmsg
}

func (m *Msg) TransToBroadCast(msg queue.Message) {
	log.Debug("TransToBroadCast", "SendTOP2P", msg.GetData())
	if m.network.node.Size() == 0 {
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	m.network.node.nodeInfo.p2pBroadcastChan <- &pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)}
	//开始广播消息
	peers := m.network.node.GetPeers()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	for _, peer := range peers {
		_, err := peer.mconn.conn.BroadCastTx(ctx, &pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			continue
		}
		peer.mconn.sendMonitor.Update(true)
	}

	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{true, []byte("ok")}))

}

//TODO 收到Mempool 模块获取mempool 的请求,从高度最高的节点 下载invs
func (m *Msg) GetMemPool(msg queue.Message) {
	log.Debug("GetMemPool", "SendTOP2P", msg.GetData())
	var Txs = make([]*pb.Transaction, 0)
	var ableInv = make([]*pb.Inventory, 0)
	peers := m.network.node.GetPeers()

	for _, peer := range peers {
		//获取远程 peer invs
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		resp, err := peer.mconn.conn.GetMemPool(ctx, &pb.P2PGetMempool{Version: m.network.node.nodeInfo.cfg.GetVersion()})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			cancel()
			continue
		}
		cancel()
		peer.mconn.sendMonitor.Update(true)
		invs := resp.GetInvs()
		//与本地mempool 对比 tx数组
		msg := m.network.c.NewMessage("mempool", pb.EventGetMempool, nil)
		m.network.c.Send(msg, true)
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
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
		p2pInvDatas, err := peer.mconn.conn.GetData(ctx, &pb.P2PGetData{Invs: ableInv, Version: m.network.node.nodeInfo.cfg.GetVersion()})
		if err != nil {
			cancel()
			continue
		}
		cancel()
		for _, invdata := range p2pInvDatas.Items {
			Txs = append(Txs, invdata.GetTx())
		}
		break
	}
	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReplyTxList, &pb.ReplyTxList{Txs: Txs}))

}
func (m *Msg) flushPeerInfos(in []*pb.Peer) {
	m.pmtx.Lock()
	defer m.pmtx.Unlock()
	//首先清空之前的数据
	for k, _ := range m.peerInfos {
		delete(m.peerInfos, k)
	}
	//重新插入新数据
	for _, peer := range in {
		log.Debug("flushPeerInfos", "info", peer)
		m.peerInfos[peer.GetName()] = peer
	}

}
func (m *Msg) getPeerInfos() []*pb.Peer {
	m.pmtx.Lock()
	defer m.pmtx.Unlock()
	var peers []*pb.Peer
	for _, peer := range m.peerInfos {
		peers = append(peers, peer)
	}
	return peers
}
func (m *Msg) monitorPeerInfo() {

	go func(m *Msg) {
	FOR_LOOP:
		for {

			ticker := time.NewTicker(time.Second * 10)

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
func (m *Msg) fetchPeerInfo() {
	var peerlist []*pb.Peer
	peerInfos := m.lastPeerInfo()
	for _, peerinfo := range peerInfos {
		peerlist = append(peerlist, peerinfo)
	}
	m.flushPeerInfos(peerlist)
}

//收到BlockChain 模块的请求，获取PeerInfo
func (m *Msg) GetPeerInfo(msg queue.Message) {

	log.Warn("GetPeerInfo", "info", m.getPeerInfos())
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventPeerList, &pb.PeerList{Peers: m.getPeerInfos()}))
	return
}

//TODO 立刻返回数据 ，然后把下载的数据用事件通知对方,异步操作
func (m *Msg) GetBlocks(msg queue.Message) {

	if m.network.node.Size() == 0 {
		log.Debug("GetBlocks", "boundNum", 0)
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{true, []byte("downloading...")}))

	// 第一步获取下载列表，第二步分配不同的节点分段下载需要的数据
	var blocks pb.Blocks
	req := msg.GetData().(*pb.ReqBlocks)
	log.Debug("GetBlocks", "req", req)
	var MaxInvs = new(pb.P2PInv)
	//获取最大的下载列表
	peers := m.network.node.GetPeers()
	log.Debug("GetBlocks", "peers", len(peers))
	for _, peer := range peers {
		invs, err := peer.mconn.conn.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: req.GetStart(), EndHeight: req.GetEnd()})
		if err != nil {
			log.Error("GetBlocks", "Err", err.Error())
			peer.mconn.sendMonitor.Update(false)
			continue
		}

		peer.mconn.sendMonitor.Update(true)
		if len(invs.Invs) > len(MaxInvs.Invs) {
			MaxInvs = invs
			if len(MaxInvs.GetInvs()) == int(req.GetEnd()-req.GetStart())+1 {
				break
			}
		}
	}
	if len(MaxInvs.GetInvs()) == 0 {
		log.Error("GetBlocks", "getInvs", 0)
		return
	}

	intervals := m.caculateInterval(len(MaxInvs.GetInvs()))
	m.tempdata = make(map[int64]*pb.Block)
	//分段下载
	for index, interval := range intervals {
		m.wg.Add(1)

		go m.downloadBlock(index, interval, MaxInvs)
	}
	//等待所有 goroutin 结束
	m.wait()
	//返回数据
	keys := m.sortKeys()
	for _, k := range keys {
		blocks.Items = append(blocks.Items, m.tempdata[int64(k)])
	}

	//作为事件，发送给blockchain,事件是 EventAddBlocks
	newmsg := m.network.node.nodeInfo.qclient.NewMessage("blockchain", pb.EventAddBlocks, &blocks)
	m.network.node.nodeInfo.qclient.Send(newmsg, false)

}

func (m *Msg) lastPeerInfo() map[string]*pb.Peer {
	var peerlist = make(map[string]*pb.Peer)
	peers := m.network.node.GetPeers()
	log.Debug("monitorPeerInfo", "peers", peers)

	for _, peer := range peers {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		peerinfo, err := peer.mconn.conn.GetPeerInfo(ctx, &pb.P2PGetPeerInfo{Version: m.network.node.nodeInfo.cfg.GetVersion()})
		if err != nil {
			cancel()
			peer.mconn.sendMonitor.Update(false)
			log.Error("monitorPeerInfo", "error", err.Error())
			continue
		}
		cancel()
		log.Debug("monitorPeerInfo", "info", peerinfo)
		peer.mconn.sendMonitor.Update(true)
		//peerlist = append(peerlist, (*pb.Peer)(peerinfo))
		peerlist[fmt.Sprintf("%v:%v", peerinfo.Addr, peerinfo.Port)] = (*pb.Peer)(peerinfo)

	}
	return peerlist
}
func (m *Msg) loadPeers() {
	m.peermtx.Lock()
	defer m.peermtx.Unlock()

	m.peers = append(m.peers, m.network.node.GetPeers()...)
}
func (m *Msg) peerSize() int {
	m.peermtx.Lock()
	defer m.peermtx.Unlock()
	return len(m.peers)
}
func (m *Msg) wait() {
	m.wg.Wait()
}

type intervalInfo struct {
	start int
	end   int
}

func (m *Msg) sortKeys() []int {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	var keys []int
	for k, _ := range m.tempdata {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	return keys

}
func (m *Msg) downloadBlock(index int, interval *intervalInfo, invs *pb.P2PInv) {

	defer m.wg.Done()
	if interval.end < interval.start {
		return
	}
	log.Debug("downloadBlock", "parminfo", index, "interval", interval)
	peersize := m.network.node.Size()
	log.Debug("downloadBlock", "peersize", peersize)
	maxInvDatas := new(pb.InvDatas)
	for i := 0; i < peersize; i++ {
		m.loadPeers()
		index = index % m.peerSize()
		log.Debug("downloadBlock", "index", index)
		var p2pdata pb.P2PGetData
		if interval.end >= len(invs.GetInvs()) || len(invs.GetInvs()) == 1 {
			interval.end = len(invs.GetInvs()) - 1
			p2pdata.Invs = invs.Invs[interval.start:]
		} else {
			p2pdata.Invs = invs.Invs[interval.start:interval.end]
		}
		log.Debug("downloadBlock", "interval invs", p2pdata.Invs)
		//判断请求的节点的高度是否在节点的实际范围内
		if index >= m.peerSize() {
			continue
		}
		ps := m.lastPeerInfo()
		if peer, ok := ps[m.peers[index].Addr()]; ok {
			if peer.GetHeader().GetHeight() < int64(interval.end) {
				continue
			}
		} else {
			continue
		}

		invdatas, err := m.peers[index].mconn.conn.GetData(context.Background(), &p2pdata)
		if err != nil {
			log.Error("downloadBlock", "err", err.Error())
			m.peers[index].mconn.sendMonitor.Update(false)
			index++
			continue
		}
		if len(invdatas.GetItems()) > len(maxInvDatas.Items) ||
			len(invdatas.GetItems()) == interval.end-interval.start+1 {
			maxInvDatas = invdatas
			break
		}

	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for _, item := range maxInvDatas.Items {
		m.tempdata[item.GetBlock().GetHeight()] = item.GetBlock()
	}

}
func (m *Msg) caculateInterval(invsNum int) map[int]*intervalInfo {
	log.Debug("caculateInterval", "invsNum", invsNum)
	var result = make(map[int]*intervalInfo)
	peerNum := m.network.node.Size()
	if invsNum < peerNum {
		result[0] = &intervalInfo{start: 0, end: invsNum - 1}
		return result
	}
	var interval = invsNum / peerNum
	var start, end int

	for i := 0; i < peerNum; i++ {
		end += interval
		if end >= invsNum || i == peerNum-1 {
			end = invsNum - 1
		}

		result[i] = &intervalInfo{start: start, end: end}
		log.Debug("caculateInterval", "createinfo", result[i])
		start = end
	}

	return result

}

func (m *Msg) BlockBroadcast(msg queue.Message) {
	log.Debug("BlockBroadcast", "SendTOP2P", msg.GetData())
	if m.network.node.Size() == 0 {
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}

	block := msg.GetData().(*pb.Block)
	peers := m.network.node.GetPeers()
	//stream blockbroadcast
	m.network.node.nodeInfo.p2pBroadcastChan <- &pb.P2PBlock{Block: block}
	for _, peer := range peers {
		resp, err := peer.mconn.conn.BroadCastBlock(context.Background(), &pb.P2PBlock{Block: block})
		if err != nil {
			log.Error("BlockBroadcast", "Error", err.Error())
			continue
		}
		log.Debug("BlockBroadcast", "Resp", resp)
	}
	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{true, []byte("ok")}))
}
