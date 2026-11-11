// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/getamis/alice/crypto/birkhoffinterpolation"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	alicggmp "github.com/getamis/alice/crypto/tss/ecdsa/cggmp"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicerefresh "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/refresh"
	alicesign "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/sign"
	alicetypes "github.com/getamis/alice/types"
)

const (
	// defaultTimeout is the default timeout of the DKG, ppk and sign phases.
	defaultTimeout = 30 * time.Second
	// refreshDefaultTimeout is the default timeout of the refresh phase. Unlike the other
	// phases, refresh generates a fresh 2048-bit Paillier key (safe primes) on every node,
	// which measures ~23s for 4 nodes on localhost alone — the 30s of the other phases would
	// time out in production, so refresh gets its own, much larger default.
	refreshDefaultTimeout = 5 * time.Minute
	// paillierKeySize is the RSA modulus bit length used for the CGGMP Paillier keys.
	paillierKeySize = 2048
)

type config struct {
	timeout    time.Duration
	timeoutSet bool
}

// Option sets configuration options for CGGMP operations.
type Option func(*config)

// WithTimeout overrides the per-phase default timeout for every phase. A non-positive value
// means no timeout (see tss.NewListener).
func WithTimeout(timeout time.Duration) Option {
	return func(o *config) {
		o.timeout = timeout
		o.timeoutSet = true
	}
}

