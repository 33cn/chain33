package p2p

import (
	"encoding/hex"
	"net"
	"strings"
	"time"

	"code.aliyun.com/chain33/chain33/common/crypto"
	pb "code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

var P2pComm Comm

type Comm struct {
}

func (c Comm) GetLocalAddr() string {

	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	defer conn.Close()
	log.Debug(strings.Split(conn.LocalAddr().String(), ":")[0])
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}

func (c Comm) DialPeerWithAddress(addr *NetAddress, persistent bool, nodeinfo **NodeInfo) (*peer, error) {

	conn, err := addr.DialTimeout(c.GrpcConfig())
	if err != nil {
		return nil, err
	}

	peer, err := c.NewPeerFromConn(conn, true, addr, nodeinfo)
	if err != nil {
		conn.Close()
		return nil, err
	}
	peer.peerAddr = addr

	log.Debug("DialPeerWithAddress", "peer", *peer, "persistent:", persistent)

	if persistent {
		peer.makePersistent()
	}

	return peer, nil
}

func (c Comm) NewPeerFromConn(rawConn *grpc.ClientConn, outbound bool, remote *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	// Key and NodeInfo are set after Handshake
	p := NewPeer(outbound, rawConn, nodeinfo, remote)

	return p, nil
}

func (c Comm) DialPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {
	log.Warn("DialPeer", "will connect", addr.String())
	var persistent bool
	for _, seed := range (*nodeinfo).cfg.Seeds { //TODO待优化
		if seed == addr.String() {
			persistent = true //种子节点要一直连接
		}
	}
	peer, err := c.DialPeerWithAddress(addr, persistent, nodeinfo)
	if err != nil {
		log.Error("DialPeer", "dial peer err:", err.Error())
		return nil, err
	}
	//获取远程节点的信息 peer
	log.Debug("DialPeer", "Peer info", peer)
	return peer, nil
}
func (c Comm) Pubkey(key string) (string, error) {

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

func (c Comm) GrpcConfig() grpc.ServiceConfig {

	var ready = true
	var defaultRespSize = 1024 * 1024 * 60
	var defaultReqSize = 1024 * 1024 * 10
	var defaulttimeout = 40 * time.Second
	var getAddrtimeout = 5 * time.Second
	var getHeadertimeout = 5 * time.Second
	var getPeerinfotimeout = 5 * time.Second
	var sendVersiontimeout = 5 * time.Second
	var pingtimeout = 10 * time.Second
	var MethodConf = map[string]grpc.MethodConfig{
		"/types.p2pgservice/Ping":        grpc.MethodConfig{WaitForReady: &ready, Timeout: &pingtimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/Version2":    grpc.MethodConfig{WaitForReady: &ready, Timeout: &sendVersiontimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/BroadCastTx": grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetMemPool":  grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		//"/types.p2pgservice/GetData":        grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetBlocks":      grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetPeerInfo":    grpc.MethodConfig{WaitForReady: &ready, Timeout: &getPeerinfotimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/BroadCastBlock": grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetAddr":        grpc.MethodConfig{WaitForReady: &ready, Timeout: &getAddrtimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetHeaders":     grpc.MethodConfig{WaitForReady: &ready, Timeout: &getHeadertimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		//"/types.p2pgservice/":               grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
	}

	return grpc.ServiceConfig{Methods: MethodConf}

}
