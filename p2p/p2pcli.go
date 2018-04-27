package p2p

import (
	//"container/list"
	"encoding/hex"
	"fmt"
	"io"
	//	"sync"

	"math/rand"
	"strconv"
	"strings"

	"sync/atomic"
	"time"

	"gitlab.33.cn/chain33/chain33/queue"
	pb "gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

//p2p 订阅的事件处理函数接口
type EventInterface interface {
	BroadCastTx(msg queue.Message, taskindex int64)
	GetMemPool(msg queue.Message, taskindex int64)
	GetPeerInfo(msg queue.Message, taskindex int64)
	GetHeaders(msg queue.Message, taskindex int64)
	GetBlocks(msg queue.Message, taskindex int64)
	BlockBroadcast(msg queue.Message, taskindex int64)
	GetNetInfo(msg queue.Message, taskindex int64)
}

//非p2p 订阅的事件处理函数接口
type NormalInterface interface {
	GetAddr(peer *Peer) ([]string, error)
	SendVersion(peer *Peer, nodeinfo *NodeInfo) (string, error)
	SendPing(peer *Peer, nodeinfo *NodeInfo) error
	GetBlockHeight(nodeinfo *NodeInfo) (int64, error)
	GetExternIP(addr string) (string, bool, error)
	CheckPeerNatOk(addr string, nodeinfo *NodeInfo) (bool, error)
}

type Cli struct {
	network *P2p
}

func NewP2PCli(network *P2p) EventInterface {
	if network == nil {
		return nil
	}
	pcli := &Cli{
		network: network,
	}

	return pcli
}

func NewNormalP2PCli() NormalInterface {
	return &Cli{}
}
func (m *Cli) BroadCastTx(msg queue.Message, taskindex int64) {
	defer func() {
		<-m.network.txFactory
		atomic.AddInt32(&m.network.txCapcity, 1)
		log.Debug("BroadCastTx", "task complete:", taskindex)
	}()
	pub.FIFOPub(&pb.P2PTx{Tx: msg.GetData().(*pb.Transaction)}, "tx")
	msg.Reply(m.network.client.NewMessage("mempool", pb.EventReply, pb.Reply{true, []byte("ok")}))

}

func (m *Cli) GetMemPool(msg queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetMemPool", "task complete:", taskindex)
	}()
	var Txs = make([]*pb.Transaction, 0)
	var ableInv = make([]*pb.Inventory, 0)
	peers, _ := m.network.node.GetActivePeers()

	for _, peer := range peers {
		//获取远程 peer invs
		resp, err := peer.mconn.gcli.GetMemPool(context.Background(), &pb.P2PGetMempool{Version: m.network.node.nodeInfo.cfg.GetVersion()})
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
		datacli, dataerr := peer.mconn.gcli.GetData(context.Background(), &pb.P2PGetData{Invs: ableInv, Version: m.network.node.nodeInfo.cfg.GetVersion()})
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
	msg.Reply(m.network.client.NewMessage("mempool", pb.EventReplyTxList, &pb.ReplyTxList{Txs: Txs}))
}

func (m *Cli) GetAddr(peer *Peer) ([]string, error) {

	resp, err := peer.mconn.gcli.GetAddr(context.Background(), &pb.P2PGetAddr{Nonce: int64(rand.Int31n(102040))})
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return nil, err
	}

	log.Debug("GetAddr Resp", "Resp", resp, "addrlist", resp.Addrlist)
	return resp.Addrlist, nil
}

func (m *Cli) SendVersion(peer *Peer, nodeinfo *NodeInfo) (string, error) {
	client := nodeinfo.client
	msg := client.NewMessage("blockchain", pb.EventGetBlockHeight, nil)
	err := client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("SendVesion", "Error", err.Error())
		return "", err
	}
	rsp, err := client.WaitTimeout(msg, time.Minute)
	if err != nil {
		log.Error("GetHeight", "Error", err.Error())
		return "", err
	}

	blockheight := rsp.GetData().(*pb.ReplyBlockHeight).GetHeight()
	randNonce := rand.Int31n(102040)
	p2pPrivKey, _ := nodeinfo.addrBook.GetPrivPubKey()
	in, err := P2pComm.Signature(p2pPrivKey, &pb.P2PPing{Nonce: int64(randNonce), Addr: nodeinfo.GetExternalAddr().IP.String(), Port: int32(nodeinfo.GetExternalAddr().Port)})
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return "", err
	}
	addrfrom := nodeinfo.GetExternalAddr().String()

	nodeinfo.blacklist.Add(addrfrom)
	resp, err := peer.mconn.gcli.Version2(context.Background(), &pb.P2PVersion{Version: nodeinfo.cfg.GetVersion(), Service: int64(nodeinfo.ServiceTy()), Timestamp: time.Now().Unix(),
		AddrRecv: peer.Addr(), AddrFrom: addrfrom, Nonce: int64(rand.Int31n(102040)),
		UserAgent: hex.EncodeToString(in.Sign.GetPubkey()), StartHeight: blockheight})
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
	port, err := strconv.Atoi(strings.Split(resp.GetAddrRecv(), ":")[1])
	if err != nil {
		return "", err
	}

	log.Debug("SHOW VERSION BACK", "VersionBack", resp, "peer", peer.Addr())

	if !peer.IsPersistent() {
		return resp.GetUserAgent(), nil //如果不是种子节点，则直接返回，不用校验自身的外网地址
	}

	if strings.Split(resp.GetAddrRecv(), ":")[0] != nodeinfo.GetExternalAddr().IP.String() {
		externalIP := strings.Split(resp.GetAddrRecv(), ":")[0]
		log.Debug("sendVersion", "externalip", externalIP)
		if exaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", externalIP, port)); err == nil {
			nodeinfo.SetExternalAddr(exaddr)
		}

	}

	return resp.GetUserAgent(), nil
}

