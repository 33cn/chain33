package p2p

import (
	"encoding/hex"
	"math/rand"
	"net"
	"strings"
	"time"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	pb "gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var P2pComm Comm

type Comm struct{}

func (Comm) AddrRouteble(addrs []string) []string {
	var enableAddrs []string
	for _, addr := range addrs {

		conn, err := net.DialTimeout("tcp", addr, time.Second*1)
		if err == nil {
			conn.Close()
			enableAddrs = append(enableAddrs, addr)
		}
	}

	return enableAddrs

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

	peer, err := c.NewPeerFromConn(conn, addr, nodeinfo)
	if err != nil {
		conn.Close()
		return nil, err
	}
	peer.peerAddr = addr

	log.Debug("DialPeerWithAddress", "peer", peer.Addr(), "persistent:", persistent)

	if persistent {
		peer.makePersistent()
	}

	return peer, nil
}

func (c Comm) NewPeerFromConn(rawConn *grpc.ClientConn, remote *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	// Key and NodeInfo are set after Handshake
	p := NewPeer(rawConn, nodeinfo, remote)

	return p, nil
}

func (c Comm) DialPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {
	log.Debug("DialPeer", "will connect", addr.String())
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

func (c Comm) GenPrivkey() ([]byte, error) {
	cr, err := crypto.New(pb.GetSignatureTypeName(pb.SECP256K1))
	if err != nil {
		log.Error("CryPto Error", "Error", err.Error())
		return nil, err
	}

	key, err := cr.GenKey()
	if err != nil {
		log.Error("GenKey", "Error", err)
		return nil, err
	}
	return key.Bytes(), nil
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
func (c Comm) NewPingData(peer *peer) (*pb.P2PPing, error) {
	randNonce := rand.Int31n(102040)
	ping := &pb.P2PPing{Nonce: int64(randNonce), Addr: (*peer.nodeInfo).GetExternalAddr().IP.String(), Port: int32((*peer.nodeInfo).GetExternalAddr().Port)}
	var err error
	ping, err = c.Signature(peer.key, ping)
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return nil, err
	}
	return ping, nil

}

func (c Comm) Signature(key string, in *pb.P2PPing) (*pb.P2PPing, error) {

	data := pb.Encode(in)
	cr, err := crypto.New(pb.GetSignatureTypeName(pb.SECP256K1))
	if err != nil {
		log.Error("CryPto Error", "Error", err.Error())
		return nil, err
	}
	pribyts, err := hex.DecodeString(key)
	if err != nil {
		log.Error("DecodeString Error", "Error", err.Error())
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(pribyts)
	if err != nil {
		log.Error("Load PrivKey", "Error", err.Error())
		return nil, err
	}
	in.Sign = new(pb.Signature)
	in.Sign.Signature = priv.Sign(data).Bytes()
	in.Sign.Ty = pb.SECP256K1
	in.Sign.Pubkey = priv.PubKey().Bytes()

	return in, nil
}
func (c Comm) CheckSign(in *pb.P2PPing) bool {

	sign := in.GetSign()
	if sign == nil {
		log.Error("CheckSign Get sign err")
		return false
	}

	cr, err := crypto.New(pb.GetSignatureTypeName(int(sign.Ty)))
	if err != nil {
		log.Error("CheckSign", "crypto.New err", err.Error())
		return false
	}
	pub, err := cr.PubKeyFromBytes(sign.Pubkey)
	if err != nil {
		log.Error("CheckSign", "PubKeyFromBytes err", err.Error())
		return false
	}
	signbytes, err := cr.SignatureFromBytes(sign.Signature)
	if err != nil {
		log.Error("CheckSign", "SignatureFromBytes err", err.Error())
		return false
	}
	in.Sign = nil
	data := pb.Encode(in)
	return pub.VerifyBytes(data, signbytes)
}

func (c Comm) CollectPeerStat(err error, peer *peer) {
	if err != nil {
		peer.peerStat.NotOk()
	} else {
		peer.peerStat.Ok()
	}
	c.reportPeerStat(peer)
}

func (c Comm) reportPeerStat(peer *peer) {
	timeout := time.NewTimer(time.Second)
	select {
	case (*peer.nodeInfo).monitorChan <- peer:
	case <-timeout.C:
		timeout.Stop()
		return
	}
	if !timeout.Stop() {
		<-timeout.C
	}
}

func (c Comm) GrpcConfig() grpc.ServiceConfig {

	var ready = false
	var defaultRespSize = 1024 * 1024 * 60
	var defaultReqSize = 1024 * 1024 * 10
	var defaulttimeout = 40 * time.Second
	var getAddrtimeout = 5 * time.Second
	var getHeadertimeout = 5 * time.Second
	var getPeerinfotimeout = 5 * time.Second
	var sendVersiontimeout = 5 * time.Second
	var pingtimeout = 10 * time.Second
	var MethodConf = map[string]grpc.MethodConfig{
		"/types.p2pgservice/Ping":           grpc.MethodConfig{WaitForReady: &ready, Timeout: &pingtimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/Version2":       grpc.MethodConfig{WaitForReady: &ready, Timeout: &sendVersiontimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/BroadCastTx":    grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetMemPool":     grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetBlocks":      grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetPeerInfo":    grpc.MethodConfig{WaitForReady: &ready, Timeout: &getPeerinfotimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/BroadCastBlock": grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetAddr":        grpc.MethodConfig{WaitForReady: &ready, Timeout: &getAddrtimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetHeaders":     grpc.MethodConfig{WaitForReady: &ready, Timeout: &getHeadertimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
	}

	return grpc.ServiceConfig{Methods: MethodConf}

}
