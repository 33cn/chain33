package p2p

import (
	"encoding/hex"

	"code.aliyun.com/chain33/chain33/queue"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
)

type msg struct {
	network *P2p
}

func NewInTrans(network *P2p) *msg {
	return &msg{
		network: network,
	}
}

func (m *msg) TransToBroadCast(msg *queue.Message) {
	//开始广播消息
	//log.Debug("TransToBroadCast", "start broadcast", data)
	for _, peer := range m.network.node.outBound {
		_, err := peer.mconn.conn.BroadCastTx(context.Background(), msg.GetData().(*pb.P2PTx))
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
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
	var blocks = make([]*pb.Block, 0)
	requst := msg.GetData().(*pb.ReqBlocks)
	for _, peer := range m.network.node.outBound {
		invs, err := peer.mconn.conn.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: requst.GetStart(), EndHeight: requst.GetEnd()})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			continue
		}
		peer.mconn.sendMonitor.Update(true)
		invdatas, err := peer.mconn.conn.GetData(context.Background(), &pb.P2PGetData{Invs: invs.GetInvs()})
		if err != nil {
			peer.mconn.sendMonitor.Update(false)
			continue
		}

		peer.mconn.sendMonitor.Update(true)
		for _, item := range invdatas.Items {
			blocks = append(blocks, item.GetBlock())
		}
		msg.Reply(m.network.c.NewMessage("blockchain", pb.EventBlocks, blocks))

	}

}

func (m *msg) BlockBroadcast(msg *queue.Message) {
	block := msg.GetData().(*pb.Block)
	for _, peer := range m.network.node.outBound {
		resp, err := peer.mconn.conn.BroadCastBlock(context.Background(), &pb.P2PBlock{Block: block})
		if err != nil {
			log.Error("SendBlock", "Error", err.Error())
			continue
		}

		log.Debug("SendBlock", "Resp", resp)
	}
}
