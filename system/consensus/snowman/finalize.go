// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package snowman
package snowman

import (
	"encoding/hex"
	"runtime"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/snow/consensus/snowball"

	smeng "github.com/ava-labs/avalanchego/snow/engine/snowman"
)

var (
	snowLog = log15.New("module", "snowman")

	_ consensus.Finalizer = (*snowman)(nil)
)

func init() {

	consensus.RegFinalizer("snowman", &snowman{})

}

type snowman struct {
	engine smeng.Engine
	vm     *chain33VM
	vs     *vdrSet
	ctx    *consensus.Context
	inMsg  chan *queue.Message
	params snowball.Parameters
}

func (s *snowman) Initialize(ctx *consensus.Context) {

	s.inMsg = make(chan *queue.Message, 32)

	params := snowball.DefaultParameters
	err := params.Verify()
	if err != nil {
		panic("Initialize snowman engine invalid snowball parameters:" + err.Error())
	}
	s.params = params
	vm := &chain33VM{}
	vm.Init(ctx)

	vs := &vdrSet{}
	vs.init(ctx)

	s.vm = vm
	s.vs = vs

	engineConfig := newSnowmanConfig(s, params, newSnowContext(ctx.Base.GetAPI().GetConfig()))

	engine, err := smeng.New(engineConfig)

	if err != nil {
		panic("Initialize snowman engine err:" + err.Error())
	}
	s.engine = engine

	go s.startRoutine()

}

func (s *snowman) startRoutine() {

	// TODO check sync status

	// check connected peers
	for {

		peers, err := s.vs.getConnectedPeers()
		if err == nil && len(peers) >= s.params.K {
			break
		}
		snowLog.Debug("startRoutine wait Peers", "getConnectedPeers", len(peers), "err", err)
		time.Sleep(2 * time.Second)
	}

	err := s.engine.Start(s.ctx.Base.Context, 0)
	if err != nil {
		panic("start snowman engine err:" + err.Error())
	}

	//启用handler协程
	concurrency := runtime.NumCPU() * 2
	for i := 0; i < concurrency; i++ {
		go s.handleMsgRountine()
	}

}

func (s *snowman) AddBlock(blk *types.Block) {

	s.vm.addNewBlock(blk)
}

func (s *snowman) SubMsg(msg *queue.Message) {

	s.inMsg <- msg
}

func (s *snowman) handleMsgRountine() {

	for {

		select {

		case <-s.ctx.Base.Context.Done():
			return

		case msg := <-s.inMsg:
			s.handleInMsg(msg)
			snowLog.Debug("handleInMsg Done", "event", types.GetEventName(int(msg.ID)))
		}

	}
}

