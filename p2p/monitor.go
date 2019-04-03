// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/33cn/chain33/types"
)

func (n *Node) destroyPeer(peer *Peer) {
	log.Debug("deleteErrPeer", "Delete peer", peer.Addr(), "running", peer.GetRunning(),
		"version support", peer.version.IsSupport())

	n.nodeInfo.addrBook.RemoveAddr(peer.Addr())
	n.remove(peer.Addr())

}

func (n *Node) monitorErrPeer() {
	for {
		peer := <-n.nodeInfo.monitorChan
		if !peer.version.IsSupport() {
			//如果版本不支持,直接删除节点
			log.Info("VersoinMonitor", "NotSupport,addr", peer.Addr())
			n.destroyPeer(peer)
			//加入黑名单12小时
			n.nodeInfo.blacklist.Add(peer.Addr(), int64(3600*12))
			continue
		}
		if peer.IsMaxInbouds {
			n.destroyPeer(peer)
		}
		if !peer.GetRunning() {
			n.destroyPeer(peer)
			continue
		}

		pstat, ok := n.nodeInfo.addrBook.setAddrStat(peer.Addr(), peer.peerStat.IsOk())
		if ok {
			if pstat.GetAttempts() > maxAttemps {
				log.Debug("monitorErrPeer", "over maxattamps", pstat.GetAttempts())
				n.destroyPeer(peer)
			}
		}
	}
}

func (n *Node) getAddrFromGithub() {
	if !n.nodeInfo.cfg.UseGithub {
		return
	}
	//从github 上下载种子节点文件
	res, err := http.Get("https://raw.githubusercontent.com/chainseed/seeds/master/bty.txt")
	if err != nil {
		log.Error("getAddrFromGithub", "http.Get", err.Error())
		return
	}

	bf := new(bytes.Buffer)
	_, err = io.Copy(bf, res.Body)
	if err != nil {
		log.Error("getAddrFromGithub", "io.Copy", err.Error())
		return
	}

	fileContent := bf.String()
	st := strings.TrimSpace(fileContent)
	strs := strings.Split(st, "\n")
	log.Info("getAddrFromGithub", "download file", fileContent)
	for _, linestr := range strs {
		pidaddr := strings.Split(linestr, "@")
		if len(pidaddr) == 2 {
			addr := pidaddr[1]
			if n.Has(addr) || n.nodeInfo.blacklist.Has(addr) {
				return
			}
			n.pubsub.FIFOPub(addr, "addr")

		}
	}
}

// getAddrFromOnline gets the address list from the online node
func (n *Node) getAddrFromOnline() {
	ticker := time.NewTicker(GetAddrFromOnlineInterval)
	defer ticker.Stop()
	pcli := NewNormalP2PCli()

	var ticktimes int
	for {

		<-ticker.C

		seedsMap := make(map[string]bool)
		//每次循环seed的排序不同
		seedArr := n.nodeInfo.cfg.Seeds
		for _, seed := range seedArr {
			seedsMap[seed] = true
		}

		ticktimes++
		if n.isClose() {
			log.Debug("GetAddrFromOnLine", "loop", "done")
			return
		}

		if n.Size() == 0 && ticktimes > 2 {
			//尝试与Seed 进行连接
			var rangeCount int
			for addr := range seedsMap {
				//先把seed 排除在外
				rangeCount++
				if rangeCount < maxOutBoundNum {
					n.pubsub.FIFOPub(addr, "addr")
				}

			}

			if rangeCount < maxOutBoundNum {
				//从innerSeeds 读取连接
				n.innerSeeds.Range(func(k, v interface{}) bool {
					rangeCount++
					if rangeCount < maxOutBoundNum {
						n.pubsub.FIFOPub(k.(string), "addr")
						return true
					}
					return false

				})
			}

			continue
		}

		peers, _ := n.GetActivePeers()
		for _, peer := range peers { //向其他节点发起请求，获取地址列表

			var addrlist []string
			var addrlistMap map[string]int64

			var err error

			addrlistMap, err = pcli.GetAddrList(peer)

			P2pComm.CollectPeerStat(err, peer)
			if err != nil {
				log.Error("getAddrFromOnline", "ERROR", err.Error())
				continue
			}

			for addr := range addrlistMap {
				addrlist = append(addrlist, addr)
			}

			for _, addr := range addrlist {

				if !n.needMore() {

					//如果已经达到25个节点，则优先删除种子节点
					localBlockHeight, err := pcli.GetBlockHeight(n.nodeInfo)
					if err != nil {
						continue
					}
					//查询对方的高度，如果不小于自己的高度,或高度差在一定范围内，则剔除一个种子
					if peerHeight, ok := addrlistMap[addr]; ok {

						if localBlockHeight-peerHeight < 1024 {
							if _, ok := seedsMap[addr]; ok {
								continue
							}

							//随机删除连接的一个种子

							n.innerSeeds.Range(func(k, v interface{}) bool {
								if n.Has(k.(string)) {
									//不能包含在cfgseed中
									if _, ok := n.cfgSeeds.Load(k.(string)); ok {
										return true
									}
									n.remove(k.(string))
									n.nodeInfo.addrBook.RemoveAddr(k.(string))
									return false
								}
								return true
							})
						}
					}
					time.Sleep(MonitorPeerInfoInterval)
					continue
				}

				if !n.nodeInfo.blacklist.Has(addr) || !Filter.QueryRecvData(addr) {
					if ticktimes < 10 {
						//如果连接了其他节点，优先不连接种子节点
						if _, ok := n.innerSeeds.Load(addr); !ok {
							//先把seed 排除在外
							n.pubsub.FIFOPub(addr, "addr")

						}
					} else {
						n.pubsub.FIFOPub(addr, "addr")
					}

				}
			}

		}

	}
}

