package p2p

import (
	"sync"
	"time"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func (p *peer) Start() {
	p.mconn.key = p.key
	go p.subStreamBlock()
	p.HeartBeat()
	return
}
func (p *peer) Close() {
	p.setRunning(false)
	p.mconn.Close()
	p.HeartBlood()
	close(p.taskPool)
	close(p.streamDone)
	close(p.sendTxLoop)
	close(p.taskChan)

}

type peer struct {
	wg         sync.WaitGroup
	pmutx      sync.Mutex
	nodeInfo   **NodeInfo
	outbound   bool
	conn       *grpc.ClientConn // source connection
	persistent bool
	isrunning  bool
	version    *Version
	key        string
	mconn      *MConnection
	peerAddr   *NetAddress
	peerStat   *Stat
	streamDone chan struct{}
	sendTxLoop chan struct{}
	heartDone  chan struct{}
	taskPool   chan struct{}
	taskChan   chan interface{} //tx block
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

// sendRoutine polls for packets to send from channels.
func (p *peer) HeartBeat() {
	go func(p *peer) {
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
			case <-p.heartDone:
				break FOR_LOOP

			}

		}
	}(p)
}

func (p *peer) HeartBlood() {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	select {
	case p.heartDone <- struct{}{}:
	case <-ticker.C:
		return
	}
}
func (p *peer) GetPeerInfo(version int32) (*pb.P2PPeerInfo, error) {
	return p.mconn.conn.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: version})
}
func (p *peer) SendData(data interface{}) (err error) {
	tick := time.NewTicker(time.Second * 1)
	defer func() {
		isErr := recover()
		if isErr != nil {
			err = isErr.(error)
		}
	}()
	defer tick.Stop()
	select {
	case <-tick.C:
		//log.Error("Peer SendData", "timeout", "return")
		return
	case <-p.sendTxLoop:
		return
	case p.taskChan <- data:
	}
	return nil
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

func (p *peer) subStreamBlock() {
	go func() { //TODO 待优化，临时不启用
		//Stream Send data
		for {
			resp, err := p.mconn.conn.RouteChat(context.Background())
			if err != nil {
				p.peerStat.NotOk()
				(*p.nodeInfo).monitorChan <- p
				time.Sleep(time.Second * 5)
				continue
			}
			for task := range p.taskChan {
				p2pdata := new(pb.BroadCastData)
				if block, ok := task.(*pb.P2PBlock); ok {
					p2pdata.Value = &pb.BroadCastData_Block{Block: block}
				} else if tx, ok := task.(*pb.P2PTx); ok {
					p2pdata.Value = &pb.BroadCastData_Tx{Tx: tx}
				}
				err := resp.Send(p2pdata)
				if err != nil {
					p.setRunning(false)
					(*p.nodeInfo).monitorChan <- p

					return
				}
			}

		}
	}()
	for {
		select {
		case <-p.streamDone:
			log.Error("SubStreamBlock", "peerdone", p.Addr())
			return

		default:

			resp, err := p.mconn.conn.RouteChat(context.Background())
			if err != nil {
				p.peerStat.NotOk()
				(*p.nodeInfo).monitorChan <- p
				time.Sleep(time.Second * 5)
				continue
			}
			log.Error("SubStreamBlock", "Start", p.Addr())

			for {
				data, err := resp.Recv()
				if err != nil {
					resp.CloseSend()
					log.Error("SubStreamBlock", "Recv Err", err.Error())
					break
				}

				if block := data.GetBlock(); block != nil {
					log.Error("SubStreamBlock", "block==+====================+==================+=>Height", block.GetBlock().GetHeight())
					if block.GetBlock() != nil {
						msg := (*p.nodeInfo).qclient.NewMessage("blockchain", pb.EventBroadcastAddBlock, block.GetBlock())
						(*p.nodeInfo).qclient.Send(msg, true)
						_, err = (*p.nodeInfo).qclient.Wait(msg)
						if err != nil {
							continue
						}
					}

				} else if tx := data.GetTx(); tx != nil {
					log.Debug("SubStreamBlock", "tx", tx.GetTx())
					if tx.GetTx() != nil {
						msg := (*p.nodeInfo).qclient.NewMessage("mempool", pb.EventTx, tx.GetTx())
						(*p.nodeInfo).qclient.Send(msg, false)
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
	//	select {
	//	case <-time.After(time.Second * 2):
	//		log.Error("read channel timeout")
	//		return fmt.Errorf("time out")
	//	case _, ok := <-p.taskPool:
	//		if !ok {
	//			log.Error("AlloclTask", "Stat", "Channel Closed")
	//			return fmt.Errorf("Channel Closed")
	//		}
	//		//补偿
	//		p.taskPool <- struct{}{}
	//		p.taskPool <- struct{}{}
	//		return nil
	//	}

}

func (p *peer) ReleaseTask() {
	<-p.taskPool
	return
}
