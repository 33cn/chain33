// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"

	"github.com/33cn/chain33/p2p/utils"

	"sync/atomic"
	"time"

	"github.com/33cn/chain33/queue"
	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type p2pEventFunc func(message *queue.Message, taskIndex int64)

// EventInterface p2p subscribe to the event hander interface
type EventInterface interface {
	BroadCastTx(msg *queue.Message, taskindex int64)
	GetMemPool(msg *queue.Message, taskindex int64)
	GetPeerInfo(msg *queue.Message, taskindex int64)
	GetHeaders(msg *queue.Message, taskindex int64)
	GetBlocks(msg *queue.Message, taskindex int64)
	BlockBroadcast(msg *queue.Message, taskindex int64)
	GetNetInfo(msg *queue.Message, taskindex int64)
}

// NormalInterface subscribe to the event hander interface
type NormalInterface interface {
	GetAddr(peer *Peer) ([]string, error)
	SendVersion(peer *Peer, nodeinfo *NodeInfo) (string, error)
	SendPing(peer *Peer, nodeinfo *NodeInfo) error
	GetBlockHeight(nodeinfo *NodeInfo) (int64, error)
	CheckPeerNatOk(addr string, nodeInfo *NodeInfo) bool
	GetAddrList(peer *Peer) (map[string]int64, error)
	GetInPeersNum(peer *Peer) (int, error)
	CheckSelf(addr string, nodeinfo *NodeInfo) bool
}

// Cli p2p client
type Cli struct {
	network *P2p
}

// NewP2PCli produce a p2p client
func NewP2PCli(network *P2p) EventInterface {
	if network == nil {
		return nil
	}
	pcli := &Cli{
		network: network,
	}

	return pcli
}

// NewNormalP2PCli produce a normal client
func NewNormalP2PCli() NormalInterface {
	return &Cli{}
}

// BroadCastTx broadcast transactions
func (m *Cli) BroadCastTx(msg *queue.Message, taskindex int64) {
	defer func() {
		<-m.network.txFactory
		atomic.AddInt32(&m.network.txCapcity, 1)
		log.Debug("BroadCastTx", "task complete:", taskindex)
	}()

	if tx, ok := msg.GetData().(*pb.Transaction); ok {
		txHash := hex.EncodeToString(tx.Hash())
		//此处使用新分配结构，避免重复修改已保存的ttl
		route := &pb.P2PRoute{TTL: 1}
		//是否已存在记录，不存在表示本节点发起的交易
		data, exist := txHashFilter.Get(txHash)
		if ttl, ok := data.(*pb.P2PRoute); exist && ok {
			route.TTL = ttl.GetTTL() + 1
		} else {
			txHashFilter.Add(txHash, true)
		}
		m.network.node.pubsub.FIFOPub(&pb.P2PTx{Tx: tx, Route: route}, "tx")
		msg.Reply(m.network.client.NewMessage("mempool", pb.EventReply, pb.Reply{IsOk: true, Msg: []byte("ok")}))
	}
}

// GetMemPool get mempool contents
func (m *Cli) GetMemPool(msg *queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetMemPool", "task complete:", taskindex)
	}()
	var Txs = make([]*pb.Transaction, 0)
	var ableInv = make([]*pb.Inventory, 0)
	peers, _ := m.network.node.GetActivePeers()

	for _, peer := range peers {
		//获取远程 peer invs
		resp, err := peer.mconn.gcli.GetMemPool(context.Background(),
			&pb.P2PGetMempool{Version: m.network.node.nodeInfo.channelVersion}, grpc.FailFast(true))
		P2pComm.CollectPeerStat(err, peer)
		if err != nil {
			if err == pb.ErrVersion {
				peer.version.SetSupport(false)
				P2pComm.CollectPeerStat(err, peer)
			}
			continue
		}

		invs := resp.GetInvs()
		//与本地mempool 对比 tx数组
		tmpMsg := m.network.client.NewMessage("mempool", pb.EventGetMempool, nil)
		txresp, err := m.network.client.Wait(tmpMsg)
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
		datacli, dataerr := peer.mconn.gcli.GetData(context.Background(),
			&pb.P2PGetData{Invs: ableInv, Version: m.network.node.nodeInfo.channelVersion}, grpc.FailFast(true))
		P2pComm.CollectPeerStat(dataerr, peer)
		if dataerr != nil {
			continue
		}

		invdatas, recerr := datacli.Recv()
		if recerr != nil && recerr != io.EOF {
			log.Error("GetMemPool", "err", recerr.Error())
			err = datacli.CloseSend()
			if err != nil {
				log.Error("datacli", "close err", err)
			}
			continue
		}

		for _, invdata := range invdatas.Items {
			Txs = append(Txs, invdata.GetTx())
		}
		err = datacli.CloseSend()
		if err != nil {
			log.Error("datacli", "CloseSend err", err)
		}
		break
	}
	msg.Reply(m.network.client.NewMessage("mempool", pb.EventReplyTxList, &pb.ReplyTxList{Txs: Txs}))
}

