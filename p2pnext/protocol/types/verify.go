// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package types

import (
	core "github.com/libp2p/go-libp2p-core"

	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

// AuthenticateMessage auth msg
func AuthenticateMessage(message types.Message, data *types.MessageComm) bool {
	// store a temp ref to signature and remove it from message data
	// sign is a string to allow easy reset to zero-value (empty string)
	sign := data.Sign
	data.Sign = nil

	// marshall data without the signature to protobufs3 binary format
	bin := types.Encode(message)

	// restore sig in message data (for possible future use)
	data.Sign = sign

	// restore peer id binary format from base58 encoded node id data
	id, err := peer.IDB58Decode(data.NodeId)
	if err != nil {
		log.Error("Failed to decode node id from base58", "err", err)
		return false
	}

	// verify the data was authored by the signing peer identified by the public key
	// and signature included in the message
	return verifyData(bin, sign, id, data.NodePubKey)
}

// SignProtoMessage sign an outgoing p2p message payload
func SignProtoMessage(message types.Message, host core.Host) ([]byte, error) {
	data := types.Encode(message)
	return signData(data, host)
}

// sign binary data using the local node's private key
func signData(data []byte, host core.Host) ([]byte, error) {
	key := host.Peerstore().PrivKey(host.ID())
	res, err := key.Sign(data)
	return res, err
}

// Verify incoming p2p message data integrity
// data: data to verify
// signature: author signature provided in the message payload
// peerId: author peer id from the message payload
// pubKeyData: author public key from the message payload
func verifyData(data []byte, signature []byte, id peer.ID, pubKeyData []byte) bool {
	key, err := crypto.UnmarshalPublicKey(pubKeyData)
	if err != nil {
		log.Error("Failed to extract key from message key data", "err", err)
		return false
	}

	// extract node id from the provided public key
	idFromKey, err := peer.IDFromPublicKey(key)

	if err != nil {
		log.Error("Failed to extract peer id from public key", "err", err)
		return false
	}

	// verify that message author node id matches the provided node public key
	if idFromKey != id {
		log.Error("Node id and provided public key mismatch", "err", err)
		return false
	}

	res, err := key.Verify(data, signature)
	if err != nil {
		log.Error("Error authenticating data", "err", err)
		return false
	}

	return res
}