func (n *Node) getAddrFromAddrBook() {
	ticker := time.NewTicker(GetAddrFromAddrBookInterval)
	defer ticker.Stop()
	var tickerTimes int64

	for {
		<-ticker.C
		tickerTimes++
		if n.isClose() {
			log.Debug("GetAddrFromOnLine", "loop", "done")
			return
		}
		//12个循环后， 则去github下载
		if tickerTimes > 12 && n.Size() == 0 {
			n.getAddrFromGithub()
			tickerTimes = 0
		}

		log.Debug("OUTBOUND NUM", "NUM", n.Size(), "start getaddr from peer,peernum", len(n.nodeInfo.addrBook.GetPeers()))

		addrNetArr := n.nodeInfo.addrBook.GetPeers()

		for _, addr := range addrNetArr {
			if !n.Has(addr.String()) && !n.nodeInfo.blacklist.Has(addr.String()) {
				log.Debug("GetAddrFromOffline", "Add addr", addr.String())

				if n.needMore() || n.CacheBoundsSize() < maxOutBoundNum {
					n.pubsub.FIFOPub(addr.String(), "addr")

				}
			}
		}

		log.Debug("Node Monitor process", "outbound num", n.Size())
	}

}

func (n *Node) nodeReBalance() {
	p2pcli := NewNormalP2PCli()
	ticker := time.NewTicker(MonitorReBalanceInterval)
	defer ticker.Stop()

	for {

		<-ticker.C
		log.Info("nodeReBalance", "cacheSize", n.CacheBoundsSize())
		if n.CacheBoundsSize() == 0 {
			continue
		}
		peers, _ := n.GetActivePeers()
		//选出当前连接的节点中，负载最大的节点
		var MaxInBounds int32
		var MaxInBoundPeer *Peer
		for _, peer := range peers {
			if peer.GetInBouns() > MaxInBounds {
				MaxInBounds = peer.GetInBouns()
				MaxInBoundPeer = peer
			}
		}
		if MaxInBoundPeer == nil {
			continue
		}

		//筛选缓存备选节点负载最大和最小的节点
		cachePeers := n.GetCacheBounds()
		var MinCacheInBounds int32 = 1000
		var MinCacheInBoundPeer *Peer
		var MaxCacheInBounds int32
		var MaxCacheInBoundPeer *Peer
		for _, peer := range cachePeers {
			inbounds, err := p2pcli.GetInPeersNum(peer)
			if err != nil {
				n.RemoveCachePeer(peer.Addr())
				peer.Close()
				continue
			}
			//选出最小负载
			if int32(inbounds) < MinCacheInBounds {
				MinCacheInBounds = int32(inbounds)
				MinCacheInBoundPeer = peer
			}

			//选出负载最大
			if int32(inbounds) > MaxCacheInBounds {
				MaxCacheInBounds = int32(inbounds)
				MaxCacheInBoundPeer = peer
			}
		}

		if MinCacheInBoundPeer == nil || MaxCacheInBoundPeer == nil {
			continue
		}

		//如果连接的节点最大负载量小于当前缓存节点的最大负载量
		if MaxInBounds < MaxCacheInBounds {
			n.RemoveCachePeer(MaxCacheInBoundPeer.Addr())
			MaxCacheInBoundPeer.Close()
		}
		//如果最大的负载量比缓存中负载最小的小，则删除缓存中所有的节点
		if MaxInBounds < MinCacheInBounds {
			cachePeers := n.GetCacheBounds()
			for _, peer := range cachePeers {
				n.RemoveCachePeer(peer.Addr())
				peer.Close()
			}

			continue
		}
		log.Info("nodeReBalance", "MaxInBounds", MaxInBounds, "MixCacheInBounds", MinCacheInBounds)
		if MaxInBounds-MinCacheInBounds < 50 {
			continue
		}

		if MinCacheInBoundPeer != nil {
			info, err := MinCacheInBoundPeer.GetPeerInfo(VERSION)
			if err != nil {
				n.RemoveCachePeer(MinCacheInBoundPeer.Addr())
				MinCacheInBoundPeer.Close()
				continue
			}
			localBlockHeight, err := p2pcli.GetBlockHeight(n.nodeInfo)
			if err != nil {
				continue
			}
			peerBlockHeight := info.GetHeader().GetHeight()
			if localBlockHeight-peerBlockHeight < 2048 {
				log.Info("noReBalance", "Repalce node new node", MinCacheInBoundPeer.Addr(), "old node", MaxInBoundPeer.Addr())
				n.addPeer(MinCacheInBoundPeer)
				n.nodeInfo.addrBook.AddAddress(MinCacheInBoundPeer.peerAddr, nil)

				n.remove(MaxInBoundPeer.Addr())
				n.RemoveCachePeer(MinCacheInBoundPeer.Addr())
			}
		}
	}
}

