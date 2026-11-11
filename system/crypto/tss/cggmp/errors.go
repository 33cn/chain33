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

	// Entry-point validation errors: a Process* call must fail instead of letting a bad peer list
	// reach alice, which indexes its parameter maps by the local peer id without checking and
	// panics when the local node is not among the participants.
	errNoPeers          = errors.New("cggmp: empty peer list")
	errInvalidPeerList  = errors.New("cggmp: invalid peer list")
	errInvalidThreshold = errors.New("cggmp: invalid threshold")
	errMissingSelfID    = errors.New("cggmp: cannot determine the local peer id")
	errSelfNotInPeers   = errors.New("cggmp: local peer id is not in the peer list")
	errNilDKGResult     = errors.New("cggmp: nil dkg result")
	errNilRefreshResult = errors.New("cggmp: nil refresh result")

	// Peer-material completeness errors, wrapped with the offending peer id.
	errMissingPeerBk  = errors.New("missing birkhoff parameter")
	errMissingPeerPPK = errors.New("missing partial public key")
	errMissingPeerPed = errors.New("missing pedersen parameter")
	errInvalidPeerPPK = errors.New("invalid partial public key")
	errThresholdPeers = errors.New("len(peers) must equal threshold")
	// errSessionRunning means another Process* call is already running for the same
	// protocol|sessionID in this process. Concurrent attempts on one session id would tear down
	// each other's partial-public-key collector (see beginPPKSession), so they are rejected.
	errSessionRunning = errors.New("cggmp: session already running")
	// errPeerNotParticipant means a message arrived from a peer id outside the round's
	// participant set, so it can never be a legitimate contribution to that round.
	errPeerNotParticipant = errors.New("cggmp: message from a non-participant peer")
	// errPeersNotReady means not every listed peer became connected within the phase timeout.
	errPeersNotReady = errors.New("cggmp: peers not connected before the phase timeout")
	// errInvalidRank means the local Birkhoff rank cannot take part in a threshold-t group.
	errInvalidRank = errors.New("cggmp: invalid birkhoff rank")
	// errPPKInconsistent means the exchanged partial public keys do not reconstruct the DKG
	// group public key: at least one participant broadcast bad material at the ppk stage.
	errPPKInconsistent = errors.New("partial public keys do not reconstruct the dkg group public key")
)
