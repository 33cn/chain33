package peer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

const diffHeightValue = 512

func (p *Protocol) getLocalPeerInfo() *types.Peer {
	msg := p.QueueClient.NewMessage(mempool, types.EventGetMempoolSize, nil)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil
	}
	resp, err := p.QueueClient.WaitTimeout(msg, time.Second*10)
	if err != nil {
		log.Error("getLocalPeerInfo", "mempool WaitTimeout", err)
		return nil
	}
	var localPeer types.Peer
	localPeer.MempoolSize = int32(resp.Data.(*types.MempoolSize).GetSize())

	msg = p.QueueClient.NewMessage(blockchain, types.EventGetLastHeader, nil)
	err = p.QueueClient.Send(msg, true)
	if err != nil {
		return nil
	}
	resp, err = p.QueueClient.WaitTimeout(msg, time.Second*10)
	if err != nil {
		log.Error("getLocalPeerInfo", "blockchain WaitTimeout", err)
		return nil
	}
	localPeer.Header = resp.Data.(*types.Header)
	localPeer.Name = p.Host.ID().Pretty()
	ip, port := parseIPAndPort(p.getExternalAddr())
	localPeer.Addr = ip
	localPeer.Port = int32(port)
	localPeer.Version = version.GetVersion() + "@" + version.GetAppVersion()
	localPeer.StoreDBVersion = version.GetStoreDBVersion()
	localPeer.LocalDBVersion = version.GetLocalDBVersion()
	return &localPeer
}

func (p *Protocol) refreshPeerInfo() {
	if !atomic.CompareAndSwapInt32(&p.refreshing, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&p.refreshing, 0)
	var wg sync.WaitGroup
	for _, remoteID := range p.ConnManager.FetchConnPeers() {
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
			pInfo, err := p.queryPeerInfoOld(pid)
			if err != nil {
				log.Error("refreshPeerInfo", "error", err, "pid", pid)
				return
			}
			p.PeerInfoManager.Refresh(pInfo)
		}(remoteID)
	}
	selfPeer := p.getLocalPeerInfo()
	if selfPeer != nil {
		selfPeer.Self = true
		p.PeerInfoManager.Refresh(selfPeer)
		p.checkOutBound(selfPeer.GetHeader().GetHeight())
	}
	wg.Wait()
}

func (p *Protocol) checkOutBound(height int64) {
	for _, pinfo := range p.PeerInfoManager.FetchAll() {
		if pinfo.GetHeader().GetHeight()+diffHeightValue < height {
			pid, err := peer.Decode(pinfo.GetName())
			if err != nil {
				continue
			}
			// 断开向外的主动连接
			for _, conn := range p.Host.Network().ConnsToPeer(pid) {
				//判断是Inbound 还是Outbound
				if conn.Stat().Direction == network.DirOutbound {
					// 拉入连接黑名单
					p.ConnBlackList.Add(pinfo.GetName(), 0)
					_ = conn.Close()
				}
			}
		}
	}
}

func (p *Protocol) detectNodeAddr() {

	// 通过bootstrap获取本节点公网ip
	for _, bootstrap := range p.SubConfig.BootStraps {
		if p.checkDone() {
			break
		}
		addr, _ := multiaddr.NewMultiaddr(bootstrap)
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			continue
		}
		err = p.queryVersionOld(peerInfo.ID)
		if err != nil {
			continue
		}
	}
	var rangeCount int
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
			time.Sleep(time.Minute)
		}
		for _, pid := range p.RoutingTable.ListPeers() {
			if p.containsPublicIP(pid) {
				continue
			}
			err := p.queryVersionOld(pid)
			if err != nil {
				log.Error("detectNodeAddr", "queryVersion error", err, "pid", pid)
				continue
			}
		}
	}
}

func (p *Protocol) queryPeerInfoOld(pid peer.ID) (*types.Peer, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, peerInfoOld)
	if err != nil {
		log.Error("refreshPeerInfo", "new stream error", err, "peer id", pid)
		return nil, err
	}
	defer protocol.CloseStream(stream)
	err = protocol.WriteStream(&types.MessagePeerInfoReq{}, stream)
	if err != nil {
		return nil, err
	}
	var resp types.MessagePeerInfoResp
	err = protocol.ReadStream(&resp, stream)
	if err != nil {
		return nil, err
	}
	if resp.Message == nil {
		return nil, types2.ErrInvalidResponse
	}
	pInfo := types.Peer{
		Addr:           resp.Message.Addr,
		Port:           resp.Message.Port,
		Name:           resp.Message.Name,
		MempoolSize:    resp.Message.MempoolSize,
		Header:         resp.Message.Header,
		Version:        resp.Message.Version,
		LocalDBVersion: resp.Message.LocalDBVersion,
		StoreDBVersion: resp.Message.StoreDBVersion,
	}
	return &pInfo, nil
}

