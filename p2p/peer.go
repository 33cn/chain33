// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	v "github.com/33cn/chain33/common/version"
	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// Start peer start
func (p *Peer) Start() {
	log.Debug("Peer", "Start", p.Addr())
	go p.heartBeat()
}

// Close peer close
func (p *Peer) Close() {
	//避免重复关闭
	if !atomic.CompareAndSwapInt32(&p.isclose, 0, 1) {
		return
	}
	p.mconn.Close()
	p.node.pubsub.Unsub(p.taskChan, "block", "tx")
	log.Info("Peer", "closed", p.Addr())

}

// Peer object information
type Peer struct {
	mutx         sync.Mutex
	node         *Node
	conn         *grpc.ClientConn // source connection
	persistent   bool
	isclose      int32
	version      *Version
	name         string //远程节点的name
	mconn        *MConnection
	peerAddr     *NetAddress
	peerStat     *Stat
	taskChan     chan interface{} //tx block
	inBounds     int32            //连接此节点的客户端节点数量
	IsMaxInbouds bool
}

// NewPeer produce a peer object
func NewPeer(conn *grpc.ClientConn, node *Node, remote *NetAddress) *Peer {
	p := &Peer{
		conn: conn,
		node: node,
	}
	p.peerStat = new(Stat)
	p.version = new(Version)
	p.version.SetSupport(true)
	p.mconn = NewMConnection(conn, remote, p)
	return p
}

// Version version object information
type Version struct {
	mtx            sync.Mutex
	version        int32
	versionSupport bool
}

// Stat object information
type Stat struct {
	mtx sync.Mutex
	ok  bool
}

// Ok start is ok
func (st *Stat) Ok() {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	st.ok = true
}

// NotOk start is not ok
func (st *Stat) NotOk() {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	st.ok = false
}

// IsOk start is ok or not
func (st *Stat) IsOk() bool {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	return st.ok
}

// SetSupport set support of version
func (v *Version) SetSupport(ok bool) {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	v.versionSupport = ok
}

// IsSupport is support version
func (v *Version) IsSupport() bool {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	return v.versionSupport
}

// SetVersion set version number
func (v *Version) SetVersion(ver int32) {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	v.version = ver
}

// GetVersion get version number
func (v *Version) GetVersion() int32 {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	return v.version
}

func (p *Peer) heartBeat() {

	pcli := NewNormalP2PCli()
	for {
		if !p.GetRunning() {
			return
		}
		peername, err := pcli.SendVersion(p, p.node.nodeInfo)
		P2pComm.CollectPeerStat(err, p)
		if err != nil || peername == "" {
			//版本不对，直接关掉
			log.Error("PeerHeartBeatSendVersion", "peerName", peername, "err", err)
			p.Close()
			return
		}

		log.Debug("sendVersion", "peer name", peername)
		p.SetPeerName(peername) //设置连接的远程节点的节点名称
		p.taskChan = p.node.pubsub.Sub("block", "tx", peername)
		go p.sendStream()
		go p.readStream()
		break
	}

	ticker := time.NewTicker(PingTimeout)
	defer ticker.Stop()
	for {
		if !p.GetRunning() {
			return
		}
		<-ticker.C
		peerNum, err := pcli.GetInPeersNum(p)
		if err == nil {
			atomic.StoreInt32(&p.inBounds, int32(peerNum))
		}
		err = pcli.SendPing(p, p.node.nodeInfo)
		if err != nil {
			log.Error("SendPeerPing", "peer", p.Addr(), "err", err)
		}
	}
}

// GetInBouns get inbounds of peer
func (p *Peer) GetInBouns() int32 {
	return atomic.LoadInt32(&p.inBounds)
}

// GetPeerInfo get peer information of peer
func (p *Peer) GetPeerInfo() (*pb.P2PPeerInfo, error) {
	return p.mconn.gcli.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: p.node.nodeInfo.channelVersion}, grpc.FailFast(true))
}

