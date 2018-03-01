package p2p

import (
	"encoding/hex"
	"sync"
	"time"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func (p *peer) Start() {
	log.Debug("Peer", "Start", p.Addr())
	p.mconn.key = p.key
	go p.HeartBeat()

	return
}
func (p *peer) Close() {
	p.setRunning(false)
	p.mconn.Close()
	close(p.allLoopDone)
	close(p.taskPool)
	close(p.filterTask.loopDone)
	ps.Unsub(p.taskChan, p.Addr())

}

type peer struct {
	wg          sync.WaitGroup
	pmutx       sync.Mutex
	nodeInfo    **NodeInfo
	conn        *grpc.ClientConn // source connection
	persistent  bool
	isrunning   bool
	version     *Version
	key         string
	mconn       *MConnection
	peerAddr    *NetAddress
	peerStat    *Stat
	filterTask  *FilterTask
	allLoopDone chan struct{}
	taskPool    chan struct{}
	taskChan    chan interface{} //tx block
}

func NewPeer(conn *grpc.ClientConn, nodeinfo **NodeInfo, remote *NetAddress) *peer {
	p := &peer{
		conn:        conn,
		taskPool:    make(chan struct{}, 50),
		allLoopDone: make(chan struct{}, 1),
		nodeInfo:    nodeinfo,
	}
	p.filterTask = new(FilterTask)
	p.filterTask.loopDone = make(chan struct{}, 1)
	p.filterTask.regTask = make(map[interface{}]time.Duration)
	p.peerStat = new(Stat)
	p.version = new(Version)
	p.version.SetSupport(true)
	p.setRunning(true)
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

func (p *peer) HeartBeat() {

	//<-(*p.nodeInfo).natDone
	ticker := time.NewTicker(PingTimeout)
	defer ticker.Stop()
	pcli := NewP2pCli(nil)
	if pcli.SendVersion(p, *p.nodeInfo) == nil {
		p.taskChan = ps.Sub(p.Addr())
		go p.filterTask.ManageFilterTask()
		go p.subStreamBlock()
	} else {
		return
	}
FOR_LOOP:
	for {
		select {
		case <-ticker.C:
			err := pcli.SendPing(p, *p.nodeInfo)
			if err != nil {
				p.peerStat.NotOk()
				(*p.nodeInfo).monitorChan <- p
			}

		case <-p.allLoopDone:
			log.Debug("Peer HeartBeat", "loop done", p.Addr())
			break FOR_LOOP

		}

	}

}

func (p *peer) GetPeerInfo(version int32) (*pb.P2PPeerInfo, error) {
	return p.mconn.conn.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: version})
}
func (p *peer) SendData(data interface{}) (err error) {
	tick := time.NewTicker(time.Second * 5)
	defer tick.Stop()
	defer func() {
		isErr := recover()
		if isErr != nil {
			err = isErr.(error)
		}
	}()

	select {
	case <-tick.C:
		return

	case p.taskChan <- data:
	}
	return nil
}

func (p *peer) subStreamBlock() {

	for {
		if p.GetRunning() == false {
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		resp, err := p.mconn.conn.RouteChat(ctx)
		if err != nil {
			p.peerStat.NotOk()
			(*p.nodeInfo).monitorChan <- p
			time.Sleep(time.Second * 5)
			cancel()
			continue
		}
	SEND_LOOP:
		for {
			timeout := time.NewTimer(time.Second * 2)
			select {
			case task := <-p.taskChan:
				if p.GetRunning() == false {
					resp.CloseSend()
					cancel()
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
					p.peerStat.NotOk()
					(*p.nodeInfo).monitorChan <- p
					resp.CloseSend()
					cancel()
					break SEND_LOOP //下一次外循环重新获取stream
				}
			case <-timeout.C:
				if p.GetRunning() == false {
					resp.CloseSend()
					cancel()
					return
				}
			}
		}

	}

}

func (p *peer) setRunning(run bool) {
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

//func (p *peer) BroadCastData() {
//	//TODO 通过Stream 广播出去subStreamBlock
//	for task := range p.taskChan {
//		if block, ok := task.(*pb.P2PBlock); ok {
//			go p.mconn.conn.BroadCastBlock(context.Background(), block)
//		} else if tx, ok := task.(*pb.P2PTx); ok {
//			go p.mconn.conn.BroadCastTx(context.Background(), tx)
//		}
//	}
//}
