// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snowman

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/proto/pb/p2p"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/networking/sender"
	"github.com/ava-labs/avalanchego/utils/set"

	"context"
)

// 向外部模块发送请求, blockchain/p2p等
type msgSender struct {
	common.Sender
}

func newMsgSender(ctx *snow.ConsensusContext) *msgSender {

	s := &msgSender{}
	sd, err := sender.New(ctx, nil, nil, nil, nil, p2p.EngineType_ENGINE_TYPE_SNOWMAN, nil)

	if err != nil {
		panic("newMsgSender err:" + err.Error())
	}
	s.Sender = sd

	return s
}

// SendChits send chits to the specified node
func (s *msgSender) SendChits(ctx context.Context, nodeID ids.NodeID, requestID uint32, preferredID ids.ID, acceptedID ids.ID) {

}

// SendGet Request that the specified node send the specified container to this node.
func (s *msgSender) SendGet(ctx context.Context, nodeID ids.NodeID, requestID uint32, blockID ids.ID) {

}

// SendPullQuery Request from the specified nodes their preferred frontier, given the existence of the specified container.
func (s *msgSender) SendPullQuery(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, containerID ids.ID) {

}

// SendPushQuery Request from the specified nodes their preferred frontier, given the
// existence of the specified container.
// This is the same as PullQuery, except that this message includes the body
// of the container rather than its ID.
func (s *msgSender) SendPushQuery(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, container []byte) {

}

// SendGossip Gossip the provided container throughout the network
func (s *msgSender) SendGossip(ctx context.Context, container []byte) {

}
