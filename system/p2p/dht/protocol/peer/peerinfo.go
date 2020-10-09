package peer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

const (
	peerInfoReq = "/chain33/peerinfoReq/1.0.0" //老版本
	//peerInfoReqV2  = "/chain33/peerinfoReq/1.0.1" //新版本，增加chain33 version 信息反馈
	peerVersionReq = "/chain33/peerVersion/1.0.0"
)

const (
	blockchain = "blockchain"
	mempool    = "mempool"
)

var log = log15.New("module", "p2p.peer")

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

//type Istream
type Protocol struct {
	*protocol.P2PEnv

	externalAddr string
	mutex        sync.Mutex
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	p := Protocol{
		P2PEnv: env,
	}

	protocol.RegisterStreamHandler(p.Host, peerInfoReq, p.handleStreamPeerInfo)
	protocol.RegisterStreamHandler(p.Host, peerVersionReq, p.handleStreamVersion)
	protocol.RegisterEventHandler(types.EventPeerInfo, p.handleEventPeerInfo)
	protocol.RegisterEventHandler(types.EventGetNetInfo, p.handleEventNetInfo)
	protocol.RegisterEventHandler(types.EventNetProtocols, p.handleEventNetProtocols)

	go p.detectNodeAddr()
	go func() {
		ticker1 := time.NewTicker(time.Second * 10)
		select {
		case <-p.Ctx.Done():
			return
		case <-ticker1.C:
			p.refreshPeerInfo()
		}
	}()
}

func (p *Protocol) getLocalPeerInfo() *types.Peer {
	msg := p.QueueClient.NewMessage(mempool, types.EventGetMempoolSize, nil)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		log.Error("getLocalPeerInfo", "sendToMempool", err)
		return nil
	}
	resp, err := p.QueueClient.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil
	}
	var localPeer types.Peer
	localPeer.MempoolSize = int32(resp.Data.(*types.MempoolSize).GetSize())

	msg = p.QueueClient.NewMessage(blockchain, types.EventGetLastHeader, nil)
	err = p.QueueClient.Send(msg, true)
	if err != nil {
		log.Error("getLocalPeerInfo", "sendToBlockChain", err)
		return nil
	}
	resp, err = p.QueueClient.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil
	}
	localPeer.Header = resp.Data.(*types.Header)
	localPeer.Name = p.Host.ID().Pretty()
	//if p.Host.Peerstore().
	//localPeer.Addr, localPeer.Port =
	splites := strings.Split(p.Host.Addrs()[0].String(), "/")
	port, err := strconv.Atoi(splites[len(splites)-1])
	if err != nil {
		log.Error("getLocalPeerInfo", "Atoi error", err)
		return nil
	}
	localPeer.Port = int32(port)
	//TODO 需要返回自身的外网地址
	if p.getExternalAddr() == "" {
		localPeer.Addr = splites[2]

	} else {
		localPeer.Addr = p.getExternalAddr()
	}
	localPeer.Version = version.GetVersion() + "@" + version.GetAppVersion()
	localPeer.StoreDBVersion = version.GetStoreDBVersion()
	localPeer.LocalDBVersion = version.GetLocalDBVersion()
	return &localPeer
}

func (p *Protocol) refreshPeerInfo() {
	var wg sync.WaitGroup
	for _, remoteID := range p.RoutingTable.ListPeers() {
		if p.checkDone() {
			log.Warn("getPeerInfo", "process", "done+++++++")
			return
		}
		if remoteID == p.Host.ID() {
			continue
		}
		//修改为并发获取peerinfo信息
		wg.Add(1)
		go func(pid peer.ID) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
			defer cancel()
			stream, err := p.Host.NewStream(ctx, pid, peerInfoReq)
			if err != nil {
				log.Error("refreshPeerInfo", "new stream error", err, "peer id", pid)
				return
			}
			var resp types.MessagePeerInfoResp
			err = protocol.ReadStream(&resp, stream)
			if err != nil {
				log.Error("refreshPeerInfo", "read stream error", err, "peer id", pid)
				return
			}
			p.PeerInfoManager.Refresh(resp.GetMessage())
		}(remoteID)

	}
	wg.Wait()
}

func (p *Protocol) setExternalAddr(addr string) {
	var spliteAddr string
	splites := strings.Split(addr, "/")
	if len(splites) == 1 {
		spliteAddr = splites[0]
	} else if len(splites) > 2 {
		spliteAddr = splites[2]
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if spliteAddr != p.externalAddr && isPublicIP(spliteAddr) {
		p.externalAddr = spliteAddr
		//设置外部地址时同时保存到peerstore里
		addr := fmt.Sprintf("/ip4/%s/tcp/%d", spliteAddr, p.SubConfig.Port)
		ma, _ := multiaddr.NewMultiaddr(addr)
		p.Host.Peerstore().AddAddr(p.Host.ID(), ma, peerstore.PermanentAddrTTL)
	}
}

func (p *Protocol) getExternalAddr() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.externalAddr
}

