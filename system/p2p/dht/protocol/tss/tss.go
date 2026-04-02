package tss

import (
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	protocolID = "/chain33/tss/1.0"
)

var log = log15.New("module", "dht.tss")

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

type tssProtocol struct {
	*protocol.P2PEnv
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	new(tssProtocol).init(env)
}

func (t *tssProtocol) init(env *protocol.P2PEnv) {
	t.P2PEnv = env
	protocol.RegisterEventHandler(types.EventCryptoTssMsg, t.handleTSSEvent)
	protocol.RegisterStreamHandler(t.Host, protocolID, t.handleTSSStream)
}

func (t *tssProtocol) handleTSSEvent(qMsg *queue.Message) {

	var wMsg *tss.MessageWrapper
	switch data := qMsg.GetData().(type) {
	case *tss.MessageWrapper:
		wMsg = data
	case tss.MessageWrapper:
		wMsg = &data
	default:
		log.Error("handleTSSEvent", "err", "invalid message type")
		return
	}
	tssMsg := &types.TSSMessage{
		Protocol: tss.ComposeProtocol(wMsg.Protocol, wMsg.SessionID),
		Msg:      wMsg.Msg,
	}
	pid, _ := peer.Decode(wMsg.PeerID)
	stream, err := t.Host.NewStream(t.Ctx, pid, protocolID)
	if err != nil {
		log.Error("handleTSSEvent", "peer", wMsg.PeerID, "session", wMsg.SessionID, "newStream err", err)
		return
	}

	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	err = protocol.WriteStream(tssMsg, stream)
	if err != nil {
		log.Error("handleTSSEvent", "peer", wMsg.PeerID, "session", wMsg.SessionID, "writeStream err", err)
	}
}

func (t *tssProtocol) handleTSSStream(stream network.Stream) {

	tssMsg := &types.TSSMessage{}
	err := protocol.ReadStream(tssMsg, stream)
	peerName := stream.Conn().RemotePeer().String()
	if err != nil {
		log.Error("handleTSSStream", "peer", peerName, "readStream err", err)
		return
	}
	baseProtocol, sessionID := tss.SplitProtocol(tssMsg.Protocol)
	wMsg := &tss.MessageWrapper{
		Protocol:  baseProtocol,
		SessionID: sessionID,
		Msg:       tssMsg.Msg,
		PeerID:    peerName,
	}
	qMsg := t.QueueClient.NewMessage("crypto", types.EventCryptoTssMsg, wMsg)
	err = t.QueueClient.Send(qMsg, false)

	if err != nil {
		log.Error("handleTSSStream", "peer", peerName, "session", sessionID, "send queue err", err)
	}
}