// GetAddr get address list
func (m *Cli) GetAddr(peer *Peer) ([]string, error) {

	resp, err := peer.mconn.gcli.GetAddr(context.Background(), &pb.P2PGetAddr{Nonce: int64(rand.Int31n(102040))},
		grpc.FailFast(true))
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return nil, err
	}

	log.Debug("GetAddr Resp", "Resp", resp, "addrlist", resp.Addrlist)
	return resp.Addrlist, nil
}

// GetInPeersNum return normal number of peers
func (m *Cli) GetInPeersNum(peer *Peer) (int, error) {
	ping, err := P2pComm.NewPingData(peer.node.nodeInfo)
	if err != nil {
		return 0, err
	}

	resp, err := peer.mconn.gcli.CollectInPeers(context.Background(), ping,
		grpc.FailFast(true))

	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return 0, err
	}

	return len(resp.GetPeers()), nil
}

// GetAddrList return a map for address-prot
func (m *Cli) GetAddrList(peer *Peer) (map[string]int64, error) {

	var addrlist = make(map[string]int64)
	if peer == nil {
		return addrlist, fmt.Errorf("pointer is nil")
	}
	resp, err := peer.mconn.gcli.GetAddrList(context.Background(), &pb.P2PGetAddr{Nonce: int64(rand.Int31n(102040))},
		grpc.FailFast(true))

	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return addrlist, err
	}
	//获取本地高度
	client := peer.node.nodeInfo.client
	msg := client.NewMessage("blockchain", pb.EventGetLastHeader, nil)
	err = client.SendTimeout(msg, true, time.Second*10)
	if err != nil {
		log.Error("getLocalPeerInfo blockchain", "Error", err.Error())
		return addrlist, err
	}
	respmsg, err := client.WaitTimeout(msg, time.Second*30)
	if err != nil {
		return addrlist, err
	}

	localBlockHeight := respmsg.GetData().(*pb.Header).GetHeight()
	peerinfos := resp.GetPeerinfo()

	for _, peerinfo := range peerinfos {
		if localBlockHeight-peerinfo.GetHeader().GetHeight() < 2048 {

			addrlist[fmt.Sprintf("%v:%v", peerinfo.GetAddr(), peerinfo.GetPort())] = peerinfo.GetHeader().GetHeight()
		}
	}
	return addrlist, nil
}

// SendVersion send version
func (m *Cli) SendVersion(peer *Peer, nodeinfo *NodeInfo) (string, error) {
	client := nodeinfo.client
	msg := client.NewMessage("blockchain", pb.EventGetBlockHeight, nil)
	err := client.SendTimeout(msg, true, time.Second*10)
	if err != nil {
		log.Error("SendVesion", "Error", err.Error())
		return "", err
	}
	rsp, err := client.WaitTimeout(msg, time.Second*20)
	if err != nil {
		log.Error("GetHeight", "Error", err.Error())
		return "", err
	}

	blockheight := rsp.GetData().(*pb.ReplyBlockHeight).GetHeight()
	randNonce := rand.Int31n(102040)
	p2pPrivKey, _ := nodeinfo.addrBook.GetPrivPubKey()
	in, err := P2pComm.Signature(p2pPrivKey,
		&pb.P2PPing{Nonce: int64(randNonce), Addr: nodeinfo.GetExternalAddr().IP.String(),
			Port: int32(nodeinfo.GetExternalAddr().Port)})
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return "", err
	}
	addrfrom := nodeinfo.GetExternalAddr().String()

	resp, err := peer.mconn.gcli.Version2(context.Background(), &pb.P2PVersion{Version: nodeinfo.channelVersion, Service: int64(nodeinfo.ServiceTy()), Timestamp: pb.Now().Unix(),
		AddrRecv: peer.Addr(), AddrFrom: addrfrom, Nonce: int64(rand.Int31n(102040)),
		UserAgent: hex.EncodeToString(in.Sign.GetPubkey()), StartHeight: blockheight}, grpc.FailFast(true))
	log.Debug("SendVersion", "resp", resp, "addrfrom", addrfrom, "sendto", peer.Addr())
	if err != nil {
		log.Error("SendVersion", "Verson", err.Error(), "peer", peer.Addr())
		if err == pb.ErrVersion {
			peer.version.SetSupport(false)
			P2pComm.CollectPeerStat(err, peer)
		}
		return "", err
	}

	P2pComm.CollectPeerStat(err, peer)
	log.Debug("SHOW VERSION BACK", "VersionBack", resp, "peer", peer.Addr())
	_, ver := utils.DecodeChannelVersion(resp.GetVersion())
	peer.version.SetVersion(ver)

	ip, _, err := net.SplitHostPort(resp.GetAddrRecv())
	if err == nil {
		if ip != nodeinfo.GetExternalAddr().IP.String() {

			log.Debug("sendVersion", "externalip", ip)
			if peer.IsPersistent() {
				//永久加入黑名单
				nodeinfo.blacklist.Add(ip, 0)
			}
		}
	}

	if exaddr, err := NewNetAddressString(resp.GetAddrRecv()); err == nil {
		nodeinfo.SetExternalAddr(exaddr)
	}
	return resp.GetUserAgent(), nil
}

