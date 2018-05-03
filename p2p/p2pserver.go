package p2p

import (
	"encoding/hex"
	"io"
	"strconv"
	"sync/atomic"

	"fmt"
	"strings"
	"sync"
	"time"

	pb "gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	pr "google.golang.org/grpc/peer"
)

type P2pServer struct {
	imtx         sync.Mutex //for inboundpeers
	smtx         sync.Mutex
	node         *Node
	streams      map[pb.P2Pgservice_ServerStreamSendServer]chan interface{}
	inboundpeers map[string]*innerpeer
	deleteSChan  chan pb.P2Pgservice_ServerStreamSendServer
	closed       int32
}
type innerpeer struct {
	addr      string
	name      string
	timestamp int64
}

func (s *P2pServer) Start() {
	s.manageStream()
}

func (s *P2pServer) Close() {
	atomic.StoreInt32(&s.closed, 1)
}

func (s *P2pServer) IsClose() bool {
	return atomic.LoadInt32(&s.closed) == 1
}

func NewP2pServer() *P2pServer {
	return &P2pServer{
		streams:      make(map[pb.P2Pgservice_ServerStreamSendServer]chan interface{}),
		deleteSChan:  make(chan pb.P2Pgservice_ServerStreamSendServer, 1024),
		inboundpeers: make(map[string]*innerpeer),
	}

}

func (s *P2pServer) Ping(ctx context.Context, in *pb.P2PPing) (*pb.P2PPong, error) {

	peeraddr := fmt.Sprintf("%s:%v", in.Addr, in.Port)
	if !P2pComm.CheckSign(in) {
		log.Error("Ping", "p2p server", "check sig err")
		return nil, pb.ErrPing
	}

	remoteNetwork, err := NewNetAddressString(fmt.Sprintf("%v:%v", peeraddr, in.GetPort()))
	if err == nil {
		if len(P2pComm.AddrRouteble([]string{remoteNetwork.String()})) == 1 {
			s.node.nodeInfo.addrBook.AddAddress(remoteNetwork)
		}
	}

	log.Debug("Send Pong", "Nonce", in.GetNonce())
	return &pb.P2PPong{Nonce: in.GetNonce()}, nil

}

// 获取地址
func (s *P2pServer) GetAddr(ctx context.Context, in *pb.P2PGetAddr) (*pb.P2PAddr, error) {
	log.Debug("GETADDR", "RECV ADDR", in, "OutBound Len", s.node.Size())
	addrBucket := make(map[string]bool)
	peers, _ := s.node.GetActivePeers()
	log.Debug("GetAddr", "GetPeers", peers)
	for _, peer := range peers {

		if stat := s.node.nodeInfo.addrBook.GetPeerStat(peer.Addr()); stat != nil {
			if stat.GetAttempts() == 0 {
				addrBucket[peer.Addr()] = true
			}
		}

	}
	addrList := s.node.nodeInfo.addrBook.GetAddrs()
	for _, addr := range addrList {

		addrBucket[addr] = true
		if len(addrBucket) > maxAddrListNum { //最多一次性返回256个地址
			break
		}
	}
	var addrlist []string
	for addr := range addrBucket {
		addrlist = append(addrlist, addr)
	}
	return &pb.P2PAddr{Nonce: in.Nonce, Addrlist: addrlist}, nil
}

// 版本
func (s *P2pServer) Version(ctx context.Context, in *pb.P2PVersion) (*pb.P2PVerAck, error) {

	return &pb.P2PVerAck{Version: s.node.nodeInfo.cfg.GetVersion(), Service: 6, Nonce: in.Nonce}, nil
}
func (s *P2pServer) Version2(ctx context.Context, in *pb.P2PVersion) (*pb.P2PVersion, error) {

	getctx, ok := pr.FromContext(ctx)
	var peeraddr string
	if ok {
		peeraddr = strings.Split(getctx.Addr.String(), ":")[0]
		log.Debug("Version2", "Addr", peeraddr)
	}

	if !s.checkVersion(in.GetVersion()) {
		return nil, pb.ErrVersion
	}

	remoteNetwork, err := NewNetAddressString(in.AddrFrom)
	if err == nil {
		if len(P2pComm.AddrRouteble([]string{remoteNetwork.String()})) == 1 {
			s.node.nodeInfo.addrBook.AddAddress(remoteNetwork)
			//broadcast again
			go func() {
				if time.Now().Unix()-in.GetTimestamp() > 5 || s.node.Has(in.AddrFrom) {
					return
				}

				peers, _ := s.node.GetActivePeers()
				for _, peer := range peers {
					peer.mconn.gcli.Version2(context.Background(), in)
				}
			}()
		}

	}
	_, pub := s.node.nodeInfo.addrBook.GetPrivPubKey()
	//addrFrom:表示自己的外网地址，addrRecv:表示对方的外网地址
	return &pb.P2PVersion{Version: s.node.nodeInfo.cfg.GetVersion(), Service: int64(s.node.nodeInfo.ServiceTy()), Nonce: in.Nonce,
		AddrFrom: in.AddrRecv, AddrRecv: fmt.Sprintf("%v:%v", peeraddr, strings.Split(in.AddrFrom, ":")[1]), UserAgent: pub}, nil

}