func (p *Protocol) queryPeerInfo(pid peer.ID) (*types.Peer, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, peerInfo)
	if err != nil {
		log.Error("refreshPeerInfo", "new stream error", err, "peer id", pid)
		return nil, err
	}
	defer protocol.CloseStream(stream)
	var resp types.Peer
	err = protocol.ReadStream(&resp, stream)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *Protocol) queryVersionOld(pid peer.ID) error {
	stream, err := p.Host.NewStream(p.Ctx, pid, peerVersionOld)
	if err != nil {
		log.Error("NewStream", "err", err, "remoteID", pid)
		return err
	}
	defer protocol.CloseStream(stream)

	req := types.MessageP2PVersionReq{
		Message: &types.P2PVersion{
			Version:  p.SubConfig.Channel,
			AddrFrom: fmt.Sprintf("/ip4/%v/tcp/%d", p.getPublicIP(), p.SubConfig.Port),
			AddrRecv: stream.Conn().RemoteMultiaddr().String(),
		},
	}
	err = protocol.WriteStream(&req, stream)
	if err != nil {
		log.Error("queryVersion", "WriteStream err", err)
		return err
	}
	var res types.MessageP2PVersionResp
	err = protocol.ReadStream(&res, stream)
	if err != nil {
		log.Error("queryVersion", "ReadStream err", err)
		return err
	}
	msg := res.Message
	addr := msg.GetAddrRecv()
	p.setExternalAddr(addr)
	if !strings.Contains(msg.AddrFrom, "/ip4") {
		_, port := parseIPAndPort(stream.Conn().RemoteMultiaddr().String())
		msg.AddrFrom = fmt.Sprintf("/ip4/%s/tcp/%d", msg.AddrFrom, port)
	}
	if ip, _ := parseIPAndPort(msg.GetAddrFrom()); isPublicIP(ip) {
		remoteMAddr, err := multiaddr.NewMultiaddr(msg.GetAddrFrom())
		if err != nil {
			return err
		}
		p.Host.Peerstore().AddAddr(pid, remoteMAddr, time.Hour*12)
	}
	return nil
}

func (p *Protocol) queryVersion(pid peer.ID) error {
	stream, err := p.Host.NewStream(p.Ctx, pid, peerVersion)
	if err != nil {
		log.Error("NewStream", "err", err, "remoteID", pid)
		return err
	}
	defer protocol.CloseStream(stream)

	req := &types.P2PVersion{
		Version:  p.SubConfig.Channel,
		AddrFrom: fmt.Sprintf("/ip4/%v/tcp/%d", p.getPublicIP(), p.SubConfig.Port),
		AddrRecv: stream.Conn().RemoteMultiaddr().String(),
	}
	err = protocol.WriteStream(req, stream)
	if err != nil {
		log.Error("queryVersion", "WriteStream err", err)
		return err
	}
	var resp types.P2PVersion
	err = protocol.ReadStream(&resp, stream)
	if err != nil {
		log.Error("queryVersion", "ReadStream err", err)
		return err
	}
	addr := resp.GetAddrRecv()
	p.setExternalAddr(addr)

	if ip, _ := parseIPAndPort(resp.GetAddrFrom()); isPublicIP(ip) {
		remoteMAddr, err := multiaddr.NewMultiaddr(resp.GetAddrFrom())
		if err != nil {
			return err
		}
		p.Host.Peerstore().AddAddr(pid, remoteMAddr, time.Hour*12)
	}
	return nil
}

func (p *Protocol) setExternalAddr(addr string) {
	ip, _ := parseIPAndPort(addr)
	if isPublicIP(ip) {
		p.mutex.Lock()
		p.externalAddr = addr
		p.mutex.Unlock()
		ma, _ := multiaddr.NewMultiaddr(addr)
		p.Host.Peerstore().AddAddr(p.Host.ID(), ma, time.Hour*24)
	}
}

func (p *Protocol) getExternalAddr() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.externalAddr
}

func (p *Protocol) getPublicIP() string {
	ip, _ := parseIPAndPort(p.getExternalAddr())
	return ip
}

func (p *Protocol) containsPublicIP(pid peer.ID) bool {
	for _, maddr := range p.Host.Peerstore().Addrs(pid) {
		if ip, _ := parseIPAndPort(maddr.String()); isPublicIP(ip) {
			return true
		}
	}
	return false
}

func parseIPAndPort(multiAddr string) (ip string, port int) {
	split := strings.Split(multiAddr, "/")
	if len(split) < 5 {
		return
	}
	port, err := strconv.Atoi(split[4])
	if err != nil {
		return
	}
	ip = split[2]
	return
}
