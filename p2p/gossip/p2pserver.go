// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gossip

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/p2p/utils"

	"github.com/33cn/chain33/common/version"
	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"

	pr "google.golang.org/grpc/peer"
)

// P2pserver object information
type P2pserver struct {
	imtx         sync.Mutex //for inboundpeers
	smtx         sync.Mutex
	node         *Node
	streams      map[string]chan interface{}
	inboundpeers map[string]*innerpeer
	closed       int32
}
type innerpeer struct {
	addr        string
	name        string
	timestamp   int64
	softversion string
	p2pversion  int32
}

// Start p2pserver start
func (s *P2pserver) Start() {
	s.manageStream()
}

// Close p2pserver close
func (s *P2pserver) Close() {
	atomic.StoreInt32(&s.closed, 1)
}

// IsClose is p2pserver running
func (s *P2pserver) IsClose() bool {
	return atomic.LoadInt32(&s.closed) == 1
}

// NewP2pServer produce a p2pserver
func NewP2pServer() *P2pserver {
	return &P2pserver{
		streams:      make(map[string]chan interface{}),
		inboundpeers: make(map[string]*innerpeer),
	}

}

// Ping p2pserver ping
func (s *P2pserver) Ping(ctx context.Context, in *pb.P2PPing) (*pb.P2PPong, error) {

	log.Debug("ping")
	if !P2pComm.CheckSign(in) {
		log.Error("Ping", "p2p server", "check sig err")
		return nil, pb.ErrPing
	}

	peerIP, _, err := resolveClientNetAddr(ctx)
	if err != nil {
		log.Error("Ping", "get grpc peer addr err", err)
		return nil, fmt.Errorf("get grpc peer addr err:%s", err.Error())
	}

	peeraddr := fmt.Sprintf("%s:%v", peerIP, in.Port)
	remoteNetwork, err := NewNetAddressString(peeraddr)
	if err == nil {
		if !s.node.nodeInfo.blacklist.Has(peeraddr) {
			s.node.nodeInfo.addrBook.AddAddress(remoteNetwork, nil)
		}

	}
	log.Debug("Send Pong", "Nonce", in.GetNonce())
	return &pb.P2PPong{Nonce: in.GetNonce()}, nil

}

// GetAddr get address
func (s *P2pserver) GetAddr(ctx context.Context, in *pb.P2PGetAddr) (*pb.P2PAddr, error) {

	log.Debug("GETADDR", "RECV ADDR", in, "OutBound Len", s.node.Size())
	var addrlist []string
	peers, _ := s.node.GetActivePeers()
	log.Debug("GetAddr", "GetPeers", peers)
	for _, peer := range peers {
		addrlist = append(addrlist, peer.Addr())
	}
	return &pb.P2PAddr{Nonce: in.Nonce, Addrlist: addrlist}, nil
}

// GetAddrList get address list , and height of address
func (s *P2pserver) GetAddrList(ctx context.Context, in *pb.P2PGetAddr) (*pb.P2PAddrList, error) {
	_, infos := s.node.GetActivePeers()
	var peerinfos []*pb.P2PPeerInfo

	for _, info := range infos {

		peerinfos = append(peerinfos, &pb.P2PPeerInfo{Addr: info.GetAddr(), Port: info.GetPort(), Name: info.GetName(), Header: info.GetHeader(),
			MempoolSize: info.GetMempoolSize()})
	}

	return &pb.P2PAddrList{Nonce: in.Nonce, Peerinfo: peerinfos}, nil
}

// Version version
func (s *P2pserver) Version(ctx context.Context, in *pb.P2PVersion) (*pb.P2PVerAck, error) {
	return &pb.P2PVerAck{Version: s.node.nodeInfo.channelVersion, Service: 6, Nonce: in.Nonce}, nil
}

