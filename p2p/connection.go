// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	pb "github.com/33cn/chain33/types"
	"google.golang.org/grpc"
)

// MConnection  contains node, grpc client, p2pgserviceClient, netaddress, peer
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

// NewTemMConnConfig return the config by grpc.clientconn, gcli
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

// NewMConnectionWithConfig return mconn by mconnconfig
func NewMConnectionWithConfig(cfg *MConnConfig) *MConnection {
	mconn := &MConnection{
		gconn: cfg.gconn,
		gcli:  cfg.gcli,
	}
	return mconn
}

// Close mconnection
func (c *MConnection) Close() {
	err := c.gconn.Close()
	if err != nil {
		log.Error("Mconnection", "Close err", err)
	}
	log.Debug("Mconnection", "Close", "^_^!")
}
