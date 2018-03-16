package p2p

import (
	"encoding/hex"
	"sync"
	"time"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (p *peer) Start() {
	log.Debug("Peer", "Start", p.Addr())
	p.mconn.key = p.key
	go p.heartBeat()

	return
}
func (p *peer) Close() {
	p.SetRunning(false)
	p.mconn.Close()
	close(p.taskPool)
	close(p.filterTask.loopDone)
	pub.Unsub(p.taskChan, "block", "tx")
}

type peer struct {
	wg         sync.WaitGroup
	pmutx      sync.Mutex
	nodeInfo   **NodeInfo
	conn       *grpc.ClientConn // source connection
	persistent bool
	isrunning  bool
	version    *Version
	key        string
	mconn      *MConnection
	peerAddr   *NetAddress
	peerStat   *Stat
	filterTask *FilterTask
	taskPool   chan struct{}
	taskChan   chan interface{} //tx block
}

func NewPeer(conn *grpc.ClientConn, nodeinfo **NodeInfo, remote *NetAddress) *peer {
	p := &peer{
		conn:     conn,
		taskPool: make(chan struct{}, 50),
		nodeInfo: nodeinfo,
	}
	p.filterTask = new(FilterTask)
	p.filterTask.loopDone = make(chan struct{}, 1)
	p.filterTask.regTask = make(map[interface{}]time.Duration)
	p.peerStat = new(Stat)
	p.version = new(Version)
	p.version.SetSupport(true)
	p.SetRunning(true)
	p.mconn = NewMConnection(conn, remote, p)
	return p
}

type FilterTask struct {
	mtx      sync.Mutex
	loopDone chan struct{}
	regTask  map[interface{}]time.Duration
}
type Version struct {
	mtx            sync.Mutex
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
func (f *FilterTask) RegTask(key interface{}) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.regTask[key] = time.Duration(time.Now().Unix())
}
func (f *FilterTask) QueryTask(key interface{}) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	_, ok := f.regTask[key]
	return ok

}
func (f *FilterTask) RemoveTask(key interface{}) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	delete(f.regTask, key)
}

func (f *FilterTask) ManageFilterTask() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {

		select {
		case <-f.loopDone:
			log.Debug("peer mangerFilterTask", "loop", "done")
			return
		case <-ticker.C:
			f.mtx.Lock()
			now := time.Now().Unix()
			for key, regtime := range f.regTask {
				if now-int64(regtime) > 50 {
					delete(f.regTask, key)
				}
			}
			f.mtx.Unlock()

		}
	}
}

func (p *peer) heartBeat() {

	pcli := NewP2pCli(nil)
	for {
		if p.GetRunning() == false {
			return
		}
		err := pcli.SendVersion(p, *p.nodeInfo)
		P2pComm.CollectPeerStat(err, p)
		if err == nil {
			p.taskChan = pub.Sub("block", "tx")
			go p.filterTask.ManageFilterTask()
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
		if p.GetRunning() == false {
			return
		}
		select {
		case <-ticker.C:
			err := pcli.SendPing(p, *p.nodeInfo)
			P2pComm.CollectPeerStat(err, p)

		}

	}

}

func (p *peer) GetPeerInfo(version int32) (*pb.P2PPeerInfo, error) {
	return p.mconn.conn.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: version})
}

