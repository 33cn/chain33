package p2p

import (
	"encoding/hex"

	"code.aliyun.com/chain33/chain33/queue"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
)

type msg struct {
	network   *P2p
	msgStatus chan bool
	tempdata  map[interface{}]*pb.Block
	peers     []*peer
}

func NewInTrans(network *P2p) *msg {
	return &msg{
		network: network,
	}
}

func (m *msg) TransToBroadCast(msg *queue.Message) {
	if len(m.network.node.outBound) == 0 {
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	//开始广播消息
	//log.Debug("TransToBroadCast", "start broadcast", data)
	for _, peer := range m.network.node.outBound {
		_, err := peer.mconn.conn.BroadCastTx(context.Background(), msg.GetData().(*pb.P2PTx))
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte(err.Error())}))
			continue
		}
		peer.mconn.sendMonitor.Update(true)
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{true, []byte("ok")}))
	}

}

//收到Mempool 模块获取mempool 的请求
func (m *msg) GetMemPool(msg *queue.Message) {

	var Txs = make([]*pb.Transaction, 0)
	var needhash = make([]*pb.Inventory, 0)
	for _, peer := range m.network.node.outBound {
		//获取远程 peer invs
		resp, err := peer.mconn.conn.GetMemPool(context.Background(), &pb.P2PGetMempool{Version: Version})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			continue
		}
		peer.mconn.sendMonitor.Update(true)
		invs := resp.GetInvs()
		//TODO 与本地mempool 对比 tx数组
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
			//txhash=tx.Hash()
			txmap[hex.EncodeToString(tx.Hash())] = tx
		}
		//check
		for _, inv := range invs {
			if _, ok := txmap[hex.EncodeToString(inv.Hash)]; !ok {

				needhash = append(needhash, inv)
			}
		}
		//TODO 获取真正的交易Tx
		//call GetData
		p2pInvDatas, err := peer.mconn.conn.GetData(context.Background(), &pb.P2PGetData{Invs: needhash, Version: Version})
		if err != nil {
			continue
		}
		for _, invdata := range p2pInvDatas.Items {
			Txs = append(Txs, invdata.GetTx())
		}

	}
	//	pb.ReplyTxList.Txs
	var rtxlist pb.ReplyTxList
	rtxlist.Txs = Txs
	msg.Reply(m.network.c.NewMessage("mempool", pb.EventReplyTxList, rtxlist))

}

//收到BlockChain 模块的请求，获取PeerInfo
func (m *msg) GetPeerInfo(msg *queue.Message) {
	var peerlist = make([]*pb.Peer, 0)
	for _, peer := range m.network.node.outBound {
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

func (m *msg) GetBlocks(msg *queue.Message) {
	if len(m.network.node.outBound) == 0 {
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventAddBlocks, pb.Reply{false, []byte("no peers")}))
		return
	}
	//TODO 第一步获取下载列表，第二步分配不同的节点分段下载需要的数据
	//var blocks = make([]*pb.Block, 0)
	var blocks pb.Blocks
	requst := msg.GetData().(*pb.ReqBlocks)
	var MaxInvs = new(pb.P2PInv)
	m.peers = make([]*peer, 0)
	//获取最大的下载列表
	for _, peer := range m.network.node.outBound {

		invs, err := peer.mconn.conn.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: requst.GetStart(), EndHeight: requst.GetEnd()})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			continue
		}
		peer.mconn.sendMonitor.Update(true)
		if len(invs.Invs) > len(MaxInvs.Invs) {
			MaxInvs = invs
			if len(MaxInvs.GetInvs()) == int(requst.GetEnd()-requst.GetStart())+1 {
				break
			}
		}

	}

	m.LoadPeers()
	intervals := m.CaculateInterval(len(MaxInvs.GetInvs()))
	m.msgStatus = make(chan bool, len(intervals))
	//分段下载
	for k, info := range intervals {
		go m.DownloadBlock(k, info, MaxInvs)
	}
	//等待所有 goroutin 结束
	m.Wait(len(intervals))
	//返回数据
	for _, block := range m.tempdata {
		blocks.Items = append(blocks.Items, block)
	}
	msg.Reply(m.network.c.NewMessage("blockchain", pb.EventBlocks, &blocks))

}
func (m *msg) LoadPeers() {
	for _, peer := range m.network.node.outBound {
		m.peers = append(m.peers, peer)
	}
}
func (m *msg) Wait(thnum int) {
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

func (m *msg) DownloadBlock(index int, interval *intervalInfo, invs *pb.P2PInv) {
	for i := 0; i < len(m.network.node.outBound); i++ {
		index = index % len(m.network.node.outBound)
		invdatas, err := m.peers[index].mconn.conn.GetData(context.Background(), &pb.P2PGetData{Invs: invs.Invs[interval.start:interval.end]})
		if err != nil {
			m.peers[index].mconn.sendMonitor.Update(false)
			index++
			continue
		}
		for _, item := range invdatas.Items {
			m.tempdata[item.GetBlock().Hash()] = item.GetBlock()
		}
		break
	}
	m.msgStatus <- true

}
func (m *msg) CaculateInterval(invsNum int) map[int]*intervalInfo {
	var result = make(map[int]*intervalInfo)
	peerNum := len(m.network.node.outBound)
	if invsNum < peerNum {
		result[0] = &intervalInfo{start: 0, end: invsNum}
		return result
	}
	var interval = invsNum / len(m.network.node.outBound)
	var start, end int
	end = interval
	for i := 0; i < len(m.network.node.outBound); i++ {
		end += start
		if end > invsNum || i == len(m.network.node.outBound)-1 {
			end = invsNum
		}
		result[i] = &intervalInfo{start: start, end: end}
		start += end
	}

	return result

}
func (m *msg) BlockBroadcast(msg *queue.Message) {
	if len(m.network.node.outBound) == 0 {
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	block := msg.GetData().(*pb.Block)
	for _, peer := range m.network.node.outBound {
		resp, err := peer.mconn.conn.BroadCastBlock(context.Background(), &pb.P2PBlock{Block: block})
		if err != nil {
			log.Error("SendBlock", "Error", err.Error())
			msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{false, []byte(err.Error())}))
			continue
		}

		log.Debug("SendBlock", "Resp", resp)
		msg.Reply(m.network.c.NewMessage("mempool", pb.EventReply, pb.Reply{true, []byte("ok")}))
	}
}
