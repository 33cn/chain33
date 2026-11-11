// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/getamis/alice/types"
)

// Backend is the alice protocol core that consumes incoming messages.
// Both the cggmp DKG/refresh/sign cores embed alice types.MessageMain and therefore
// implement AddMessage(senderID, msg).
type Backend interface {
	AddMessage(senderID string, msg types.Message) error
}

type sessionCore struct {
	// mu serialises AddMessage calls into this session's backend. The registry lock only guards
	// the map, so a protocol core doing an expensive round (Paillier generation, DKG rounds) does
	// not block message handling for every other session. Exception: registerSession flushes its
	// buffered batch while holding the registry write lock and calls the core under it, so other
	// sessions' lookups wait for that flush — a known, bounded window (registration is rare and the
	// batch is capped by maxPendingMessagesPerSession).
	mu      sync.Mutex
	backend Backend
	// participants is the participant set of this round, immutable after registration. alice's
	// CGGMP cores pass incoming messages through EchoMsgMain, which attributes a message by its
	// own Id field (honest nodes echo it, and the core then treats the echoed Id as the sender)
	// rather than by the transport peer id it was delivered with. The allowlist is what closes
	// that hole: a connected peer outside the round cannot contribute a message at all, so it
	// cannot forge another participant's Id by getting an honest node to echo it.
	participants map[string]struct{}
}

// isParticipant reports whether peerID belongs to this round.
func (s *sessionCore) isParticipant(peerID string) bool {
	_, ok := s.participants[peerID]
	return ok
}

// pendingMessage is a buffered message together with the transport-authenticated peer id it
// arrived from. The peer id is captured at arrival time because the message is flushed into the
// alice core only once the local session is registered, when the transport identity is no
// longer available.
type pendingMessage struct {
	peerID string
	msg    types.Message
}

// The session registry is per-process: in a real deployment every TSS node is its own
// process, so the same protocol|sessionID key does not collide across nodes. This mirrors
// the gg18 wrapper. CGGMP keeps a private registry so GG18 and CGGMP sessions coexist.
var (
	sessionsMu                   sync.RWMutex
	sessions                     = make(map[string]*sessionCore)
	maxPendingSessions           = 100
	pendingMessages              = make(map[string][]pendingMessage, maxPendingSessions)
	maxPendingMessagesPerSession = 32

	// runningSessions holds the protocol|sessionID keys with a Process* call in flight in this
	// process. One session id names one round, so two concurrent attempts are never legitimate:
	// the second would reset the first one's ppk collector (beginPPKSession deletes it) and both
	// would fail. Reject the second attempt up front instead.
	runningSessions = make(map[string]struct{})
)

// beginSessionRun claims protocol|sessionID for the calling Process*. It must be paired with
// endSessionRun (defer) on every return path.
func beginSessionRun(protocol, sessionID string) error {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if _, running := runningSessions[id]; running {
		return fmt.Errorf("%w: %s", errSessionRunning, id)
	}
	runningSessions[id] = struct{}{}
	return nil
}

func endSessionRun(protocol, sessionID string) {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(runningSessions, id)
}

// addMessage hands an incoming tss message to the alice core of its session. peerID is the
// transport-authenticated sender (the libp2p peer id the message arrived from, i.e.
// tss.MessageWrapper.PeerID), never the message's own Id field.
//
// It is not enough to prefer the transport id: every CGGMP core wraps its message main in alice's
// EchoMsgMain, which ignores the senderID argument, keys its echo table by msg.Id, and hands the
// core m.originalMsg.GetId() as the sender once NumPeers copies agree. A connected peer outside
// the round could therefore send a body with Id=<victim>, have the honest nodes echo it, and be
// processed as that participant (DoS: the round fails once the real message arrives with a
// different hash). The participant allowlist below is what actually binds a message to a
// participant: a peer id outside the round is rejected before the core ever sees it. (The GG18
// wrapper relies on alice's MsgMain, which does compare senderID with msg.Id.)
func addMessage(protocol, sessionID, peerID string, msg types.Message) error {
	if peerID == "" {
		// Without a transport-authenticated identity there is nothing to attribute the message
		// to; falling back to msg.Id would be trusting attacker-controlled data.
		return errMissingPeerID
	}
	id := tss.ComposeProtocol(protocol, sessionID)
	// Look the session up under the registry lock, but hand the message to the core *after*
	// releasing it: the per-session lock below keeps one session's AddMessage calls serialised,
	// while other sessions stay free to make progress.
	sessionsMu.RLock()
	session, ok := sessions[id]
	sessionsMu.RUnlock()
	if ok {
		return session.add(peerID, msg)
	}

	sessionsMu.Lock()
	// Re-check under the write lock: a session may have been registered between the read above
	// and this acquisition. Buffering it for a session that is already live would leave the
	// message unflushed for ever.
	if session, ok := sessions[id]; ok {
		sessionsMu.Unlock()
		return session.add(peerID, msg)
	}
	defer sessionsMu.Unlock()
	if len(pendingMessages[id]) >= maxPendingMessagesPerSession {
		return fmt.Errorf("addMessage max pending messages reached")
	}
	if len(pendingMessages) >= maxPendingSessions {
		log.Debug("addMessage max pending sessions reached, clear pending messages")
		for id := range pendingMessages {
			delete(pendingMessages, id)
		}
	}
	pendingMessages[id] = append(pendingMessages[id], pendingMessage{peerID: peerID, msg: msg})
	return nil
}

