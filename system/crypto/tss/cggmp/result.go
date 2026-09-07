// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
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

// SelfBkString returns the string form of the local node's Birkhoff parameter used to derive
// the CGGMP ssid. It must be called with the same field order on every node.
func (d *DKGResult) SelfBkString(selfID string, fieldOrder *big.Int) (string, error) {
	bk, ok := d.Bks[selfID]
	if !ok || bk == nil {
		return "", errMissingSelfBk
	}
	b := birkhoffinterpolation.NewBkParameter(new(big.Int).SetBytes(bk.X), bk.Rank)
	return b.String(fieldOrder), nil
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
