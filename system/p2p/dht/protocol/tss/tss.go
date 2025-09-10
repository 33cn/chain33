package tss

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	. "github.com/33cn/chain33/system/crypto/tss"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

const (
	protocolID = "/chain33/tss/1.0"
)

var log = log15.New("module", "dht.tss")

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

type tss struct {
	*protocol.P2PEnv
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	new(tss).init(env)
}

func (t *tss) init(env *protocol.P2PEnv) {
	t.P2PEnv = env
	protocol.RegisterEventHandler(types.EventCryptoTssMsg, t.handleTSSEvent)
	protocol.RegisterStreamHandler(t.Host, protocolID, t.handleTSSStream)
}

func (t *tss) handleTSSEvent(qMsg *queue.Message) {

	wMsg := qMsg.GetData().(MessageWrapper)
	tssMsg := &types.TSSMessage{
		Protocol: wMsg.Protocol,
		Msg:      wMsg.Msg,
	}
	pid, _ := peer.Decode(wMsg.PeerID)
	stream, err := t.Host.NewStream(t.Ctx, pid, protocolID)
	if err != nil {
		log.Error("handleTSSEvent", "peer", wMsg.PeerID, "newStream err", err)
		return
	}

	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	err = protocol.WriteStream(tssMsg, stream)
	if err != nil {
		log.Error("handleTSSEvent", "peer", wMsg.PeerID, "writeStream err", err)
	}
}

func (t *tss) handleTSSStream(stream network.Stream) {

	tssMsg := &types.TSSMessage{}
	err := protocol.ReadStream(tssMsg, stream)
	peerName := stream.Conn().RemotePeer().String()
	if err != nil {
		log.Error("handleTSSStream", "peer", peerName, "readStream err", err)
		return
	}
	wMsg := &MessageWrapper{
		Protocol: tssMsg.Protocol,
		Msg:      tssMsg.Msg,
		PeerID:   peerName,
	}
	qMsg := t.QueueClient.NewMessage("crypto", types.EventCryptoTssMsg, wMsg)
	err = t.QueueClient.Send(qMsg, false)

	if err != nil {
		log.Error("handleTSSStream", "peer", peerName, "send queue err", err)
	}
}
