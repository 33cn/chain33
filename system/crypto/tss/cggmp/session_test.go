// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"testing"

	"github.com/33cn/chain33/system/crypto/tss"
	chaintypes "github.com/33cn/chain33/types"
	"github.com/getamis/alice/crypto/birkhoffinterpolation"
	"github.com/getamis/alice/crypto/elliptic"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicetypes "github.com/getamis/alice/types"
	alicemsg "github.com/getamis/alice/types/message"
	siriuslog "github.com/getamis/sirius/log"
	"github.com/stretchr/testify/require"
)

// The tests below pin down who an incoming tss message is attributed to. alice's core rejects a
// message whose body id disagrees with the sender it is handed (types/message/msg_main.go), so the
// guarantee rests on the session layer handing it the *transport-authenticated* peer id rather
// than the id the sender wrote into the message body. The transport fills
// tss.MessageWrapper.PeerID with the libp2p peer the stream came from (system/p2p/dht/protocol/
// tss/tss.go) — the same peer id space the alice peer manager uses (see also the ppk exchange in
// handlers.go, which is attributed the same way for the same reason).

// stubHandler is a minimal alice types.Handler. The sessions under test only consult MessageType;
// the remaining methods are never reached.
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

// dkgMessage returns a valid alice cggmp dkg message whose body claims to come from id.
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
	const sessionID = "cggmp-session-forged-sender-id"
	require.NoError(t, registerSession(DkgProtocol, sessionID, newAliceCore()))
	defer removeSession(DkgProtocol, sessionID)

	err := addMessage(DkgProtocol, sessionID, "peer-real", dkgMessage("peer-claimed"))
	require.ErrorIs(t, err, alicemsg.ErrBadMsg,
		"a message claiming another peer must be rejected by the core")

	// Control: the same message, truthfully attributed, is accepted.
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-claimed", dkgMessage("peer-claimed")))
}

// TestAddMessageSenderIDIsTransportPeerID asserts the sender id the core receives is the transport
// peer id, not the (attacker-controlled) id in the message body.
func TestAddMessageSenderIDIsTransportPeerID(t *testing.T) {
	const sessionID = "cggmp-session-sender-id-source"
	backend := &captureBackend{}
	require.NoError(t, registerSession(DkgProtocol, sessionID, backend))
	defer removeSession(DkgProtocol, sessionID)

	msg := dkgMessage("peer-claimed")
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-transport", msg))

	require.Len(t, backend.calls, 1)
	require.Equal(t, "peer-transport", backend.calls[0].senderID)
	require.NotEqual(t, msg.GetId(), backend.calls[0].senderID)
	require.Equal(t, alicetypes.Message(msg), backend.calls[0].msg)

	// Every cggmp phase is wired the same way.
	for _, protocol := range []string{RefreshProtocol, SignProtocol} {
		phaseSession := sessionID + protocol
		phaseBackend := &captureBackend{}
		require.NoError(t, registerSession(protocol, phaseSession, phaseBackend))
		require.NoError(t, addMessage(protocol, phaseSession, "peer-transport", dkgMessage("peer-claimed")))
		require.Len(t, phaseBackend.calls, 1)
		require.Equal(t, "peer-transport", phaseBackend.calls[0].senderID)
		removeSession(protocol, phaseSession)
	}
}

