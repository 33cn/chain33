// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"math/big"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/33cn/chain33/types"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicerefresh "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/refresh"
	alicesign "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/sign"
)

const (
	// DkgProtocol is the CGGMP DKG protocol tag.
	DkgProtocol = "/cggmp/dkg"
	// RefreshProtocol is the CGGMP refresh (key refresh + Paillier/Pedersen provisioning) tag.
	RefreshProtocol = "/cggmp/refresh"
	// SignProtocol is the CGGMP (4-round) sign protocol tag.
	SignProtocol = "/cggmp/sign"
	// PpkProtocol is the CGGMP partial-public-key exchange protocol tag, run right after DKG.
	PpkProtocol = "/cggmp/ppk"
)

var (
	log = log15.New("module", "tss.cggmp")
)

func init() {
	tss.RegisterMsgHandler(DkgProtocol, handleDkgMsg)
	tss.RegisterMsgHandler(RefreshProtocol, handleRefreshMsg)
	tss.RegisterMsgHandler(SignProtocol, handleSignMsg)
	tss.RegisterMsgHandler(PpkProtocol, handlePpkMsg)
}

func handleDkgMsg(wMsg *tss.MessageWrapper) {
	if wMsg.Protocol != DkgProtocol {
		log.Error("handleDkgMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	msg := &alicedkg.Message{}
	if err := types.Decode(wMsg.Msg, msg); err != nil {
		log.Error("handleDkgMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	if err := addMessage(DkgProtocol, wMsg.SessionID, msg); err != nil {
		log.Error("handleDkgMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "Cannot add message to core, err", err)
	}
}

func handleRefreshMsg(wMsg *tss.MessageWrapper) {
	if wMsg.Protocol != RefreshProtocol {
		log.Error("handleRefreshMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	msg := &alicerefresh.Message{}
	if err := types.Decode(wMsg.Msg, msg); err != nil {
		log.Error("handleRefreshMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	if err := addMessage(RefreshProtocol, wMsg.SessionID, msg); err != nil {
		log.Error("handleRefreshMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "Cannot add message to core, err", err)
	}
}

func handleSignMsg(wMsg *tss.MessageWrapper) {
	if wMsg.Protocol != SignProtocol {
		log.Error("handleSignMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	msg := &alicesign.Message{}
	if err := types.Decode(wMsg.Msg, msg); err != nil {
		log.Error("handleSignMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	if err := addMessage(SignProtocol, wMsg.SessionID, msg); err != nil {
		log.Error("handleSignMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "Cannot add message to core, err", err)
	}
}

// ---- partial-public-key exchange (post-DKG) ----

// ppkCollector accumulates the partial public keys broadcast by the other participants of a
// CGGMP DKG session. It is a lightweight per-process registry, separate from the alice-core
// session registry, because partial public keys travel as PartialPublicKey protos rather than
// alice types.Message.
//
// A partial public key is always attributed to the *transport-authenticated* peer id
// (tss.MessageWrapper.PeerID), never to the peer-supplied PartialPublicKey.Sender field:
// Sender is attacker-controlled and keying the collection by it would let one peer satisfy the
// quorum alone (filling the map with forged sender ids) or impersonate another participant's
// g^{share}. mu guards got: add runs on the queue message-handler goroutine while result is
// read by the ProcessDKG goroutine.
type ppkCollector struct {
	mu    sync.Mutex
	total int                                 // number of other participants to collect
	got   map[string]*ecpointgrouplaw.ECPoint // authenticated peer id -> point
	done  chan struct{}
	once  sync.Once
}

func newPPKCollector(total int) *ppkCollector {
	return &ppkCollector{
		total: total,
		got:   make(map[string]*ecpointgrouplaw.ECPoint, total),
		done:  make(chan struct{}),
	}
}

func (c *ppkCollector) add(peerID string, p *ecpointgrouplaw.ECPoint) {
	if p == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, dup := c.got[peerID]; dup {
		return
	}
	c.got[peerID] = p
	if len(c.got) >= c.total {
		c.once.Do(func() { close(c.done) })
	}
}

func (c *ppkCollector) wait(timeout time.Duration) error {
	if c.total <= 0 {
		return nil
	}
	select {
	case <-c.done:
		return nil
	case <-time.After(timeout):
		return errPPKTimeout
	}
}

// result returns a snapshot of the collected partial public keys, so the caller never shares a
// map with the add path (which may still be running for messages that arrive late).
func (c *ppkCollector) result() map[string]*ecpointgrouplaw.ECPoint {
	c.mu.Lock()
	defer c.mu.Unlock()
	got := make(map[string]*ecpointgrouplaw.ECPoint, len(c.got))
	for id, p := range c.got {
		got[id] = p
	}
	return got
}

// pendingPPKMsg is a buffered partial public key together with the authenticated peer id it
// came from, so it can be attributed correctly when it is flushed into a late collector.
type pendingPPKMsg struct {
	peerID string
	msg    *PartialPublicKey
}

// The ppk buffer is bounded twice over — participants per session and number of sessions — like
// the alice-core message buffer in session.go, so a malicious peer cannot grow it without bound
// with duplicates or with random session ids.
const (
	maxPendingPPKPerSession = 32
	maxPendingPPKSessions   = 100
)

var (
	ppkMu         sync.Mutex
	ppkCollectors = make(map[string]*ppkCollector)
	// pendingPPK buffers partial public keys that arrive before the local ProcessDKG has
	// registered its collector (a node may finish DKG and broadcast before the others do).
	pendingPPK = make(map[string][]pendingPPKMsg)
)

func registerPPKCollector(sessionID string, c *ppkCollector) {
	key := tss.ComposeProtocol(PpkProtocol, sessionID)
	ppkMu.Lock()
	defer ppkMu.Unlock()
	ppkCollectors[key] = c
	for _, pending := range pendingPPK[key] {
		addPPKToCollector(c, pending.peerID, pending.msg)
	}
	delete(pendingPPK, key)
}

func removePPKCollector(sessionID string) {
	key := tss.ComposeProtocol(PpkProtocol, sessionID)
	ppkMu.Lock()
	defer ppkMu.Unlock()
	delete(ppkCollectors, key)
	delete(pendingPPK, key)
}

// addPPKToCollector records the partial public key of the authenticated peer peerID. It is
// deliberately independent of msg.Sender: the point is attributed by transport identity, so a
// forged Sender cannot add a second entry for the same peer.
func addPPKToCollector(c *ppkCollector, peerID string, msg *PartialPublicKey) {
	p, err := ecpointgrouplaw.NewECPoint(elliptic.Secp256k1(),
		new(big.Int).SetBytes(msg.X), new(big.Int).SetBytes(msg.Y))
	if err != nil {
		log.Error("addPPKToCollector invalid point", "peerID", peerID, "err", err)
		return
	}
	c.add(peerID, p)
}

func handlePpkMsg(wMsg *tss.MessageWrapper) {
	if wMsg.Protocol != PpkProtocol {
		log.Error("handlePpkMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	// The transport fills PeerID with the authenticated remote peer id; without it the partial
	// public key cannot be attributed to a participant, so it must be dropped rather than
	// buffered under an unauthenticated key.
	if wMsg.PeerID == "" {
		log.Error("handlePpkMsg", "session", wMsg.SessionID, "missing authenticated peer id")
		return
	}
	msg := &PartialPublicKey{}
	if err := types.Decode(wMsg.Msg, msg); err != nil {
		log.Error("handlePpkMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	// PartialPublicKey.Sender is untrusted and is never used for attribution or validation: a
	// disagreement with the authenticated peer id is reported and then ignored.
	if msg.Sender != "" && msg.Sender != wMsg.PeerID {
		log.Warn("handlePpkMsg sender does not match authenticated peer id, sender ignored",
			"peerID", wMsg.PeerID, "sender", msg.Sender, "session", wMsg.SessionID)
	}
	key := tss.ComposeProtocol(PpkProtocol, wMsg.SessionID)
	ppkMu.Lock()
	defer ppkMu.Unlock()
	if c, ok := ppkCollectors[key]; ok {
		addPPKToCollector(c, wMsg.PeerID, msg)
		return
	}
	if _, ok := pendingPPK[key]; !ok && len(pendingPPK) >= maxPendingPPKSessions {
		log.Warn("handlePpkMsg max pending ppk sessions reached, clear pending ppk messages")
		for id := range pendingPPK {
			delete(pendingPPK, id)
		}
	}
	// At most one buffered point per authenticated peer, first one wins — the same rule the
	// collector applies when it is flushed (see ppkCollector.add).
	for _, pending := range pendingPPK[key] {
		if pending.peerID == wMsg.PeerID {
			return
		}
	}
	if len(pendingPPK[key]) >= maxPendingPPKPerSession {
		log.Warn("handlePpkMsg max pending ppk messages reached, drop message",
			"peerID", wMsg.PeerID, "session", wMsg.SessionID)
		return
	}
	pendingPPK[key] = append(pendingPPK[key], pendingPPKMsg{peerID: wMsg.PeerID, msg: msg})
}

// partialPublicKeyMsg builds the on-wire partial public key message for the local node.
// sender is informational only (a receiver must attribute the point by the authenticated peer
// id, see handlePpkMsg), but honest nodes fill it in with their own id so a mismatch is
// visible in the logs.
func partialPublicKeyMsg(sender string, p *ecpointgrouplaw.ECPoint) *PartialPublicKey {
	return &PartialPublicKey{
		Sender: sender,
		X:      p.GetX().Bytes(),
		Y:      p.GetY().Bytes(),
	}
}