// SendPing send ping
func (m *Cli) SendPing(peer *Peer, nodeinfo *NodeInfo) error {
	randNonce := rand.Int31n(102040)
	ping := &pb.P2PPing{Nonce: int64(randNonce), Addr: nodeinfo.GetExternalAddr().IP.String(), Port: int32(nodeinfo.GetExternalAddr().Port)}
	p2pPrivKey, _ := nodeinfo.addrBook.GetPrivPubKey()
	_, err := P2pComm.Signature(p2pPrivKey, ping)
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return err
	}

	r, err := peer.mconn.gcli.Ping(context.Background(), ping, grpc.FailFast(true))
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return err
	}

	log.Debug("SendPing", "Peer", peer.Addr(), "nonce", randNonce, "recv", r.Nonce)
	return nil
}

// GetBlockHeight return block height
func (m *Cli) GetBlockHeight(nodeinfo *NodeInfo) (int64, error) {
	client := nodeinfo.client
	msg := client.NewMessage("blockchain", pb.EventGetLastHeader, nil)
	err := client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("GetBlockHeight", "Error", err.Error())
		return 0, err
	}
	resp, err := client.WaitTimeout(msg, time.Minute)
	if err != nil {
		return 0, err
	}

	header := resp.GetData().(*pb.Header)
	return header.GetHeight(), nil
}

// GetPeerInfo return peer information
func (m *Cli) GetPeerInfo(msg *queue.Message, taskindex int64) {
	defer func() {
		log.Debug("GetPeerInfo", "task complete:", taskindex)
	}()

	peerinfo, err := m.getLocalPeerInfo()
	if err != nil {
		log.Error("GetPeerInfo", "p2p cli Err", err.Error())
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventPeerList, &pb.PeerList{Peers: m.peerInfos()}))
		return
	}

	var peers = m.peerInfos()
	var peer pb.Peer
	peer.Addr = peerinfo.GetAddr()
	peer.Port = peerinfo.GetPort()
	peer.Name = peerinfo.GetName()
	peer.MempoolSize = peerinfo.GetMempoolSize()
	peer.Self = true
	peer.Header = peerinfo.GetHeader()
	peers = append(peers, &peer)
	msg.Reply(m.network.client.NewMessage("blockchain", pb.EventPeerList, &pb.PeerList{Peers: peers}))
}

