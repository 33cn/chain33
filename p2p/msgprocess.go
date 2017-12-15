package p2p

import (
	"encoding/hex"
	"sync"

	"code.aliyun.com/chain33/chain33/queue"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
)

type msg struct {
	mtx       sync.Mutex
	network   *P2p
	msgStatus chan bool
	tempdata  map[int64]*pb.Block
	peers     []*peer
}

func NewInTrans(network *P2p) *msg {
	return &msg{
		network: network,
	}
}

func (m *msg) TransToBroadCast(msg queue.Message) {
	log.Debug("TransToBroadCast", "SendTOP2P", msg.GetData())
	if m.network.node.Size() == 0 {
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	//开始广播消息
	peers := m.network.node.GetPeers()
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
func (m *msg) GetMemPool(msg queue.Message) {
	log.Debug("GetMemPool", "SendTOP2P", msg.GetData())
	var Txs = make([]*pb.Transaction, 0)
	var ableInv = make([]*pb.Inventory, 0)
	peers := m.network.node.GetPeers()
	for _, peer := range peers {
		//获取远程 peer invs
		resp, err := peer.mconn.conn.GetMemPool(context.Background(), &pb.P2PGetMempool{Version: Version})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			continue
		}
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
		p2pInvDatas, err := peer.mconn.conn.GetData(context.Background(), &pb.P2PGetData{Invs: ableInv, Version: Version})
		if err != nil {
			continue
		}
		for _, invdata := range p2pInvDatas.Items {
			Txs = append(Txs, invdata.GetTx())
		}
		break
	}
	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReplyTxList, &pb.ReplyTxList{Txs: Txs}))

}

//收到BlockChain 模块的请求，获取PeerInfo
func (m *msg) GetPeerInfo(msg queue.Message) {
	log.Debug("GetPeerInfo", "SendTOP2P", msg.GetData())
	var peerlist = make([]*pb.Peer, 0)
	peers := m.network.node.GetPeers()
	for _, peer := range peers {
		//peer.mconn.conn.get
		peerinfo, err := peer.mconn.conn.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: Version})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			continue
		}
		peer.mconn.sendMonitor.Update(true)
		peerlist = append(peerlist, (*pb.Peer)(peerinfo))

	}

	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventPeerList, &pb.PeerList{Peers: peerlist}))

	return
}

//TODO 立刻返回数据 ，然后把下载的数据用事件通知对方,异步操作
func (m *msg) GetBlocks(msg queue.Message) {
	log.Debug("GetBlocks", "SendTOP2P", msg.GetData())
	if len(m.network.node.outBound) == 0 {
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventReply, pb.Reply{true, []byte("downloading...")}))

	// 第一步获取下载列表，第二步分配不同的节点分段下载需要的数据
	var blocks pb.Blocks
	req := msg.GetData().(*pb.ReqBlocks)
	var MaxInvs = new(pb.P2PInv)
	m.peers = make([]*peer, 0)
	//获取最大的下载列表
	peers := m.network.node.GetPeers()
	for _, peer := range peers {
		invs, err := peer.mconn.conn.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: req.GetStart(), EndHeight: req.GetEnd()})
		if err != nil {
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
	log.Debug("GetBlocks", "invs", MaxInvs.GetInvs())
	m.loadPeers()
	intervals := m.caculateInterval(len(MaxInvs.GetInvs()))
	log.Debug("GetBlocks", "intervals", intervals)
	m.msgStatus = make(chan bool, len(intervals))
	m.tempdata = make(map[int64]*pb.Block)
	//分段下载
	for index, interval := range intervals {
		go m.downloadBlock(index, interval, MaxInvs)
	}
	//等待所有 goroutin 结束
	m.wait(len(intervals))
	//返回数据
	keys := m.sortKeys()
	for _, k := range keys {
		blocks.Items = append(blocks.Items, m.tempdata[k])
	}

	//作为事件，发送给blockchain,事件是 EventAddBlocks
	newmsg := m.network.node.nodeInfo.qclient.NewMessage("blockchain", pb.EventAddBlocks, &blocks)
	m.network.node.nodeInfo.qclient.Send(newmsg, false)

}
func (m *msg) loadPeers() {
	m.peers = append(m.peers, m.network.node.GetPeers()...)
}
func (m *msg) wait(thnum int) {
	var count int
	for {
		<-m.msgStatus
		count++
		if count == thnum {
			break
		}
	}
}

type intervalInfo struct {
	start int
	end   int
}

func (m *msg) sortKeys() []int64 {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	var keys []int64
	for k, _ := range m.tempdata {
		keys = append(keys, k)
	}
	return keys

}
func (m *msg) downloadBlock(index int, interval *intervalInfo, invs *pb.P2PInv) {
	peersize := m.network.node.Size()
	for i := 0; i < peersize; i++ {
		index = index % peersize
		invdatas, err := m.peers[index].mconn.conn.GetData(context.Background(), &pb.P2PGetData{Invs: invs.Invs[interval.start:interval.end]})
		if err != nil {
			m.peers[index].mconn.sendMonitor.Update(false)
			index++
			continue
		}
		m.mtx.Lock()
		defer m.mtx.Unlock()
		for _, item := range invdatas.Items {

			m.tempdata[item.GetBlock().GetHeight()] = item.GetBlock()
		}
		break
	}
	m.msgStatus <- true

}
func (m *msg) caculateInterval(invsNum int) map[int]*intervalInfo {
	var result = make(map[int]*intervalInfo)
	peerNum := m.network.node.Size()
	if invsNum < peerNum {
		result[0] = &intervalInfo{start: 0, end: invsNum}
		return result
	}
	var interval = invsNum / peerNum
	var start, end int
	end = interval
	for i := 0; i < peerNum; i++ {
		end += start
		if end > invsNum || i == peerNum-1 {
			end = invsNum
		}
		result[i] = &intervalInfo{start: start, end: end}
		start += end
	}

	return result

}
func (m *msg) BlockBroadcast(msg queue.Message) {
	log.Debug("BlockBroadcast", "SendTOP2P", msg.GetData())
	if m.network.node.Size() == 0 {
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	block := msg.GetData().(*pb.Block)
	peers := m.network.node.GetPeers()
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
