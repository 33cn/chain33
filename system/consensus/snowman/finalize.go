// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package snowman
package snowman

import (
	"runtime"

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
	ctx    *consensus.Context
	inMsg  chan *queue.Message
}

func (s *snowman) Initialize(ctx *consensus.Context) {

	s.inMsg = make(chan *queue.Message, 32)

	params := snowball.Parameters{}
	vm := &chain33VM{}
	vm.Init(ctx)
	engineConfig := newSnowmanConfig(vm, params, newSnowContext(ctx.Base.GetAPI().GetConfig()))

	engine, err := smeng.New(engineConfig)

	if err != nil {
		panic("Initialize snowman engine err:" + err.Error())
	}
	s.engine = engine
	s.vm = vm

}

func (s *snowman) Start() error {

	err := s.engine.Start(s.ctx.Base.Context, 0)

	if err != nil {

		return err
	}

	//使用多个协程并发处理，提高效率
	concurrency := runtime.NumCPU() * 2
	for i := 0; i < concurrency; i++ {
		go s.handleMsgRountine()
	}
	return nil

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
			s.handleMsg(msg)
		}

	}
}

func (s *snowman) handleMsg(msg *queue.Message) {

	switch msg.ID {

	}
}
