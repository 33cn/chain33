// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snowman

import (
	"os"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/consensus/snowball"
	smcon "github.com/ava-labs/avalanchego/snow/consensus/snowman"
	snowcom "github.com/ava-labs/avalanchego/snow/engine/common"
	smeng "github.com/ava-labs/avalanchego/snow/engine/snowman"
	snowgetter "github.com/ava-labs/avalanchego/snow/engine/snowman/getter"
	"github.com/ava-labs/avalanchego/utils/logging"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newSnowmanConfig(sm *snowman, params snowball.Parameters, snowCtx *snow.ConsensusContext) smeng.Config {

	sender := newMsgSender(sm.vs, sm.ctx.Base.GetQueueClient(), snowCtx)
	engineConfig := smeng.Config{
		Ctx:         snowCtx,
		VM:          sm.vm,
		Validators:  sm.vs,
		Params:      params,
		Consensus:   &smcon.Topological{},
		PartialSync: false,
		Sender:      sender,
	}

	ccfg := snowcom.Config{

		Ctx:                            snowCtx,
		SampleK:                        params.K,
		Sender:                         sender,
		MaxTimeGetAncestors:            time.Second * 5,
		AncestorsMaxContainersSent:     2000,
		AncestorsMaxContainersReceived: 2000,
	}

	getter, err := snowgetter.New(sm.vm, ccfg)
	if err != nil {
		panic("newSnowmanConfig newSnowGetter err" + err.Error())
	}
	engineConfig.AllGetsServer = getter
	return engineConfig
}

func newSnowContext(cfg *types.Chain33Config) *snow.ConsensusContext {

	ctx := snow.DefaultConsensusContextTest()
	// TODO use chain33 log writer
	logCfg := cfg.GetModuleConfig().Log

	logger := &lumberjack.Logger{
		Filename:   "logs/snowman.log",
		MaxSize:    int(logCfg.MaxFileSize),
		MaxBackups: int(logCfg.MaxBackups),
		MaxAge:     int(logCfg.MaxAge),
		LocalTime:  logCfg.LocalTime,
		Compress:   logCfg.Compress,
	}

	fileCore := logging.NewWrappedCore(logging.Verbo, logger, logging.Colors.FileEncoder())
	consoleCore := logging.NewWrappedCore(logging.Info, os.Stderr, logging.Colors.ConsoleEncoder())
	ctx.Context.Log = logging.NewLogger("", fileCore, consoleCore)

	return ctx
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
		s.params.BetaRogue = cfg.BetaVirtuous + 1
	}

	if cfg.BetaRogue > 0 {
		s.params.BetaRogue = cfg.BetaRogue
	}

	if cfg.ConcurrentRepolls > 0 {
		s.params.ConcurrentRepolls = cfg.ConcurrentRepolls
	}
}