// add serialises AddMessage calls for one session. removeSession only drops the registry entry;
// the core pointer stays valid, so a message that raced with removal is still handled by the
// (possibly stopped) core rather than dereferencing freed memory.
//
// Messages from peers outside the round are dropped here: see the comment on addMessage.
func (s *sessionCore) add(peerID string, msg types.Message) error {
	if !s.isParticipant(peerID) {
		log.Warn("sessionCore.add drop message from non-participant", "peerID", peerID)
		return errPeerNotParticipant
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.AddMessage(peerID, msg)
}

// registerSession binds backend to protocol|sessionID and flushes the messages that were buffered
// for it while no session was registered.
//
// Registration is all-or-nothing. A session id has to be reusable after a failed round: the DKG
// session name is a constant shared by every participant (alice derives the DKG ZK challenges from
// the sid, so the names must match across nodes — a per-node retry-unique name would split the
// group into different rounds), which means the retry loop that restarts a failed DKG calls this
// with exactly the same id again. Registering the backend *before* the flush therefore left a
// half-registered session behind on a flush error, and every later attempt then failed with
// "session already registered" — a node that timed out once never came back (2026-09-19 E2E:
// all four participants wedged this way after a single DKG timeout). Roll the entry back instead.
//
// The buffered batch is taken out of the map before the flush, so it is consumed either way: a
// message the core rejects means the batch mixes two rounds (alice's echo layer keys messages by
// (type, sender) and rejects a second, different body with ErrDifferentHash), and the rest of that
// batch is no more trustworthy than the message that failed. Dropping it makes the next attempt
// start from a clean slate instead of failing on the same stale batch for ever.
func registerSession(protocol, sessionID string, backend Backend, participants []string) error {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	_, ok := sessions[id]
	if ok {
		return fmt.Errorf("session already registered")
	}
	allowed := make(map[string]struct{}, len(participants))
	for _, peerID := range participants {
		allowed[peerID] = struct{}{}
	}
	pending := pendingMessages[id]
	delete(pendingMessages, id)
	sessions[id] = &sessionCore{
		backend:      backend,
		participants: allowed,
	}
	// flush buffer messages
	for _, pendingMsg := range pending {
		// The participant set is only known now, so a message buffered before registration is
		// filtered here rather than at arrival time (addMessage cannot know the round's members
		// yet). Dropping it silently is correct: it could never have been a contribution to this
		// round.
		if _, member := allowed[pendingMsg.peerID]; !member {
			log.Warn("registerSession drop buffered message from non-participant",
				"session", sessionID, "peerID", pendingMsg.peerID)
			continue
		}
		err := backend.AddMessage(pendingMsg.peerID, pendingMsg.msg)
		if err != nil {
			log.Error("registerSession", "session", sessionID, "Cannot add pending message to core, err", err)
			delete(sessions, id)
			return err
		}
	}
	return nil
}

// removeSession unregisters a finished (or failed) session and drops anything still buffered for
// it. A session id names one round: leftovers of the round that just ended must never be flushed
// into the next one, where they arrive as a second, conflicting body for the same (message type,
// sender) and break it (see registerSession). This mirrors the ppk registry in handlers.go, which
// clears its buffer in removePPKCollector for the same reason.
func removeSession(protocol, sessionID string) {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(sessions, id)
	delete(pendingMessages, id)
}
