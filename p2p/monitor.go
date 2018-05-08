package p2p

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
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
		if !peer.version.IsSupport() { //如果版本不支持,直接删除节点
			log.Debug("VersoinMonitor", "NotSupport,addr", peer.Addr())
			n.destroyPeer(peer)
			//加入黑名单24小时
			n.nodeInfo.blacklist.Add(peer.Addr(), int64(time.Duration(time.Hour*24)))
			continue
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
	ticker := time.NewTicker(GetAddrFromGitHubInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		if n.isClose() {
			log.Debug("getAddrFromGithub", "loop", "done")
			return
		}
		if n.needMore() {
			//从github 上下载种子节点文件
			res, err := http.Get("https://raw.githubusercontent.com/chainseed/seeds/master/bty.txt")
			if err != nil {
				log.Error("getAddrFromGithub", "http.Get", err.Error())
				continue
			}

			bf := new(bytes.Buffer)
			_, err = io.Copy(bf, res.Body)
			if err != nil {
				log.Error("getAddrFromGithub", "io.Copy", err.Error())
				continue
			}

			fileContent := bf.String()
			st := strings.TrimSpace(fileContent)
			strs := strings.Split(st, "\n")
			log.Info("getAddrFromGithub", "download file", fileContent)
			for _, linestr := range strs {
				pidaddr := strings.Split(linestr, "@")
				if len(pidaddr) == 2 {
					addr := pidaddr[1]
					if n.Has(addr) || n.nodeInfo.blacklist.Has(addr) ||
						len(P2pComm.AddrRouteble([]string{addr})) == 0 {
						continue
					}
					pub.FIFOPub(addr, "addr")

				}
			}
		}

	}
}

