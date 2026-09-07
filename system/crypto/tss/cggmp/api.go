// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"math/big"
	"time"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	alicggmp "github.com/getamis/alice/crypto/tss/ecdsa/cggmp"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicerefresh "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/refresh"
	alicesign "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/sign"
)

const (
	defaultTimeout = 30 * time.Second
	// paillierKeySize is the RSA modulus bit length used for the CGGMP Paillier keys.
	paillierKeySize = 2048
)

type config struct {
	timeout time.Duration
}

// Option sets configuration options for CGGMP operations.
type Option func(*config)

// WithTimeout overrides the default per-phase timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *config) {
		o.timeout = timeout
	}
}

func newConfig(opts ...Option) *config {
	cfg := &config{timeout: defaultTimeout}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// ProcessDKG runs the CGGMP DKG protocol and then the CGGMP partial-public-key exchange,
// returning a persisted DKGResult (group public key, this node's share, Birkhoff parameters,
// rid and every participant's partial public key). sessionID must be identical across nodes;
// peers must include the local node id. CGGMP sessions are independent from the GG18 wrapper.
func ProcessDKG(peers []string, threshold, rank uint32, sessionID string, opts ...Option) (*DKGResult, error) {
	cfg := newConfig(opts...)
	curve := elliptic.Secp256k1()

	// Phase 1: alice CGGMP DKG.
	pm := tss.NewReadyPeerManager(peers, DkgProtocol, sessionID)
	listener := tss.NewListener(DkgProtocol, cfg.timeout)
	dkgCore, err := alicedkg.NewDKG(curve, pm, []byte(sessionID), threshold, rank, listener)
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "NewDKG err", err)
		return nil, err
	}
	if err := registerSession(DkgProtocol, sessionID, dkgCore); err != nil {
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
	selfID := pm.SelfID()
	partialPubKeys, err := exchangePartialPubKeys(peers, selfID, aliceRes.Share, curve, sessionID, cfg.timeout)
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "partial pubkey exchange err", err)
		return nil, err
	}
	return newDKGResult(aliceRes, partialPubKeys), nil
}

// exchangePartialPubKeys broadcasts the local g^{share} to all peers and collects theirs.
func exchangePartialPubKeys(peers []string, selfID string, share *big.Int, curve elliptic.Curve, sessionID string, timeout time.Duration) (map[string]*ecpointgrouplaw.ECPoint, error) {
	myPPK := ecpointgrouplaw.ScalarBaseMult(curve, share)
	pm := tss.NewReadyPeerManager(peers, PpkProtocol, sessionID)
	collector := newPPKCollector(len(peers) - 1)
	registerPPKCollector(sessionID, collector)
	defer removePPKCollector(sessionID)

	myMsg := partialPublicKeyMsg(selfID, myPPK)
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
func ProcessRefresh(peers []string, threshold uint32, dkgRes *DKGResult, sessionID string, opts ...Option) (*RefreshResult, error) {
	cfg := newConfig(opts...)

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

	pm := tss.NewReadyPeerManager(peers, RefreshProtocol, sessionID)
	listener := tss.NewListener(RefreshProtocol, cfg.timeout)
	ssid := computeSSID(sessionID, dkgRes.Rid)

	refreshCore, err := alicerefresh.NewRefresh(aRes.Share, aRes.PublicKey, pm, threshold,
		partialPubKeys, aRes.Bks, paillierKeySize, ssid, listener)
	if err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "NewRefresh err", err)
		return nil, err
	}
	if err := registerSession(RefreshProtocol, sessionID, refreshCore); err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "register session failed, err", err)
		return nil, err
	}
	defer removeSession(RefreshProtocol, sessionID)
	refreshCore.Start()
	defer refreshCore.Stop()
	if err := listener.Wait(); err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "listener done err", err)
		return nil, err
	}
	aliceRefresh, err := refreshCore.GetResult()
	if err != nil {
		log.Error("ProcessRefresh", "session", sessionID, "GetResult err", err)
		return nil, err
	}
	return newRefreshResult(aliceRefresh)
}

// ProcessSign runs the CGGMP (4-round) threshold sign for msg. dkgRes is the DKG output of the
// key and refreshRes the corresponding refresh output. peers is the signer subset (includes the
// local node id, size == threshold); each node must call with the same peers list. sessionID
// must be identical across the signing nodes.
func ProcessSign(peers []string, threshold uint32, msg []byte, dkgRes *DKGResult, refreshRes *RefreshResult, sessionID string, opts ...Option) (*alicesign.Result, error) {
	cfg := newConfig(opts...)

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

	pm := tss.NewReadyPeerManager(peers, SignProtocol, sessionID)
	listener := tss.NewListener(SignProtocol, cfg.timeout)
	ssid := computeSSID(sessionID, dkgRes.Rid)

	signCore, err := alicesign.NewSign(threshold, ssid, new(big.Int).SetBytes(refreshRes.Share),
		aRes.PublicKey, partialPubKeys, paillierKey, ped, aRes.Bks, msg, pm, listener)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "NewSign err", err)
		return nil, err
	}
	if err := registerSession(SignProtocol, sessionID, signCore); err != nil {
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

// ToBtcecSignature converts a CGGMP signer result (R,S) to a btcec ecdsa signature, ready for
// verification with the DKG group public key. It is the CGGMP counterpart of
// gg18.AliceToBtcecSignature.
func ToBtcecSignature(result *alicesign.Result) (*ecdsa.Signature, error) {
	if result == nil {
		return nil, errNilResult
	}
	return tss.BuildBtcecSignature(result.R, result.S)
}
