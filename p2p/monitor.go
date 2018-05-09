package p2p

import (
	"bytes"
	"io"
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
		if !peer.version.IsSupport() {
			//如果版本不支持,直接删除节点
			log.Debug("VersoinMonitor", "NotSupport,addr", peer.Addr())
			n.destroyPeer(peer)
			//加入黑名单12小时
			n.nodeInfo.blacklist.Add(peer.Addr(), int64(3600*12))
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
	if !n.nodeInfo.cfg.GetUseGithub() {
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
			if n.Has(addr) || n.nodeInfo.blacklist.Has(addr) ||
				len(P2pComm.AddrRouteble([]string{addr})) == 0 {
				return
			}
			pub.FIFOPub(addr, "addr")

		}
	}
}

//从在线节点获取地址列表
func (n *Node) getAddrFromOnline() {
	ticker := time.NewTicker(GetAddrFromOnlineInterval)
	defer ticker.Stop()
	pcli := NewNormalP2PCli()

	var ticktimes int
	for {

		<-ticker.C

		seedsMap := make(map[string]bool)
		//每次循环seed的排序不同
		seedArr := n.nodeInfo.cfg.GetSeeds()
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
			for addr, _ := range seedsMap {
				//先把seed 排除在外
				rangeCount++
				if rangeCount < maxOutBoundNum {
					pub.FIFOPub(addr, "addr")
				}

			}
			continue
		}

		peers, _ := n.GetActivePeers()
		for _, peer := range peers { //向其他节点发起请求，获取地址列表

			var addrlist []string
			var addrlistMap map[string]int64
			var err error

			if peer.version.GetVersion() >= VERSION {
				addrlistMap, err = pcli.GetAddrList(peer)
				for addr, _ := range addrlistMap {
					addrlist = append(addrlist, addr)
				}
			} else {
				//老版本
				addrlist, err = pcli.GetAddr(peer)
			}

			P2pComm.CollectPeerStat(err, peer)
			if err != nil {
				log.Error("getAddrFromOnline", "ERROR", err.Error())
				continue
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
						if peerHeight >= localBlockHeight || localBlockHeight-peerHeight > 1024 {
							//随机删除连接的一个种子
							for _, seed := range seedArr {
								if n.Has(seed) {
									n.remove(seed)
									break
								}
							}

						}
					}

				}

				if !n.nodeInfo.blacklist.Has(addr) || !Filter.QueryRecvData(addr) {
					if ticktimes < 10 {
						//如果连接了其他节点，优先不连接种子节点
						if _, ok := seedsMap[addr]; !ok {
							//先把seed 排除在外
							pub.FIFOPub(addr, "addr")

						}
					} else {
						pub.FIFOPub(addr, "addr")
					}

				}
			}

		}

	}
}

//从addrBook 中获取地址
func (n *Node) getAddrFromAddrBook() {
	ticker := time.NewTicker(GetAddrFromAddrBookInterval)
	defer ticker.Stop()
	var tickerTimes int64
	seedsMap := make(map[string]bool) //每次循环seed的排序不同
	seedArr := n.nodeInfo.cfg.GetSeeds()
	for _, seed := range seedArr {
		seedsMap[seed] = true
	}

	for {
		<-ticker.C
		tickerTimes++
		if n.isClose() {
			log.Debug("GetAddrFromOnLine", "loop", "done")
			return
		}
		//5个循环后， 则去github下载
		if tickerTimes > 5 && n.Size() == 0 {
			n.getAddrFromGithub()
			tickerTimes = 0
		}

		log.Debug("OUTBOUND NUM", "NUM", n.Size(), "start getaddr from peer", n.nodeInfo.addrBook.GetPeers())

		addrNetArr := n.nodeInfo.addrBook.GetPeers()

		for _, addr := range addrNetArr {
			if !n.Has(addr.String()) && !n.nodeInfo.blacklist.Has(addr.String()) {
				log.Debug("GetAddrFromOffline", "Add addr", addr.String())

				if _, ok := seedsMap[addr.String()]; !ok {
					if n.needMore() {
						pub.FIFOPub(addr.String(), "addr")
					}

				}
			}
		}
		log.Debug("Node Monitor process", "outbound num", n.Size())
	}

}

func (n *Node) monitorPeers() {

	p2pcli := NewNormalP2PCli()
	ticker := time.NewTicker(time.Second * 30)
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
			if pinfo.GetName() == selfName { //发现连接到自己，立即删除
				//删除节点数过低的节点
				n.remove(paddr)
				n.nodeInfo.blacklist.Add(paddr, 0)
			}

			if localBlockHeight-peerheight > 2048 { //比自己较低的节点删除
				if addrMap, err := p2pcli.GetAddrList(peers[paddr]); err == nil {

					for addr := range addrMap {
						if !n.Has(addr) && !n.nodeInfo.blacklist.Has(addr) {
							pub.FIFOPub(addr, "addr")
						}
					}

				}
				//删除节点数过低的节点
				n.remove(paddr)
				//短暂加入黑名单20分钟
				n.nodeInfo.blacklist.Add(paddr, int64(60*20))

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

//并发连接节点地址
func (n *Node) monitorDialPeers() {

	addrChan := pub.Sub("addr")
	for addr := range addrChan {

		if n.isClose() {
			log.Info("monitorDialPeers", "loop", "done")
			return
		}
		if Filter.QueryRecvData(addr.(string)) {
			//先查询有没有注册进去，避免同时重复连接相同的地址
			continue
		}

		Filter.RegRecvData(addr.(string))
		//把待连接的节点增加到过滤容器中
		netAddr, err := NewNetAddressString(addr.(string))
		if err != nil {
			continue
		}

		if n.nodeInfo.addrBook.ISOurAddress(netAddr) {
			continue
		}

		//不对已经连接上的地址重新发起连接
		if n.Has(netAddr.String()) || n.nodeInfo.blacklist.Has(netAddr.String()) {
			log.Debug("DialPeers", "find hash", netAddr.String())
			continue
		}

		//注册的节点超过最大节点数暂不连接
		if !n.needMore() || len(n.GetRegisterPeers()) > maxOutBoundNum {
			time.Sleep(time.Second * 10)
			continue
		}
		log.Info("DialPeers", "peer", netAddr.String())
		//并发连接节点，增加连接效率
		go func(netAddr *NetAddress) {

			defer Filter.RemoveRecvData(netAddr.String())
			peer, err := P2pComm.dialPeer(netAddr, &n.nodeInfo)
			if err != nil {
				//连接失败后
				Filter.RemoveRecvData(netAddr.String())
				log.Error("monitorDialPeers", "Err", err.Error())
				n.nodeInfo.blacklist.Add(netAddr.String(), int64(60*10))
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
		now := time.Now().Unix()
		for badPeer, intime := range badPeers {
			if n.nodeInfo.addrBook.IsOurStringAddress(badPeer) {
				continue
			}
			//0表示永久加入黑名单
			if 0 == intime {
				continue
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
