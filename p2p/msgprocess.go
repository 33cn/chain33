package p2p

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/queue"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
)

type Msg struct {
	mtx sync.Mutex

	network   *P2p
	peerInfos map[string]*pb.Peer
	done      chan struct{}
}
type intervalInfo struct {
	start int
	end   int
}

func NewMsg(network *P2p) *Msg {
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
	//通过grpc stream发送到各个节点
	m.network.node.nodeInfo.p2pBroadcastChan <- &pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)}
	peers, _ := m.network.node.GetPeers()

	for _, peer := range peers {
		_, err := peer.mconn.conn.BroadCastTx(context.Background(), &pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)})
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
	peers, _ := m.network.node.GetPeers()

	for _, peer := range peers {
		//获取远程 peer invs
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
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
		datacli, dataerr := peer.mconn.conn.GetData(context.Background(), &pb.P2PGetData{Invs: ableInv, Version: m.network.node.nodeInfo.cfg.GetVersion()})
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
func (m *Msg) flushPeerInfos(in []*pb.Peer) {
	m.network.node.nodeInfo.peerInfos.flushPeerInfos(in)

}

func (m *Msg) getPeerInfos() []*pb.Peer {

	peerinfos := m.network.node.nodeInfo.peerInfos.GetPeerInfos()
	var peers []*pb.Peer
	for _, peer := range peerinfos {
		peers = append(peers, peer)
	}
	return peers
}
func (m *Msg) monitorPeerInfo() {

	go func(m *Msg) {
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
func (m *Msg) Stop() {
	m.done <- struct{}{}
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

	log.Info("GetPeerInfo", "info", m.getPeerInfos())
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventPeerList, &pb.PeerList{Peers: m.getPeerInfos()}))
	return
}

func (m *Msg) GetBlocks(msg queue.Message) {

	if m.network.node.Size() == 0 {
		log.Debug("GetBlocks", "boundNum", 0)
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{true, []byte("downloading...")}))

	// 第一步获取下载列表，第二步分配不同的节点分段下载需要的数据

	req := msg.GetData().(*pb.ReqBlocks)
	log.Debug("GetBlocks", "req", req)
	var MaxInvs = new(pb.P2PInv)
	//获取最大的下载列表
	peers, _ := m.network.node.GetPeers()
	//log.Error("peers", "show", peers)
	for _, peer := range peers {
		log.Error("peer", "addr", peer.Addr())
		peerinfo := m.network.node.nodeInfo.peerInfos.GetPeerInfo(peer.Addr())
		if peerinfo.GetHeader().GetHeight() < req.GetEnd() {
			continue
		}
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
	var bChan = make(chan *pb.Block, 256) //下次的区块不超过256个
	//分段下载
	log.Error("downloadblock", "intervals", intervals)
	var gcount int
	var wg sync.WaitGroup
	for index, interval := range intervals {
		gcount++
		wg.Add(1)
		go m.downloadBlock(index, interval, MaxInvs, bChan, &wg)
	}
	//等待所有 goroutin 结束
	log.Error("downloadblock", "wait", "befor", "groutin num", gcount)
	m.wait(&wg)
	//返回数据
	log.Error("downloadblock", "wait", "after")
	close(bChan)                   //关闭channal
	bks, keys := m.sortKeys(bChan) //区块排序
	var blocks pb.Blocks
	for _, k := range keys {
		log.Error("downloadblock", "index sort", int64(k))
		blocks.Items = append(blocks.Items, bks[int64(k)])
	}
	//作为事件，发送给blockchain,事件是 EventAddBlocks
	newmsg := m.network.node.nodeInfo.qclient.NewMessage("blockchain", pb.EventAddBlocks, &blocks)
	m.network.node.nodeInfo.qclient.Send(newmsg, false)

}

func (m *Msg) lastPeerInfo() map[string]*pb.Peer {
	var peerlist = make(map[string]*pb.Peer)
	peers := m.network.node.GetRegisterPeers()
	for _, peer := range peers {
		if peer.mconn.sendMonitor.GetCount() > 0 {
			continue
		}

		peerinfo, err := peer.mconn.conn.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: m.network.node.nodeInfo.cfg.GetVersion()})
		if err != nil {
			m.network.node.nodeInfo.monitorChan <- peer //直接删掉问题节点
			continue
		}
		peer.mconn.sendMonitor.Update(true)
		peerlist[fmt.Sprintf("%v:%v", peerinfo.Addr, peerinfo.Port)] = (*pb.Peer)(peerinfo)
	}
	return peerlist
}

func (m *Msg) wait(wg *sync.WaitGroup) {
	wg.Wait()
}

func (m *Msg) sortKeys(bchan chan *pb.Block) (map[int64]*pb.Block, []int) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	var keys []int
	var blocks = make(map[int64]*pb.Block)
	for block := range bchan {
		keys = append(keys, int(block.GetHeight()))
		blocks[block.GetHeight()] = block
	}

	sort.Ints(keys)
	return blocks, keys

}
func (m *Msg) downloadBlock(index int, interval *intervalInfo, invs *pb.P2PInv, bchan chan *pb.Block, wg *sync.WaitGroup) {

	defer wg.Done()
	if interval.end < interval.start {
		return
	}

	peers, pinfos := m.network.node.GetPeers()
	peersize := len(peers)
	log.Debug("downloadBlock", "download from index", index, "interval", interval, "peersize", peersize)
FOOR_LOOP:
	for i := 0; i < peersize; i++ {

		index = index % peersize
		log.Debug("downloadBlock", "index", index)
		var p2pdata pb.P2PGetData
		p2pdata.Version = m.network.node.nodeInfo.cfg.GetVersion()
		if interval.end >= len(invs.GetInvs()) || len(invs.GetInvs()) == 1 {
			interval.end = len(invs.GetInvs()) - 1
			p2pdata.Invs = invs.Invs[interval.start:]
		} else {
			p2pdata.Invs = invs.Invs[interval.start:interval.end]
		}
		log.Debug("downloadBlock", "interval invs", p2pdata.Invs)
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
			if pinfo.GetHeader().GetHeight() < int64(invs.Invs[interval.end].GetHeight()) {
				index++
				log.Debug("download", "much height", pinfo.GetHeader().GetHeight(), "invs height", int64(invs.Invs[interval.end].GetHeight()))
				continue
			}
		} else {
			log.Debug("download", "pinfo", "no this addr", peer.Addr())
			index++
			continue
		}
		log.Debug("downloadBlock", "index", index, "peersize", peersize, "peeraddr", peer.Addr(), "p2pdata", p2pdata)
		resp, err := peer.mconn.conn.GetData(context.Background(), &p2pdata)
		if err != nil {
			log.Error("downloadBlock", "GetData err", err.Error())
			peer.mconn.sendMonitor.Update(false)
			index++
			continue
		}
		//var InvDatas *pb.InvDatas
		var count int
		for {
			count++
			invdatas, err := resp.Recv()
			if err == io.EOF {
				log.Error("download", "recv", "IO.EOF", "count", count)
				resp.CloseSend()
				if count != (interval.end - interval.end + 1) {
					index++
					break
				}

				break FOOR_LOOP
			}
			if err != nil && err != io.EOF {
				log.Error("download", "resp,Recv err", err.Error())
				resp.CloseSend()
				break FOOR_LOOP
			}
			for _, item := range invdatas.Items {

				bchan <- item.GetBlock()
			}
		}

	}
	log.Error("download", "out of func", "ok")
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
	peers, _ := m.network.node.GetPeers()
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

func (m *Msg) GetTaskInfo(msg queue.Message) {
	//TODO  查询任务状态
}
