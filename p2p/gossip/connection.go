// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gossip

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

// Close mconnection
func (c *MConnection) Close() {
	err := c.gconn.Close()
	if err != nil {
		log.Error("Mconnection", "Close err", err)
	}
	log.Debug("Mconnection", "Close", "^_^!")
}