// GetHeaders get headers information
func (m *Cli) GetHeaders(msg *queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetHeaders", "task complete:", taskindex)
	}()
	if m.network.node.Size() == 0 {
		log.Debug("GetHeaders", "boundNum", 0)
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{Msg: []byte("no peers")}))
		return
	}
	req := msg.GetData().(*pb.ReqBlocks)
	pid := req.GetPid()
	if len(pid) == 0 {
		log.Debug("GetHeaders:pid is nil")
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{IsOk: true, Msg: []byte("ok")}))
	peers, infos := m.network.node.GetActivePeers()

	if peer, ok := peers[pid[0]]; ok && peer != nil {
		var err error
		headers, err := peer.mconn.gcli.GetHeaders(context.Background(), &pb.P2PGetHeaders{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
			Version: m.network.node.nodeInfo.channelVersion}, grpc.FailFast(true))
		P2pComm.CollectPeerStat(err, peer)
		if err != nil {
			log.Error("GetBlocks", "Err", err.Error())
			if err == pb.ErrVersion {
				peer.version.SetSupport(false)
				P2pComm.CollectPeerStat(err, peer) //把no support 消息传递过去
			}
			return
		}

		client := m.network.node.nodeInfo.client
		msg := client.NewMessage("blockchain", pb.EventAddBlockHeaders, &pb.HeadersPid{Pid: pid[0], Headers: &pb.Headers{Items: headers.GetHeaders()}})
		err = client.Send(msg, false)
		if err != nil {
			log.Error("send", "to blockchain EventAddBlockHeaders msg Err", err.Error())
		}
	} else {
		//当请求的pid不是ActivePeer时需要打印日志方便问题定位
		log.Debug("GetHeaders", "pid", pid[0], "ActivePeers", peers, "infos", infos)
	}

}

// GetBlocks get blocks information
func (m *Cli) GetBlocks(msg *queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetBlocks", "task complete:", taskindex)

	}()
	if m.network.node.Size() == 0 {
		log.Debug("GetBlocks", "boundNum", 0)
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{Msg: []byte("no peers")}))
		return
	}

	req := msg.GetData().(*pb.ReqBlocks)
	log.Debug("GetBlocks", "start", req.GetStart(), "end", req.GetEnd())
	pids := req.GetPid()
	log.Debug("GetBlocks", "pids", pids)
	var Inventorys = make([]*pb.Inventory, 0)
	for i := req.GetStart(); i <= req.GetEnd(); i++ {
		var inv pb.Inventory
		inv.Ty = msgBlock
		inv.Height = i
		Inventorys = append(Inventorys, &inv)
	}

	MaxInvs := &pb.P2PInv{Invs: Inventorys}

	var downloadPeers []*Peer
	peers, infos := m.network.node.GetActivePeers()
	if len(pids) > 0 && pids[0] != "" { //指定Pid 下载数据
		log.Debug("fetch from peer in pids", "pids", pids)
		for _, pid := range pids {
			if peer, ok := peers[pid]; ok && peer != nil {
				downloadPeers = append(downloadPeers, peer)
			}
		}

	} else {
		log.Debug("fetch from all peers in pids")
		for name, peer := range peers {
			info, ok := infos[name]
			if !ok || info.GetHeader().GetHeight() < req.GetStart() { //高度不符合要求
				continue
			}
			downloadPeers = append(downloadPeers, peer)
		}
	}

	if len(downloadPeers) == 0 {
		log.Error("GetBlocks", "downloadPeers", 0, "peers", peers, "infos", infos)
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{Msg: []byte("no downloadPeers")}))
		return
	}

	msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{IsOk: true, Msg: []byte("downloading...")}))

	//使用新的下载模式进行下载
	var bChan = make(chan *pb.BlockPid, 512)
	invs := MaxInvs.GetInvs()
	job := NewDownloadJob(m, downloadPeers)
	var jobcancel int32
	go func(cancel *int32, invs []*pb.Inventory) {
		for {
			if atomic.LoadInt32(cancel) == 1 {
				return
			}

			invs = job.DownloadBlock(invs, bChan)
			if len(invs) == 0 {
				return
			}

			if job.avalidPeersNum() <= 0 {
				job.ResetDownloadPeers(downloadPeers)
				continue
			}

			if job.isCancel() {
				return
			}

		}
	}(&jobcancel, invs)

	i := 0
	for {
		if job.isCancel() {
			return
		}
		timeout := time.NewTimer(time.Minute * 10)
		select {
		case <-timeout.C:
			atomic.StoreInt32(&jobcancel, 1)
			job.CancelJob()
			log.Error("download timeout")
			return
		case blockpid := <-bChan:
			newmsg := m.network.node.nodeInfo.client.NewMessage("blockchain", pb.EventSyncBlock, blockpid)
			err := m.network.node.nodeInfo.client.SendTimeout(newmsg, false, 60*time.Second)
			if err != nil {
				log.Error("send", "to blockchain EventSyncBlock msg err", err)
			}
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

// BlockBroadcast block broadcast
func (m *Cli) BlockBroadcast(msg *queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("BlockBroadcast", "task complete:", taskindex)
	}()

	if block, ok := msg.GetData().(*pb.Block); ok {
		pb.AssertConfig(m.network.client)
		blockHashFilter.Add(hex.EncodeToString(block.Hash(m.network.client.GetConfig())), true)
		m.network.node.pubsub.FIFOPub(&pb.P2PBlock{Block: block}, "block")
	}
}

// GetNetInfo get network information
func (m *Cli) GetNetInfo(msg *queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetNetInfo", "task complete:", taskindex)
	}()

	var netinfo pb.NodeNetInfo
	netinfo.Externaladdr = m.network.node.nodeInfo.GetExternalAddr().String()
	netinfo.Localaddr = m.network.node.nodeInfo.GetListenAddr().String()
	netinfo.Service = m.network.node.nodeInfo.IsOutService()
	netinfo.Outbounds = int32(m.network.node.Size())
	netinfo.Inbounds = int32(len(m.network.node.server.p2pserver.getInBoundPeers()))
	msg.Reply(m.network.client.NewMessage("rpc", pb.EventReplyNetInfo, &netinfo))

}