// Version2 p2pserver version
func (s *P2pserver) Version2(ctx context.Context, in *pb.P2PVersion) (*pb.P2PVersion, error) {

	channel, ver := utils.DecodeChannelVersion(in.GetVersion())
	log.Debug("p2pServer Version2", "p2pChannel", channel, "p2p version", ver)

	if !s.node.verifyP2PChannel(channel) {
		return nil, pb.ErrP2PChannel
	}

	log.Debug("Version2", "before", "GetPrivPubKey")
	_, pub := s.node.nodeInfo.addrBook.GetPrivPubKey()
	log.Debug("Version2", "after", "GetPrivPubKey")
	peerIP, _, err := resolveClientNetAddr(ctx)
	if err != nil {
		log.Error("Version2", "get grpc peer addr err", err)
		return nil, fmt.Errorf("get grpc peer addr err:%s", err.Error())
	}
	//addrFrom:表示发送方外网地址，addrRecv:表示接收方外网地址
	_, port, err := net.SplitHostPort(in.AddrFrom)
	if err != nil {
		return nil, fmt.Errorf("AddrFrom format err")
	}

	peerAddr := fmt.Sprintf("%v:%v", peerIP, port)

	remoteNetwork, err := NewNetAddressString(peerAddr)
	if err == nil {
		if !s.node.nodeInfo.blacklist.Has(remoteNetwork.String()) {
			s.node.nodeInfo.addrBook.AddAddress(remoteNetwork, nil)
		}
	}

	return &pb.P2PVersion{Version: s.node.nodeInfo.channelVersion,
		Service: int64(s.node.nodeInfo.ServiceTy()), Nonce: in.Nonce,
		AddrFrom: in.AddrRecv, AddrRecv: fmt.Sprintf("%v:%v", peerIP, port), UserAgent: pub}, nil
}

// SoftVersion software version
func (s *P2pserver) SoftVersion(ctx context.Context, in *pb.P2PPing) (*pb.Reply, error) {

	if !P2pComm.CheckSign(in) {
		log.Error("Ping", "p2p server", "check sig err")
		return nil, pb.ErrPing
	}
	ver := version.GetVersion()
	return &pb.Reply{IsOk: true, Msg: []byte(ver)}, nil

}

// BroadCastTx broadcast transactions of p2pserver
func (s *P2pserver) BroadCastTx(ctx context.Context, in *pb.P2PTx) (*pb.Reply, error) {
	log.Debug("p2pServer RECV TRANSACTION", "in", in)

	client := s.node.nodeInfo.client
	msg := client.NewMessage("mempool", pb.EventTx, in.Tx)
	err := client.Send(msg, false)
	if err != nil {
		return nil, err
	}
	return &pb.Reply{IsOk: true, Msg: []byte("ok")}, nil
}

// GetBlocks get blocks of p2pserver
func (s *P2pserver) GetBlocks(ctx context.Context, in *pb.P2PGetBlocks) (*pb.P2PInv, error) {

	channel, ver := utils.DecodeChannelVersion(in.GetVersion())
	log.Debug("p2pServer GetBlocks", "p2pChannel", channel, "p2p version", ver)
	if !s.node.verifyP2PChannel(channel) {
		return nil, pb.ErrP2PChannel
	}

	client := s.node.nodeInfo.client
	msg := client.NewMessage("blockchain", pb.EventGetHeaders, &pb.ReqBlocks{Start: in.StartHeight, End: in.EndHeight,
		IsDetail: false})
	err := client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("GetBlocks", "Error", err.Error())
		return nil, err
	}
	resp, err := client.WaitTimeout(msg, time.Minute)
	if err != nil {
		return nil, err
	}
	headers := resp.Data.(*pb.Headers)
	var invs = make([]*pb.Inventory, 0)
	for _, item := range headers.Items {
		var inv pb.Inventory
		inv.Ty = msgBlock
		inv.Height = item.GetHeight()
		invs = append(invs, &inv)
	}
	return &pb.P2PInv{Invs: invs}, nil
}

// GetMemPool p2pserver queries the local mempool
func (s *P2pserver) GetMemPool(ctx context.Context, in *pb.P2PGetMempool) (*pb.P2PInv, error) {
	channel, ver := utils.DecodeChannelVersion(in.GetVersion())
	log.Debug("p2pServer GetMemPool", "p2pChannel", channel, "p2p version", ver)
	if !s.node.verifyP2PChannel(channel) {
		return nil, pb.ErrP2PChannel
	}

	memtx, err := s.loadMempool()
	if err != nil {
		return nil, err
	}

	var invlist = make([]*pb.Inventory, 0)
	for _, tx := range memtx {
		invlist = append(invlist, &pb.Inventory{Hash: tx.Hash(), Ty: msgTx})
	}

	return &pb.P2PInv{Invs: invlist}, nil
}

