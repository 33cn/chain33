// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"fmt"
	"math/big"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/getamis/alice/crypto/birkhoffinterpolation"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	"github.com/getamis/alice/crypto/homo/paillier"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicerefresh "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/refresh"
	zkpaillier "github.com/getamis/alice/crypto/zkproof/paillier"
)

// ECPoint is a serialisable elliptic-curve point (big-endian X/Y, curve is fixed to secp256k1).
type ECPoint struct {
	X []byte `json:"x"`
	Y []byte `json:"y"`
}

// toECPoint converts to an alice ECPoint on secp256k1.
func toECPoint(p *ECPoint) (*ecpointgrouplaw.ECPoint, error) {
	if p == nil {
		return nil, errNilPoint
	}
	return ecpointgrouplaw.NewECPoint(elliptic.Secp256k1(),
		new(big.Int).SetBytes(p.X), new(big.Int).SetBytes(p.Y))
}

// fromECPoint serialises an alice ECPoint.
func fromECPoint(p *ecpointgrouplaw.ECPoint) *ECPoint {
	if p == nil {
		return nil
	}
	return &ECPoint{X: p.GetX().Bytes(), Y: p.GetY().Bytes()}
}

// PedersenParam carries one peer's Pedersen (N,S,T) opening parameters used by CGGMP MtA proofs.
type PedersenParam struct {
	N []byte `json:"n"`
	S []byte `json:"s"`
	T []byte `json:"t"`
}

// toPedersen converts back to an alice PederssenOpenParameter.
func toPedersen(p *PedersenParam) *zkpaillier.PederssenOpenParameter {
	if p == nil {
		return nil
	}
	return zkpaillier.NewPedersenOpenParameter(new(big.Int).SetBytes(p.N),
		new(big.Int).SetBytes(p.S), new(big.Int).SetBytes(p.T))
}

// fromPedersen serialises an alice PederssenOpenParameter.
func fromPedersen(p *zkpaillier.PederssenOpenParameter) *PedersenParam {
	if p == nil {
		return nil
	}
	return &PedersenParam{N: p.GetN().Bytes(), S: p.GetS().Bytes(), T: p.GetT().Bytes()}
}

// DKGResult is the per-node output of the CGGMP DKG phase, persisted for later use by
// ProcessRefresh / ProcessSign. It carries the group public key, this node's share, the
// Birkhoff parameters of all participants, the DKG rid, and every participant's partial
// public key (g^{share_i}) exchanged right after DKG. It is independent from the GG18
// tss.DKGResult used by the gg18 wrapper.
type DKGResult struct {
	PubX           []byte              `json:"pubX"`
	PubY           []byte              `json:"pubY"`
	Share          []byte              `json:"share"`
	Bks            map[string]*tss.BK  `json:"bks"`
	Rid            []byte              `json:"rid"`
	PartialPubKeys map[string]*ECPoint `json:"partialPubKeys"`
}

// GroupPublicKey returns the DKG group public key as an alice ECPoint.
func (d *DKGResult) GroupPublicKey() (*ecpointgrouplaw.ECPoint, error) {
	return ecpointgrouplaw.NewECPoint(elliptic.Secp256k1(),
		new(big.Int).SetBytes(d.PubX), new(big.Int).SetBytes(d.PubY))
}

// toAliceBks rebuilds the alice Birkhoff parameter set for the given participant list,
// keeping only entries that are present in the DKG result. The peer order must match the
// calling convention (every participating node uses the same list).
func (d *DKGResult) toAliceBks(peers []string) map[string]*birkhoffinterpolation.BkParameter {
	bks := make(map[string]*birkhoffinterpolation.BkParameter, len(peers))
	for _, id := range peers {
		bk, ok := d.Bks[id]
		if !ok || bk == nil {
			continue
		}
		bks[id] = birkhoffinterpolation.NewBkParameter(new(big.Int).SetBytes(bk.X), bk.Rank)
	}
	return bks
}

// aliceResult rebuilds the alice cggmp dkg.Result restricted to the given participant list.
// peers must include the local node id and match across all participants of a phase.
func (d *DKGResult) aliceResult(peers []string) (*alicedkg.Result, error) {
	pub, err := d.GroupPublicKey()
	if err != nil {
		return nil, err
	}
	return &alicedkg.Result{
		PublicKey: pub,
		Share:     new(big.Int).SetBytes(d.Share),
		Bks:       d.toAliceBks(peers),
		Rid:       d.Rid,
	}, nil
}

