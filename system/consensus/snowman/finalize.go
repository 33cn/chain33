// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package snowman
package snowman

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/consensus"
	"github.com/ava-labs/avalanchego/snow/consensus/snowball"

	smeng "github.com/ava-labs/avalanchego/snow/engine/snowman"
)

var (
	log = log15.New("module", "snowman")

	_ consensus.Finalizer = (*snowman)(nil)
)

func init() {

	consensus.RegFinalizer("snowman", &snowman{})

}

type snowman struct {
	engine smeng.Engine
	ctx    *consensus.Context
}

func (s *snowman) Initialize(ctx *consensus.Context) error {

	params := snowball.Parameters{}
	engineConfig := newSnowmanConfig(&chain33VM{}, params, newSnowContext(ctx.Base.GetAPI().GetConfig()))

	engine, err := smeng.New(engineConfig)

	if err != nil {
		panic("Initialize snowman engine err:" + err.Error())
	}
	s.engine = engine

	return nil

}

func (s *snowman) Start() error {

	err := s.engine.Start(s.ctx.Base.Context, 0)

	if err != nil {

		return err
	}
	return nil

}
func (s *snowman) ProcessMsg(msg *queue.Message) (processed bool) {

	return false
}
