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
type ppkCollector struct {
	total int                                 // number of other participants to collect
	got   map[string]*ecpointgrouplaw.ECPoint // sender id -> point
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

func (c *ppkCollector) add(sender string, p *ecpointgrouplaw.ECPoint) {
	if p == nil {
		return
	}
	if _, dup := c.got[sender]; dup {
		return
	}
	c.got[sender] = p
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

func (c *ppkCollector) result() map[string]*ecpointgrouplaw.ECPoint {
	return c.got
}

var (
	ppkMu         sync.Mutex
	ppkCollectors = make(map[string]*ppkCollector)
	// pendingPPK buffers partial public keys that arrive before the local ProcessDKG has
	// registered its collector (a node may finish DKG and broadcast before the others do).
	pendingPPK = make(map[string][]*PartialPublicKey)
)

func registerPPKCollector(sessionID string, c *ppkCollector) {
	key := tss.ComposeProtocol(PpkProtocol, sessionID)
	ppkMu.Lock()
	defer ppkMu.Unlock()
	ppkCollectors[key] = c
	for _, msg := range pendingPPK[key] {
		addPPKToCollector(c, msg)
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

func addPPKToCollector(c *ppkCollector, msg *PartialPublicKey) {
	p, err := ecpointgrouplaw.NewECPoint(elliptic.Secp256k1(),
		new(big.Int).SetBytes(msg.X), new(big.Int).SetBytes(msg.Y))
	if err != nil {
		log.Error("addPPKToCollector invalid point", "sender", msg.Sender, "err", err)
		return
	}
	c.add(msg.Sender, p)
}

func handlePpkMsg(wMsg *tss.MessageWrapper) {
	if wMsg.Protocol != PpkProtocol {
		log.Error("handlePpkMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	msg := &PartialPublicKey{}
	if err := types.Decode(wMsg.Msg, msg); err != nil {
		log.Error("handlePpkMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	key := tss.ComposeProtocol(PpkProtocol, wMsg.SessionID)
	ppkMu.Lock()
	defer ppkMu.Unlock()
	if c, ok := ppkCollectors[key]; ok {
		addPPKToCollector(c, msg)
		return
	}
	pendingPPK[key] = append(pendingPPK[key], msg)
}

// partialPublicKeyMsg builds the on-wire partial public key message for the local node.
func partialPublicKeyMsg(sender string, p *ecpointgrouplaw.ECPoint) *PartialPublicKey {
	return &PartialPublicKey{
		Sender: sender,
		X:      p.GetX().Bytes(),
		Y:      p.GetY().Bytes(),
	}
}