// toAlicePartialPubKeys rebuilds partial public keys restricted to the given participant list.
func (d *DKGResult) toAlicePartialPubKeys(peers []string) (map[string]*ecpointgrouplaw.ECPoint, error) {
	m := make(map[string]*ecpointgrouplaw.ECPoint, len(peers))
	for _, id := range peers {
		pp, ok := d.PartialPubKeys[id]
		if !ok || pp == nil {
			continue
		}
		p, err := toECPoint(pp)
		if err != nil {
			return nil, err
		}
		m[id] = p
	}
	return m, nil
}

// SelfBkString returns the string form of the local node's Birkhoff parameter, using the same
// encoding alice uses when it binds a peer's bk into a zero-knowledge challenge
// (cggmp.ComputeZKSsid). It is a diagnostics/debugging helper only — the CGGMP ssid this
// wrapper uses is NOT derived from the local bk (that would differ per node and break the
// shared challenges); see computeSSID for how the ssid is built.
func (d *DKGResult) SelfBkString(selfID string, fieldOrder *big.Int) (string, error) {
	bk, ok := d.Bks[selfID]
	if !ok || bk == nil {
		return "", errMissingSelfBk
	}
	b := birkhoffinterpolation.NewBkParameter(new(big.Int).SetBytes(bk.X), bk.Rank)
	return b.String(fieldOrder), nil
}

// validatePartialPubKeys checks the material gathered by the partial-public-key (ppk) exchange:
// every participant in peers must have a bk and a g^{share}, and those points must reconstruct
// the DKG group public key through the Birkhoff parameters.
//
// The DKG discards its Feldman commitments (alice never exposes them), so the Birkhoff
// reconstruction is the only local consistency check available at this point — the same one
// alice itself runs at the end of DKG and of refresh
// (birkhoffinterpolation.BkParameters.ValidatePublicKey). Running it as soon as the ppks are
// collected rejects a participant that broadcast a bad g^{share} at the ppk stage, instead of
// carrying the corruption into the refresh phase (where it would surface as an opaque
// zero-knowledge failure, or produce a share set that never signs).
//
// The reconstruction is a single group equation, so on a mismatch the error reports the phase
// and the whole participant set that took part in it; a *missing* entry, by contrast, is
// reported with the exact peer id.
func (d *DKGResult) validatePartialPubKeys(peers []string, threshold uint32) error {
	pub, err := d.GroupPublicKey()
	if err != nil {
		return fmt.Errorf("cggmp: ppk check: %w", err)
	}
	bks := make(birkhoffinterpolation.BkParameters, 0, len(peers))
	sgs := make([]*ecpointgrouplaw.ECPoint, 0, len(peers))
	for _, id := range peers {
		bk, ok := d.Bks[id]
		if !ok || bk == nil {
			return fmt.Errorf("cggmp: ppk check: peer %s: %w", id, errMissingPeerBk)
		}
		pp, ok := d.PartialPubKeys[id]
		if !ok || pp == nil {
			return fmt.Errorf("cggmp: ppk check: peer %s: %w", id, errMissingPeerPPK)
		}
		p, err := toECPoint(pp)
		if err != nil {
			return fmt.Errorf("cggmp: ppk check: peer %s: %w: %v", id, errInvalidPeerPPK, err)
		}
		bks = append(bks, birkhoffinterpolation.NewBkParameter(new(big.Int).SetBytes(bk.X), bk.Rank))
		sgs = append(sgs, p)
	}
	// #nosec G115 -- len(peers) is bounded by the participant count of a session
	if uint32(len(peers)) < threshold {
		return fmt.Errorf("cggmp: ppk check: %d participants < threshold %d", len(peers), threshold)
	}
	if err := bks.ValidatePublicKey(sgs, threshold, pub); err != nil {
		return fmt.Errorf("cggmp: ppk check: peers %v: %w (%v)", peers, errPPKInconsistent, err)
	}
	return nil
}

// RefreshResult is the per-node output of the CGGMP refresh (key refresh + Paillier /
// Pedersen provisioning) phase. Together with the DKGResult it carries everything ProcessSign
// needs: this node's refreshed share, its own Paillier primes (secret), every participant's
// partial public key and Pedersen parameters, and the y/ySecret values required by the
// six-round signer.
type RefreshResult struct {
	Share          []byte                    `json:"share"`
	PaillierP      []byte                    `json:"paillierP"`
	PaillierQ      []byte                    `json:"paillierQ"`
	PartialPubKeys map[string]*ECPoint       `json:"partialPubKeys"`
	Y              map[string]*ECPoint       `json:"y"`
	PedParams      map[string]*PedersenParam `json:"pedParams"`
	YSecret        []byte                    `json:"ySecret"`
}