func (p *Protocol) checkDone() bool {
	select {
	case <-p.Ctx.Done():
		return true
	default:
		return false
	}
}

func (p *Protocol) getNodeAddr(pid peer.ID) {
	for _, maddr := range p.Host.Peerstore().Addrs(pid) {
		// 如果已保存公网地址则不需要再持续请求
		ip, _ := parseIPAndPort(maddr.String())
		if isPublicIP(ip) {
			return
		}
	}
	stream, err := p.Host.NewStream(p.Ctx, pid, peerVersionReq)
	if err != nil {
		log.Error("NewStream", "err", err, "remoteID", pid)
		return
	}
	defer protocol.CloseStream(stream)

	req := &types.MessageP2PVersionReq{
		Message: &types.P2PVersion{
			Version:  p.SubConfig.Channel,
			AddrFrom: fmt.Sprintf("/ip4/%v/tcp/%d", p.getExternalAddr(), p.SubConfig.Port),
			AddrRecv: stream.Conn().RemoteMultiaddr().String(),
		},
	}
	err = protocol.WriteStream(req, stream)
	if err != nil {
		log.Error("DetectNodeAddr", "WriteStream err", err)
		return
	}
	var resp types.MessageP2PVersionResp
	err = protocol.ReadStream(&resp, stream)
	if err != nil {
		log.Error("DetectNodeAddr", "ReadStream err", err)
		return
	}
	addr := resp.GetMessage().GetAddrRecv()
	p.setExternalAddr(addr)
	remoteMAddr, err := multiaddr.NewMultiaddr(resp.GetMessage().GetAddrFrom())
	if err != nil {
		return
	}
	p.Host.Peerstore().AddAddr(pid, remoteMAddr, time.Minute)
}

func (p *Protocol) detectNodeAddr() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("detectNodeAddr", "recover", r)
		}
	}()
	//通常libp2p监听的地址列表，第一个为局域网地址，最后一个为外部，先进行外部地址预设置
	addrs := p.Host.Addrs()
	preExternalAddr := strings.Split(addrs[len(addrs)-1].String(), "/")[2]
	p.mutex.Lock()
	p.externalAddr = preExternalAddr
	p.mutex.Unlock()

	if isPublicIP(preExternalAddr) { //检测是PubIp不用继续通过其他节点获取
		log.Debug("detectNodeAddr", "testPubIp", preExternalAddr)
	}
	log.Info("detectNodeAddr", "+++++++++++++++", preExternalAddr, "addrs", addrs)
	var rangeCount int
	queryInterval := time.Minute
	for {
		if p.checkDone() {
			break
		}
		if p.RoutingTable.Size() == 0 {
			time.Sleep(time.Second)
			continue
		}
		//启动后间隔1分钟，以充分获得节点外网地址
		rangeCount++
		if rangeCount > 2 {
			time.Sleep(queryInterval)
		}
		log.Info("detectNodeAddr", "conns amount", len(p.Host.Network().Conns()))
		for _, pid := range p.RoutingTable.ListPeers() {
			stream, err := p.Host.NewStream(p.Ctx, pid, peerVersionReq)
			if err != nil {
				log.Error("NewStream", "err", err, "remoteID", pid)
				continue
			}
			defer protocol.CloseStream(stream)

			req := &types.MessageP2PVersionReq{
				Message: &types.P2PVersion{
					Version:  p.SubConfig.Channel,
					AddrFrom: fmt.Sprintf("/ip4/%v/tcp/%d", p.getExternalAddr(), p.SubConfig.Port),
					AddrRecv: stream.Conn().RemoteMultiaddr().String(),
				},
			}
			err = protocol.WriteStream(req, stream)
			if err != nil {
				log.Error("DetectNodeAddr", "WriteStream err", err)
				continue
			}
			var resp types.MessageP2PVersionResp
			err = protocol.ReadStream(&resp, stream)
			if err != nil {
				log.Error("DetectNodeAddr", "ReadStream err", err)
				continue
			}
			addr := resp.GetMessage().GetAddrRecv()
			p.setExternalAddr(addr)
			remoteMAddr, err := multiaddr.NewMultiaddr(resp.GetMessage().GetAddrFrom())
			if err != nil {
				continue
			}
			p.Host.Peerstore().AddAddr(pid, remoteMAddr, queryInterval*3)
		}
	}
}

func parseIPAndPort(multiAddr string) (ip string, port int) {
	split := strings.Split(multiAddr, "/")
	if len(split) < 5 {
		return
	}
	port, err := strconv.Atoi(split[5])
	if err != nil {
		return
	}
	ip = split[2]
	return
}
