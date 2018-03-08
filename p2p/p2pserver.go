package p2p

import (
	"encoding/hex"
	"io"

	"fmt"
	"strings"
	"sync"
	"time"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	pr "google.golang.org/grpc/peer"
)

type p2pServer struct {
	imtx         sync.Mutex //for inboundpeers
	smtx         sync.Mutex
	node         *Node
	streams      map[pb.P2Pgservice_ServerStreamSendServer]chan interface{}
	inboundpeers map[string]*innerpeer
	deleteSChan  chan pb.P2Pgservice_ServerStreamSendServer
	loopdone     chan struct{}
}
type innerpeer struct {
	addr string
	name string
}

func NewP2pServer() *p2pServer {
	return &p2pServer{
		streams:      make(map[pb.P2Pgservice_ServerStreamSendServer]chan interface{}),
		deleteSChan:  make(chan pb.P2Pgservice_ServerStreamSendServer, 1024),
		inboundpeers: make(map[string]*innerpeer),
		loopdone:     make(chan struct{}, 1),
	}

}

func (s *p2pServer) Ping(ctx context.Context, in *pb.P2PPing) (*pb.P2PPong, error) {

	peeraddr := fmt.Sprintf("%s:%v", in.Addr, in.Port)
	if P2pComm.CheckSign(in) {
		log.Info("Ping", "p2p server", "recv ping")
	}

	remoteNetwork, err := NewNetAddressString(fmt.Sprintf("%v:%v", peeraddr, in.GetPort()))
	if err == nil {
		if len(P2pComm.AddrRouteble([]string{remoteNetwork.String()})) == 1 {
			s.node.addrBook.AddAddress(remoteNetwork)
		}

	}

	log.Debug("Send Pong", "Nonce", in.GetNonce())
	return &pb.P2PPong{Nonce: in.GetNonce()}, nil

}

// 获取地址
func (s *p2pServer) GetAddr(ctx context.Context, in *pb.P2PGetAddr) (*pb.P2PAddr, error) {
	log.Debug("GETADDR", "RECV ADDR", in, "OutBound Len", s.node.Size())
	addrBucket := make(map[string]bool)
	peers, _ := s.node.GetActivePeers()
	log.Debug("GetAddr", "GetPeers", peers)
	for _, peer := range peers {
		if stat := s.node.addrBook.GetPeerStat(peer.Addr()); stat != nil {
			if stat.GetAttempts() == 0 {
				addrBucket[peer.Addr()] = true
			}
		}

	}
	addrList := s.node.addrBook.GetAddrs()
	for _, addr := range addrList {

		addrBucket[addr] = true
		if len(addrBucket) > MaxAddrListNum { //最多一次性返回256个地址
			break
		}
	}
	var addrlist []string
	for addr, _ := range addrBucket {
		addrlist = append(addrlist, addr)
	}
	return &pb.P2PAddr{Nonce: in.Nonce, Addrlist: addrlist}, nil
}

// 版本
func (s *p2pServer) Version(ctx context.Context, in *pb.P2PVersion) (*pb.P2PVerAck, error) {

	return &pb.P2PVerAck{Version: s.node.nodeInfo.cfg.GetVersion(), Service: 6, Nonce: in.Nonce}, nil
}
func (s *p2pServer) Version2(ctx context.Context, in *pb.P2PVersion) (*pb.P2PVersion, error) {

	getctx, ok := pr.FromContext(ctx)
	var peeraddr string
	if ok {
		peeraddr = strings.Split(getctx.Addr.String(), ":")[0]
		log.Debug("Version2", "Addr", peeraddr)
	}

	if s.checkVersion(in.GetVersion()) == false {
		return nil, fmt.Errorf(VersionNotSupport)
	}

	remoteNetwork, err := NewNetAddressString(in.AddrFrom)
	if err == nil {
		if len(P2pComm.AddrRouteble([]string{remoteNetwork.String()})) == 1 {
			s.node.addrBook.AddAddress(remoteNetwork)
			//broadcast again
			go func() {
				if time.Now().Unix()-in.GetTimestamp() > 5 || s.node.Has(in.AddrFrom) {
					return
				}

				peers, _ := s.node.GetActivePeers()
				for _, peer := range peers {

					peer.mconn.conn.Version2(context.Background(), in)

				}
			}()
		}

	}

	//addrFrom:表示自己的外网地址，addrRecv:表示对方的外网地址
	return &pb.P2PVersion{Version: s.node.nodeInfo.cfg.GetVersion(), Service: SERVICE, Nonce: in.Nonce,
		AddrFrom: in.AddrRecv, AddrRecv: fmt.Sprintf("%v:%v", peeraddr, strings.Split(in.AddrFrom, ":")[1])}, nil
}