func (p *Peer) sendStream() {
	//Stream Send data
	for {
		if !p.GetRunning() {
			log.Info("sendStream peer connect closed", "peerid", p.GetPeerName())
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		resp, err := p.mconn.gcli.ServerStreamRead(ctx)
		P2pComm.CollectPeerStat(err, p)
		if err != nil {
			cancel()
			log.Error("sendStream", "ServerStreamRead", err)
			time.Sleep(time.Second * 5)
			continue
		}
		//send ping package
		ping, err := P2pComm.NewPingData(p.node.nodeInfo)
		if err != nil {
			errs := resp.CloseSend()
			if errs != nil {
				log.Error("CloseSend", "err", errs)
			}
			cancel()
			time.Sleep(time.Second)
			continue
		}
		p2pdata := new(pb.BroadCastData)
		p2pdata.Value = &pb.BroadCastData_Ping{Ping: ping}
		if err := resp.Send(p2pdata); err != nil {
			P2pComm.CollectPeerStat(err, p)
			errs := resp.CloseSend()
			if errs != nil {
				log.Error("CloseSend", "err", errs)
			}
			cancel()
			log.Error("sendStream", "sendping", err)
			time.Sleep(time.Second)
			continue
		}

		//send softversion&p2pversion
		_, peername := p.node.nodeInfo.addrBook.GetPrivPubKey()
		p2pdata.Value = &pb.BroadCastData_Version{Version: &pb.Versions{P2Pversion: p.node.nodeInfo.channelVersion,
			Softversion: v.GetVersion(), Peername: peername}}

		if err := resp.Send(p2pdata); err != nil {
			P2pComm.CollectPeerStat(err, p)
			errs := resp.CloseSend()
			if errs != nil {
				log.Error("CloseSend", "err", errs)
			}
			cancel()
			log.Error("sendStream", "sendversion", err)
			continue
		}
		timeout := time.NewTimer(time.Second * 2)
		defer timeout.Stop()
	SEND_LOOP:
		for {

			if !p.GetRunning() {
				return
			}
			select {
			case task := <-p.taskChan:
				if !p.GetRunning() {
					errs := resp.CloseSend()
					if errs != nil {
						log.Error("CloseSend", "err", errs)
					}
					cancel()
					log.Error("sendStream peer connect closed", "peerName", p.GetPeerName())
					return
				}
				sendData, doSend := p.node.processSendP2P(task, p.version.GetVersion(), p.Addr())
				if !doSend {
					continue
				}
				err := resp.Send(sendData)
				P2pComm.CollectPeerStat(err, p)
				if err != nil {
					log.Error("sendStream", "send", err)
					if grpc.Code(err) == codes.Unimplemented { //maybe order peers delete peer to BlackList
						p.node.nodeInfo.blacklist.Add(p.Addr(), 3600)
					}
					time.Sleep(time.Second) //have a rest
					errs := resp.CloseSend()
					if errs != nil {
						log.Error("CloseSend", "err", errs)
					}
					cancel()

					break SEND_LOOP //下一次外循环重新获取stream
				}
				log.Debug("sendStream", "send data", "ok")

			case <-timeout.C:
				if !p.GetRunning() {
					log.Error("sendStream timeout")
					errs := resp.CloseSend()
					if errs != nil {
						log.Error("CloseSend", "err", errs)
					}
					cancel()
					return
				}
				timeout.Reset(time.Second * 2)
			}
		}
	}
}

func (p *Peer) readStream() {

	for {
		if !p.GetRunning() {
			log.Debug("readstream", "loop", "done")
			return
		}
		ping, err := P2pComm.NewPingData(p.node.nodeInfo)
		if err != nil {
			log.Error("readStream", "err:", err.Error())
			continue
		}
		resp, err := p.mconn.gcli.ServerStreamSend(context.Background(), ping)
		P2pComm.CollectPeerStat(err, p)
		if err != nil {
			log.Error("readStream", "serverstreamsend,err:", err, "peer", p.Addr())
			time.Sleep(time.Second)
			continue
		}

		log.Debug("SubStreamBlock", "Start", p.Addr())
		for {
			if !p.GetRunning() {
				errs := resp.CloseSend()
				if errs != nil {
					log.Error("CloseSend", "err", errs)
				}
				return
			}

			data, err := resp.Recv()
			P2pComm.CollectPeerStat(err, p)
			if err != nil {
				log.Error("readStream", "recv,err:", err.Error())
				errs := resp.CloseSend()
				if errs != nil {
					log.Error("CloseSend", "err", errs)
				}
				if grpc.Code(err) == codes.Unimplemented { //maybe order peers delete peer to BlackList
					p.node.nodeInfo.blacklist.Add(p.Addr(), 3600)
				}
				//beyound max inbound num
				if strings.Contains(err.Error(), "beyound max inbound num") {
					log.Info("readStream", "peer inbounds num", p.GetInBouns())
					p.IsMaxInbouds = true
					P2pComm.CollectPeerStat(err, p)
				}
				time.Sleep(time.Second) //have a rest
				break
			}

			p.node.processRecvP2P(data, p.GetPeerName(), p.node.pubToPeer, p.Addr())
		}
	}
}

// GetRunning get running ok or not
func (p *Peer) GetRunning() bool {
	if p.node.isClose() {
		return false
	}

	return atomic.LoadInt32(&p.isclose) != 1

}

// MakePersistent marks the peer as persistent.
func (p *Peer) MakePersistent() {

	p.persistent = true
}

// SetAddr set address of peer
func (p *Peer) SetAddr(addr *NetAddress) {
	p.peerAddr = addr
}

// Addr returns peer's remote network address.
func (p *Peer) Addr() string {
	return p.peerAddr.String()
}

// IsPersistent returns true if the peer is persitent, false otherwise.
func (p *Peer) IsPersistent() bool {
	return p.persistent
}

// SetPeerName set name of peer
func (p *Peer) SetPeerName(name string) {
	p.mutx.Lock()
	defer p.mutx.Unlock()
	if name == "" {
		return
	}
	p.name = name
}

// GetPeerName get name of peer
func (p *Peer) GetPeerName() string {
	p.mutx.Lock()
	defer p.mutx.Unlock()
	return p.name
}
