// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/types"
	"google.golang.org/grpc"
)

var P2pComm Comm

type Comm struct{}

func (Comm) AddrRouteble(addrs []string) []string {
	var enableAddrs []string

	for _, addr := range addrs {
		netaddr, err := NewNetAddressString(addr)
		if err != nil {
			log.Error("AddrRouteble", "NewNetAddressString", err.Error())
			continue
		}
		conn, err := netaddr.DialTimeout(VERSION)
		if err != nil {
			//log.Error("AddrRouteble", "DialTimeout", err.Error())
			continue
		}
		conn.Close()
		enableAddrs = append(enableAddrs, addr)
	}
	return enableAddrs
}

func (c Comm) RandStr(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	r := rand.New(rand.NewSource(types.Now().Unix()))
	b := make([]rune, n)
	for i := range b {

		b[i] = letters[r.Intn(len(letters))]
	}

	return string(b)
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

func (c Comm) dialPeerWithAddress(addr *NetAddress, persistent bool, node *Node) (*Peer, error) {
	log.Info("dialPeerWithAddress")
	conn, err := addr.DialTimeout(node.nodeInfo.cfg.Version)
	if err != nil {
		return nil, err
	}

	peer, err := c.newPeerFromConn(conn, addr, node)
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

func (c Comm) newPeerFromConn(rawConn *grpc.ClientConn, remote *NetAddress, node *Node) (*Peer, error) {

	// Key and NodeInfo are set after Handshake
	p := NewPeer(rawConn, node, remote)
	return p, nil
}

func (c Comm) dialPeer(addr *NetAddress, node *Node) (*Peer, error) {
	log.Debug("dialPeer", "will connect", addr.String())
	var persistent bool
	for _, seed := range node.nodeInfo.cfg.Seeds { //TODO待优化
		if seed == addr.String() {
			persistent = true //种子节点要一直连接
		}
	}
	peer, err := c.dialPeerWithAddress(addr, persistent, node)
	if err != nil {
		log.Error("dialPeer", "dial peer err:", err.Error())
		return nil, err
	}
	//获取远程节点的信息 peer
	log.Debug("dialPeer", "Peer info", peer)
	return peer, nil
}

func (c Comm) GenPrivPubkey() ([]byte, []byte, error) {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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

	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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
func (c Comm) NewPingData(nodeInfo *NodeInfo) (*types.P2PPing, error) {
	randNonce := rand.Int31n(102040)
	ping := &types.P2PPing{Nonce: int64(randNonce), Addr: nodeInfo.GetExternalAddr().IP.String(), Port: int32(nodeInfo.GetExternalAddr().Port)}
	var err error
	p2pPrivKey, _ := nodeInfo.addrBook.GetPrivPubKey()
	ping, err = c.Signature(p2pPrivKey, ping)
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return nil, err
	}
	return ping, nil

}

func (c Comm) Signature(key string, in *types.P2PPing) (*types.P2PPing, error) {

	data := types.Encode(in)
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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
	in.Sign = new(types.Signature)
	in.Sign.Signature = priv.Sign(data).Bytes()
	in.Sign.Ty = types.SECP256K1
	in.Sign.Pubkey = priv.PubKey().Bytes()

	return in, nil
}
func (c Comm) CheckSign(in *types.P2PPing) bool {

	sign := in.GetSign()
	if sign == nil {
		log.Error("CheckSign Get sign err")
		return false
	}

	cr, err := crypto.New(types.GetSignName("", int(sign.Ty)))
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
	data := types.Encode(in)
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
	case peer.node.nodeInfo.monitorChan <- peer:
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

	var defaulttimeout = 20 * time.Second

	var MethodConf = map[string]grpc.MethodConfig{
		"/types.p2pgservice/Ping":            {Timeout: &defaulttimeout},
		"/types.p2pgservice/Version2":        {Timeout: &defaulttimeout},
		"/types.p2pgservice/BroadCastTx":     {Timeout: &defaulttimeout},
		"/types.p2pgservice/GetMemPool":      {Timeout: &defaulttimeout},
		"/types.p2pgservice/GetBlocks":       {Timeout: &defaulttimeout},
		"/types.p2pgservice/GetPeerInfo":     {Timeout: &defaulttimeout},
		"/types.p2pgservice/BroadCastBlock":  {Timeout: &defaulttimeout},
		"/types.p2pgservice/GetAddr":         {Timeout: &defaulttimeout},
		"/types.p2pgservice/GetHeaders":      {Timeout: &defaulttimeout},
		"/types.p2pgservice/RemotePeerAddr":  {Timeout: &defaulttimeout},
		"/types.p2pgservice/RemotePeerNatOk": {Timeout: &defaulttimeout},
	}

	return grpc.ServiceConfig{Methods: MethodConf}

}
