package p2p

import (
	"time"
)

func (n *Node) checkActivePeers() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-n.loopDone:
			log.Debug("checkActivePeers", "loop", "done")
			break FOR_LOOP
		case <-ticker.C:
			peers := n.GetRegisterPeers()
			for _, peer := range peers {
				if peer.mconn == nil {
					n.destroyPeer(peer)
					continue
				}

				log.Debug("checkActivePeers", "remotepeer", peer.mconn.remoteAddress.String())
				if stat := n.addrBook.GetPeerStat(peer.Addr()); stat != nil {
					if stat.GetAttempts() > 20 || peer.GetRunning() == false {
						log.Info("checkActivePeers", "Delete peer", peer.Addr(), "Attemps", stat.GetAttempts(), "ISRUNNING", peer.GetRunning())

						n.destroyPeer(peer)
					}
				}

			}
		}

	}
}

func (n *Node) destroyPeer(peer *peer) {
	log.Debug("deleteErrPeer", "Delete peer", peer.Addr(), "RUNNING", peer.GetRunning(), "IsSuuport", peer.version.IsSupport())
	n.addrBook.RemoveAddr(peer.Addr())
	n.addrBook.Save()
	n.Remove(peer.Addr())
}

func (n *Node) monitorErrPeer() {

	for {

		peer := <-n.nodeInfo.monitorChan
		if peer.version.IsSupport() == false { //如果版本不支持，则加入黑名单，下次不再发起连接
			log.Debug("VersoinMonitor", "NotSupport", "DELETE")
			//n.nodeInfo.blacklist.Add(peer.Addr()) //加入黑名单
			n.destroyPeer(peer)
		}

		n.addrBook.SetAddrStat(peer.Addr(), peer.peerStat.IsOk())

	}

}

func (n *Node) getAddrFromOnline() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	pcli := NewP2pCli(nil)
FOR_LOOP:
	for {
		select {
		case <-n.loopDone:
			log.Debug("GetAddrFromOnLine", "loop", "Done")
			break FOR_LOOP
		case <-ticker.C:
			if n.needMore() {
				peers, _ := n.GetActivePeers()
				for _, peer := range peers { //向其他节点发起请求，获取地址列表
					log.Debug("Getpeer", "addr", peer.Addr())
					addrlist, err := pcli.GetAddr(peer)
					if err != nil {
						log.Error("getAddrFromOnline", "ERROR", err.Error())
						continue
					}
					//log.Info("getAddrFromOnline", "ADDRLIST", addrlist)
					//过滤无法连接的节点

					//过滤黑名单的地址
					oklist := P2pComm.AddrTest(addrlist)
					var whitlist = make(map[string]bool)
					for _, addr := range oklist {
						if n.nodeInfo.blacklist.Has(addr) == false {
							whitlist[addr] = true
						} else {
							log.Debug("Filter addr", "BlackList", addr)
						}
					}

					go n.DialPeers(whitlist) //对获取的地址列表发起连接

				}
			}

		}
	}
}

func (n *Node) getAddrFromOffline() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-n.loopDone:
			log.Debug("GetAddrFromOffLine", "loop", "Done")
			break FOR_LOOP
		case <-ticker.C:
			if n.needMore() {
				var savelist = make(map[string]bool)
				for _, seed := range n.nodeInfo.cfg.Seeds {
					if n.Has(seed) == false && n.nodeInfo.blacklist.Has(seed) == false {
						log.Debug("GetAddrFromOffline", "Add Seed", seed)
						savelist[seed] = true
					}
				}

				log.Debug("OUTBOUND NUM", "NUM", n.Size(), "start getaddr from peer", n.addrBook.GetPeers())
				peeraddrs := n.addrBook.GetPeers()

				if len(peeraddrs) != 0 {
					var testlist []string
					for _, peer := range peeraddrs {
						testlist = append(testlist, peer.String())
					}
					oklist := P2pComm.AddrTest(testlist)
					for _, addr := range oklist {

						if n.Has(addr) == false && n.nodeInfo.blacklist.Has(addr) == false {
							log.Debug("GetAddrFromOffline", "Add addr", addr)
							savelist[addr] = true
						}
						//log.Debug("getAddrFromOffline", "list", savelist)
					}
				}

				if len(savelist) == 0 {
					continue
				}

				go n.DialPeers(savelist)
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
