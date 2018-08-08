package p2p

import (
	pb "gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

type MConnection struct {
	node          *Node
	gconn         *grpc.ClientConn
	gcli          pb.P2PgserviceClient // source connection
	remoteAddress *NetAddress          //peer 的地址
	peer          *Peer
}

// MConnConfig is a MConnection configuration.
type MConnConfig struct {
	gconn *grpc.ClientConn
	gcli  pb.P2PgserviceClient
}

// DefaultMConnConfig returns the default config.
func DefaultMConnConfig() *MConnConfig {
	return &MConnConfig{}
}

func NewTemMConnConfig(gconn *grpc.ClientConn, gcli pb.P2PgserviceClient) *MConnConfig {
	return &MConnConfig{
		gconn: gconn,
		gcli:  gcli,
	}
}

// NewMConnection wraps net.Conn and creates multiplex connection
func NewMConnection(conn *grpc.ClientConn, remote *NetAddress, peer *Peer) *MConnection {
	log.Info("NewMConnection p2p client", "addr", remote)
	mconn := &MConnection{
		gconn: conn,
		gcli:  pb.NewP2PgserviceClient(conn),
		peer:  peer,
	}
	mconn.node = peer.node
	mconn.remoteAddress = remote
	return mconn
}

func NewMConnectionWithConfig(cfg *MConnConfig) *MConnection {
	mconn := &MConnection{
		gconn: cfg.gconn,
		gcli:  cfg.gcli,
	}
	return mconn
}

func (c *MConnection) Close() {
	c.gconn.Close()
	log.Debug("Mconnection", "Close", "^_^!")
}
