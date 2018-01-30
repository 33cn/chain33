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
	p.mconn.key = p.key
	p.taskChan = ps.Sub(p.Addr())
	go p.subStreamBlock()
	go p.filterTask.ManageFilterTask()
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

func NewPeer(isout bool, conn *grpc.ClientConn, nodeinfo **NodeInfo, remote *NetAddress) *peer {
	p := &peer{
		outbound: isout,
		conn:     conn,

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

type peer struct {
	wg          sync.WaitGroup
	pmutx       sync.Mutex
	nodeInfo    **NodeInfo
	outbound    bool
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

// sendRoutine polls for packets to send from channels.
func (p *peer) HeartBeat() {

	var count int64
	ticker := time.NewTicker(PingTimeout)
	defer ticker.Stop()
	pcli := NewP2pCli(nil)
FOR_LOOP:
	for {
		select {
		case <-ticker.C:
			err := pcli.SendPing(p, *p.nodeInfo)
			if err != nil {
				if count == 0 {
					p.setRunning(false)
				}
			}
			count++
		case <-p.allLoopDone:
			log.Error("Peer HeartBeat", "loop done", p.Addr())
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
		//log.Error("Peer SendData", "timeout", "return")
		return

	case p.taskChan <- data:
	}
	return nil
}

func (p *peer) subStreamBlock() {
	//p.taskChan = ps.Sub(p.Addr())
	pcli := NewP2pCli(nil)
	go func(p *peer) {
		//Stream Send data
		for {
			select {
			case <-p.allLoopDone:
				log.Debug("peer SubStreamBlock", "Send Stream  Done", p.Addr())
				return

			default:
				resp, err := p.mconn.conn.RouteChat(context.Background())
				if err != nil {
					p.peerStat.NotOk()
					(*p.nodeInfo).monitorChan <- p
					time.Sleep(time.Second * 5)
					continue
				}
				//for task := range p.taskChan {
				for task := range p.taskChan {
					p2pdata := new(pb.BroadCastData)
					if block, ok := task.(*pb.P2PBlock); ok {
						height := block.GetBlock().GetHeight()
						pinfo, err := p.GetPeerInfo((*p.nodeInfo).cfg.GetVersion())
						if err == nil {
							if pinfo.GetHeader().GetHeight() >= height {
								continue
							}
						}
						if p.filterTask.QueryTask(height) == true {
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
						break //下一次外循环重新获取stream
					}
				}

			}
		}
	}(p)
	for {
		select {
		case <-p.allLoopDone:
			log.Debug("Peer SubStreamBlock", "RecvStreamDone", p.Addr())
			return

		default:

			resp, err := p.mconn.conn.RouteChat(context.Background())
			if err != nil {
				p.peerStat.NotOk()
				(*p.nodeInfo).monitorChan <- p
				time.Sleep(time.Second * 5)
				continue
			}
			log.Debug("SubStreamBlock", "Start", p.Addr())

			for {
				data, err := resp.Recv()
				if err != nil {
					resp.CloseSend()
					p.peerStat.NotOk()
					(*p.nodeInfo).monitorChan <- p
					//log.Error("SubStreamBlock", "Recv Err", err.Error())
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
						log.Info("SubStreamBlock", "block==+======+====+=>Height", block.GetBlock().GetHeight())
						msg := (*p.nodeInfo).qclient.NewMessage("blockchain", pb.EventBroadcastAddBlock, block.GetBlock())
						(*p.nodeInfo).qclient.Send(msg, true)
						_, err = (*p.nodeInfo).qclient.Wait(msg)
						if err != nil {
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
	if !p.outbound {
		panic("inbound peers can't be made persistent")
	}
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
