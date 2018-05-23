package p2p

import (
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	pb "gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (p *Peer) Start() {
	log.Debug("Peer", "Start", p.Addr())
	go p.heartBeat()
}

func (p *Peer) Close() {
	atomic.StoreInt32(&p.isclose, 1)
	p.mconn.Close()
	pub.Unsub(p.taskChan, "block", "tx")
	log.Info("Peer", "closed", p.Addr())

}

type Peer struct {
	mutx       sync.Mutex
	nodeInfo   **NodeInfo
	conn       *grpc.ClientConn // source connection
	persistent bool
	isclose    int32
	version    *Version
	name       string //远程节点的name
	mconn      *MConnection
	peerAddr   *NetAddress
	peerStat   *Stat
	taskChan   chan interface{} //tx block
	inBounds   int32            //连接此节点的客户端节点数量
}

func NewPeer(conn *grpc.ClientConn, nodeinfo **NodeInfo, remote *NetAddress) *Peer {
	p := &Peer{
		conn:     conn,
		nodeInfo: nodeinfo,
	}

	p.peerStat = new(Stat)
	p.version = new(Version)
	p.version.SetSupport(true)
	p.mconn = NewMConnection(conn, remote, p)
	return p
}

type Version struct {
	mtx            sync.Mutex
	version        int32
	versionSupport bool
}
type Stat struct {
	mtx sync.Mutex
	ok  bool
}

func (st *Stat) Ok() {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	st.ok = true
}

func (st *Stat) NotOk() {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	st.ok = false
}

func (st *Stat) IsOk() bool {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	return st.ok
}

func (v *Version) SetSupport(ok bool) {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	v.versionSupport = ok
}

func (v *Version) IsSupport() bool {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	return v.versionSupport
}

func (v *Version) SetVersion(ver int32) {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	v.version = ver
}

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
		peername, err := pcli.SendVersion(p, *p.nodeInfo)
		P2pComm.CollectPeerStat(err, p)
		if err == nil {
			log.Debug("sendVersion", "peer name", peername)
			p.SetPeerName(peername) //设置连接的远程节点的节点名称
			p.taskChan = pub.Sub("block", "tx")
			go p.sendStream()
			go p.readStream()
			break
		} else {
			time.Sleep(time.Second * 5)
			continue
		}
	}

	ticker := time.NewTicker(PingTimeout)
	defer ticker.Stop()
	for {
		if !p.GetRunning() {
			return
		}

		<-ticker.C
		err := pcli.SendPing(p, *p.nodeInfo)
		P2pComm.CollectPeerStat(err, p)
		peernum, err := pcli.GetInPeersNum(p)
		P2pComm.CollectPeerStat(err, p)
		atomic.StoreInt32(&p.inBounds, int32(peernum))
	}
}

func (p *Peer) GetInBouns() int32 {
	return atomic.LoadInt32(&p.inBounds)
}

func (p *Peer) GetPeerInfo(version int32) (*pb.P2PPeerInfo, error) {
	return p.mconn.gcli.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: version}, grpc.FailFast(true))
}