// GetData get data of p2pserver
func (s *P2pserver) GetData(in *pb.P2PGetData, stream pb.P2Pgservice_GetDataServer) error {

	channel, ver := utils.DecodeChannelVersion(in.GetVersion())
	log.Debug("p2pServer Recv GetDataTx", "p2pChannel", channel, "p2p version", ver)
	if !s.node.verifyP2PChannel(channel) {
		return pb.ErrP2PChannel
	}
	var p2pInvData = make([]*pb.InvData, 0)
	var count = 0

	invs := in.GetInvs()
	client := s.node.nodeInfo.client
	for _, inv := range invs { //过滤掉不需要的数据
		var invdata pb.InvData
		var memtx = make(map[string]*pb.Transaction)
		if inv.GetTy() == msgTx {
			//loadMempool
			if count == 0 {
				var err error
				memtx, err = s.loadMempool()
				if err != nil {
					continue
				}
			}
			count++
			txhash := hex.EncodeToString(inv.GetHash())
			if tx, ok := memtx[txhash]; ok {
				invdata.Value = &pb.InvData_Tx{Tx: tx}
				invdata.Ty = msgTx
				p2pInvData = append(p2pInvData, &invdata)
			}

		} else if inv.GetTy() == msgBlock {
			height := inv.GetHeight()
			reqblock := &pb.ReqBlocks{Start: height, End: height}
			msg := client.NewMessage("blockchain", pb.EventGetBlocks, reqblock)
			err := client.Send(msg, true)
			if err != nil {
				log.Error("GetBlocks", "Error", err.Error())
				return err //blockchain 模块关闭，直接返回，不需要continue
			}
			resp, err := client.WaitTimeout(msg, time.Second*20)
			if err != nil {
				log.Error("GetBlocks Err", "Err", err.Error())
				return err
			}

			blocks := resp.Data.(*pb.BlockDetails)
			for _, item := range blocks.Items {
				invdata.Ty = msgBlock
				invdata.Value = &pb.InvData_Block{Block: item.Block}
				p2pInvData = append(p2pInvData, &invdata)
			}

		}
	}

	var counts int
	for _, invdata := range p2pInvData {
		counts++
		var InvDatas []*pb.InvData
		InvDatas = append(InvDatas, invdata)
		err := stream.Send(&pb.InvDatas{Items: InvDatas})
		if err != nil {
			log.Error("sendBlock", "err", err.Error())
			return err
		}
	}
	log.Debug("sendblock", "count", counts, "invs", len(invs))

	return nil

}

// GetHeaders ger headers of p2pServer
func (s *P2pserver) GetHeaders(ctx context.Context, in *pb.P2PGetHeaders) (*pb.P2PHeaders, error) {

	channel, ver := utils.DecodeChannelVersion(in.GetVersion())
	log.Debug("p2pServer GetHeaders", "p2pChannel", channel, "p2p version", ver)
	if !s.node.verifyP2PChannel(channel) {
		return nil, pb.ErrP2PChannel
	}

	if in.GetEndHeight()-in.GetStartHeight() > 2000 || in.GetEndHeight() < in.GetStartHeight() {
		return nil, fmt.Errorf("out of range")
	}

	client := s.node.nodeInfo.client
	msg := client.NewMessage("blockchain", pb.EventGetHeaders, &pb.ReqBlocks{Start: in.GetStartHeight(), End: in.GetEndHeight()})
	err := client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("GetHeaders", "Error", err.Error())
		return nil, err
	}
	resp, err := client.WaitTimeout(msg, time.Minute)
	if err != nil {
		return nil, err
	}

	headers := resp.GetData().(*pb.Headers)

	return &pb.P2PHeaders{Headers: headers.GetItems()}, nil
}