func newConfig(opts ...Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// phaseTimeout returns the timeout to use for a protocol phase: the explicit WithTimeout
// override when the caller set one, otherwise the per-phase default.
func (c *config) phaseTimeout(protocol string) time.Duration {
	if c.timeoutSet {
		return c.timeout
	}
	if protocol == RefreshProtocol {
		return refreshDefaultTimeout
	}
	return defaultTimeout
}

// validateNonEmptyPeers rejects the participant sets that would otherwise reach alice or a peer
// manager: no peers, a blank or duplicated peer id, or a threshold alice cannot work with
// (CGGMP requires at least 2). Phase-specific checks (e.g. the signer count must equal the
// threshold) are applied by the caller; both refresh and DKG accept any threshold up to the
// participant count, see validateParticipants.
func validateNonEmptyPeers(peers []string, threshold uint32) error {
	if len(peers) == 0 {
		return errNoPeers
	}
	if threshold < 2 {
		return fmt.Errorf("%w: threshold=%d must be at least 2", errInvalidThreshold, threshold)
	}
	seen := make(map[string]struct{}, len(peers))
	for _, id := range peers {
		if id == "" {
			return fmt.Errorf("%w: blank peer id", errInvalidPeerList)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("%w: duplicate peer id %q", errInvalidPeerList, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// validateParticipants rejects a participant set that cannot run DKG/refresh: besides the
// non-empty checks, the threshold may not exceed the number of participants. It runs before a
// peer manager is created, so an impossible set fails immediately instead of blocking on peer
// discovery.
func validateParticipants(peers []string, threshold uint32) error {
	if err := validateNonEmptyPeers(peers, threshold); err != nil {
		return err
	}
	if uint64(threshold) > uint64(len(peers)) {
		return fmt.Errorf("%w: threshold=%d, peers=%d", errInvalidThreshold, threshold, len(peers))
	}
	return nil
}

// validateSessionID rejects session ids the registry keys cannot represent. The registry key is
// protocol + separator + sessionID, so an empty id collapses every session of a protocol onto the
// bare protocol tag, and an id containing the separator makes two different (protocol, sessionID)
// pairs collide on the same key — a peer could then drive another session's core.
func validateSessionID(sessionID string) error {
	if sessionID == "" {
		return errEmptySessionID
	}
	if strings.Contains(sessionID, tss.SessionProtocolSeparator) {
		return fmt.Errorf("%w: %q", errInvalidSessionID, sessionID)
	}
	return nil
}

// validateSelf returns the transport-authenticated local peer id and requires it to be part of
// peers. alice indexes its bks / partial public key / Pedersen maps by SelfID without checking, so
// a peer list that omits the local node panics inside the protocol core; reject it here instead.
func validateSelf(pm alicetypes.PeerManager, peers []string) (string, error) {
	selfID := pm.SelfID()
	if selfID == "" {
		return "", errMissingSelfID
	}
	for _, id := range peers {
		if id == selfID {
			return selfID, nil
		}
	}
	return "", fmt.Errorf("%w: %q, peers=%v", errSelfNotInPeers, selfID, peers)
}

// otherPeers returns peers without selfID, i.e. the participants whose partial public keys the
// local node has to collect.
func otherPeers(peers []string, selfID string) []string {
	others := make([]string, 0, len(peers))
	for _, id := range peers {
		if id != selfID {
			others = append(others, id)
		}
	}
	return others
}

// ProcessDKG runs the CGGMP DKG protocol and then the CGGMP partial-public-key exchange,
// returning a persisted DKGResult (group public key, this node's share, Birkhoff parameters,
// rid and every participant's partial public key). sessionID must be identical across nodes;
// peers must include the local node id. CGGMP sessions are independent from the GG18 wrapper.
func ProcessDKG(peers []string, threshold, rank uint32, sessionID string, opts ...Option) (*DKGResult, error) {
	if err := validateSessionID(sessionID); err != nil {
		log.Error("ProcessDKG", "session", sessionID, "invalid session id", err)
		return nil, err
	}
	if err := validateParticipants(peers, threshold); err != nil {
		log.Error("ProcessDKG", "session", sessionID, "invalid participants", err)
		return nil, err
	}
	// alice's EnsureRank rejects rank+1 >= threshold (rank <= threshold-2 is the valid range).
	// Check it before the peer manager, so an impossible input fails at once instead of blocking
	// on peer discovery. threshold >= 2 is already enforced by validateParticipants, so
	// threshold-1 cannot underflow.
	if rank >= threshold-1 {
		err := fmt.Errorf("%w: rank=%d, threshold=%d", errInvalidRank, rank, threshold)
		log.Error("ProcessDKG", "session", sessionID, "invalid rank", err)
		return nil, err
	}
	if err := beginSessionRun(DkgProtocol, sessionID); err != nil {
		log.Error("ProcessDKG", "session", sessionID, "session already running", err)
		return nil, err
	}
	defer endSessionRun(DkgProtocol, sessionID)
	cfg := newConfig(opts...)
	curve := elliptic.Secp256k1()

	// Phase 1: alice CGGMP DKG.
	pm, err := tss.NewReadyPeerManagerWithTimeout(peers, DkgProtocol, sessionID, cfg.phaseTimeout(DkgProtocol))
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "peers not ready", err)
		return nil, err
	}
	selfID, err := validateSelf(pm, peers)
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "invalid local peer", err)
		return nil, err
	}
	// A retry reuses this session id; drop the partial public keys buffered by a previous failed
	// round so they cannot be flushed into this round's collector (see beginPPKSession).
	beginPPKSession(sessionID, otherPeers(peers, selfID))
	defer removePPKCollector(sessionID)
	listener := tss.NewListener(DkgProtocol, cfg.phaseTimeout(DkgProtocol))
	dkgCore, err := alicedkg.NewDKG(curve, pm, []byte(sessionID), threshold, rank, listener)
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "NewDKG err", err)
		return nil, err
	}
	if err := registerSession(DkgProtocol, sessionID, dkgCore, peers); err != nil {
		log.Error("ProcessDKG", "session", sessionID, "register session failed, err", err)
		return nil, err
	}
	defer removeSession(DkgProtocol, sessionID)
	dkgCore.Start()
	defer dkgCore.Stop()
	if err := listener.Wait(); err != nil {
		log.Error("ProcessDKG", "session", sessionID, "listener done err", err)
		return nil, err
	}
	aliceRes, err := dkgCore.GetResult()
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "GetResult err", err)
		return nil, err
	}

	// Phase 2: partial public key (g^{share}) exchange. The refresh phase needs every
	// participant's partial public key as an input, so we exchange them right after DKG.
	partialPubKeys, err := exchangePartialPubKeys(peers, selfID, aliceRes.Share, curve, sessionID, aliceRes.Rid, cfg.phaseTimeout(PpkProtocol))
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "partial pubkey exchange err", err)
		return nil, err
	}
	dkgResult := newDKGResult(aliceRes, partialPubKeys, threshold, peers)
	// The ppk exchange is the last point at which the DKG material is still local and complete:
	// verify that the collected g^{share_i} reconstruct the group public key so a participant
	// that broadcast bad material is rejected here rather than in the refresh phase.
	if err := dkgResult.validatePartialPubKeys(peers, threshold); err != nil {
		log.Error("ProcessDKG", "session", sessionID, "partial pubkey check err", err)
		return nil, err
	}
	return dkgResult, nil
}