func (n *Node) monitorPeers() {

	p2pcli := NewNormalP2PCli()

	ticker := time.NewTicker(MonitorPeerNumInterval)
	defer ticker.Stop()
	_, selfName := n.nodeInfo.addrBook.GetPrivPubKey()
	for {

		<-ticker.C
		localBlockHeight, err := p2pcli.GetBlockHeight(n.nodeInfo)
		if err != nil {
			continue
		}

		peers, infos := n.GetActivePeers()
		for paddr, pinfo := range infos {
			peerheight := pinfo.GetHeader().GetHeight()
			if pinfo.GetName() == selfName && !pinfo.GetSelf() { //发现连接到自己，立即删除
				//删除节点数过低的节点
				n.remove(paddr)
				n.nodeInfo.addrBook.RemoveAddr(paddr)
				n.nodeInfo.blacklist.Add(paddr, 0)
			}

			if localBlockHeight-peerheight > 2048 {
				//删除比自己较低的节点
				if addrMap, err := p2pcli.GetAddrList(peers[paddr]); err == nil {

					for addr := range addrMap {
						if !n.Has(addr) && !n.nodeInfo.blacklist.Has(addr) {
							n.pubsub.FIFOPub(addr, "addr")
						}
					}

				}

				if n.Size() <= stableBoundNum {
					continue
				}
				//如果是配置节点，则不删除
				if _, ok := n.cfgSeeds.Load(paddr); ok {
					continue
				}
				//删除节点数过低的节点
				n.remove(paddr)
				n.nodeInfo.addrBook.RemoveAddr(paddr)
			}

		}

	}

}

func (n *Node) monitorPeerInfo() {

	go func() {
		n.nodeInfo.FetchPeerInfo(n)
		ticker := time.NewTicker(MonitorPeerInfoInterval)
		defer ticker.Stop()
		for {
			if n.isClose() {
				return
			}

			<-ticker.C
			n.nodeInfo.FetchPeerInfo(n)
		}
	}()
}

