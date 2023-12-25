// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snowman

import (
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/consensus/snowball"
	smcon "github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	smeng "github.com/ava-labs/avalanchego/snow/engine/snowman"
	"github.com/ava-labs/avalanchego/snow/validators"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config wraps all the parameters needed for a snowman engine
type Config struct {
	common.AllGetsServer

	//ctx  *consensus.Context
	chainCfg *types.Chain33Config

	ctx        *snow.ConsensusContext
	log        logging.Logger
	registerer prometheus.Registerer
	//VM         block.ChainVM
	//Sender     common.Sender
	validators validators.Set
	params     snowball.Parameters
	consensus  smcon.Consensus
}

func newSnowmanConfig(vm *chain33VM, vs *vdrSet, params snowball.Parameters, ctx *snow.ConsensusContext) smeng.Config {

	engineConfig := smeng.Config{
		Ctx:         ctx,
		VM:          vm,
		Validators:  newVdrSet(),
		Params:      params,
		Consensus:   &smcon.Topological{},
		PartialSync: false,
		Sender:      newMsgSender(ctx),
	}

	return engineConfig
}

func newSnowContext(cfg *types.Chain33Config) *snow.ConsensusContext {

	ctx := snow.DefaultConsensusContextTest()

	// TODO use chain33 log writer
	logCfg := cfg.GetModuleConfig().Log

	logger := &lumberjack.Logger{
		Filename:   logCfg.LogFile,
		MaxSize:    int(logCfg.MaxFileSize),
		MaxBackups: int(logCfg.MaxBackups),
		MaxAge:     int(logCfg.MaxAge),
		LocalTime:  logCfg.LocalTime,
		Compress:   logCfg.Compress,
	}

	fileCore := logging.NewWrappedCore(logging.Debug, logger, logging.Colors.FileEncoder())
	ctx.Context.Log = logging.NewLogger("", fileCore)

	return ctx
}