func (s *p2pServer) BroadCastTx(ctx context.Context, in *pb.P2PTx) (*pb.Reply, error) {
	log.Debug("p2pServer RECV TRANSACTION", "in", in)
	client := s.node.nodeInfo.qclient
	msg := client.NewMessage("mempool", pb.EventTx, in.Tx)
	client.Send(msg, false)
	return &pb.Reply{IsOk: true, Msg: []byte("ok")}, nil
}

func (s *p2pServer) GetBlocks(ctx context.Context, in *pb.P2PGetBlocks) (*pb.P2PInv, error) {

	log.Debug("p2pServer GetBlocks", "P2P Recv", in)
	if s.checkVersion(in.GetVersion()) == false {
		return nil, fmt.Errorf(VersionNotSupport)
	}

	client := s.node.nodeInfo.qclient
	msg := client.NewMessage("blockchain", pb.EventGetHeaders, &pb.ReqBlocks{Start: in.StartHeight, End: in.EndHeight,
		Isdetail: false})
	err := client.Send(msg, true)
	if err != nil {
		log.Error("GetBlocks", "Error", err.Error())
		return nil, err
	}
	resp, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	headers := resp.Data.(*pb.Headers)
	var invs = make([]*pb.Inventory, 0)
	for _, item := range headers.Items {
		var inv pb.Inventory
		inv.Ty = MSG_BLOCK
		inv.Height = item.GetHeight()
		invs = append(invs, &inv)
	}
	return &pb.P2PInv{Invs: invs}, nil
}

//服务端查询本地mempool
func (s *p2pServer) GetMemPool(ctx context.Context, in *pb.P2PGetMempool) (*pb.P2PInv, error) {
	log.Debug("p2pServer Recv GetMempool", "version", in)
	if s.checkVersion(in.GetVersion()) == false {
		return nil, fmt.Errorf(VersionNotSupport)
	}
	memtx, err := s.loadMempool()
	if err != nil {
		return nil, err
	}

	var invlist = make([]*pb.Inventory, 0)
	for _, tx := range memtx {
		invlist = append(invlist, &pb.Inventory{Hash: tx.Hash(), Ty: MSG_TX})
	}

	return &pb.P2PInv{Invs: invlist}, nil
}

