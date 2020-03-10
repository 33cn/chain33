package peer

import (
	"time"
	//prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	//"github.com/33cn/chain33/queue"
	"math/rand"

	"github.com/33cn/chain33/types"

	uuid "github.com/google/uuid"
	core "github.com/libp2p/go-libp2p-core"
)

//p2p版本区间 10020, 11000

//历史版本
const (
	//p2p广播交易哈希而非完整区块数据
	lightBroadCastVersion = 10030
)

// VERSION number
const VERSION = lightBroadCastVersion

// MainNet Channel = 0x0000

const (
	versionMask = 0xFFFF
)

func decodeChannelVersion(channelVersion int32) (channel int32, version int32) {
	channel = channelVersion >> 16
	version = channelVersion & versionMask
	return
}

func (p *peerInfoProtol) OnVersionReq(req *types.MessageP2PVersionReq, s core.Stream) {

	log.Info("OnVersionReq", "peerproto", s.Protocol(), "req", req)
	if p.GetExternalAddr() == "" {
		p.SetExternalAddr(req.GetMessage().GetAddrRecv())
		log.Info("OnVersionReq", "externalAddr", p.GetExternalAddr())
	}

	channel, _ := decodeChannelVersion(req.GetMessage().GetVersion())
	if channel < p.p2pCfg.Channel {
		//TODO 协议不匹配
		log.Error("OnVersionReq", "channel err", channel)
		return
	}

	pid := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()

	mutiAddr := s.Conn().RemoteMultiaddr()
	log.Info("OnVersionReq", "mutiAddr", mutiAddr.String())
	var version types.P2PVersion
	version.AddrFrom = p.GetExternalAddr()
	version.AddrRecv = mutiAddr.String()
	version.Nonce = rand.Int63n(102400)
	version.Timestamp = time.Now().Unix()

	resp := &types.MessageP2PVersionResp{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false),
		Message: &version}

	err := p.SendProtoMessage(resp, s)
	if err != nil {
		log.Error("SendProtoMessage", "err", err)
		return
	}

	log.Info("OnVersionReq", "localPeer", s.Conn().LocalPeer().String(), "remotePeer", s.Conn().RemotePeer().String())

}
