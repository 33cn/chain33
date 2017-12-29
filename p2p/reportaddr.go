package p2p

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/common/crypto"
	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pr "google.golang.org/grpc/peer"
)

func Pubkey(key string) (string, error) {

	cr, err := crypto.New(pb.GetSignatureTypeName(pb.SECP256K1))
	if err != nil {
		log.Error("CryPto Error", "Error", err.Error())
		return "", err
	}
	pribyts, err := hex.DecodeString(key)
	if err != nil {
		log.Error("DecodeString Error", "Error", err.Error())
		return "", err
	}
	priv, err := cr.PrivKeyFromBytes(pribyts)
	if err != nil {
		log.Error("Load PrivKey", "Error", err.Error())
		return "", err
	}

	return hex.EncodeToString(priv.PubKey().Bytes()), nil
}

type RemoteListener interface {
	Stop() bool
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
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.server = grpc.NewServer()
	pb.RegisterP2PremoteaddrServer(r.server, &p2pRemote{})
	r.server.Serve(r.listener)

}

func (r *RemoteAddrListener) Stop() bool {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.server.Stop()
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

func getSelfExternalAddr(serveraddr string) []string {
	var addrlist = make([]string, 0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	conn, err := grpc.DialContext(ctx, serveraddr, grpc.WithInsecure())
	if err != nil {
		log.Error("grpc DialCon", "did not connect: %v", err)
		return addrlist
	}
	defer conn.Close()
	gconn := pb.NewP2PremoteaddrClient(conn)
	resp, err := gconn.RemotePeerAddr(ctx, &pb.P2PGetAddr{Nonce: 12})
	if err != nil {
		return addrlist
	}

	return resp.Addrlist
}