func (p *Peer) sendStream() {
	//Stream Send data
	for {
		if !p.GetRunning() {
			log.Info("sendStream peer is not running")
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
		ping, err := P2pComm.NewPingData(*p.nodeInfo)
		if err != nil {
			resp.CloseSend()
			cancel()
			time.Sleep(time.Second)
			continue
		}
		p2pdata := new(pb.BroadCastData)
		p2pdata.Value = &pb.BroadCastData_Ping{Ping: ping}
		if err := resp.Send(p2pdata); err != nil {
			resp.CloseSend()
			cancel()
			log.Error("sendStream", "sendping", err)
			time.Sleep(time.Second)
			continue
		}

		timeout := time.NewTimer(time.Second * 2)
		defer timeout.Stop()
		var hash [64]byte
	SEND_LOOP:
		for {

			select {
			case task := <-p.taskChan:
				if !p.GetRunning() {
					resp.CloseSend()
					cancel()
					log.Error("sendStream peer is not running")
					return
				}
				p2pdata := new(pb.BroadCastData)
				if block, ok := task.(*pb.P2PBlock); ok {
					height := block.GetBlock().GetHeight()
					hex.Encode(hash[:], block.GetBlock().Hash())
					blockhash := string(hash[:])
					log.Debug("sendStream", "will send block", blockhash)
					pinfo, err := p.GetPeerInfo((*p.nodeInfo).cfg.GetVersion())
					P2pComm.CollectPeerStat(err, p)
					if err == nil {
						if pinfo.GetHeader().GetHeight() >= height {
							log.Debug("sendStream", "find peer height>this broadblock ,send process", "break")
							continue
						}
					}

					p2pdata.Value = &pb.BroadCastData_Block{Block: block}
					Filter.RegRecvData(blockhash)

				} else if tx, ok := task.(*pb.P2PTx); ok {
					hex.Encode(hash[:], tx.GetTx().Hash())
					txhash := string(hash[:])
					log.Debug("sendStream", "will send tx", txhash)
					p2pdata.Value = &pb.BroadCastData_Tx{Tx: tx}
					Filter.RegRecvData(txhash)
				}

				err := resp.Send(p2pdata)
				P2pComm.CollectPeerStat(err, p)
				if err != nil {
					log.Error("sendStream", "send", err)
					if grpc.Code(err) == codes.Unimplemented { //maybe order peers delete peer to BlackList
						(*p.nodeInfo).blacklist.Add(p.Addr(), 3600)
					}
					time.Sleep(time.Second) //have a rest
					resp.CloseSend()
					cancel()

					break SEND_LOOP //下一次外循环重新获取stream
				}
				log.Debug("sendStream", "send data", "ok")

			case <-timeout.C:
				if !p.GetRunning() {
					log.Error("sendStream timeout")
					resp.CloseSend()
					cancel()
					return
				}
				timeout.Reset(time.Second * 2)

			}
		}

	}
}

func (p *Peer) readStream() {

	pcli := NewNormalP2PCli()

	for {
		if !p.GetRunning() {
			log.Debug("readstream", "loop", "done")
			return
		}
		ping, err := P2pComm.NewPingData(*p.nodeInfo)
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
		var hash [64]byte
		for {
			if !p.GetRunning() {
				resp.CloseSend()
				return
			}
			data, err := resp.Recv()
			P2pComm.CollectPeerStat(err, p)
			if err != nil {
				log.Error("readStream", "recv,err:", err)
				resp.CloseSend()
				if grpc.Code(err) == codes.Unimplemented { //maybe order peers delete peer to BlackList
					(*p.nodeInfo).blacklist.Add(p.Addr(), 3600)
				}
				time.Sleep(time.Second) //have a rest
				break
			}

			if block := data.GetBlock(); block != nil {
				if block.GetBlock() != nil {
					//如果已经有登记过的消息记录，则不发送给本地blockchain
					hex.Encode(hash[:], block.GetBlock().Hash())
					blockhash := string(hash[:])
					if Filter.QueryRecvData(blockhash) {
						continue
					}

					//判断比自己低的区块，则不发送给blockchain

					height, err := pcli.GetBlockHeight((*p.nodeInfo))
					if err == nil {
						if height >= block.GetBlock().GetHeight() {
							continue
						}
					}

					log.Info("readStream", "block==+======+====+=>Height", block.GetBlock().GetHeight(), "from peer", p.Addr(),
						"block size(KB)", float32(len(pb.Encode(block)))/1024, "block hash",
						blockhash)
					msg := (*p.nodeInfo).client.NewMessage("blockchain", pb.EventBroadcastAddBlock, &pb.BlockPid{p.GetPeerName(), block.GetBlock()})
					err = (*p.nodeInfo).client.Send(msg, false)
					if err != nil {
						log.Error("readStream", "send to blockchain Error", err.Error())
						continue
					}
					Filter.RegRecvData(blockhash) //添加发送登记，下次通过stream 接收同样的消息的时候可以过滤
				}

			} else if tx := data.GetTx(); tx != nil {

				if tx.GetTx() != nil {
					hex.Encode(hash[:], tx.Tx.Hash())
					txhash := string(hash[:])
					log.Debug("readStream", "tx", txhash)
					if Filter.QueryRecvData(txhash) {
						continue //处理方式同上
					}
					msg := (*p.nodeInfo).client.NewMessage("mempool", pb.EventTx, tx.GetTx())
					(*p.nodeInfo).client.Send(msg, false)
					Filter.RegRecvData(txhash) //登记
				}
			}
		}
	}
}

func (p *Peer) GetRunning() bool {
	return atomic.LoadInt32(&p.isclose) != 1

}

// makePersistent marks the peer as persistent.
func (p *Peer) MakePersistent() {

	p.persistent = true
}

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

func (p *Peer) SetPeerName(name string) {
	p.mutx.Lock()
	defer p.mutx.Unlock()
	if name == "" {
		return
	}
	p.name = name
}

func (p *Peer) GetPeerName() string {
	p.mutx.Lock()
	defer p.mutx.Unlock()
	return p.name
}