// GetPeerInfo get peer information of p2pServer
func (s *P2pserver) GetPeerInfo(ctx context.Context, in *pb.P2PGetPeerInfo) (*pb.P2PPeerInfo, error) {
	channel, ver := utils.DecodeChannelVersion(in.GetVersion())
	log.Debug("p2pServer GetPeerInfo", "p2pChannel", channel, "p2p version", ver)
	if !s.node.verifyP2PChannel(channel) {
		return nil, pb.ErrP2PChannel
	}

	client := s.node.nodeInfo.client
	log.Debug("GetPeerInfo", "GetMempoolSize", "befor")
	msg := client.NewMessage("mempool", pb.EventGetMempoolSize, nil)
	err := client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("GetPeerInfo mempool", "Error", err.Error())
		return nil, err
	}
	log.Debug("GetPeerInfo", "GetMempoolSize", "after")
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}

	meminfo := resp.GetData().(*pb.MempoolSize)
	var peerinfo pb.P2PPeerInfo

	_, pub := s.node.nodeInfo.addrBook.GetPrivPubKey()

	log.Debug("GetPeerInfo", "EventGetLastHeader", "befor")
	//get header
	msg = client.NewMessage("blockchain", pb.EventGetLastHeader, nil)
	err = client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("GetPeerInfo blockchain", "Error", err.Error())
		return nil, err
	}
	resp, err = client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	log.Debug("GetPeerInfo", "EventGetLastHeader", "after")
	header := resp.GetData().(*pb.Header)

	peerinfo.Header = header
	peerinfo.Name = pub
	peerinfo.MempoolSize = int32(meminfo.GetSize())
	peerinfo.Addr = s.node.nodeInfo.GetExternalAddr().IP.String()
	peerinfo.Port = int32(s.node.nodeInfo.GetExternalAddr().Port)
	return &peerinfo, nil
}

// BroadCastBlock broadcast block of p2pserver
func (s *P2pserver) BroadCastBlock(ctx context.Context, in *pb.P2PBlock) (*pb.Reply, error) {
	log.Debug("BroadCastBlock")

	client := s.node.nodeInfo.client
	msg := client.NewMessage("blockchain", pb.EventBroadcastAddBlock, in.GetBlock())
	err := client.Send(msg, false)
	if err != nil {
		log.Error("BroadCastBlock", "Error", err.Error())
		return nil, err
	}
	return &pb.Reply{IsOk: true, Msg: []byte("ok")}, nil
}

// ServerStreamSend serverstream send of p2pserver
func (s *P2pserver) ServerStreamSend(in *pb.P2PPing, stream pb.P2Pgservice_ServerStreamSendServer) error {
	if len(s.getInBoundPeers()) > int(s.node.nodeInfo.cfg.InnerBounds) {
		return fmt.Errorf("beyound max inbound num")
	}

	peerIP, _, err := resolveClientNetAddr(stream.Context())
	if err != nil {
		log.Error("ServerStreamSend", "get grpc peer addr err", err)
		return fmt.Errorf("get grpc peer addr err:%s", err.Error())
	}
	peerAddr := fmt.Sprintf("%s:%v", peerIP, in.GetPort())
	//等待ReadStream接收节点version信息
	var peerInfo *innerpeer
	var reTry int32
	peerName := hex.EncodeToString(in.GetSign().GetPubkey())
	//此处不能用IP:Port 作为key,因为存在内网多个节点共享一个IP的可能,用peerName 不会有这个问题
	for ; peerInfo == nil || peerInfo.p2pversion == 0; peerInfo = s.getInBoundPeerInfo(peerName) {
		time.Sleep(time.Second)
		reTry++
		if reTry > 5 { //如果一直不跳出循环，这个goroutine 一直存在，潜在的风险点
			return fmt.Errorf("can not find peer:%v", peerAddr)
		}
	}
	log.Debug("ServerStreamSend")

	dataChain := s.addStreamHandler(peerName)
	defer s.deleteStream(peerName, dataChain)
	for data := range dataChain {
		if s.IsClose() {
			return fmt.Errorf("node close")
		}
		sendData, doSend := s.node.processSendP2P(data, peerInfo.p2pversion, peerName, peerInfo.addr)
		if !doSend {
			continue
		}
		err := stream.Send(sendData)
		if err != nil {
			return err
		}
	}
	return nil
}

