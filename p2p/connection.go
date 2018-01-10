package p2p

import (
	"sync"
	"time"

	pb "code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

type MConnection struct {
	nodeInfo      **NodeInfo
	gconn         *grpc.ClientConn
	conn          pb.P2PgserviceClient // source connection
	config        *MConnConfig
	key           string //privkey
	quit          chan bool
	versionDone   chan struct{}
	remoteAddress *NetAddress //peer 的地址
	peer          *peer
	sendMonitor   *Monitor
	once          sync.Once
}

// MConnConfig is a MConnection configuration.
type MConnConfig struct {
	gconn *grpc.ClientConn
	conn  pb.P2PgserviceClient
}

// DefaultMConnConfig returns the default config.
func DefaultMConnConfig() *MConnConfig {
	return &MConnConfig{}
}

func NewTemMConnConfig(gconn *grpc.ClientConn, conn pb.P2PgserviceClient) *MConnConfig {
	return &MConnConfig{
		gconn: gconn,
		conn:  conn,
	}
}

// NewMConnection wraps net.Conn and creates multiplex connection
func NewMConnection(conn *grpc.ClientConn, remote *NetAddress, peer *peer) *MConnection {

	mconn := &MConnection{
		gconn:       conn,
		conn:        pb.NewP2PgserviceClient(conn),
		sendMonitor: NewMonitor(),
		peer:        peer,
		quit:        make(chan bool, 1),
	}
	mconn.nodeInfo = peer.nodeInfo
	mconn.remoteAddress = remote

	return mconn

}

func NewMConnectionWithConfig(cfg *MConnConfig) *MConnection {
	mconn := &MConnection{
		gconn: cfg.gconn,
		conn:  cfg.conn,
	}

	return mconn
}

// sendRoutine polls for packets to send from channels.
func (c *MConnection) PingRoutine() {
	go func(c *MConnection) {

		ticker := time.NewTicker(PingTimeout)
		defer ticker.Stop()
		pcli := NewP2pCli(nil)
	FOR_LOOP:
		for {
			select {
			case <-ticker.C:
				pcli.SendPing(c.peer, *c.nodeInfo)
			case <-c.quit:
				break FOR_LOOP

			}

		}
	}(c)
}

func (c *MConnection) Start() error {
	c.PingRoutine()
	return nil
}

func (c *MConnection) closePingRoutine() {
	c.quit <- false
}
func (c *MConnection) Close() {

	c.sendMonitor.Close()
	c.gconn.Close()
	c.closePingRoutine()
	log.Debug("Mconnection", "Close+++++++++++++++++++++++++++++++++", "^_^!")
}
