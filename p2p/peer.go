package p2p

import (
	"sync"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

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
	streamDone chan struct{}
}

type Version struct {
	mtx            sync.Mutex
	versionSupport bool
}

func (v *Version) Set(ok bool) {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	v.versionSupport = ok
}

func (v *Version) Get() bool {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	return v.versionSupport
}

func (p *peer) Start() error {
	p.mconn.key = p.key //TODO setKey
	go p.subStreamBlock()
	return p.mconn.Start()
}

func (p *peer) subStreamBlock() {
BEGIN:
	resp, err := p.mconn.conn.RouteChat(context.Background(), &pb.ReqNil{})
	if err != nil {
		(*p.nodeInfo).monitorChan <- p //直接删除节点
		return
	}
	for {
		select {
		case <-p.streamDone:
			log.Debug("SubStreamBlock", "break", "peerdone")
			resp.CloseSend()
			return

		default:

			data, err := resp.Recv()
			if err != nil {
				resp.CloseSend()
				log.Error("SubStreamBlock", "Recv Err", err.Error())
				goto BEGIN

			}
			log.Debug("SubStreamBlock", "recv blockorTxXXXXXXXXXXXXXXXXXXXXXXXXX", data)

			if block := data.GetBlock(); block != nil {
				log.Debug("SubStreamBlock", "block", block.GetBlock())
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

func (p *peer) Close() {
	p.setRunning(false)
	p.mconn.Close()
	close(p.streamDone)

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