// ServerStreamRead server stream read of p2pserver
func (s *P2pserver) ServerStreamRead(stream pb.P2Pgservice_ServerStreamReadServer) error {
	if len(s.getInBoundPeers()) > int(s.node.nodeInfo.cfg.InnerBounds) {
		return fmt.Errorf("beyound max inbound num:%v>%v", len(s.getInBoundPeers()), int(s.node.nodeInfo.cfg.InnerBounds))
	}
	log.Debug("StreamRead")
	peerIP, _, err := resolveClientNetAddr(stream.Context())
	if err != nil {
		log.Error("ServerStreamRead", "get grpc peer addr err", err)
		return fmt.Errorf("get grpc peer addr err:%s", err.Error())
	}

	var peeraddr, peername string
	//此处delete是defer调用, 提前绑定变量,需要传入指针, peeraddr的值才能被获取
	defer s.deleteInBoundPeerInfo(&peername)
	defer stream.SendAndClose(&pb.ReqNil{})

	for {
		if s.IsClose() {
			return fmt.Errorf("node close")
		}
		in, err := stream.Recv()
		if err != nil {
			log.Error("ServerStreamRead", "Recv", err)
			return err
		}

		if s.node.processRecvP2P(in, peername, s.pubToStream, peeraddr) {

		} else if ver := in.GetVersion(); ver != nil {
			//接收版本信息
			peername = ver.GetPeername()
			softversion := ver.GetSoftversion()
			innerpeer := s.getInBoundPeerInfo(peername)
			channel, p2pVersion := utils.DecodeChannelVersion(ver.GetP2Pversion())
			if !s.node.verifyP2PChannel(channel) {
				return pb.ErrP2PChannel
			}
			if innerpeer != nil {
				//这里如果直接修改原值, 可能data race
				info := *innerpeer
				info.p2pversion = p2pVersion
				info.softversion = softversion
				s.addInBoundPeerInfo(peername, info)
			} else {
				//没有获取到peer 的信息，说明没有获取ping的消息包
				return pb.ErrStreamPing
			}

		} else if ping := in.GetPing(); ping != nil { ///被远程节点初次连接后，会收到ping 数据包，收到后注册到inboundpeers.
			//Ping package
			if !P2pComm.CheckSign(ping) {
				log.Error("ServerStreamRead", "check stream", "check sig err")
				return pb.ErrStreamPing
			}

			if s.node.Size() > 0 {

				if peerIP != s.node.nodeInfo.GetListenAddr().IP.String() && peerIP != s.node.nodeInfo.GetExternalAddr().IP.String() {
					s.node.nodeInfo.SetServiceTy(Service)
				}
			}
			peername = hex.EncodeToString(ping.GetSign().GetPubkey())
			peeraddr = fmt.Sprintf("%s:%v", peerIP, ping.GetPort())
			s.addInBoundPeerInfo(peername, innerpeer{addr: peeraddr, name: peername, timestamp: pb.Now().Unix()})
		}
	}
}

// CollectInPeers collect external network nodes of connect their own
func (s *P2pserver) CollectInPeers(ctx context.Context, in *pb.P2PPing) (*pb.PeerList, error) {
	log.Debug("CollectInPeers")
	if !P2pComm.CheckSign(in) {
		log.Debug("CollectInPeers", "ping", "signatrue err")
		return nil, pb.ErrPing
	}
	inPeers := s.getInBoundPeers()
	var p2pPeers []*pb.Peer
	for _, inpeer := range inPeers {
		ip, portstr, err := net.SplitHostPort(inpeer.addr)
		if err != nil {
			continue
		}
		port, err := strconv.Atoi(portstr)
		if err != nil {
			continue
		}

		p2pPeers = append(p2pPeers, &pb.Peer{Name: inpeer.name, Addr: ip, Port: int32(port)}) ///仅用name,addr,port字段，用于统计peer num.
	}
	return &pb.PeerList{Peers: p2pPeers}, nil
}

