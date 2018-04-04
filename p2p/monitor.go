package p2p

import (
	"time"
)

func (n *Node) destroyPeer(peer *peer) {
	log.Debug("deleteErrPeer", "Delete peer", peer.Addr(), "running", peer.GetRunning(),
		"version support", peer.version.IsSupport())
	n.nodeInfo.addrBook.RemoveAddr(peer.Addr())
	n.Remove(peer.Addr())

}

func (n *Node) monitorErrPeer() {
	for {
		peer := <-n.nodeInfo.monitorChan
		if peer.version.IsSupport() == false { //如果版本不支持,直接删除节点
			log.Debug("VersoinMonitor", "NotSupport,addr", peer.Addr())
			n.destroyPeer(peer)
			//加入黑名单
			n.nodeInfo.blacklist.Add(peer.Addr())
			continue
		}

		if peer.GetRunning() == false {
			n.destroyPeer(peer)
			continue
		}

		pstat, ok := n.nodeInfo.addrBook.SetAddrStat(peer.Addr(), peer.peerStat.IsOk())
		if ok {
			if pstat.GetAttempts() > MaxAttemps {
				log.Debug("monitorErrPeer", "over maxattamps", pstat.GetAttempts())
				n.destroyPeer(peer)
			}
		}
	}
}

func (n *Node) getAddrFromOnline() {
	ticker := time.NewTicker(GetAddrFromOnlineInterval)
	defer ticker.Stop()
	pcli := NewP2pCli(nil)

	for {
		select {
		case <-ticker.C:
			if n.IsClose() {
				log.Debug("GetAddrFromOnLine", "loop", "done")
				return
			}
			if n.needMore() {
				peers, _ := n.GetActivePeers()
				for _, peer := range peers { //向其他节点发起请求，获取地址列表
					log.Debug("Getpeer", "addr", peer.Addr())
					addrlist, err := pcli.GetAddr(peer)
					P2pComm.CollectPeerStat(err, peer)
					if err != nil {
						log.Error("getAddrFromOnline", "ERROR", err.Error())
						continue
					}

					log.Debug("GetAddrFromOnline", "addrlist", addrlist)
					//过滤黑名单的地址
					oklist := P2pComm.AddrRouteble(addrlist)
					for _, addr := range oklist {
						if n.nodeInfo.blacklist.Has(addr) == false {
							pub.FIFOPub(addr, "addr")
						} else {
							log.Debug("Filter addr", "BlackList", addr)
						}
					}

				}
			}

		}
	}
}

func (n *Node) getAddrFromOffline() {
	ticker := time.NewTicker(GetAddrFromOfflineInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if n.IsClose() {
				log.Debug("GetAddrFromOnLine", "loop", "done")
				return
			}
			if n.needMore() {
				var testlist []string
				for _, seed := range n.nodeInfo.cfg.Seeds {
					if n.Has(seed) == false && n.nodeInfo.blacklist.Has(seed) == false {
						log.Debug("GetAddrFromOffline", "Add Seed", seed)
						testlist = append(testlist, seed)

					}
				}

				log.Debug("OUTBOUND NUM", "NUM", n.Size(), "start getaddr from peer", n.nodeInfo.addrBook.GetPeers())
				peeraddrs := n.nodeInfo.addrBook.GetPeers()

				if len(peeraddrs) != 0 {

					for _, peer := range peeraddrs {
						testlist = append(testlist, peer.String())
					}
				}

				oklist := P2pComm.AddrRouteble(testlist)
				for _, addr := range oklist {

					if n.Has(addr) == false && n.nodeInfo.blacklist.Has(addr) == false {
						log.Debug("GetAddrFromOffline", "Add addr", addr)
						pub.FIFOPub(addr, "addr")
					}

				}

			} else {
				log.Debug("getAddrFromOffline", "nodestable", n.needMore())
				for _, seed := range n.nodeInfo.cfg.Seeds {
					//如果达到稳定节点数量，则断开种子节点
					if n.Has(seed) == true {
						n.Remove(seed)
					}
				}
			}

			log.Debug("Node Monitor process", "outbound num", n.Size())
		}
	}

}

func (n *Node) monitorPeerInfo() {

	go func() {
		n.nodeInfo.FetchPeerInfo(n)
		ticker := time.NewTicker(MonitorPeerInfoInterval)
		defer ticker.Stop()
		for {
			if n.IsClose() {
				return
			}
			select {
			case <-ticker.C:
				n.nodeInfo.FetchPeerInfo(n)

			}
		}
	}()
}

func (n *Node) monitorDialPeers() {

	addrChan := pub.Sub("addr")
	for addr := range addrChan {
		if n.IsClose() {
			log.Info("monitorDialPeers", "loop", "done")
			return
		}
		netAddr, err := NewNetAddressString(addr.(string))
		if err != nil {
			continue
		}

		if n.nodeInfo.addrBook.ISOurAddress(netAddr) == true {
			continue
		}

		//不对已经连接上的地址重新发起连接
		if n.Has(netAddr.String()) {
			log.Debug("DialPeers", "find hash", netAddr.String())
			continue
		}

		if n.needMore() == false {
			time.Sleep(time.Second * 10)
			continue
		}
		log.Debug("DialPeers", "peer", netAddr.String())
		peer, err := P2pComm.DialPeer(netAddr, &n.nodeInfo)
		if err != nil {
			log.Error("DialPeers", "Err", err.Error())
			continue
		}
		n.AddPeer(peer)
		n.nodeInfo.addrBook.AddAddress(netAddr)

	}

}

func (n *Node) monitorBlackList() {
	ticker := time.NewTicker(CheckBlackListInterVal)
	defer ticker.Stop()
	for {
		if n.IsClose() {
			log.Info("monitorBlackList", "loop", "done")
			return
		}

		select {
		case <-ticker.C:
			badPeers := n.nodeInfo.blacklist.GetBadPeers()
			now := time.Now().Unix()
			for badPeer, intime := range badPeers {
				if n.nodeInfo.addrBook.IsOurStringAddress(badPeer) {
					continue
				}
				if now-intime > 3600 { //one hour
					n.nodeInfo.blacklist.Delete(badPeer)
				}
			}
		}

	}
}

func (n *Node) monitorFilter() {
	Filter.ManageRecvFilter()
}