// monitorDialPeers connect the node address concurrently
func (n *Node) monitorDialPeers() {
	var dialCount int
	addrChan := n.pubsub.Sub("addr")
	p2pcli := NewNormalP2PCli()
	for addr := range addrChan {

		if n.isClose() {
			log.Info("monitorDialPeers", "loop", "done")
			return
		}
		if Filter.QueryRecvData(addr.(string)) {
			//先查询有没有注册进去，避免同时重复连接相同的地址
			continue
		}

		netAddr, err := NewNetAddressString(addr.(string))
		if err != nil {
			continue
		}

		if n.nodeInfo.addrBook.ISOurAddress(netAddr) {
			continue
		}

		//不对已经连接上的地址或者黑名单地址发起连接
		if n.Has(netAddr.String()) || n.nodeInfo.blacklist.Has(netAddr.String()) || n.HasCacheBound(netAddr.String()) {
			log.Debug("DialPeers", "find hash", netAddr.String())
			continue
		}

		//注册的节点超过最大节点数暂不连接
		if !n.needMore() && n.CacheBoundsSize() >= maxOutBoundNum {
			n.pubsub.FIFOPub(addr, "addr")
			time.Sleep(time.Second * 10)
			continue
		}

		log.Info("DialPeers", "peer", netAddr.String())
		//并发连接节点，增加连接效率
		if dialCount >= maxOutBoundNum*2 {
			n.pubsub.FIFOPub(addr, "addr")
			time.Sleep(time.Second * 10)
			dialCount = len(n.GetRegisterPeers()) + n.CacheBoundsSize()
			continue
		}
		dialCount++
		//把待连接的节点增加到过滤容器中
		Filter.RegRecvData(addr.(string))
		log.Info("monitorDialPeer", "dialCount", dialCount)
		go func(netAddr *NetAddress) {
			defer Filter.RemoveRecvData(netAddr.String())
			peer, err := P2pComm.dialPeer(netAddr, n)
			if err != nil {
				//连接失败后
				n.nodeInfo.addrBook.RemoveAddr(netAddr.String())
				log.Error("monitorDialPeers", "Err", err.Error())
				if err == types.ErrVersion { //版本不支持，加入黑名单12小时
					peer.version.SetSupport(false)
					P2pComm.CollectPeerStat(err, peer)
					return
				}
				//其他原因，黑名单10分钟
				if peer != nil {
					peer.Close()
				}
				if _, ok := n.cfgSeeds.Load(netAddr.String()); !ok {
					n.nodeInfo.blacklist.Add(netAddr.String(), int64(60*10))
				}
				return
			}
			//查询远程节点的负载
			inbounds, err := p2pcli.GetInPeersNum(peer)
			if err != nil {
				peer.Close()
				return
			}
			//判断负载情况,负载达到额定90%，负载过大，就删除节点
			if int32(inbounds*100)/n.nodeInfo.cfg.InnerBounds > 90 {
				peer.Close()
				return
			}
			//注册的节点超过最大节点数暂不连接
			if len(n.GetRegisterPeers()) >= maxOutBoundNum {
				if n.CacheBoundsSize() < maxOutBoundNum {
					n.AddCachePeer(peer)
				} else {
					peer.Close()
				}
				return
			}

			n.addPeer(peer)
			n.nodeInfo.addrBook.AddAddress(netAddr, nil)

		}(netAddr)

	}

}

func (n *Node) monitorBlackList() {
	ticker := time.NewTicker(CheckBlackListInterVal)
	defer ticker.Stop()
	for {
		if n.isClose() {
			log.Info("monitorBlackList", "loop", "done")
			return
		}

		<-ticker.C
		badPeers := n.nodeInfo.blacklist.GetBadPeers()
		now := types.Now().Unix()
		for badPeer, intime := range badPeers {
			if n.nodeInfo.addrBook.IsOurStringAddress(badPeer) {
				continue
			}
			//0表示永久加入黑名单
			if 0 == intime {
				continue
			}
			if now > intime {
				n.nodeInfo.blacklist.Delete(badPeer)
			}
		}
	}
}

func (n *Node) monitorFilter() {
	Filter.ManageRecvFilter()
}

//独立goroutine 监控配置的

func (n *Node) monitorCfgSeeds() {

	ticker := time.NewTicker(CheckCfgSeedsInterVal)
	defer ticker.Stop()

	for {
		if n.isClose() {
			log.Info("monitorCfgSeeds", "loop", "done")
			return
		}

		<-ticker.C
		n.cfgSeeds.Range(func(k, v interface{}) bool {

			if !n.Has(k.(string)) {
				//尝试连接此节点
				if n.needMore() { //如果需要更多的节点
					n.pubsub.FIFOPub(k.(string), "addr")
				} else {
					//腾笼换鸟
					peers, _ := n.GetActivePeers()
					//选出当前连接的节点中，负载最大的节点
					var MaxInBounds int32
					var MaxInBoundPeer *Peer
					for _, peer := range peers {
						if peer.GetInBouns() > MaxInBounds {
							MaxInBounds = peer.GetInBouns()
							MaxInBoundPeer = peer
						}
					}

					n.remove(MaxInBoundPeer.Addr())
					n.pubsub.FIFOPub(k.(string), "addr")

				}

			}
			return true
		})
	}

}
