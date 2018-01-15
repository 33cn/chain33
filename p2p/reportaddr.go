package p2p

import (
	"fmt"
	"net"
	"strings"
	"sync"

	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"

	"google.golang.org/grpc"
	pr "google.golang.org/grpc/peer"
)

type RemoteListener interface {
	Close() bool
}
type RemoteAddrListener struct {
	mtx      sync.Mutex
	server   *grpc.Server
	listener net.Listener
}

func NewRemotePeerAddrServer() RemoteListener {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", DefalutP2PRemotePort))
	if err != nil {
		log.Crit("Failed to listen", "Error", err.Error())
		return nil
	}
	var remoteListener RemoteAddrListener
	remoteListener.listener = listener
	go remoteListener.Start()
	return &remoteListener

}

func (r *RemoteAddrListener) Start() {
	go r.listenRoutine()
}
func (r *RemoteAddrListener) listenRoutine() {

	r.server = grpc.NewServer()
	pb.RegisterP2PremoteaddrServer(r.server, &p2pRemote{})
	r.server.Serve(r.listener)
}
func (r *RemoteAddrListener) Close() bool {

	r.server.Stop()
	r.listener.Close()
	return true
}

type p2pRemote struct {
}

func (p *p2pRemote) RemotePeerAddr(ctx context.Context, in *pb.P2PGetAddr) (*pb.P2PAddr, error) {
	var addrlist = make([]string, 0)
	getctx, ok := pr.FromContext(ctx)
	if ok {
		remoteaddr := strings.Split(getctx.Addr.String(), ":")[0]
		addrlist = append(addrlist, remoteaddr)
	}
	return &pb.P2PAddr{Addrlist: addrlist}, nil
}