// exchangePartialPubKeys broadcasts the local g^{share} to all peers and collects theirs. rid is
// the DKG rid of this round: the collector only accepts points tagged with it, so a point from a
// previous round that reused the same session id cannot occupy a slot.
func exchangePartialPubKeys(peers []string, selfID string, share *big.Int, curve elliptic.Curve, sessionID string, rid []byte, timeout time.Duration) (map[string]*ecpointgrouplaw.ECPoint, error) {
	myPPK := ecpointgrouplaw.ScalarBaseMult(curve, share)
	pm, err := tss.NewReadyPeerManagerWithTimeout(peers, PpkProtocol, sessionID, timeout)
	if err != nil {
		return nil, err
	}
	collector := newPPKCollector(rid, otherPeers(peers, selfID))
	registerPPKCollector(sessionID, collector)
	defer removePPKCollector(sessionID)

	myMsg := partialPublicKeyMsg(selfID, rid, myPPK)
	for _, id := range pm.PeerIDs() {
		pm.MustSend(id, myMsg)
	}
	if err := collector.wait(timeout); err != nil {
		log.Error("exchangePartialPubKeys", "session", sessionID, "err", err)
		return nil, err
	}
	got := collector.result()
	got[selfID] = myPPK
	return got, nil
}

// ProcessRefresh runs the CGGMP refresh phase (key refresh plus Paillier/Pedersen
// provisioning) for the key produced by ProcessDKG. It must be run once per key before any
// ProcessSign; the returned RefreshResult together with the DKGResult is all ProcessSign
// needs. sessionID must be identical across nodes; peers includes the local node id.
// Repeated refreshes of the same key use ProcessRefreshAgain.
func ProcessRefresh(peers []string, threshold uint32, dkgRes *DKGResult, sessionID string, opts ...Option) (*RefreshResult, error) {
	if dkgRes == nil {
		return nil, errNilDKGResult
	}
	if err := validateSessionID(sessionID); err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "invalid session id", err)
		return nil, err
	}
	if err := validateParticipants(peers, threshold); err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "invalid participants", err)
		return nil, err
	}
	if err := dkgRes.validateThresholdAndPeers(peers, threshold); err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "dkg result mismatch", err)
		return nil, err
	}
	if err := beginSessionRun(RefreshProtocol, sessionID); err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "session already running", err)
		return nil, err
	}
	defer endSessionRun(RefreshProtocol, sessionID)
	cfg := newConfig(opts...)

	// Validate the persisted ppk material before handing it to alice: refresh needs every
	// participant's bk and g^{share}, and a missing or inconsistent entry would otherwise be
	// dropped silently (toAlicePartialPubKeys) or fail opaquely inside the refresh rounds.
	if err := dkgRes.validatePartialPubKeys(peers, threshold); err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "partial pubkey check err", err)
		return nil, err
	}
	aRes, err := dkgRes.aliceResult(peers)
	if err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "aliceResult err", err)
		return nil, err
	}
	partialPubKeys, err := dkgRes.toAlicePartialPubKeys(peers)
	if err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "partial pubkeys err", err)
		return nil, err
	}
	ssid := computeSSID(sessionID, dkgRes.Rid)
	return refreshRound(peers, threshold, aRes.PublicKey, aRes.Bks, aRes.Share, partialPubKeys,
		ssid, sessionID, cfg.phaseTimeout(RefreshProtocol))
}