func (s *p2pServer) GetData(in *pb.P2PGetData, stream pb.P2Pgservice_GetDataServer) error {
	log.Debug("p2pServer Recv GetDataTx", "p2p version", in.GetVersion())
	var p2pInvData = make([]*pb.InvData, 0)
	var count = 0
	if s.checkVersion(in.GetVersion()) == false {
		return fmt.Errorf(VersionNotSupport)
	}
	invs := in.GetInvs()
	client := s.node.nodeInfo.qclient
	for _, inv := range invs { //过滤掉不需要的数据
		var invdata pb.InvData
		var memtx = make(map[string]*pb.Transaction)
		if inv.GetTy() == MSG_TX {
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
				invdata.Ty = MSG_TX
				p2pInvData = append(p2pInvData, &invdata)
			}

		} else if inv.GetTy() == MSG_BLOCK {
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
				invdata.Ty = MSG_BLOCK
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

func (s *p2pServer) GetHeaders(ctx context.Context, in *pb.P2PGetHeaders) (*pb.P2PHeaders, error) {
	log.Debug("p2pServer GetHeaders", "p2p version", in.GetVersion())
	if s.checkVersion(in.GetVersion()) == false {
		return nil, fmt.Errorf(VersionNotSupport)
	}
	if in.GetEndHeight()-in.GetStartHeight() > 2000 || in.GetEndHeight() < in.GetStartHeight() {
		return nil, fmt.Errorf("out of range")
	}

	client := s.node.nodeInfo.qclient
	msg := client.NewMessage("blockchain", pb.EventGetHeaders, &pb.ReqBlocks{Start: in.GetStartHeight(), End: in.GetEndHeight()})
	err := client.Send(msg, true)
	if err != nil {
		log.Error("GetHeaders", "Error", err.Error())
		return nil, err
	}
	resp, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}

	headers := resp.GetData().(*pb.Headers)

	return &pb.P2PHeaders{Headers: headers.GetItems()}, nil
}

func (s *p2pServer) GetPeerInfo(ctx context.Context, in *pb.P2PGetPeerInfo) (*pb.P2PPeerInfo, error) {
	log.Debug("p2pServer GetPeerInfo", "p2p version", in.GetVersion())
	if s.checkVersion(in.GetVersion()) == false {
		return nil, fmt.Errorf(VersionNotSupport)
	}
	client := s.node.nodeInfo.qclient
	log.Debug("GetPeerInfo", "GetMempoolSize", "befor")
	msg := client.NewMessage("mempool", pb.EventGetMempoolSize, nil)
	err := client.Send(msg, true)
	if err != nil {
		log.Error("GetPeerInfo mempool", "Error", err.Error())
		return nil, err
	}
	log.Debug("GetPeerInfo", "GetMempoolSize", "after")
	resp, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}

	meminfo := resp.GetData().(*pb.MempoolSize)
	var peerinfo pb.P2PPeerInfo

	pub, err := P2pComm.Pubkey(s.node.addrBook.key)
	if err != nil {
		log.Error("getpubkey", "error", err.Error())
	}

	log.Debug("GetPeerInfo", "EventGetLastHeader", "befor")
	//get header
	msg = client.NewMessage("blockchain", pb.EventGetLastHeader, nil)
	err = client.Send(msg, true)
	if err != nil {
		log.Error("GetPeerInfo blockchain", "Error", err.Error())
		return nil, err
	}
	resp, err = client.Wait(msg)
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

func (s *p2pServer) BroadCastBlock(ctx context.Context, in *pb.P2PBlock) (*pb.Reply, error) {
	client := s.node.nodeInfo.qclient
	msg := client.NewMessage("blockchain", pb.EventBroadcastAddBlock, in.GetBlock())
	err := client.Send(msg, false)
	if err != nil {
		log.Error("BroadCastBlock", "Error", err.Error())
		return nil, err
	}
	return &pb.Reply{IsOk: true, Msg: []byte("ok")}, nil
}

func (s *p2pServer) ServerStreamSend(in *pb.P2PPing, stream pb.P2Pgservice_ServerStreamSendServer) error {
	peername := hex.EncodeToString(in.GetSign().GetPubkey())
	dataChain := s.addStreamHandler(stream)
	for data := range dataChain {
		p2pdata := new(pb.BroadCastData)
		if block, ok := data.(*pb.P2PBlock); ok {
			p2pdata.Value = &pb.BroadCastData_Block{Block: block}
		} else if tx, ok := data.(*pb.P2PTx); ok {
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

func (s *p2pServer) ServerStreamRead(stream pb.P2Pgservice_ServerStreamReadServer) error {
	var peeraddr, peername string
	defer s.deleteInBoundPeerInfo(peername)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Info("ServerStreamRead", "Recv", "EOF")
			return err
		}
		if err != nil {
			log.Error("ServerStreamRead", "Recv", err)
			return err
		}
		if block := in.GetBlock(); block != nil {
			log.Info("ServerStreamRead", " Recv block==+=====+=====+=>Height", block.GetBlock().GetHeight())
			if block.GetBlock() != nil {
				msg := s.node.nodeInfo.qclient.NewMessage("blockchain", pb.EventBroadcastAddBlock, block.GetBlock())
				err := s.node.nodeInfo.qclient.Send(msg, false)
				if err != nil {
					log.Error("ServerStreamRead", "Error", err.Error())
					continue
				}
			}

		} else if tx := in.GetTx(); tx != nil {
			log.Debug("RouteChat", "tx", tx.GetTx())
			if tx.GetTx() != nil {
				msg := s.node.nodeInfo.qclient.NewMessage("mempool", pb.EventTx, tx.GetTx())
				s.node.nodeInfo.qclient.Send(msg, false)
			}

		} else if ping := in.GetPing(); ping != nil {
			//Ping package
			peername = hex.EncodeToString(ping.GetSign().GetPubkey())
			peeraddr = fmt.Sprintf("%s:%v", in.GetPing().GetAddr(), in.GetPing().GetPort())
			s.addInBoundPeerInfo(peername, innerpeer{addr: peeraddr, name: peername})
		}
	}

}

func (s *p2pServer) RemotePeerAddr(ctx context.Context, in *pb.P2PGetAddr) (*pb.P2PExternalInfo, error) {
	var remoteaddr string
	var outside bool
	getctx, ok := pr.FromContext(ctx)
	if ok {
		remoteaddr = strings.Split(getctx.Addr.String(), ":")[0]

		if len(P2pComm.AddrRouteble([]string{fmt.Sprintf("%v:%v", remoteaddr, DefaultPort)})) == 0 {

			outside = false
		} else {
			outside = true

		}
	}
	return &pb.P2PExternalInfo{Addr: remoteaddr, Isoutside: outside}, nil
}

func (s *p2pServer) checkVersion(version int32) bool {
	if version < s.node.nodeInfo.cfg.GetVerMix() || version > s.node.nodeInfo.cfg.GetVerMax() {
		//版本不支持
		return false
	}

	return true
}
func (s *p2pServer) loadMempool() (map[string]*pb.Transaction, error) {

	var txmap = make(map[string]*pb.Transaction)
	client := s.node.nodeInfo.qclient
	msg := client.NewMessage("mempool", pb.EventGetMempool, nil)
	err := client.Send(msg, true)
	if err != nil {
		log.Error("loadMempool", "Error", err.Error())
		return txmap, err
	}
	resp, err := client.Wait(msg)
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

func (s *p2pServer) ManageStream() {
	go s.deleteDisableStream()
	go func() { //发送空的block stream ping
		ticker := time.NewTicker(StreamPingTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.addStreamData(&pb.P2PBlock{})
			case <-s.loopdone:
				return
			}
		}

	}()
	go func() {
		fifoChan := pub.Sub("Stream")
		for data := range fifoChan {
			s.addStreamData(data)
		}
		log.Info("p2pserver", "manageStream", "close")
	}()
}

func (s *p2pServer) addStreamHandler(stream pb.P2Pgservice_ServerStreamSendServer) chan interface{} {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	s.streams[stream] = make(chan interface{}, 1024)
	return s.streams[stream]

}

func (s *p2pServer) addStreamData(data interface{}) {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	timetikc := time.NewTicker(time.Second * 1)
	defer timetikc.Stop()
	for stream, _ := range s.streams {
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

func (s *p2pServer) deleteDisableStream() {
	for stream := range s.deleteSChan {
		s.deleteStream(stream)
	}
}
func (s *p2pServer) deleteStream(stream pb.P2Pgservice_ServerStreamSendServer) {
	s.smtx.Lock()
	defer s.smtx.Unlock()
	close(s.streams[stream])
	delete(s.streams, stream)
}

func (s *p2pServer) addInBoundPeerInfo(peername string, info innerpeer) {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	s.inboundpeers[peername] = &info
}

func (s *p2pServer) deleteInBoundPeerInfo(peername string) {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	delete(s.inboundpeers, peername)

}

func (s *p2pServer) getInBoundPeerInfo(peername string) *innerpeer {
	s.imtx.Lock()
	defer s.imtx.Unlock()
	if key, ok := s.inboundpeers[peername]; ok {
		return key
	}

	return nil
}
