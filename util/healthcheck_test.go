// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"testing"

	"time"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	q := queue.New("channel")
	health := NewHealthCheckServer(q.Client())

	api := new(mocks.QueueProtocolAPI)
	reply := &types.Reply{IsOk: true}
	api.On("IsSync").Return(reply, nil)
	peer1 := &types.Peer{Addr: "addr1"}
	peer2 := &types.Peer{Addr: "addr2"}
	peers := &types.PeerList{Peers: []*types.Peer{peer1, peer2}}
	api.On("PeerInfo").Return(peers, nil)
	api.On("Close").Return()
	health.api = api

	cfg, _ := types.InitCfg("../cmd/chain33/chain33.test.toml")
	health.Start(cfg.Health)
	time.Sleep(time.Second * 3)
	health.Close()
	time.Sleep(time.Second * 1)
}

func TestGetHealth(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	reply := &types.Reply{IsOk: true}
	api.On("IsSync").Return(reply, nil).Once()
	peer2 := &types.Peer{Addr: "addr2"}
	peerlist := &types.PeerList{Peers: []*types.Peer{peer2}}
	api.On("PeerInfo").Return(peerlist, nil).Once()

	q := queue.New("channel")
	health := NewHealthCheckServer(q.Client())
	health.api = api
	ret, err := health.getHealth(true)
	assert.Nil(t, err)
	assert.Equal(t, false, ret)

}