// ProcessRefreshAgain runs a further CGGMP refresh (key rotation) of the same key. Instead of the
// DKG material, it seeds the round with prev's share and partial public keys and returns the next
// generation, which ProcessSign can use exactly like a first-refresh result. Use ProcessRefresh
// for the first refresh of a key, and this entry point for every refresh after it.
//
// dkgRes supplies the immutable parts (group public key, Birkhoff parameters, rid and the
// threshold/participant check); prev must have been produced by a refresh of exactly the same
// participant set. sessionID must be identical across nodes and names this round.
//
// alice's refresh.Result carries no rid of its own, and the ssid is an opaque, caller-supplied
// domain separator for the zero-knowledge challenges. The next ssid is therefore derived from the
// previous round's ssid (see computeRefreshAgainSSID), so consecutive refreshes never reuse the
// same challenge domain even if the caller reuses a session id.
func ProcessRefreshAgain(peers []string, threshold uint32, dkgRes *DKGResult, prev *RefreshResult, sessionID string, opts ...Option) (*RefreshResult, error) {
	if dkgRes == nil {
		return nil, errNilDKGResult
	}
	if prev == nil {
		return nil, errNilRefreshResult
	}
	if err := validateSessionID(sessionID); err != nil {
		log.Error("ProcessRefreshAgain", "session", sessionID, "invalid session id", err)
		return nil, err
	}
	if err := validateParticipants(peers, threshold); err != nil {
		log.Error("ProcessRefreshAgain", "session", sessionID, "invalid participants", err)
		return nil, err
	}
	if err := dkgRes.validateThresholdAndPeers(peers, threshold); err != nil {
		log.Error("ProcessRefreshAgain", "session", sessionID, "dkg result mismatch", err)
		return nil, err
	}
	if err := beginSessionRun(RefreshProtocol, sessionID); err != nil {
		log.Error("ProcessRefreshAgain", "session", sessionID, "session already running", err)
		return nil, err
	}
	defer endSessionRun(RefreshProtocol, sessionID)
	cfg := newConfig(opts...)

	// dkgRes's own share / partial public keys belong to the generation that prev already rotated
	// away from; only its immutable parts are used here.
	aRes, err := dkgRes.aliceResult(peers)
	if err != nil {
		log.Error("ProcessRefreshAgain", "session", sessionID, "aliceResult err", err)
		return nil, err
	}
	share, partialPubKeys, err := prev.refreshInputs(dkgRes, peers, threshold)
	if err != nil {
		log.Error("ProcessRefreshAgain", "session", sessionID, "refresh input check err", err)
		return nil, err
	}
	ssid := computeRefreshAgainSSID(sessionID, prev.Ssid, dkgRes.Rid)
	return refreshRound(peers, threshold, aRes.PublicKey, aRes.Bks, share, partialPubKeys,
		ssid, sessionID, cfg.phaseTimeout(RefreshProtocol))
}

// checkSelfShare rejects a local share that does not encode the same point as the local partial
// public key. The refresh takes both as separate inputs, so a share and a ppk from different
// generations would only be caught by alice's final Birkhoff check, after the Paillier work.
func checkSelfShare(curve elliptic.Curve, share *big.Int, selfPPK *ecpointgrouplaw.ECPoint, selfID string) error {
	if selfPPK == nil {
		return fmt.Errorf("%w: %s", errMissingPeerPPK, selfID)
	}
	if !selfPPK.Equal(ecpointgrouplaw.ScalarBaseMult(curve, share)) {
		return fmt.Errorf("%w: %s", errRefreshShareMismatch, selfID)
	}
	return nil
}

