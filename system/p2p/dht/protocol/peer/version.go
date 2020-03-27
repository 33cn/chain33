package peer

import (
	"fmt"
	"time"

	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"

	"math/rand"

	"github.com/33cn/chain33/types"

	core "github.com/libp2p/go-libp2p-core"
)

// MainNet Channel = 0x0000

func (p *peerInfoProtol) processVerReq(req *types.MessageP2PVersionReq, muaddr string) (*types.MessageP2PVersionResp, error) {
	if p.getExternalAddr() == "" {
		p.setExternalAddr(req.GetMessage().GetAddrRecv())
		log.Debug("OnVersionReq", "externalAddr", p.getExternalAddr())
	}

	channel := req.GetMessage().GetVersion()
	if channel != p.p2pCfg.Channel {
		//TODO 协议不匹配
		log.Error("OnVersionReq", "channel err", channel, "cfg channel", p.p2pCfg.Channel)
		return nil, fmt.Errorf("channel err,cfg channel:%d", p.p2pCfg.Channel)

	}

	pid := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()

	var version types.P2PVersion
	version.AddrFrom = p.getExternalAddr()
	version.AddrRecv = muaddr
	version.Nonce = rand.Int63n(102400)
	version.Timestamp = time.Now().Unix()

	resp := &types.MessageP2PVersionResp{MessageData: p.NewMessageCommon(req.GetMessageData().GetId(), pid.Pretty(), pubkey, false),
		Message: &version}
	return resp, nil
}

func (p *peerInfoProtol) onVersionReq(req *types.MessageP2PVersionReq, s core.Stream) {
	log.Debug("onVersionReq", "peerproto", s.Protocol(), "req", req)
	remoteMAddr := s.Conn().RemoteMultiaddr()

	senddata, err := p.processVerReq(req, remoteMAddr.String())
	if err != nil {
		log.Error("onVersionReq", "err", err.Error())
		return
	}
	err = prototypes.WriteStream(senddata, s)
	if err != nil {
		log.Error("WriteStream", "err", err)
		return
	}

	log.Debug("OnVersionReq", "localPeer", s.Conn().LocalPeer().String(), "remotePeer", s.Conn().RemotePeer().String())

}