func (s *snowman) handleInMsg(msg *queue.Message) {

	switch msg.ID {

	case types.EventSnowmanChits:

		req := msg.Data.(*types.SnowChits)
		snowLog.Debug("handleInMsg chits", "reqID", req.GetRequestID(), "peerName", req.GetPeerName(),
			"prefer", hex.EncodeToString(req.GetPreferredBlkHash()), "acept", hex.EncodeToString(req.GetAcceptedBlkHash()))
		nodeID, err := s.vs.toNodeID(req.PeerName)
		if err != nil {
			snowLog.Error("handleInMsg chits", "reqID", req.RequestID, "peerName", req.PeerName, "toNodeID err", err)
			return
		}

		err = s.engine.Chits(s.ctx.Base.Context, nodeID, req.RequestID,
			toSnowID(req.PreferredBlkHash), toSnowID(req.AcceptedBlkHash))
		if err != nil {

			snowLog.Error("handleInMsg chits", "reqID", req.RequestID, "peerName", req.PeerName,
				"prefer", hex.EncodeToString(req.PreferredBlkHash), "acept", hex.EncodeToString(req.AcceptedBlkHash),
				"chits err", err)
		}

	case types.EventSnowmanGetBlock:

		req := msg.Data.(*types.SnowGetBlock)
		snowLog.Debug("handleInMsg getBlock", "reqID", req.GetRequestID(), "peerName", req.GetPeerName(),
			"hash", hex.EncodeToString(req.BlockHash))
		nodeID, err := s.vs.toNodeID(req.PeerName)
		if err != nil {
			snowLog.Error("handleInMsg getBlock", "reqID", req.RequestID, "peerName", req.PeerName, "toNodeID err", err)
			return
		}

		err = s.engine.Get(s.ctx.Base.Context, nodeID, req.RequestID, toSnowID(req.BlockHash))
		if err != nil {
			snowLog.Error("handleInMsg getBlock", "reqID", req.RequestID, "peerName", req.PeerName, "Get err", err)
		}
	case types.EventSnowmanPutBlock:

		req := msg.Data.(*types.SnowPutBlock)
		snowLog.Debug("handleInMsg putBlock", "reqID", req.GetRequestID(), "peerName", req.GetPeerName())
		nodeID, err := s.vs.toNodeID(req.PeerName)
		if err != nil {
			snowLog.Error("handleInMsg putBlock", "reqID", req.RequestID, "peerName", req.PeerName, "toNodeID err", err)
			return
		}

		err = s.engine.Put(s.ctx.Base.Context, nodeID, req.RequestID, req.BlockData)
		if err != nil {
			snowLog.Error("handleInMsg putBlock", "reqID", req.RequestID, "peerName", req.PeerName, "Put err", err)
		}

	case types.EventSnowmanPullQuery:

		req := msg.Data.(*types.SnowPullQuery)
		snowLog.Debug("handleInMsg pullQuery", "reqID", req.GetRequestID(), "peerName", req.GetPeerName(),
			"hash", hex.EncodeToString(req.BlockHash))
		nodeID, err := s.vs.toNodeID(req.PeerName)
		if err != nil {
			snowLog.Error("handleInMsg pullQuery", "reqID", req.RequestID, "peerName", req.PeerName, "toNodeID err", err)
			return
		}

		err = s.engine.PullQuery(s.ctx.Base.Context, nodeID, req.RequestID, toSnowID(req.BlockHash))
		if err != nil {
			snowLog.Error("handleInMsg pullQuery", "reqID", req.RequestID, "peerName", req.PeerName, "pullQuery err", err)
		}

	case types.EventSnowmanPushQuery:

		req := msg.Data.(*types.SnowPushQuery)
		snowLog.Debug("handleInMsg pushQuery", "reqID", req.GetRequestID(), "peerName", req.GetPeerName())
		nodeID, err := s.vs.toNodeID(req.PeerName)
		if err != nil {
			snowLog.Error("handleInMsg pushQuery", "reqID", req.RequestID, "peerName", req.PeerName, "toNodeID err", err)
			return
		}

		err = s.engine.PushQuery(s.ctx.Base.Context, nodeID, req.RequestID, req.BlockData)
		if err != nil {
			snowLog.Error("handleInMsg pushQuery", "reqID", req.RequestID, "peerName", req.PeerName, "pushQuery err", err)
		}

	case types.EventSnowmanQueryFailed:

		req := msg.Data.(*types.SnowFailedQuery)
		snowLog.Debug("handleInMsg failQuery", "reqID", req.RequestID, "peerName", req.GetPeerName())
		nodeID, err := s.vs.toNodeID(req.PeerName)
		if err != nil {
			snowLog.Error("handleInMsg failQuery", "reqID", req.RequestID, "peerName", req.PeerName, "toNodeID err", err)
			return
		}

		err = s.engine.QueryFailed(s.ctx.Base.Context, nodeID, req.RequestID)
		if err != nil {
			snowLog.Error("handleInMsg failQuery", "reqID", req.RequestID, "peerName", req.PeerName, "failQuery err", err)
		}

	case types.EventSnowmanGetFailed:

		req := msg.Data.(*types.SnowFailedQuery)
		snowLog.Debug("handleInMsg failGet", "reqID", req.RequestID, "peerName", req.GetPeerName())
		nodeID, err := s.vs.toNodeID(req.PeerName)
		if err != nil {
			snowLog.Error("handleInMsg failGet", "reqID", req.RequestID, "peerName", req.PeerName, "toNodeID err", err)
			return
		}

		err = s.engine.GetFailed(s.ctx.Base.Context, nodeID, req.RequestID)
		if err != nil {
			snowLog.Error("handleInMsg failGet", "reqID", req.RequestID, "peerName", req.PeerName, "failGet err", err)
		}

	default:
		snowLog.Error("snowman handleInMsg, recv unknow msg", "id", msg.ID)

	}
}
