package gg18

import (
	"testing"

	"github.com/33cn/chain33/system/crypto/tss"
	chaintypes "github.com/33cn/chain33/types"
	alicedkg "github.com/getamis/alice/crypto/tss/dkg"
	alicetypes "github.com/getamis/alice/types"
	alicemsg "github.com/getamis/alice/types/message"
	siriuslog "github.com/getamis/sirius/log"
	"github.com/stretchr/testify/require"
)

// The tests below pin down who a tss message is attributed to. alice's core rejects a message
// whose body id disagrees with the sender it is handed (types/message/msg_main.go), so the whole
// guarantee rests on the session layer handing it the *transport-authenticated* peer id rather
// than the id the sender wrote into the message body. The production transport fills
// tss.MessageWrapper.PeerID with the libp2p peer the stream came from (system/p2p/dht/protocol/
// tss/tss.go), which is the same peer id space the alice peer manager uses.

// stubHandler is a minimal alice types.Handler. The sessions under test only consult
// MessageType; the remaining methods are never reached.
type stubHandler struct {
	msgType alicetypes.MessageType
}

func (h *stubHandler) MessageType() alicetypes.MessageType { return h.msgType }

func (h *stubHandler) GetRequiredMessageCount() uint32 { return 0 }

func (h *stubHandler) IsHandled(siriuslog.Logger, string) bool { return false }

func (h *stubHandler) HandleMessage(siriuslog.Logger, alicetypes.Message) error { return nil }

func (h *stubHandler) Finalize(siriuslog.Logger) (alicetypes.Handler, error) { return nil, nil }

type stubListener struct{}

func (stubListener) OnStateChanged(alicetypes.MainState, alicetypes.MainState) {}

// newAliceCore builds a real alice protocol core, so the forged-sender test exercises alice's own
// check instead of a re-implementation of it.
func newAliceCore() *alicemsg.MsgMain {
	msgType := alicetypes.MessageType(alicedkg.Type_Peer)
	return alicemsg.NewMsgMain("self", 4, stubListener{},
		&stubHandler{msgType: msgType}, msgType)
}

// dkgMessage returns a valid alice dkg message whose body claims to come from id.
func dkgMessage(id string) *alicedkg.Message {
	return &alicedkg.Message{
		Type: alicedkg.Type_Peer,
		Id:   id,
		Body: &alicedkg.Message_Peer{Peer: &alicedkg.BodyPeer{}},
	}
}

// capturedMsg is one AddMessage call observed by captureBackend.
type capturedMsg struct {
	senderID string
	msg      alicetypes.Message
}

// captureBackend records what the session layer hands the protocol core.
type captureBackend struct {
	calls []capturedMsg
}

func (b *captureBackend) AddMessage(senderID string, msg alicetypes.Message) error {
	b.calls = append(b.calls, capturedMsg{senderID: senderID, msg: msg})
	return nil
}

// TestAddMessageRejectsForgedSenderID feeds the session a message that claims to be another
// participant. Its transport-authenticated sender is the peer it actually arrived from, so the
// claim disagrees with the attribution and alice's core must reject it with ErrBadMsg rather than
// accept it as the peer it claims to be.
func TestAddMessageRejectsForgedSenderID(t *testing.T) {
	const sessionID = "gg18-session-forged-sender-id"
	require.NoError(t, registerSession(DkgProtocol, sessionID, newAliceCore()))
	defer removeSession(DkgProtocol, sessionID)

	err := addMessage(DkgProtocol, sessionID, "peer-real", dkgMessage("peer-claimed"))
	require.ErrorIs(t, err, alicemsg.ErrBadMsg,
		"a message claiming another peer must be rejected by the core")

	// Control: the same message, truthfully attributed, is accepted.
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-claimed", dkgMessage("peer-claimed")))
}

// TestAddMessageSenderIDIsTransportPeerID asserts the sender id the core receives is the
// transport peer id, not the (attacker-controlled) id in the message body.
func TestAddMessageSenderIDIsTransportPeerID(t *testing.T) {
	const sessionID = "gg18-session-sender-id-source"
	backend := &captureBackend{}
	require.NoError(t, registerSession(DkgProtocol, sessionID, backend))
	defer removeSession(DkgProtocol, sessionID)

	msg := dkgMessage("peer-claimed")
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-transport", msg))

	require.Len(t, backend.calls, 1)
	require.Equal(t, "peer-transport", backend.calls[0].senderID)
	require.NotEqual(t, msg.GetId(), backend.calls[0].senderID)
	require.Equal(t, alicetypes.Message(msg), backend.calls[0].msg)
}

// TestAddMessageRejectsMissingPeerID asserts fail-closed behaviour: without a transport-
// authenticated peer id there is nothing to attribute the message to, so it must neither reach
// the core nor be buffered for a later session (where it would be flushed with no sender).
func TestAddMessageRejectsMissingPeerID(t *testing.T) {
	const (
		registered   = "gg18-session-missing-peer-id"
		unregistered = "gg18-session-missing-peer-id-unregistered"
	)
	backend := &captureBackend{}
	require.NoError(t, registerSession(DkgProtocol, registered, backend))
	defer removeSession(DkgProtocol, registered)

	require.ErrorIs(t, addMessage(DkgProtocol, registered, "", dkgMessage("peer-claimed")),
		errMissingPeerID)
	require.Empty(t, backend.calls)

	require.ErrorIs(t, addMessage(DkgProtocol, unregistered, "", dkgMessage("peer-claimed")),
		errMissingPeerID)
	sessionsMu.RLock()
	_, buffered := pendingMessages[tss.ComposeProtocol(DkgProtocol, unregistered)]
	sessionsMu.RUnlock()
	require.False(t, buffered, "a message without an authenticated sender must not be buffered")
}

// TestBufferedMessageKeepsTransportPeerID covers the late-session path: a message arriving before
// the local session is registered is buffered, and the flush into the core must still attribute
// it to the peer it arrived from.
func TestBufferedMessageKeepsTransportPeerID(t *testing.T) {
	const sessionID = "gg18-session-buffered-peer-id"
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-transport", dkgMessage("peer-claimed")))

	backend := &captureBackend{}
	require.NoError(t, registerSession(DkgProtocol, sessionID, backend))
	defer removeSession(DkgProtocol, sessionID)

	require.Len(t, backend.calls, 1)
	require.Equal(t, "peer-transport", backend.calls[0].senderID)
}

// TestHandleDkgMsgUsesTransportPeerID asserts the wire handler forwards the peer id the transport
// authenticated (MessageWrapper.PeerID) and not the id carried inside the message body.
func TestHandleDkgMsgUsesTransportPeerID(t *testing.T) {
	const sessionID = "gg18-session-handler-peer-id"
	backend := &captureBackend{}
	require.NoError(t, registerSession(DkgProtocol, sessionID, backend))
	defer removeSession(DkgProtocol, sessionID)

	handleDkgMsg(&tss.MessageWrapper{
		Protocol:  DkgProtocol,
		SessionID: sessionID,
		PeerID:    "peer-transport",
		Msg:       chaintypes.Encode(dkgMessage("peer-claimed")),
	})

	require.Len(t, backend.calls, 1)
	require.Equal(t, "peer-transport", backend.calls[0].senderID)
}