// refreshRound runs one alice CGGMP refresh round with the given key material and returns the
// persisted result tagged with the ssid it ran with. share and partialPubKeys must belong to the
// same generation and be consistent with publicKey/bks; the caller checks that before calling.
func refreshRound(peers []string, threshold uint32, publicKey *ecpointgrouplaw.ECPoint,
	bks map[string]*birkhoffinterpolation.BkParameter, share *big.Int,
	partialPubKeys map[string]*ecpointgrouplaw.ECPoint, ssid []byte,
	sessionID string, timeout time.Duration) (*RefreshResult, error) {
	pm, err := tss.NewReadyPeerManagerWithTimeout(peers, RefreshProtocol, sessionID, timeout)
	if err != nil {
		log.Error("refreshRound", "session", sessionID, "peers not ready", err)
		return nil, err
	}
	selfID, err := validateSelf(pm, peers)
	if err != nil {
		log.Error("refreshRound", "session", sessionID, "invalid local peer", err)
		return nil, err
	}
	// alice seeds the new generation from the share and, independently, from the ppk map. If the
	// two come from different generations, the round only fails at its final Birkhoff check,
	// after generating the Paillier key; reject the mismatch before starting it.
	if err := checkSelfShare(publicKey.GetCurve(), share, partialPubKeys[selfID], selfID); err != nil {
		log.Error("refreshRound", "session", sessionID, "invalid local share", err)
		return nil, err
	}
	listener := tss.NewListener(RefreshProtocol, timeout)
	refreshCore, err := alicerefresh.NewRefresh(share, publicKey, pm, threshold,
		partialPubKeys, bks, paillierKeySize, ssid, listener)
	if err != nil {
		log.Error("refreshRound", "session", sessionID, "NewRefresh err", err)
		return nil, err
	}
	if err := registerSession(RefreshProtocol, sessionID, refreshCore, peers); err != nil {
		log.Error("refreshRound", "session", sessionID, "register session failed, err", err)
		return nil, err
	}
	defer removeSession(RefreshProtocol, sessionID)
	refreshCore.Start()
	defer refreshCore.Stop()
	if err := listener.Wait(); err != nil {
		log.Error("refreshRound", "session", sessionID, "listener done err", err)
		return nil, err
	}
	aliceRefresh, err := refreshCore.GetResult()
	if err != nil {
		log.Error("refreshRound", "session", sessionID, "GetResult err", err)
		return nil, err
	}
	return newRefreshResult(aliceRefresh, ssid), nil
}