func (p *peer) sendStream() {
	//Stream Send data
	for {
		if p.GetRunning() == false {
			log.Info("sendStream peer is not running")
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		resp, err := p.mconn.conn.ServerStreamRead(ctx)
		P2pComm.CollectPeerStat(err, p)
		if err != nil {
			cancel()
			log.Error("sendStream", "CollectPeerStat", err)
			time.Sleep(time.Second * 5)
			continue
		}
		//send ping package
		ping, err := P2pComm.NewPingData(p)
		if err == nil {
			p2pdata := new(pb.BroadCastData)
			p2pdata.Value = &pb.BroadCastData_Ping{Ping: ping}
			if err := resp.Send(p2pdata); err != nil {
				resp.CloseSend()
				cancel()
				log.Error("sendStream", "sendping", err)
				time.Sleep(time.Second)
				continue
			}
		}

		timeout := time.NewTimer(time.Second * 2)
		defer timeout.Stop()
	SEND_LOOP:
		for {

			select {
			case task := <-p.taskChan:
				if p.GetRunning() == false {
					resp.CloseSend()
					cancel()
					log.Error("sendStream peer is not running")
					return
				}
				p2pdata := new(pb.BroadCastData)
				if block, ok := task.(*pb.P2PBlock); ok {
					height := block.GetBlock().GetHeight()
					pinfo, err := p.GetPeerInfo((*p.nodeInfo).cfg.GetVersion())
					if err == nil {
						if pinfo.GetHeader().GetHeight() >= height {
							timeout.Stop()
							continue
						}
					}
					if p.filterTask.QueryTask(height) == true {
						timeout.Stop()
						continue //已经接收的消息不再发送
					}
					p2pdata.Value = &pb.BroadCastData_Block{Block: block}
					//登记新的发送消息
					p.filterTask.RegTask(height)

				} else if tx, ok := task.(*pb.P2PTx); ok {
					sig := tx.GetTx().GetSignature().GetSignature()
					if p.filterTask.QueryTask(hex.EncodeToString(sig)) == true {
						continue
					}
					p2pdata.Value = &pb.BroadCastData_Tx{Tx: tx}
					p.filterTask.RegTask(hex.EncodeToString(sig))
				}
				err := resp.Send(p2pdata)
				if err != nil {
					log.Error("sendStream", "send", err)
					p.peerStat.NotOk()
					if grpc.Code(err) == codes.Unimplemented { //maybe order peers delete peer to BlackList
						(*p.nodeInfo).blacklist.Add(p.Addr())
					}
					time.Sleep(time.Second) //have a rest
					(*p.nodeInfo).monitorChan <- p
					resp.CloseSend()
					cancel()
					break SEND_LOOP //下一次外循环重新获取stream
				}
			case <-timeout.C:
				if p.GetRunning() == false {
					log.Error("sendStream timeout")
					resp.CloseSend()
					cancel()

					return
				}
			}
		}

	}
}

func (p *peer) readStream() {

	pcli := NewP2pCli(nil)

	for {
		if p.GetRunning() == false {
			return
		}
		ping, err := P2pComm.NewPingData(p)
		if err != nil {
			log.Error("readStream", "err:", err.Error())
			continue
		}
		resp, err := p.mconn.conn.ServerStreamSend(context.Background(), ping)
		P2pComm.CollectPeerStat(err, p)
		if err != nil {
			log.Error("readStream", "serverstreamsend,err:", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Debug("SubStreamBlock", "Start", p.Addr())

		for {
			if p.GetRunning() == false {
				return
			}
			data, err := resp.Recv()
			P2pComm.CollectPeerStat(err, p)
			if err != nil {
				log.Error("readStream", "recv,err:", err)
				resp.CloseSend()
				if grpc.Code(err) == codes.Unimplemented { //maybe order peers delete peer to BlackList
					(*p.nodeInfo).blacklist.Add(p.Addr())
				}
				time.Sleep(time.Second) //have a rest
				break
			}

			if block := data.GetBlock(); block != nil {
				if block.GetBlock() != nil {
					//如果已经有登记过的消息记录，则不发送给本地blockchain
					if p.filterTask.QueryTask(block.GetBlock().GetHeight()) == true {
						continue
					}

					//判断比自己低的区块，则不发送给blockchain

					height, err := pcli.GetBlockHeight((*p.nodeInfo))
					if err == nil {
						if height >= block.GetBlock().GetHeight() {
							continue
						}
					}
					log.Info("SubStreamBlock", "block==+======+====+=>Height", block.GetBlock().GetHeight(), "from peer", p.Addr())
					msg := (*p.nodeInfo).qclient.NewMessage("blockchain", pb.EventBroadcastAddBlock, block.GetBlock())
					err = (*p.nodeInfo).qclient.Send(msg, false)
					if err != nil {
						log.Error("subStreamBlock", "send to blockchain Error", err.Error())
						continue
					}
					p.filterTask.RegTask(block.GetBlock().GetHeight()) //添加发送登记，下次通过stream 接收同样的消息的时候可以过滤
				}

			} else if tx := data.GetTx(); tx != nil {

				if tx.GetTx() != nil {
					log.Debug("SubStreamBlock", "tx", tx.GetTx())
					sig := tx.GetTx().GetSignature().GetSignature()
					if p.filterTask.QueryTask(hex.EncodeToString(sig)) == true {
						continue //处理方式同上
					}
					msg := (*p.nodeInfo).qclient.NewMessage("mempool", pb.EventTx, tx.GetTx())
					(*p.nodeInfo).qclient.Send(msg, false)
					p.filterTask.RegTask(hex.EncodeToString(sig)) //登记
				}
			}
		}
	}
}

func (p *peer) SetRunning(run bool) {
	p.pmutx.Lock()
	defer p.pmutx.Unlock()
	p.isrunning = run
}
func (p *peer) GetRunning() bool {
	p.pmutx.Lock()
	defer p.pmutx.Unlock()
	return p.isrunning
}

// makePersistent marks the peer as persistent.
func (p *peer) makePersistent() {

	p.persistent = true
}

// Addr returns peer's remote network address.
func (p *peer) Addr() string {
	return p.peerAddr.String()

}

// IsPersistent returns true if the peer is persitent, false otherwise.
func (p *peer) IsPersistent() bool {
	return p.persistent
}

func (p *peer) AllockTask() (err error) {
	defer func() {
		isErr := recover()
		if isErr != nil {
			err = isErr.(error)
		}
	}()
	p.taskPool <- struct{}{}
	return err

}

func (p *peer) ReleaseTask() {
	<-p.taskPool
	return
}