func (s *P2pServer) BroadCastTx(ctx context.Context, in *pb.P2PTx) (*pb.Reply, error) {
	log.Debug("p2pServer RECV TRANSACTION", "in", in)
	client := s.node.nodeInfo.client
	msg := client.NewMessage("mempool", pb.EventTx, in.Tx)
	client.Send(msg, false)
	return &pb.Reply{IsOk: true, Msg: []byte("ok")}, nil
}

func (s *P2pServer) GetBlocks(ctx context.Context, in *pb.P2PGetBlocks) (*pb.P2PInv, error) {

	log.Debug("p2pServer GetBlocks", "P2P Recv", in)
	if !s.checkVersion(in.GetVersion()) {
		return nil, pb.ErrVersion
	}

	client := s.node.nodeInfo.client
	msg := client.NewMessage("blockchain", pb.EventGetHeaders, &pb.ReqBlocks{Start: in.StartHeight, End: in.EndHeight,
		Isdetail: false})
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

//服务端查询本地mempool
func (s *P2pServer) GetMemPool(ctx context.Context, in *pb.P2PGetMempool) (*pb.P2PInv, error) {
	log.Debug("p2pServer Recv GetMempool", "version", in)
	if !s.checkVersion(in.GetVersion()) {
		return nil, pb.ErrVersion
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

func (s *P2pServer) GetData(in *pb.P2PGetData, stream pb.P2Pgservice_GetDataServer) error {
	log.Debug("p2pServer Recv GetDataTx", "p2p version", in.GetVersion())
	var p2pInvData = make([]*pb.InvData, 0)
	var count = 0
	if !s.checkVersion(in.GetVersion()) {
		return pb.ErrVersion
	}
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
			msg := client.NewMessage("blockchain", pb.EventGetBlocks, &pb.ReqBlocks{height, height, false, []string{""}})
			err := client.Send(msg, true)
			if err != nil {
				log.Error("GetBlocks", "Error", err.Error())
				return err //blockchain 模块关闭，直接返回，不需要continue
			}
			resp, err := client.Wait(msg)
			if err != nil {
				log.Error("GetBlocks Err", "Err", err.Error())
				continue
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

func (s *P2pServer) GetHeaders(ctx context.Context, in *pb.P2PGetHeaders) (*pb.P2PHeaders, error) {
	log.Debug("p2pServer GetHeaders", "p2p version", in.GetVersion())
	if !s.checkVersion(in.GetVersion()) {
		return nil, pb.ErrVersion
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

func (s *P2pServer) GetPeerInfo(ctx context.Context, in *pb.P2PGetPeerInfo) (*pb.P2PPeerInfo, error) {
	log.Debug("p2pServer GetPeerInfo", "p2p version", in.GetVersion())
	if !s.checkVersion(in.GetVersion()) {
		return nil, pb.ErrVersion
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

func (s *P2pServer) BroadCastBlock(ctx context.Context, in *pb.P2PBlock) (*pb.Reply, error) {
	client := s.node.nodeInfo.client
	msg := client.NewMessage("blockchain", pb.EventBroadcastAddBlock, in.GetBlock())
	err := client.Send(msg, false)
	if err != nil {
		log.Error("BroadCastBlock", "Error", err.Error())
		return nil, err
	}
	return &pb.Reply{IsOk: true, Msg: []byte("ok")}, nil
}

func (s *P2pServer) ServerStreamSend(in *pb.P2PPing, stream pb.P2Pgservice_ServerStreamSendServer) error {
	peername := hex.EncodeToString(in.GetSign().GetPubkey())
	dataChain := s.addStreamHandler(stream)
	for data := range dataChain {
		if s.IsClose() {
			return fmt.Errorf("node close")
		}
		p2pdata := new(pb.BroadCastData)
		if block, ok := data.(*pb.P2PBlock); ok {
			if block.GetBlock() != nil {
				log.Debug("ServerStreamSend", "blockhash", hex.EncodeToString(block.GetBlock().GetTxHash()))
			}

			p2pdata.Value = &pb.BroadCastData_Block{Block: block}
		} else if tx, ok := data.(*pb.P2PTx); ok {
			log.Debug("ServerStreamSend", "txhash", hex.EncodeToString(tx.GetTx().Hash()))
			p2pdata.Value = &pb.BroadCastData_Tx{Tx: tx}
		} else {
			log.Error("RoutChate", "Convert error", data)
			continue
		}
		//增加过滤，如果自己连接了远程节点，则不需要通过stream send 重复发送数据给这个节点
		if peerinfo := s.getInBoundPeerInfo(peername); peerinfo != nil {
			if s.node.Has(peerinfo.addr) {
				continue
			}
		}

		err := stream.Send(p2pdata)
		if err != nil {
			s.deleteSChan <- stream
			s.deleteInBoundPeerInfo(peername)
			return err
		}
	}

	return nil
}

func (s *P2pServer) ServerStreamRead(stream pb.P2Pgservice_ServerStreamReadServer) error {
	var hash [64]byte
	var peeraddr, peername string
	defer s.deleteInBoundPeerInfo(peername)
	var in = new(pb.BroadCastData)
	var err error
	for {
		if s.IsClose() {
			return fmt.Errorf("node close")
		}
		in, err = stream.Recv()
		if err == io.EOF {
			log.Info("ServerStreamRead", "Recv", "EOF")
			return err
		}
		if err != nil {
			log.Error("ServerStreamRead", "Recv", err)
			return err
		}

		if block := in.GetBlock(); block != nil {
			hex.Encode(hash[:], block.GetBlock().Hash())
			blockhash := string(hash[:])
			if Filter.QueryRecvData(blockhash) { //已经注册了相同的区块hash，则不会再发送给blockchain
				continue
			}

			log.Info("ServerStreamRead", " Recv block==+=====+=>Height", block.GetBlock().GetHeight(),
				"block size(KB)", len(pb.Encode(block))/1024, "block hash", blockhash)
			if block.GetBlock() != nil {
				//msg := s.node.nodeInfo.client.NewMessage("blockchain", pb.EventBroadcastAddBlock, block.GetBlock())
				msg := s.node.nodeInfo.client.NewMessage("blockchain", pb.EventBroadcastAddBlock, &pb.BlockPid{peername, block.GetBlock()})
				s.node.nodeInfo.client.Send(msg, false)
			}
			Filter.RegRecvData(blockhash) //注册已经收到的区块

		} else if tx := in.GetTx(); tx != nil {
			hex.Encode(hash[:], tx.GetTx().Hash())
			txhash := string(hash[:])
			log.Debug("ServerStreamRead", "txhash:", txhash)
			if Filter.QueryRecvData(txhash) { //同上
				continue
			}
			if tx.GetTx() != nil {
				msg := s.node.nodeInfo.client.NewMessage("mempool", pb.EventTx, tx.GetTx())
				s.node.nodeInfo.client.Send(msg, false)
			}
			Filter.RegRecvData(txhash)

		} else if ping := in.GetPing(); ping != nil { ///被远程节点初次连接后，会收到ping 数据包，收到后注册到inboundpeers.
			//Ping package

			if !P2pComm.CheckSign(ping) {
				log.Error("ServerStreamRead", "check stream", "check sig err")
				return pb.ErrStreamPing
			}

			peername = hex.EncodeToString(ping.GetSign().GetPubkey())
			peeraddr = fmt.Sprintf("%s:%v", in.GetPing().GetAddr(), in.GetPing().GetPort())
			s.addInBoundPeerInfo(peername, innerpeer{addr: peeraddr, name: peername, timestamp: time.Now().Unix()})

		}
	}

}

/**
* 提供远程节点自身网络定位服务
 */
func (s *P2pServer) RemotePeerAddr(ctx context.Context, in *pb.P2PGetAddr) (*pb.P2PExternalInfo, error) {
	var remoteaddr string
	var outside bool
	getctx, ok := pr.FromContext(ctx)
	if ok {
		remoteaddr = strings.Split(getctx.Addr.String(), ":")[0]

		if len(P2pComm.AddrRouteble([]string{fmt.Sprintf("%v:%v", remoteaddr, defaultPort)})) == 0 {

			outside = false
		} else {
			outside = true
		}
	}
	return &pb.P2PExternalInfo{Addr: remoteaddr, Isoutside: outside}, nil
}

/**
* 提供检验自身节点网络映射后服务是否开启的服务
 */
func (s *P2pServer) RemotePeerNatOk(ctx context.Context, in *pb.P2PPing) (*pb.P2PExternalInfo, error) {
	if !P2pComm.CheckSign(in) {
		return nil, pb.ErrPing
	}
	var natok bool
	remoteaddr := fmt.Sprintf("%v:%v", in.GetAddr(), in.GetPort())
	_, pub := s.node.nodeInfo.addrBook.GetPrivPubKey()
	if hex.EncodeToString(in.GetSign().GetPubkey()) == pub {
		//发现是自己节点发来的验证消息
		natok = false
	} else {
		if len(P2pComm.AddrRouteble([]string{remoteaddr})) != 0 {
			natok = true
		}
	}

	return &pb.P2PExternalInfo{remoteaddr, natok}, nil

}

/**
* 统计连接自己的外网节点
 */

func (s *P2pServer) CollectInPeers(ctx context.Context, in *pb.P2PPing) (*pb.PeerList, error) {
	if !P2pComm.CheckSign(in) {
		log.Info("CollectInPeers", "ping", "signatrue err")
		return nil, pb.ErrPing
	}
	inPeers := s.getInBoundPeers()
	var p2pPeers []*pb.Peer
	for _, inpeer := range inPeers {
		addrport := strings.Split(inpeer.addr, ":")
		var addr string
		var port int
		if len(addrport) == 2 {
			addr = addrport[0]
			port, _ = strconv.Atoi(addrport[1])
		}
		p2pPeers = append(p2pPeers, &pb.Peer{Name: inpeer.name, Addr: addr, Port: int32(port)}) ///仅用name,addr,port字段，用于统计peer num.
	}

	return &pb.PeerList{Peers: p2pPeers}, nil
}

func (s *P2pServer) checkVersion(version int32) bool {
	if version < s.node.nodeInfo.cfg.GetVerMix() || version > s.node.nodeInfo.cfg.GetVerMax() {
		//版本不支持
		return false
	}

	return true
}
func (s *P2pServer) loadMempool() (map[string]*pb.Transaction, error) {

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

func (s *P2pServer) manageStream() {
	go s.deleteDisableStream()
	go func() { //发送空的block stream ping
		ticker := time.NewTicker(StreamPingTimeout)
		defer ticker.Stop()
		for {
			if s.IsClose() {
				return
			}
			<-ticker.C
			s.addStreamData(&pb.P2PBlock{})
		}
	}()
	go func() {
		fifoChan := pub.Sub("block", "tx")
		for data := range fifoChan {
			if s.IsClose() {
				return
			}
			s.addStreamData(data)
		}
		log.Info("p2pserver", "manageStream", "close")
	}()
}

func (s *P2pServer) addStreamHandler(stream pb.P2Pgservice_ServerStreamSendServer) chan interface{} {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	s.streams[stream] = make(chan interface{}, 1024)
	return s.streams[stream]

}

func (s *P2pServer) addStreamData(data interface{}) {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	timetikc := time.NewTicker(time.Second * 1)
	defer timetikc.Stop()
	for stream := range s.streams {
		if _, ok := s.streams[stream]; !ok {
			log.Error("AddStreamBLock", "No this Stream", "++++++")
			continue
		}
		select {
		case s.streams[stream] <- data:

		case <-timetikc.C:
			continue
		}

	}

}

func (s *P2pServer) deleteDisableStream() {
	for stream := range s.deleteSChan {
		s.deleteStream(stream)
	}
}
func (s *P2pServer) deleteStream(stream pb.P2Pgservice_ServerStreamSendServer) {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	close(s.streams[stream])
	delete(s.streams, stream)
}

func (s *P2pServer) addInBoundPeerInfo(peername string, info innerpeer) {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	s.inboundpeers[peername] = &info
}

func (s *P2pServer) deleteInBoundPeerInfo(peername string) {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	delete(s.inboundpeers, peername)

}

func (s *P2pServer) getInBoundPeerInfo(peername string) *innerpeer {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	if key, ok := s.inboundpeers[peername]; ok {
		return key
	}

	return nil
}

func (s *P2pServer) getInBoundPeers() []*innerpeer {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	var peers []*innerpeer
	for _, innerpeer := range s.inboundpeers {
		peers = append(peers, innerpeer)
	}
	return peers
}
