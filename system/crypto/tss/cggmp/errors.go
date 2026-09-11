// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import "errors"

var (
	errNilPoint      = errors.New("cggmp: nil ec point")
	errPPKTimeout    = errors.New("cggmp: partial public key exchange timed out")
	errMissingSelfBk = errors.New("cggmp: missing self bk in dkg result")
	errNilResult     = errors.New("cggmp: nil result")
	// errMissingPeerID means an incoming tss message carried no transport-authenticated peer id,
	// so it cannot be attributed to a participant and must be rejected rather than trusted.
	errMissingPeerID = errors.New("cggmp: missing authenticated peer id")

	// Peer-material completeness errors, wrapped with the offending peer id.
	errMissingPeerBk  = errors.New("missing birkhoff parameter")
	errMissingPeerPPK = errors.New("missing partial public key")
	errMissingPeerPed = errors.New("missing pedersen parameter")
	errInvalidPeerPPK = errors.New("invalid partial public key")
	errThresholdPeers = errors.New("len(peers) must equal threshold")
	// errPPKInconsistent means the exchanged partial public keys do not reconstruct the DKG
	// group public key: at least one participant broadcast bad material at the ppk stage.
	errPPKInconsistent = errors.New("partial public keys do not reconstruct the dkg group public key")
)
