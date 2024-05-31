package peer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common/utils"

	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	diffHeightValue = 512
	maxPeers        = 20
	latestTag       = "latestBlock"
	highestTag      = "highest"
)

type lastBlock struct {
	Height    int64
	LastScene time.Time
}

func (p *Protocol) getLocalPeerInfo() *types.Peer {
	msg := p.QueueClient.NewMessage(mempool, types.EventGetMempoolSize, nil)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil
	}
	resp, err := p.QueueClient.WaitTimeout(msg, time.Second*5)
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
	resp, err = p.QueueClient.WaitTimeout(msg, time.Second)
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
	//localPeer.StoreDBVersion = version.GetStoreDBVersion()
	//localPeer.LocalDBVersion = version.GetLocalDBVersion()
	//增加节点运行时间，以及节点节点类型：全节点，分片节点
	localPeer.RunningTime = caculteRunningTime()
	localPeer.FullNode = p.SubConfig.IsFullNode
	//增加节点高度增长是否是阻塞状态监测
	localPeer.Blocked = p.getBlocked()
	localPeer.Finalized, err = p.API.GetFinalizedBlock()
	if err != nil {
		log.Error("getLocalPeerInfo", "GetFinalizedBlock err", err)
		return nil
	}
	return &localPeer
}

func caculteRunningTime() string {
	var runningTime string
	mins := time.Since(processStart).Minutes()
	runningTime = fmt.Sprintf("%.3f minutes", mins)
	if mins > 60 {
		hours := mins / 60
		runningTime = fmt.Sprintf("%.3f hours", hours)
	}

	return runningTime
}
func (p *Protocol) refreshSelf() {
	selfPeer := p.getLocalPeerInfo()
	if selfPeer != nil {
		selfPeer.Self = true
		p.PeerInfoManager.Refresh(selfPeer)
	}
}

func (p *Protocol) refreshPeerInfo(peers []peer.ID) {
	var wg sync.WaitGroup
	// 限制最大并发数量为20
	ch := make(chan struct{}, 20)
	for _, remoteID := range peers {
		if p.checkDone() {
			log.Warn("getPeerInfo", "process", "done+++++++")
			return
		}
		if remoteID == p.Host.ID() {
			continue
		}
		//修改为并发获取peerinfo信息
		wg.Add(1)
		ch <- struct{}{}
		go func(pid peer.ID) {
			defer func() {
				<-ch
				wg.Done()
			}()
			pInfo, err := p.queryPeerInfo(pid)
			if err != nil {
				log.Error("refreshPeerInfo", "error", err, "pid", pid)
				return
			}
			if p.checkVersionLimit(pInfo.GetVersion()) {
				p.PeerInfoManager.Refresh(pInfo)
			} else {
				//add blacklist
				log.Info("refreshPeerInfo", "AddBlacklist,peerName:", pInfo.GetName(), "version:", pInfo.GetVersion(), "runningTime:", pInfo.GetRunningTime())
				p.P2PEnv.Host.Network().ClosePeer(peer.ID(pInfo.GetName()))
				p.P2PEnv.ConnBlackList.Add(pInfo.GetName(), time.Hour*24)
			}

		}(remoteID)
	}
	wg.Wait()
}

func (p *Protocol) checkOutBound(height int64) {
	if height < diffHeightValue {
		return
	}
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
		err = p.queryVersion(peerInfo.ID)
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
		//启动后间隔10秒钟，以充分获得节点外网地址，稳定后间隔10分钟
		rangeCount++
		if rangeCount > 100 {
			time.Sleep(time.Minute * 10)
		} else if rangeCount > 2 {
			time.Sleep(time.Second * 10)
		}
		for _, pid := range p.RoutingTable.ListPeers() {
			if utils.IsPublicIP(p.getPublicIP()) && p.containsPublicIP(pid) {
				continue
			}
			err := p.queryVersion(pid)
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
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer stream.Close()
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
		Addr:        resp.Message.Addr,
		Port:        resp.Message.Port,
		Name:        resp.Message.Name,
		MempoolSize: resp.Message.MempoolSize,
		Header:      resp.Message.Header,
		Version:     resp.Message.Version,
		//LocalDBVersion: resp.Message.LocalDBVersion,
		//StoreDBVersion: resp.Message.StoreDBVersion,
		FullNode:    resp.Message.GetFullNode(),
		RunningTime: resp.Message.GetRunningTime(),
		Blocked:     resp.Message.GetBlocked(),
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
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer stream.Close()
	var resp types.Peer
	err = protocol.ReadStream(&resp, stream)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *Protocol) queryVersionOld(pid peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, peerVersionOld)
	if err != nil {
		log.Error("NewStream", "err", err, "remoteID", pid)
		return err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer stream.Close()

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
	if ip, _ := parseIPAndPort(msg.GetAddrFrom()); utils.IsPublicIP(ip) {
		remoteMAddr, err := multiaddr.NewMultiaddr(msg.GetAddrFrom())
		if err != nil {
			return err
		}
		p.Host.Peerstore().AddAddr(pid, remoteMAddr, time.Hour*12)
	}
	return nil
}

func (p *Protocol) queryVersion(pid peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, peerVersion)
	if err != nil {
		log.Error("NewStream", "err", err, "remoteID", pid)
		return err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer stream.Close()

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

	if ip, _ := parseIPAndPort(resp.GetAddrFrom()); utils.IsPublicIP(ip) {
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
	if utils.IsPublicIP(ip) {
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
		if ip, _ := parseIPAndPort(maddr.String()); utils.IsPublicIP(ip) {
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

func (p *Protocol) checkBlocked() {

	var check = time.NewTicker(time.Second * 10)
	defer check.Stop()

	for {
		select {

		case <-p.Ctx.Done():
			return

		case <-check.C:
			msg := p.QueueClient.NewMessage(blockchain, types.EventGetLastHeader, nil)
			err := p.QueueClient.Send(msg, true)
			if err != nil {
				continue
			}
			resp, err := p.QueueClient.WaitTimeout(msg, time.Second)
			if err != nil {
				log.Error("checkBlocked EventGetLastHeader", "blockchain WaitTimeout", err)
				continue
			}

			header := resp.Data.(*types.Header)
			var storeInfo lastBlock
			storeInfo.LastScene = time.Now()
			storeInfo.Height = header.GetHeight()
			p.latestBlock.Store(latestTag, &storeInfo)

			if info, ok := p.latestBlock.Load(highestTag); ok {
				preMaxHigh := info.(*lastBlock)
				if header.GetHeight() > preMaxHigh.Height {
					preMaxHigh.Height = header.GetHeight()
					preMaxHigh.LastScene = storeInfo.LastScene
					atomic.StoreInt32(&p.blocked, 0)
					continue
				}
				if storeInfo.LastScene.Sub(preMaxHigh.LastScene) > time.Minute { //1分钟高度没有更新
					atomic.StoreInt32(&p.blocked, 1)
				}
			}

		}
	}
}

func (p *Protocol) getBlocked() bool {
	return atomic.LoadInt32(&p.blocked) == 1
}
