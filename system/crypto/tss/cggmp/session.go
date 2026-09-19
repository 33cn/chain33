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
	backend Backend
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
)

// addMessage hands an incoming tss message to the alice core of its session. peerID is the
// transport-authenticated sender (the libp2p peer id the message arrived from, i.e.
// tss.MessageWrapper.PeerID), never the message's own Id field: alice's core rejects a message
// whose Id disagrees with the sender it is attributed to, and Id is filled in by the sender, so
// trusting it would let a peer impersonate another participant.
func addMessage(protocol, sessionID, peerID string, msg types.Message) error {
	if peerID == "" {
		// Without a transport-authenticated identity there is nothing to attribute the message
		// to; falling back to msg.Id would be trusting attacker-controlled data.
		return errMissingPeerID
	}
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	session, ok := sessions[id]
	if ok {
		return session.backend.AddMessage(peerID, msg)
	}
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
func registerSession(protocol, sessionID string, backend Backend) error {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	_, ok := sessions[id]
	if ok {
		return fmt.Errorf("session already registered")
	}
	pending := pendingMessages[id]
	delete(pendingMessages, id)
	sessions[id] = &sessionCore{
		backend: backend,
	}
	// flush buffer messages
	for _, pendingMsg := range pending {
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
