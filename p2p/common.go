package p2p

import (
	"bytes"
	"encoding/binary"
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

func (c Comm) dialPeerWithAddress(addr *NetAddress, persistent bool, nodeinfo **NodeInfo) (*Peer, error) {

	conn, err := addr.DialTimeout(c.GrpcConfig(), (*nodeinfo).cfg.GetVersion())
	if err != nil {
		return nil, err
	}

	peer, err := c.newPeerFromConn(conn, addr, nodeinfo)
	if err != nil {
		conn.Close()
		return nil, err
	}
	peer.SetAddr(addr)
	log.Debug("dialPeerWithAddress", "peer", peer.Addr(), "persistent:", persistent)

	if persistent {
		peer.MakePersistent()
	}

	return peer, nil
}

func (c Comm) newPeerFromConn(rawConn *grpc.ClientConn, remote *NetAddress, nodeinfo **NodeInfo) (*Peer, error) {

	// Key and NodeInfo are set after Handshake
	p := NewPeer(rawConn, nodeinfo, remote)

	return p, nil
}

func (c Comm) dialPeer(addr *NetAddress, nodeinfo **NodeInfo) (*Peer, error) {
	log.Debug("dialPeer", "will connect", addr.String())
	var persistent bool
	for _, seed := range (*nodeinfo).cfg.Seeds { //TODO待优化
		if seed == addr.String() {
			persistent = true //种子节点要一直连接
		}
	}
	peer, err := c.dialPeerWithAddress(addr, persistent, nodeinfo)
	if err != nil {
		log.Error("dialPeer", "dial peer err:", err.Error())
		return nil, err
	}
	//获取远程节点的信息 peer
	log.Debug("dialPeer", "Peer info", peer)
	return peer, nil
}

func (c Comm) GenPrivPubkey() ([]byte, []byte, error) {
	cr, err := crypto.New(pb.GetSignatureTypeName(pb.SECP256K1))
	if err != nil {
		log.Error("CryPto Error", "Error", err.Error())
		return nil, nil, err
	}

	key, err := cr.GenKey()
	if err != nil {
		log.Error("GenKey", "Error", err)
		return nil, nil, err
	}
	return key.Bytes(), key.PubKey().Bytes(), nil
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
func (c Comm) NewPingData(nodeInfo *NodeInfo) (*pb.P2PPing, error) {
	randNonce := rand.Int31n(102040)
	ping := &pb.P2PPing{Nonce: int64(randNonce), Addr: nodeInfo.GetExternalAddr().IP.String(), Port: int32(nodeInfo.GetExternalAddr().Port)}
	var err error
	p2pPrivKey, _ := nodeInfo.addrBook.GetPrivPubKey()
	ping, err = c.Signature(p2pPrivKey, ping)
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
	if pub.VerifyBytes(data, signbytes) {
		in.Sign = sign
		return true
	}
	return false
}

func (c Comm) CollectPeerStat(err error, peer *Peer) {
	if err != nil {
		peer.peerStat.NotOk()
	} else {
		peer.peerStat.Ok()
	}
	c.reportPeerStat(peer)
}

func (c Comm) reportPeerStat(peer *Peer) {
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

func (c Comm) BytesToInt32(b []byte) int32 {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int32
	binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
	return tmp
}

func (c Comm) Int32ToBytes(n int32) []byte {
	tmp := n
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.LittleEndian, tmp)
	return bytesBuffer.Bytes()
}

func (c Comm) GrpcConfig() grpc.ServiceConfig {

	var ready = false
	var defaultRespSize = 1024 * 1024 * 60
	var defaultReqSize = 1024 * 1024 * 10
	var defaulttimeout = 40 * time.Second
	var getAddrtimeout = 5 * time.Second
	var getHeadertimeout = 2 * time.Second
	var getPeerinfotimeout = 5 * time.Second
	var sendVersiontimeout = 5 * time.Second
	var pingtimeout = 10 * time.Second
	var MethodConf = map[string]grpc.MethodConfig{
		"/types.p2pgservice/Ping":           {WaitForReady: &ready, Timeout: &pingtimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/Version2":       {WaitForReady: &ready, Timeout: &sendVersiontimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/BroadCastTx":    {WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetMemPool":     {WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetBlocks":      {WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetPeerInfo":    {WaitForReady: &ready, Timeout: &getPeerinfotimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/BroadCastBlock": {WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetAddr":        {WaitForReady: &ready, Timeout: &getAddrtimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/GetHeaders":     {WaitForReady: &ready, Timeout: &getHeadertimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types.p2pgservice/CheckPeerNatOk": {WaitForReady: &ready, Timeout: &getHeadertimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
	}

	return grpc.ServiceConfig{Methods: MethodConf}

}
