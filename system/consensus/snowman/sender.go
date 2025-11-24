// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snowman

import (
	"encoding/hex"

	client "github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
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
	cli  client.Client
	vdrs *vdrSet
}

func newMsgSender(vdrs *vdrSet, cli client.Client, snowCtx *snow.ConsensusContext) *msgSender {

	s := &msgSender{vdrs: vdrs, cli: cli}
	sd, err := sender.New(snowCtx, nil, nil, nil, nil, p2p.EngineType_ENGINE_TYPE_SNOWMAN, nil)

	if err != nil {
		panic("newMsgSender err:" + err.Error())
	}
	s.Sender = sd

	return s
}

// SendChits send chits to the specified node
func (s *msgSender) SendChits(_ context.Context, nodeID ids.NodeID, requestID uint32, preferredID ids.ID, acceptedID ids.ID) {

	peerName := s.vdrs.toLibp2pID(nodeID)
	snowLog.Debug("msgSender SendChits", "reqID", requestID, "peer", peerName,
		"preferHash", hex.EncodeToString(preferredID[:]),
		"acceptHash", hex.EncodeToString(acceptedID[:]))

	req := &types.SnowChits{
		RequestID:        requestID,
		PreferredBlkHash: preferredID[:],
		AcceptedBlkHash:  acceptedID[:],
		PeerName:         peerName,
	}

	msg := s.cli.NewMessage("p2p", types.EventSnowmanChits, req)
	err := s.cli.Send(msg, false)
	if err != nil {
		snowLog.Error("msgSender SendChits", "peer", req.PeerName, "reqID", requestID,
			"perferHash", hex.EncodeToString(req.PreferredBlkHash),
			"accept", hex.EncodeToString(req.AcceptedBlkHash), "client.Send err:", err)
	}
}

// SendGet Request that the specified node send the specified container to this node.
func (s *msgSender) SendGet(_ context.Context, nodeID ids.NodeID, requestID uint32, blockID ids.ID) {

	peerName := s.vdrs.toLibp2pID(nodeID)
	snowLog.Debug("msgSender SendGet", "reqID", requestID, "peer", peerName, "blkHash", hex.EncodeToString(blockID[:]))
	req := &types.SnowGetBlock{
		RequestID: requestID,
		BlockHash: blockID[:],
		PeerName:  peerName,
	}

	msg := s.cli.NewMessage("p2p", types.EventSnowmanGetBlock, req)
	err := s.cli.Send(msg, false)
	if err != nil {
		snowLog.Error("msgSender SendGet", "peer", req.PeerName, "reqID", requestID,
			"blkHash", hex.EncodeToString(req.BlockHash), "client.Send err:", err)
	}
}

// SendPut Tell the specified node about [container].
func (s *msgSender) SendPut(_ context.Context, nodeID ids.NodeID, requestID uint32, blkData []byte) {

	peerName := s.vdrs.toLibp2pID(nodeID)
	snowLog.Debug("msgSender SendPut", "reqID", requestID, "peer", peerName)
	req := &types.SnowPutBlock{
		RequestID: requestID,
		PeerName:  peerName,
		BlockData: blkData,
	}

	msg := s.cli.NewMessage("p2p", types.EventSnowmanPutBlock, req)
	err := s.cli.Send(msg, false)
	if err != nil {
		snowLog.Error("msgSender SendPut", "peer", req.PeerName, "reqID", requestID, "client.Send err:", err)
	}
}

// SendPullQuery Request from the specified nodes their preferred frontier, given the existence of the specified container.
func (s *msgSender) SendPullQuery(_ context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, blockID ids.ID) {

	snowLog.Debug("msgSender SendPullQuery", "reqID", requestID, "peerLen", nodeIDs.Len(),
		"blkHash", hex.EncodeToString(blockID[:]))

	for nodeID := range nodeIDs {

		req := &types.SnowPullQuery{
			RequestID: requestID,
			BlockHash: blockID[:],
			PeerName:  s.vdrs.toLibp2pID(nodeID),
		}

		msg := s.cli.NewMessage("p2p", types.EventSnowmanPullQuery, req)
		err := s.cli.Send(msg, false)
		if err != nil {
			snowLog.Error("msgSender SendPullQuery", "peer", req.PeerName, "reqID", requestID,
				"blkHash", hex.EncodeToString(req.BlockHash), "client.Send err:", err)
		}
	}

}

// SendPushQuery Request from the specified nodes their preferred frontier, given the
// existence of the specified container.
// This is the same as PullQuery, except that this message includes the body
// of the container rather than its ID.
func (s *msgSender) SendPushQuery(_ context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, blockData []byte) {

	snowLog.Debug("msgSender SendPushQuery", "reqID", requestID, "peerLen", nodeIDs.Len())

	for nodeID := range nodeIDs {

		req := &types.SnowPushQuery{
			RequestID: requestID,
			BlockData: blockData,
			PeerName:  s.vdrs.toLibp2pID(nodeID),
		}

		msg := s.cli.NewMessage("p2p", types.EventSnowmanPushQuery, req)
		err := s.cli.Send(msg, false)
		if err != nil {
			snowLog.Error("msgSender SendPushQuery", "peer", req.PeerName, "reqID", requestID, "client.Send err:", err)
		}
	}
}

// SendGossip Gossip the provided container throughout the network
func (s *msgSender) SendGossip(ctx context.Context, container []byte) {

	recordUnimplementedError("msgSender SendGossip")
}

// SendGetAncestors requests that node [nodeID] send container [containerID]
// and its ancestors.
func (s *msgSender) SendGetAncestors(ctx context.Context, nodeID ids.NodeID, requestID uint32, containerID ids.ID) {
	recordUnimplementedError("msgSender SendGetAncestors")
}

// SendAncestors Give the specified node several containers at once. Should be in response
// to a GetAncestors message with request ID [requestID] from the node.
func (s *msgSender) SendAncestors(ctx context.Context, nodeID ids.NodeID, requestID uint32, containers [][]byte) {
	recordUnimplementedError("msgSender SendAncestors")
}