func (m *Cli) SendPing(peer *Peer, nodeinfo *NodeInfo) error {
	randNonce := rand.Int31n(102040)
	ping := &pb.P2PPing{Nonce: int64(randNonce), Addr: nodeinfo.GetExternalAddr().IP.String(), Port: int32(nodeinfo.GetExternalAddr().Port)}
	p2pPrivKey, _ := nodeinfo.addrBook.GetPrivPubKey()
	_, err := P2pComm.Signature(p2pPrivKey, ping)
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return err
	}

	r, err := peer.mconn.gcli.Ping(context.Background(), ping)
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		return err
	}

	log.Debug("SendPing", "Peer", peer.Addr(), "nonce", randNonce, "recv", r.Nonce)
	return nil
}

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

func (m *Cli) GetPeerInfo(msg queue.Message, taskindex int64) {
	defer func() {
		log.Debug("GetPeerInfo", "task complete:", taskindex)
	}()

	peerinfo, err := m.getLocalPeerInfo()
	if err != nil {
		log.Error("GetPeerInfo", "Err", err.Error())
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

func (m *Cli) GetHeaders(msg queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetHeaders", "task complete:", taskindex)
	}()
	if m.network.node.Size() == 0 {
		log.Debug("GetHeaders", "boundNum", 0)
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	req := msg.GetData().(*pb.ReqBlocks)
	pid := req.GetPid()
	if len(pid) == 0 {
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no pid")}))
		return
	}
	msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{true, []byte("ok")}))
	peers, infos := m.network.node.GetActivePeers()
	for paddr, info := range infos {
		if info.GetName() == pid[0] { //匹配成功
			peer := m.network.node.GetRegisterPeer(paddr)
			peer, ok := peers[paddr]
			if ok && peer != nil {
				var err error

				headers, err := peer.mconn.gcli.GetHeaders(context.Background(), &pb.P2PGetHeaders{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
					Version: m.network.node.nodeInfo.cfg.GetVersion()})
				P2pComm.CollectPeerStat(err, peer)
				if err != nil {
					log.Error("GetBlocks", "Err", err.Error())
					if err == pb.ErrVersion {
						peer.version.SetSupport(false)
						P2pComm.CollectPeerStat(err, peer)
					}
					return
				}

				client := m.network.node.nodeInfo.client
				msg := client.NewMessage("blockchain", pb.EventAddBlockHeaders, &pb.HeadersPid{pid[0], &pb.Headers{headers.GetHeaders()}})
				client.Send(msg, false)
			}
		}
	}
}
func (m *Cli) GetBlocks(msg queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetBlocks", "task complete:", taskindex)

	}()
	if m.network.node.Size() == 0 {
		log.Debug("GetBlocks", "boundNum", 0)
		msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{false, []byte("no peers")}))
		return
	}
	msg.Reply(m.network.client.NewMessage("blockchain", pb.EventReply, pb.Reply{true, []byte("downloading...")}))

	req := msg.GetData().(*pb.ReqBlocks)
	log.Info("GetBlocks", "start", req.GetStart(), "end", req.GetEnd())
	pids := req.GetPid()
	var MaxInvs = new(pb.P2PInv)
	var downloadPeers []*Peer
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
						MaxInvs, err = peer.mconn.gcli.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
							Version: m.network.node.nodeInfo.cfg.GetVersion()})
						P2pComm.CollectPeerStat(err, peer)
						if err != nil {
							log.Error("GetBlocks", "Err", err.Error())
							if err == pb.ErrVersion {
								peer.version.SetSupport(false)
								P2pComm.CollectPeerStat(err, peer)
							}
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

			if peerinfo.GetHeader().GetHeight() < req.GetEnd() {
				continue
			}

			invs, err := peer.mconn.gcli.GetBlocks(context.Background(), &pb.P2PGetBlocks{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
				Version: m.network.node.nodeInfo.cfg.GetVersion()})
			P2pComm.CollectPeerStat(err, peer)
			if err != nil {
				log.Error("GetBlocks", "Err", err.Error())
				if err == pb.ErrVersion {
					peer.version.SetSupport(false)
					P2pComm.CollectPeerStat(err, peer)
				}
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
			peerinfo, ok := infos[peer.Addr()]
			if !ok {
				continue
			}
			if peerinfo.GetHeader().GetHeight() < req.GetStart() { //高度不符合要求
				continue
			}

			downloadPeers = append(downloadPeers, peer)
		}
	}

	log.Debug("Invs", "Invs show", MaxInvs.GetInvs())
	if len(MaxInvs.GetInvs()) == 0 {
		log.Error("GetBlocks", "getInvs", 0)
		return
	}

	//使用新的下载模式进行下载
	var bChan = make(chan *pb.BlockPid, 256)
	invs := MaxInvs.GetInvs()
	job := NewDownloadJob(m)
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
		}
	}(&jobcancel, invs)
	i := 0
	for {
		timeout := time.NewTimer(time.Minute)
		select {
		case <-timeout.C:
			atomic.StoreInt32(&jobcancel, 1)
			job.CancelJob()
			log.Error("download timeout")
			return
		case blockpid := <-bChan:
			newmsg := m.network.node.nodeInfo.client.NewMessage("blockchain", pb.EventSyncBlock, blockpid)
			m.network.node.nodeInfo.client.SendTimeout(newmsg, false, 60*time.Second)
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

func (m *Cli) BlockBroadcast(msg queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("BlockBroadcast", "task complete:", taskindex)
	}()
	pub.FIFOPub(&pb.P2PBlock{Block: msg.GetData().(*pb.Block)}, "block")

}

func (m *Cli) GetNetInfo(msg queue.Message, taskindex int64) {
	defer func() {
		<-m.network.otherFactory
		log.Debug("GetNetInfo", "task complete:", taskindex)
	}()

	var netinfo pb.NodeNetInfo
	netinfo.Externaladdr = m.network.node.nodeInfo.GetExternalAddr().String()
	netinfo.Localaddr = m.network.node.nodeInfo.GetListenAddr().String()
	netinfo.Service = m.network.node.nodeInfo.IsOutService()

	msg.Reply(m.network.client.NewMessage("rpc", pb.EventReplyNetInfo, &netinfo))

}

func (m *Cli) GetExternIP(addr string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		log.Error("grpc DialConn", "err", err.Error())
		return "", false, err
	}
	defer conn.Close()
	gconn := pb.NewP2PgserviceClient(conn)
	resp, err := gconn.RemotePeerAddr(context.Background(), &pb.P2PGetAddr{Nonce: 12})
	if err != nil {
		return "", false, err
	}
	return resp.Addr, resp.Isoutside, nil
}