// CheckPeerNatOk check peer is ok or not
func (m *Cli) CheckPeerNatOk(addr string, info *NodeInfo) bool {
	//连接自己的地址信息做测试
	return !(len(P2pComm.AddrRouteble([]string{addr}, info.channelVersion)) == 0)

}

// CheckSelf check addrbook privPubKey
func (m *Cli) CheckSelf(addr string, nodeinfo *NodeInfo) bool {
	netaddr, err := NewNetAddressString(addr)
	if err != nil {
		log.Error("AddrRouteble", "NewNetAddressString", err.Error())
		return false
	}
	conn, err := netaddr.DialTimeout(nodeinfo.channelVersion)
	if err != nil {
		return false
	}
	defer conn.Close()

	cli := pb.NewP2PgserviceClient(conn)
	resp, err := cli.GetPeerInfo(context.Background(),
		&pb.P2PGetPeerInfo{Version: nodeinfo.channelVersion}, grpc.FailFast(true))
	if err != nil {
		return false
	}
	_, selfName := nodeinfo.addrBook.GetPrivPubKey()
	return resp.GetName() == selfName

}
func (m *Cli) peerInfos() []*pb.Peer {
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

func (m *Cli) getLocalPeerInfo() (*pb.P2PPeerInfo, error) {
	client := m.network.client
	msg := client.NewMessage("mempool", pb.EventGetMempoolSize, nil)
	err := client.SendTimeout(msg, true, time.Second*10) //发送超时
	if err != nil {
		log.Error("GetPeerInfo mempool", "Error", err.Error())
		return nil, err
	}

	resp, err := client.WaitTimeout(msg, time.Second*30)
	if err != nil {
		return nil, err
	}

	log.Debug("getLocalPeerInfo", "GetMempoolSize", "after")
	meminfo := resp.GetData().(*pb.MempoolSize)
	var localpeerinfo pb.P2PPeerInfo

	_, pub := m.network.node.nodeInfo.addrBook.GetPrivPubKey()

	log.Debug("getLocalPeerInfo", "EventGetLastHeader", "befor")
	//get header
	msg = client.NewMessage("blockchain", pb.EventGetLastHeader, nil)
	err = client.SendTimeout(msg, true, time.Second*10)
	if err != nil {
		log.Error("getLocalPeerInfo blockchain", "Error", err.Error())
		return nil, err
	}
	resp, err = client.WaitTimeout(msg, time.Second*30)
	if err != nil {
		return nil, err
	}
	log.Debug("getLocalPeerInfo", "EventGetLastHeader", "after")
	header := resp.GetData().(*pb.Header)

	localpeerinfo.Header = header
	localpeerinfo.Name = pub
	localpeerinfo.MempoolSize = int32(meminfo.GetSize())
	if m.network.node.nodeInfo.GetExternalAddr().IP == nil {
		localpeerinfo.Addr = m.network.node.nodeInfo.GetListenAddr().IP.String()
		localpeerinfo.Port = int32(m.network.node.nodeInfo.GetListenAddr().Port)
	} else {
		localpeerinfo.Addr = m.network.node.nodeInfo.GetExternalAddr().IP.String()
		localpeerinfo.Port = int32(m.network.node.nodeInfo.GetExternalAddr().Port)
	}

	return &localpeerinfo, nil
}
