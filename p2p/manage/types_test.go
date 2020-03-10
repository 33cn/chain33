// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package manage

import (
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

var (
	testTy = "testP2P"
)

type testP2P struct {
}

func newTestP2P(mgr *P2PMgr, subCfg []byte) IP2P {

	return &testP2P{}
}

func (p *testP2P) StartP2P() {}

func (p *testP2P) CloseP2P() {}

func initRegP2P() {
	p2pRegTypes = make(map[string]CreateP2P)
}

func TestRegisterLoad(t *testing.T) {

	initRegP2P()
	RegisterP2PCreate(testTy, newTestP2P)
	RegisterP2PCreate(testTy+"test", newTestP2P)
	assert.Equal(t, 2, len(p2pRegTypes))
	create := LoadP2PCreate(testTy)
	assert.NotNil(t, create)
}

func TestEvent(t *testing.T) {

	cfg := types.NewChain33Config(types.ReadFile("../../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	q.SetConfig(cfg)
	go q.Start()
	initRegP2P()
	RegisterP2PCreate(testTy, newTestP2P)
	p2pCfg := cfg.GetModuleConfig().P2P
	p2pCfg.Types = []string{testTy}

	mgr := NewP2PMgr(cfg)
	defer mgr.Close()
	mgr.SetQueueClient(q.Client())

	subChan := mgr.PubSub.Sub(testTy)

	events := []int64{types.EventTxBroadcast, types.EventBlockBroadcast,
		types.EventFetchBlocks, types.EventGetMempool, types.EventFetchBlockHeaders,
		types.EventPeerInfo, types.EventGetNetInfo}

	for _, ty := range events {

		mgr.Client.Send(mgr.Client.NewMessage("p2p", ty, nil), false)
		msg, ok := (<-subChan).(*queue.Message)
		assert.True(t, ok)
		assert.Equal(t, ty, msg.Ty)
	}
}