func (m *Cli) CheckPeerNatOk(addr string, nodeinfo *NodeInfo) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		log.Error("grpc DialConn", "err", err.Error())
		return false, err
	}
	defer conn.Close()
	gconn := pb.NewP2PgserviceClient(conn)
	ping, err := P2pComm.NewPingData(nodeinfo)
	if err != nil {
		return false, err
	}
	resp, err := gconn.RemotePeerNatOk(ctx, ping)
	if err != nil {
		return false, err
	}
	return resp.Isoutside, nil
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
	err := client.SendTimeout(msg, true, time.Minute) //发送超时
	if err != nil {
		log.Error("GetPeerInfo mempool", "Error", err.Error())
		return nil, err
	}
	log.Debug("GetPeerInfo", "GetMempoolSize", "after")

	resp, err := client.WaitTimeout(msg, time.Minute)
	if err != nil {
		return nil, err
	}

	meminfo := resp.GetData().(*pb.MempoolSize)
	var localpeerinfo pb.P2PPeerInfo

	_, pub := m.network.node.nodeInfo.addrBook.GetPrivPubKey()

	log.Debug("getLocalPeerInfo", "EventGetLastHeader", "befor")
	//get header
	msg = client.NewMessage("blockchain", pb.EventGetLastHeader, nil)
	err = client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("getLocalPeerInfo blockchain", "Error", err.Error())
		return nil, err
	}
	resp, err = client.WaitTimeout(msg, time.Minute)
	if err != nil {
		return nil, err
	}
	log.Debug("getLocalPeerInfo", "EventGetLastHeader", "after")
	header := resp.GetData().(*pb.Header)

	localpeerinfo.Header = header
	localpeerinfo.Name = pub
	localpeerinfo.MempoolSize = int32(meminfo.GetSize())
	localpeerinfo.Addr = m.network.node.nodeInfo.GetExternalAddr().IP.String()
	localpeerinfo.Port = int32(m.network.node.nodeInfo.GetExternalAddr().Port)
	return &localpeerinfo, nil
}
