// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package snowman
package snowman

import (
	"encoding/hex"
	"runtime"
	"sync/atomic"
	"time"

	sncom "github.com/ava-labs/avalanchego/snow/engine/common"

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

// Config snowman 参数配置
type Config struct {
	K                 int `json:"k" yaml:"k"`
	Alpha             int `json:"alpha" yaml:"alpha"`
	BetaVirtuous      int `json:"betaVirtuous" yaml:"betaVirtuous"`
	BetaRogue         int `json:"betaRogue" yaml:"betaRogue"`
	ConcurrentRepolls int `json:"concurrentRepolls" yaml:"concurrentRepolls"`
}

type snowman struct {
	engine        smeng.Engine
	vm            *chain33VM
	vs            *vdrSet
	ctx           *consensus.Context
	inMsg         chan *queue.Message
	engineNotify  chan struct{}
	params        snowball.Parameters
	engineStarted atomic.Bool
}

func (s *snowman) Initialize(ctx *consensus.Context) {

	s.ctx = ctx
	s.params = snowball.DefaultParameters
	s.applyConfig(ctx.Base.GetAPI().GetConfig().GetSubConfig())
	err := s.params.Verify()
	if err != nil {
		panic("Initialize invalid snowball parameters:" + err.Error())
	}

	s.vm = &chain33VM{}
	s.vm.Init(ctx)

	s.vs = &vdrSet{}
	s.vs.init(ctx)

	engineConfig := newSnowmanConfig(s, s.params, newSnowContext(ctx.Base.GetAPI().GetConfig()))
	engine, err := smeng.New(engineConfig)
	if err != nil {
		panic("Initialize snowman engine err:" + err.Error())
	}
	s.engine = engine

	s.inMsg = make(chan *queue.Message, 32)
	s.engineNotify = make(chan struct{}, 32)
	go s.startRoutine()
}

func (s *snowman) applyConfig(subCfg *types.ConfigSubModule) {

	cfg := &Config{}
	types.MustDecode(subCfg.Consensus["snowman"], cfg)

	if cfg.K > 0 {
		s.params.K = cfg.K
	}

	if cfg.Alpha > 0 {
		s.params.Alpha = cfg.Alpha
	}

	if cfg.BetaVirtuous > 0 {
		s.params.BetaVirtuous = cfg.BetaVirtuous
	}

	if cfg.BetaRogue > 0 {
		s.params.BetaRogue = cfg.BetaRogue
	}

	if cfg.ConcurrentRepolls > 0 {
		s.params.ConcurrentRepolls = cfg.ConcurrentRepolls
	}
}

func (s *snowman) getChainSyncStatus() bool {

	reply, err := s.ctx.Base.GetAPI().IsSync()

	if err != nil {
		snowLog.Error("getChainSyncStatus", "err", err)
		return false
	}

	return reply.GetIsOk()
}

func (s *snowman) startRoutine() {

	// check chain sync status
	for !s.getChainSyncStatus() {
		snowLog.Info("startRoutine wait chain state syncing...")
		time.Sleep(5 * time.Second)
	}

	// check connected peers
	for {

		peers, err := s.vs.getConnectedPeers()
		if err == nil && len(peers) >= s.params.K {
			break
		}
		snowLog.Info("startRoutine wait more snowman peer connected...",
			"currConnected", len(peers), "minRequiredNum", s.params.K, "err", err)
		time.Sleep(5 * time.Second)
	}

	err := s.engine.Start(s.ctx.Base.Context, 0)
	if err != nil {
		panic("start snowman engine err:" + err.Error())
	}

	go s.dispatchSyncMsg()
	go s.handleNotifyAddBlock()
	s.engineStarted.Store(true)
	snowLog.Debug("snowman startRoutine done")

}

func (s *snowman) AddBlock(blk *types.Block) {

	if !s.engineStarted.Load() {
		return
	}
	s.vm.addNewBlock(blk)
	s.engineNotify <- struct{}{}

}

func (s *snowman) SubMsg(msg *queue.Message) {

	if !s.engineStarted.Load() {
		snowLog.Debug("snowman SubMsg ignore", "id", msg.ID, "name", types.GetEventName(int(msg.ID)))
		return
	}
	s.inMsg <- msg
}

// 通知新区快事件
func (s *snowman) handleNotifyAddBlock() {

	for {

		select {

		case <-s.ctx.Base.Context.Done():
			return

		case <-s.engineNotify:
			err := s.engine.Notify(s.ctx.Base.Context, sncom.PendingTxs)
			if err != nil {
				snowLog.Error("snowman NotifyAddBlock", "err", err)
			}

		}
	}
}

func (s *snowman) dispatchSyncMsg() {

	for {

		select {

		case <-s.ctx.Base.Context.Done():
			return

		case msg := <-s.inMsg:
			s.handleSyncMsg(msg)
			snowLog.Debug("handleInMsg Done", "event", types.GetEventName(int(msg.ID)))
		}

	}
}

func (s *snowman) handleSyncMsg(msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
			var buf [4048]byte
			n := runtime.Stack(buf[:], false)
			snowLog.Error("handleInMsg", "err", r, "stack", buf[:n])
		}
	}()

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
			snowLog.Error("handleInMsg getBlock", "reqID", req.RequestID,
				"hash", hex.EncodeToString(req.BlockHash), "peerName", req.PeerName, "Get err", err)
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
			snowLog.Error("handleInMsg pullQuery", "reqID", req.RequestID,
				"hash", hex.EncodeToString(req.BlockHash), "peerName", req.PeerName, "pullQuery err", err)
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
