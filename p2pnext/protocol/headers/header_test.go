// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package headers

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/client"
	p2pmgr "github.com/33cn/chain33/p2p/manage"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

type headersProtoMock struct {
	*headersProtocol
}

func (b *headersProtoMock) sendStream(pid string, data interface{}) error {
	return nil
}

func newTestEnv() *prototypes.P2PEnv {

	cfg := types.NewChain33Config(types.ReadFile("../../../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	q.SetConfig(cfg)
	go q.Start()
	defer q.Close()

	mgr := p2pmgr.NewP2PMgr(cfg)
	mgr.Client = q.Client()
	mgr.SysApi, _ = client.New(mgr.Client, nil)

	subCfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[p2pmgr.DHTTypeName], subCfg)
	subCfg.MinLtBlockTxNum = 1

	env := &prototypes.P2PEnv{
		ChainCfg:        cfg,
		QueueClient:     q.Client(),
		Host:            nil,
		ConnManager:     nil,
		PeerInfoManager: nil,
		Discovery:       nil,
		P2PManager:      mgr,
		SubConfig:       subCfg,
	}
	return env
}