// CollectInPeers2 collect external network nodes of connect their own
func (s *P2pserver) CollectInPeers2(ctx context.Context, in *pb.P2PPing) (*pb.PeersReply, error) {
	log.Debug("CollectInPeers2")
	if !P2pComm.CheckSign(in) {
		log.Debug("CollectInPeers", "ping", "signatrue err")
		return nil, pb.ErrPing
	}
	inPeers := s.getInBoundPeers()
	var p2pPeers []*pb.PeersInfo
	for _, inpeer := range inPeers {
		ip, portstr, err := net.SplitHostPort(inpeer.addr)
		if err != nil {
			continue
		}
		port, err := strconv.Atoi(portstr)
		if err != nil {
			continue
		}

		p2pPeers = append(p2pPeers, &pb.PeersInfo{Name: inpeer.name, Ip: ip, Port: int32(port),
			Softversion: inpeer.softversion, P2Pversion: inpeer.p2pversion}) ///仅用name,addr,port字段，用于统计peer num.
	}

	return &pb.PeersReply{Peers: p2pPeers}, nil
}

func (s *P2pserver) loadMempool() (map[string]*pb.Transaction, error) {

	var txmap = make(map[string]*pb.Transaction)
	client := s.node.nodeInfo.client
	msg := client.NewMessage("mempool", pb.EventGetMempool, nil)
	err := client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("loadMempool", "Error", err.Error())
		return txmap, err
	}
	resp, err := client.WaitTimeout(msg, time.Minute)
	if err != nil {
		return txmap, err
	}

	txlist := resp.GetData().(*pb.ReplyTxList)
	txs := txlist.GetTxs()

	for _, tx := range txs {
		txmap[hex.EncodeToString(tx.Hash())] = tx
	}
	return txmap, nil
}

func (s *P2pserver) manageStream() {

	go func() { //发送空的block stream ping
		ticker := time.NewTicker(StreamPingTimeout)
		defer ticker.Stop()
		for {
			if s.IsClose() {
				return
			}
			<-ticker.C
			s.pubToAllStream(&pb.P2PPing{})
		}
	}()
	go func() {
		fifoChan := s.node.pubsub.Sub("block", "tx")
		for data := range fifoChan {
			if s.IsClose() {
				return
			}
			s.pubToAllStream(data)
		}
		log.Info("p2pserver", "manageStream", "close")
	}()
}

func (s *P2pserver) addStreamHandler(peerName string) chan interface{} {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	if dataChan, ok := s.streams[peerName]; ok {
		//一个节点对应一个流, 重复打开两个流, 关闭老的数据管道
		close(dataChan)
	}

	s.streams[peerName] = make(chan interface{}, 1024)
	return s.streams[peerName]
}

//发布数据到所有服务流
func (s *P2pserver) pubToAllStream(data interface{}) {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for _, dataChan := range s.streams {
		select {
		case dataChan <- data:

		case <-ticker.C:
			continue
		}
	}
}

//发布数据到指定流
func (s *P2pserver) pubToStream(data interface{}, peerName string) {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
	if dataChan, ok := s.streams[peerName]; ok {
		select {
		case dataChan <- data:

		case <-ticker.C:
			return
		}
	}
}

func (s *P2pserver) deleteStream(peerName string, delChan chan interface{}) {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	if dataChan, ok := s.streams[peerName]; ok && dataChan == delChan {
		close(s.streams[peerName])
		delete(s.streams, peerName)
	}
}

func (s *P2pserver) addInBoundPeerInfo(peerName string, info innerpeer) {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	s.inboundpeers[peerName] = &info
}

func (s *P2pserver) deleteInBoundPeerInfo(peerName *string) {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	delete(s.inboundpeers, *peerName)
}

func (s *P2pserver) getInBoundPeerInfo(peerName string) *innerpeer {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	if key, ok := s.inboundpeers[peerName]; ok {
		return key
	}

	return nil
}

func (s *P2pserver) getInBoundPeers() []*innerpeer {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	var peers []*innerpeer
	for _, innerpeer := range s.inboundpeers {
		peers = append(peers, innerpeer)
	}
	return peers
}

func resolveClientNetAddr(ctx context.Context) (host, port string, err error) {

	grpcPeer, ok := pr.FromContext(ctx)
	if ok {
		return net.SplitHostPort(grpcPeer.Addr.String())
	}

	return "", "", fmt.Errorf("get grpc peer from ctx err")
}
