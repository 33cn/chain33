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
	heartDone  chan struct{}
	taskPool   chan struct{}
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
	select {
	case p.heartDone <- struct{}{}:
	case <-ticker.C:
		return
	}
}
func (p *peer) GetPeerInfo(version int32) (*pb.P2PPeerInfo, error) {
	return p.mconn.conn.GetPeerInfo(context.Background(), &pb.P2PGetPeerInfo{Version: version})
}

func (p *peer) subStreamBlock() {

	for {
		select {
		case <-p.streamDone:
			//log.Error("SubStreamBlock", "peerdone", p.Addr())
			return

		default:

			resp, err := p.mconn.conn.RouteChat(context.Background(), &pb.ReqNil{})
			if err != nil {
				p.peerStat.NotOk()
				(*p.nodeInfo).monitorChan <- p
				time.Sleep(time.Second * 5)
				continue
			}
			log.Info("SubStreamBlock", "Start", p.Addr())

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

func (p *peer) AllockTask() {
	p.taskPool <- struct{}{}
	return
}

func (p *peer) ReleaseTask() {
	<-p.taskPool
	return
}
