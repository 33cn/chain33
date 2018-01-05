package p2p

import (
	"fmt"
	"sync"
	"time"

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
	return p.mconn.start()
}
func (p *peer) subStreamBlock() {
BEGIN:
	//defer p.wg.Done()
	resp, err := p.mconn.conn.RouteChat(context.Background(), &pb.ReqNil{})
	if err != nil {
		log.Error("SubStreamBlock", "call RouteChat err", err.Error()+p.Addr())
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
				log.Error("SubStreamBlock", "Recv Err", err.Error())
				time.Sleep(time.Second * 1)
				resp.CloseSend()
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

func (p *peer) Stop() {
	close(p.streamDone)
	p.setRunning(false)
	p.mconn.stop()

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

func dial(addr *NetAddress) (*grpc.ClientConn, error) {
	conn, err := addr.DialTimeout(DialTimeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func dialPeerWithAddress(addr *NetAddress, persistent bool, nodeinfo **NodeInfo) (*peer, error) {

	log.Debug("dialPeerWithAddress", "Dial peer address", addr.String())

	peer, err := newOutboundPeer(addr, nodeinfo)
	if err != nil {
		log.Error("dialPeerWithAddress", "address", addr, "err", err)
		return nil, err
	}
	log.Debug("dialPeerWithAddress", "peer", *peer, "persistent:", persistent)

	if persistent {
		peer.makePersistent()
	}

	return peer, nil
}

//连接server out=往其他节点连接
func newOutboundPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	conn, err := dial(addr)
	if err != nil {
		return nil, fmt.Errorf("Error creating peer")
	}

	peer, err := newPeerFromConn(conn, true, addr, nodeinfo)
	if err != nil {
		conn.Close()
		return nil, err
	}
	peer.peerAddr = addr
	return peer, nil
}

func newPeerFromConn(rawConn *grpc.ClientConn, outbound bool, remote *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	conn := rawConn
	// Key and NodeInfo are set after Handshake
	p := &peer{
		outbound:   outbound,
		conn:       conn,
		streamDone: make(chan struct{}, 1),
		nodeInfo:   nodeinfo,
	}
	p.version = new(Version)
	p.version.Set(true)
	p.setRunning(true)
	p.mconn = NewMConnection(conn, remote, p)

	return p, nil
}
func DialPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {
	log.Warn("DialPeer", "will connect", addr.String())
	var persistent bool
	for _, seed := range (*nodeinfo).cfg.Seeds { //TODO待优化
		if seed == addr.String() {
			persistent = true //种子节点要一直连接
		}
	}
	peer, err := dialPeerWithAddress(addr, persistent, nodeinfo)
	if err != nil {
		log.Error("DialPeer", "dial peer err:", err.Error())
		return nil, err
	}
	//获取远程节点的信息 peer
	log.Debug("DialPeer", "Peer info", peer)
	return peer, nil
}
