package p2p

import (
	"net"
	"time"
)

func (n *Node) AddrTest(addrs []string) []string {
	var enableAddrs []string
	for _, addr := range addrs {

		conn, err := net.DialTimeout("tcp", addr, time.Second*1)
		if err == nil {
			conn.Close()
			enableAddrs = append(enableAddrs, addr)
		}
	}

	return enableAddrs

}
func (n *Node) checkActivePeers() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-n.loopDone:
			log.Error("checkActivePeers", "loop", "done")
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
						log.Error("checkActivePeers", "Delete peer", peer.Addr(), "Attemps", stat.GetAttempts(), "ISRUNNING", peer.GetRunning())

						n.destroyPeer(peer)
					}
				}

			}
		}

	}
}
func (n *Node) destroyPeer(peer *peer) {
	log.Error("deleteErrPeer", "Delete peer", peer.Addr(), "RUNNING", peer.GetRunning(), "IsSuuport", peer.version.IsSupport())
	n.addrBook.RemoveAddr(peer.Addr())
	n.addrBook.Save()
	n.Remove(peer.Addr())
}
func (n *Node) monitorErrPeer() {

	for {

		peer := <-n.nodeInfo.monitorChan
		log.Debug("deleteErrPeer", "REMOVE", peer.Addr())
		if peer.version.IsSupport() == false { //如果版本不支持，则加入黑名单，下次不再发起连接

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
			log.Error("GetAddrFromOnLine", "loop", "Done")
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
					//log.Error("getAddrFromOnline", "ADDRLIST", addrlist)
					//过滤无法连接的节点

					//过滤黑名单的地址
					oklist := n.AddrTest(addrlist)
					var whitlist = make(map[string]bool)
					for _, addr := range oklist {
						if n.nodeInfo.blacklist.Has(addr) == false {
							whitlist[addr] = true
						} else {
							log.Warn("Filter addr", "BlackList", addr)
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
			log.Error("GetAddrFromOffLine", "loop", "Done")
			break FOR_LOOP
		case <-ticker.C:
			if n.needMore() {
				var savelist = make(map[string]bool)
				for _, seed := range n.nodeInfo.cfg.Seeds {
					if n.Has(seed) == false && n.nodeInfo.blacklist.Has(seed) == false {
						log.Info("GetAddrFromOffline", "Add Seed", seed)
						savelist[seed] = true
					}
				}

				log.Info("OUTBOUND NUM", "NUM", n.Size(), "start getaddr from peer", n.addrBook.GetPeers())
				peeraddrs := n.addrBook.GetPeers()

				if len(peeraddrs) != 0 {
					var testlist []string
					for _, peer := range peeraddrs {
						testlist = append(testlist, peer.String())
					}
					oklist := n.AddrTest(testlist)
					for _, addr := range oklist {

						if n.Has(addr) == false && n.nodeInfo.blacklist.Has(addr) == false {
							log.Info("GetAddrFromOffline", "Add addr", addr)
							savelist[addr] = true
						}
						//log.Error("getAddrFromOffline", "list", savelist)
					}
				}

				if len(savelist) == 0 {
					continue
				}

				go n.DialPeers(savelist)
			} else {
				log.Info("getAddrFromOffline", "nodestable", n.needMore())
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