// TestAddMessageRejectsMissingPeerID asserts fail-closed behaviour: without a transport-
// authenticated peer id there is nothing to attribute the message to, so it must neither reach the
// core nor be buffered for a later session (where it would be flushed with no sender).
func TestAddMessageRejectsMissingPeerID(t *testing.T) {
	const (
		registered   = "cggmp-session-missing-peer-id"
		unregistered = "cggmp-session-missing-peer-id-unregistered"
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
// the local session is registered is buffered, and the flush into the core must still attribute it
// to the peer it arrived from.
func TestBufferedMessageKeepsTransportPeerID(t *testing.T) {
	const sessionID = "cggmp-session-buffered-peer-id"
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-transport", dkgMessage("peer-claimed")))

	backend := &captureBackend{}
	require.NoError(t, registerSession(DkgProtocol, sessionID, backend))
	defer removeSession(DkgProtocol, sessionID)

	require.Len(t, backend.calls, 1)
	require.Equal(t, "peer-transport", backend.calls[0].senderID)
}

// stubPeerManager is a minimal alice types.PeerManager. The retry tests below drive the session
// registry in-process, so nothing is ever sent over a transport (the echo layer's relay calls
// land here as no-ops).
type stubPeerManager struct {
	self  string
	peers []string
}

func (p *stubPeerManager) NumPeers() uint32             { return uint32(len(p.peers)) }
func (p *stubPeerManager) PeerIDs() []string            { return p.peers }
func (p *stubPeerManager) SelfID() string               { return p.self }
func (p *stubPeerManager) MustSend(string, interface{}) {}

// newSessionDKG builds a real alice cggmp DKG core, so the retry tests hit alice's own echo layer
// (the source of ErrDifferentHash) instead of a re-implementation of it. The core is never started:
// only its AddMessage path is exercised.
func newSessionDKG(t *testing.T) *alicedkg.DKG {
	t.Helper()
	pm := &stubPeerManager{self: "peer-self", peers: []string{"peer-a", "peer-b"}}
	core, err := alicedkg.NewDKG(elliptic.Secp256k1(), pm, []byte("retry-session"), 2, 0, stubListener{})
	require.NoError(t, err)
	return core
}

// sessionPeerMsg returns a Type_Peer message from id. Its body carries the sender's Birkhoff
// parameter, whose x is drawn randomly by alice on every DKG round (newPeerHandler) — so two
// rounds of the same participant produce two *different* bodies under the same (type, sender) key,
// which is exactly what makes a cross-round batch collide in the echo layer.
func sessionPeerMsg(id string, x byte) *alicedkg.Message {
	return &alicedkg.Message{
		Type: alicedkg.Type_Peer,
		Id:   id,
		Body: &alicedkg.Message_Peer{Peer: &alicedkg.BodyPeer{
			Bk: &birkhoffinterpolation.BkParameterMessage{X: []byte{x}},
		}},
	}
}

// requireNoSessionState asserts nothing is kept for sessionID: neither a registered session nor a
// pending buffer. Both would outlive the round they belong to, and the next round (which must reuse
// the same session name, see TestRegisterSessionRecoversAfterFailedRound) would inherit them.
func requireNoSessionState(t *testing.T, sessionID string) {
	t.Helper()
	id := tss.ComposeProtocol(DkgProtocol, sessionID)
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	_, registered := sessions[id]
	_, buffered := pendingMessages[id]
	require.False(t, registered, "session %q must not stay registered", sessionID)
	require.False(t, buffered, "session %q must not keep a pending buffer", sessionID)
}

// TestRegisterSessionRecoversAfterFailedRound is the regression test for the DKG retry path: a node
// whose DKG round failed must be able to run the DKG again under the same session name.
//
// The session name is a constant shared by every participant (alice derives the DKG zero-knowledge
// challenges from the sid, so a per-node unique name would split the group into different rounds),
// so a retry necessarily re-registers the *same* id. Reproduced here is what the 2026-09-19 E2E log
// showed: round 1 is registered and then torn down (its listener gave up on the deadline), and the
// next round's flush throws ErrDifferentHash — the late Type_Peer of the dead round and the one the
// peer broadcast when it restarted sit in the same buffer, and alice keeps a single body per
// (message type, sender). Before the fix that flush error left the half-registered session behind,
// so every later attempt died on "session already registered" and the node never came back.
func TestRegisterSessionRecoversAfterFailedRound(t *testing.T) {
	const sessionID = "cggmp-session-retry-after-failed-round"

	// Round 1, registered and torn down the way ProcessDKG does it (registerSession, then
	// removeSession on return).
	require.NoError(t, registerSession(DkgProtocol, sessionID, newSessionDKG(t)))
	removeSession(DkgProtocol, sessionID)
	requireNoSessionState(t, sessionID)

	// While no session is registered, two Type_Peer messages of the same participant arrive: the
	// late one from the round that just died, and the one it sent when it restarted.
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-b", sessionPeerMsg("peer-b", 1)))
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-b", sessionPeerMsg("peer-b", 2)))

	// Round 2 fails on that batch: alice rejects the second, different body for (Type_Peer, peer-b)
	// instead of letting the round run to its own timeout. Failing fast is deliberate — what matters
	// is that it is recoverable.
	err := registerSession(DkgProtocol, sessionID, newSessionDKG(t))
	require.ErrorIs(t, err, alicemsg.ErrDifferentHash)
	// The failed registration left nothing behind: no session, and the poisoned batch is gone too.
	requireNoSessionState(t, sessionID)

	// The very next attempt under the same name registers fine — this is the behaviour the node
	// needs to come back without a restart.
	require.NoError(t, registerSession(DkgProtocol, sessionID, newSessionDKG(t)))
	defer removeSession(DkgProtocol, sessionID)

	// Live messages still reach the registered core ...
	require.NoError(t, addMessage(DkgProtocol, sessionID, "peer-b", sessionPeerMsg("peer-b", 3)))

	// ... and a second registration of a *live* session is still refused: that rejection is what
	// keeps two concurrent rounds of the same name from trampling each other.
	require.ErrorContains(t, registerSession(DkgProtocol, sessionID, newSessionDKG(t)),
		"session already registered")

	// Teardown drops the session state again, so the next round starts from a clean slate.
	removeSession(DkgProtocol, sessionID)
	requireNoSessionState(t, sessionID)
}

// TestHandleDkgMsgUsesTransportPeerID asserts the wire handler forwards the peer id the transport
// authenticated (MessageWrapper.PeerID) and not the id carried inside the message body.
func TestHandleDkgMsgUsesTransportPeerID(t *testing.T) {
	const sessionID = "cggmp-session-handler-peer-id"
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