// ownPaillier rebuilds this node's secret Paillier key from the persisted primes.
func (r *RefreshResult) ownPaillier() (*paillier.Paillier, error) {
	return paillier.NewPaillierWithGivenPrimes(new(big.Int).SetBytes(r.PaillierP),
		new(big.Int).SetBytes(r.PaillierQ))
}

// toAlicePartialPubKeys rebuilds partial public keys restricted to the given participant list.
func (r *RefreshResult) toAlicePartialPubKeys(peers []string) (map[string]*ecpointgrouplaw.ECPoint, error) {
	m := make(map[string]*ecpointgrouplaw.ECPoint, len(peers))
	for _, id := range peers {
		pp, ok := r.PartialPubKeys[id]
		if !ok || pp == nil {
			continue
		}
		p, err := toECPoint(pp)
		if err != nil {
			return nil, err
		}
		m[id] = p
	}
	return m, nil
}

// toAlicePedersen rebuilds the Pedersen parameter map restricted to the given participant list.
func (r *RefreshResult) toAlicePedersen(peers []string) map[string]*zkpaillier.PederssenOpenParameter {
	m := make(map[string]*zkpaillier.PederssenOpenParameter, len(peers))
	for _, id := range peers {
		ped, ok := r.PedParams[id]
		if !ok || ped == nil {
			continue
		}
		m[id] = toPedersen(ped)
	}
	return m
}

// validateSignMaterial checks that every signer in peers has all the key material ProcessSign
// hands to alice: a bk (from the DKG), a partial public key and Pedersen parameters (from the
// refresh). alice indexes those maps by peer id without checking, so a missing entry becomes a
// nil dereference inside alice (e.g. cggmp/sign/peer.go reading partialPubKey[self]) rather
// than an error. threshold must equal the signer count: alice derives its Birkhoff coefficients
// for exactly threshold participants.
func (r *RefreshResult) validateSignMaterial(d *DKGResult, peers []string, threshold uint32) error {
	if uint32(len(peers)) != threshold {
		return fmt.Errorf("cggmp: sign: %w: len(peers)=%d, threshold=%d", errThresholdPeers, len(peers), threshold)
	}
	for _, id := range peers {
		if bk, ok := d.Bks[id]; !ok || bk == nil {
			return fmt.Errorf("cggmp: sign: peer %s: %w", id, errMissingPeerBk)
		}
		if pp, ok := r.PartialPubKeys[id]; !ok || pp == nil {
			return fmt.Errorf("cggmp: sign: peer %s: %w", id, errMissingPeerPPK)
		}
		if ped, ok := r.PedParams[id]; !ok || ped == nil {
			return fmt.Errorf("cggmp: sign: peer %s: %w", id, errMissingPeerPed)
		}
	}
	return nil
}

// newDKGResult converts an alice cggmp dkg result plus the exchanged partial public keys
// into the persisted chain33 representation.
func newDKGResult(res *alicedkg.Result, partialPubKeys map[string]*ecpointgrouplaw.ECPoint) *DKGResult {
	d := &DKGResult{
		PubX:           res.PublicKey.GetX().Bytes(),
		PubY:           res.PublicKey.GetY().Bytes(),
		Share:          res.Share.Bytes(),
		Bks:            make(map[string]*tss.BK, len(res.Bks)),
		Rid:            res.Rid,
		PartialPubKeys: make(map[string]*ECPoint, len(partialPubKeys)),
	}
	for id, bk := range res.Bks {
		d.Bks[id] = &tss.BK{Rank: bk.GetRank(), X: bk.GetX().Bytes()}
	}
	for id, p := range partialPubKeys {
		d.PartialPubKeys[id] = fromECPoint(p)
	}
	return d
}

// newRefreshResult converts an alice cggmp refresh result into the persisted representation.
func newRefreshResult(res *alicerefresh.Result) (*RefreshResult, error) {
	p, q := res.PaillierKey.GetPQ()
	r := &RefreshResult{
		Share:          res.Share.Bytes(),
		PaillierP:      p.Bytes(),
		PaillierQ:      q.Bytes(),
		PartialPubKeys: make(map[string]*ECPoint, len(res.PartialPubKey)),
		Y:              make(map[string]*ECPoint, len(res.Y)),
		PedParams:      make(map[string]*PedersenParam, len(res.PedParameter)),
		YSecret:        res.YSecret.Bytes(),
	}
	for id, pp := range res.PartialPubKey {
		r.PartialPubKeys[id] = fromECPoint(pp)
	}
	for id, y := range res.Y {
		r.Y[id] = fromECPoint(y)
	}
	for id, ped := range res.PedParameter {
		r.PedParams[id] = fromPedersen(ped)
	}
	return r, nil
}