// ProcessSign runs the CGGMP (4-round) threshold sign for msg. dkgRes is the DKG output of the
// key and refreshRes the corresponding refresh output. peers is the signer subset (includes the
// local node id, size == threshold, which is enforced); each node must call with the same peers
// list, and every listed signer must have a bk, a partial public key and Pedersen parameters in
// the two results. sessionID must be identical across the signing nodes.
func ProcessSign(peers []string, threshold uint32, msg []byte, dkgRes *DKGResult, refreshRes *RefreshResult, sessionID string, opts ...Option) (*alicesign.Result, error) {
	if dkgRes == nil {
		return nil, errNilDKGResult
	}
	if refreshRes == nil {
		return nil, errNilRefreshResult
	}
	if err := validateSessionID(sessionID); err != nil {
		log.Error("ProcessSign", "session", sessionID, "invalid session id", err)
		return nil, err
	}
	if err := validateNonEmptyPeers(peers, threshold); err != nil {
		log.Error("ProcessSign", "session", sessionID, "invalid participants", err)
		return nil, err
	}
	if err := dkgRes.validateThresholdAndPeers(peers, threshold); err != nil {
		log.Error("ProcessSign", "session", sessionID, "dkg result mismatch", err)
		return nil, err
	}
	if err := beginSessionRun(SignProtocol, sessionID); err != nil {
		log.Error("ProcessSign", "session", sessionID, "session already running", err)
		return nil, err
	}
	defer endSessionRun(SignProtocol, sessionID)
	cfg := newConfig(opts...)

	// Refuse incomplete signer material up front: alice indexes bks / partial public keys /
	// Pedersen parameters by peer id without checking, so a missing entry reached the alice
	// sign core as a nil dereference instead of an error. It also enforces the wrapper's signer
	// subset policy that the signer count equals the threshold; alice itself accepts
	// len(bks) >= threshold.
	if err := refreshRes.validateSignMaterial(dkgRes, peers, threshold); err != nil {
		log.Error("ProcessSign", "session", sessionID, "sign material check err", err)
		return nil, err
	}
	aRes, err := dkgRes.aliceResult(peers)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "aliceResult err", err)
		return nil, err
	}
	partialPubKeys, err := refreshRes.toAlicePartialPubKeys(peers)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "partial pubkeys err", err)
		return nil, err
	}
	ped := refreshRes.toAlicePedersen(peers)
	paillierKey, err := refreshRes.ownPaillier()
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "paillier err", err)
		return nil, err
	}

	pm, err := tss.NewReadyPeerManagerWithTimeout(peers, SignProtocol, sessionID, cfg.phaseTimeout(SignProtocol))
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "peers not ready", err)
		return nil, err
	}
	if _, err := validateSelf(pm, peers); err != nil {
		log.Error("ProcessSign", "session", sessionID, "invalid local peer", err)
		return nil, err
	}
	listener := tss.NewListener(SignProtocol, cfg.phaseTimeout(SignProtocol))
	ssid := computeSSID(sessionID, dkgRes.Rid)

	signCore, err := alicesign.NewSign(threshold, ssid, new(big.Int).SetBytes(refreshRes.Share),
		aRes.PublicKey, partialPubKeys, paillierKey, ped, aRes.Bks, msg, pm, listener)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "NewSign err", err)
		return nil, err
	}
	if err := registerSession(SignProtocol, sessionID, signCore, peers); err != nil {
		log.Error("ProcessSign", "session", sessionID, "register session failed, err", err)
		return nil, err
	}
	defer removeSession(SignProtocol, sessionID)
	signCore.Start()
	defer signCore.Stop()
	if err := listener.Wait(); err != nil {
		log.Error("ProcessSign", "session", sessionID, "listener done err", err)
		return nil, err
	}
	return signCore.GetResult()
}

// computeSSID derives the CGGMP ssid used by the refresh and sign phases. It MUST be identical
// across all participants of a session: the refresh/sign zero-knowledge challenges are derived
// with ComputeZKSsid(ssid, peerBk) on both the prover and the verifier side, which only match
// when the ssid is shared (see alice's own cggmp tests, which pass one shared nonce). The DKG
// rid is included to bind the ssid to a specific key.
func computeSSID(sessionID string, rid []byte) []byte {
	return alicggmp.ComputeSSID([]byte(sessionID), nil, rid)
}

// computeRefreshAgainSSID derives the ssid of a repeated refresh. alice's refresh.Result carries
// no rid of its own: the ssid is only a domain separator for the refresh zero-knowledge
// challenges, so the requirement is that all participants of a round share it and that it does
// not repeat across rounds. Chaining the previous round's ssid (and the caller's session id, which
// names this round) satisfies both without a new rid: every refresh in the chain gets a distinct
// value even if the caller reuses a session id.
func computeRefreshAgainSSID(sessionID string, prevSSID, rid []byte) []byte {
	return alicggmp.ComputeSSID([]byte(sessionID), prevSSID, rid)
}

// ToBtcecSignature converts a CGGMP signer result (R,S) to a btcec ecdsa signature, ready for
// verification with the DKG group public key. It is the CGGMP counterpart of
// gg18.AliceToBtcecSignature.
func ToBtcecSignature(result *alicesign.Result) (*ecdsa.Signature, error) {
	if result == nil {
		return nil, errNilResult
	}
	return tss.BuildBtcecSignature(result.R, result.S)
}