func (n *Node) getAddrFromOnline() {
	ticker := time.NewTicker(GetAddrFromOnlineInterval)
	defer ticker.Stop()
	pcli := NewNormalP2PCli()

	for {
		<-ticker.C
		if n.isClose() {
			log.Debug("GetAddrFromOnLine", "loop", "done")
			return
		}
		if n.needMore() {
			peers, _ := n.GetActivePeers()
			for _, peer := range peers { //向其他节点发起请求，获取地址列表
				log.Debug("Getpeer", "addr", peer.Addr())
				var addrlist []string
				var err error
				if peer.version.GetVersion() >= VERSION {
					log.Info("peer", "VERSION", peer.version.GetVersion())
					addrlist, err = pcli.GetAddrList(peer)
				} else {
					addrlist, err = pcli.GetAddr(peer)
				}

				P2pComm.CollectPeerStat(err, peer)
				if err != nil {
					log.Error("getAddrFromOnline", "ERROR", err.Error())
					continue
				}

				log.Debug("GetAddrFromOnline", "addrlist", addrlist)
				//过滤黑名单的地址
				for _, addr := range addrlist {
					if !n.nodeInfo.blacklist.Has(addr) {
						pub.FIFOPub(addr, "addr")
					} else {
						log.Debug("Filter addr", "BlackList", addr)
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
		<-ticker.C
		if n.isClose() {
			log.Debug("GetAddrFromOnLine", "loop", "done")
			return
		}
		if n.needMore() {
			var testlist []string
			//随机选择种子进行连接
			seeds := n.nodeInfo.cfg.GetSeeds()
			index := rand.Intn(len(seeds))
			if !n.Has(seeds[index]) && !n.nodeInfo.blacklist.Has(seeds[index]) {

				testlist = append(testlist, seeds[index])
			}
			log.Debug("OUTBOUND NUM", "NUM", n.Size(), "start getaddr from peer", n.nodeInfo.addrBook.GetPeers())
			peeraddrs := n.nodeInfo.addrBook.GetPeers()

			if len(peeraddrs) != 0 {
				for _, peer := range peeraddrs {
					testlist = append(testlist, peer.String())
				}
			}
			//log.Info("getaddrfromoffline")
			for _, addr := range testlist {

				if !n.Has(addr) && !n.nodeInfo.blacklist.Has(addr) {
					log.Debug("GetAddrFromOffline", "Add addr", addr)
					pub.FIFOPub(addr, "addr")
				}
			}

		} else {

			for _, seed := range n.nodeInfo.cfg.Seeds {
				//如果达到稳定节点数量，则断开种子节点
				if n.Has(seed) {
					if !n.needMore() && len(n.GetRegisterPeers()) > stableBoundNum {
						n.remove(seed)
					}

				}
			}

			//如果删除种子节点后，依然有过多的节点连接，则继续删除其他非种子节点
			if !n.needMore() && len(n.GetRegisterPeers()) > stableBoundNum {
				peerinfos := n.nodeInfo.peerInfos.GetPeerInfos()
				//把连接的高度最低的节点删掉
				var lowest int64
				var lowestPeer string
				var index int
				for peeraddr, pbpeer := range peerinfos {
					if index == 0 {
						lowest = pbpeer.GetHeader().GetHeight()
					}
					if lowest > pbpeer.GetHeader().GetHeight() {
						lowest = pbpeer.GetHeader().GetHeight()
						lowestPeer = peeraddr
					}
					index++
				}

				n.remove(lowestPeer)
			}

		}

		log.Debug("Node Monitor process", "outbound num", n.Size())
	}

}

func (n *Node) monitorPeersHeight() {

	p2pcli := NewNormalP2PCli()
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	_, pname := n.nodeInfo.addrBook.GetPrivPubKey()
	for {

		<-ticker.C
		localBlockHeight, err := p2pcli.GetBlockHeight(n.nodeInfo)
		if err != nil {
			continue
		}

		peers, infos := n.GetActivePeers()
		for paddr, pinfo := range infos {
			peerheight := pinfo.GetHeader().GetHeight()
			if pinfo.GetName() == pname { //发现连接到自己，立即删除
				//删除节点数过低的节点
				n.remove(paddr)
				n.nodeInfo.blacklist.Add(paddr, 0)
			}

			if localBlockHeight-peerheight > 2048 { //比自己较低的节点删除
				if addrlist, err := p2pcli.GetAddrList(peers[paddr]); err == nil {

					for _, addr := range addrlist {
						if !n.Has(addr) && !n.nodeInfo.blacklist.Has(addr) {
							pub.FIFOPub(addr, "addr")
						}
					}

				}
				//删除节点数过低的节点
				n.remove(paddr)
				//短暂加入黑名单20分钟
				n.nodeInfo.blacklist.Add(paddr, int64(time.Duration(time.Minute*20)))

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

func (n *Node) monitorDialPeers() {

	addrChan := pub.Sub("addr")
	for addr := range addrChan {
		if n.isClose() {
			log.Info("monitorDialPeers", "loop", "done")
			return
		}
		netAddr, err := NewNetAddressString(addr.(string))
		if err != nil {
			continue
		}

		if n.nodeInfo.addrBook.ISOurAddress(netAddr) {
			continue
		}

		//不对已经连接上的地址重新发起连接
		if n.Has(netAddr.String()) {
			log.Debug("DialPeers", "find hash", netAddr.String())
			continue
		}

		if !n.needMore() || len(n.GetRegisterPeers()) > maxOutBoundNum { //注册的节点超过最大节点数暂不连接
			time.Sleep(time.Second * 10)
			continue
		}
		log.Info("DialPeers", "peer", netAddr.String())
		peer, err := P2pComm.dialPeer(netAddr, &n.nodeInfo)
		if err != nil {
			log.Error("monitorDialPeers", "Err", err.Error())
			n.nodeInfo.blacklist.Add(netAddr.String(), int64(time.Duration(time.Minute*10)))
			continue
		}
		n.addPeer(peer)
		n.nodeInfo.addrBook.AddAddress(netAddr, nil)

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
		now := time.Now().Unix()
		for badPeer, intime := range badPeers {
			if n.nodeInfo.addrBook.IsOurStringAddress(badPeer) {
				continue
			}
			if 0 == intime {
				continue //0表示永久加入黑名单
			}
			if now-intime > 0 {
				n.nodeInfo.blacklist.Delete(badPeer)
			}
		}

	}
}

func (n *Node) monitorFilter() {
	Filter.ManageRecvFilter()
}
