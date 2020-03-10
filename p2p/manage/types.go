// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package manage 实现多种类型p2p兼容管理
package manage

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/pubsub"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	lru "github.com/hashicorp/golang-lru"
)

// IP2P p2p interface
type IP2P interface {
	StartP2P()
	CloseP2P()
}

// P2PMgr p2p manager
type P2PMgr struct {
	SysAPI   client.QueueProtocolAPI
	Client   queue.Client
	ChainCfg *types.Chain33Config
	PubSub   *pubsub.PubSub
	//subChan         chan interface{}
	broadcastFilter *lru.Cache
	p2ps            []IP2P
	p2pCfg          *types.P2P
}

// CreateP2P p2p creator
type CreateP2P func(mgr *P2PMgr, subCfg []byte) IP2P

var (
	p2pRegTypes = make(map[string]CreateP2P)
)

// RegisterP2PCreate register p2p create func
func RegisterP2PCreate(p2pType string, create CreateP2P) {

	if create == nil {
		panic("RegisterP2P, nil CreateP2P, p2pType = " + p2pType)
	}
	if _, ok := p2pRegTypes[p2pType]; ok {
		panic("RegisterP2P, duplicate, p2pType = " + p2pType)
	}

	p2pRegTypes[p2pType] = create
}

// LoadP2PCreate load p2p create func
func LoadP2PCreate(p2pType string) CreateP2P {

	if create, ok := p2pRegTypes[p2pType]; ok {
		return create
	}
	panic("LoadP2P, not exist, p2pType = " + p2pType)
}

// NewP2PMgr new p2p manager
func NewP2PMgr(cfg *types.Chain33Config) *P2PMgr {
	mgr := &P2PMgr{
		PubSub: pubsub.NewPubSub(1024),
	}
	var err error
	mgr.broadcastFilter, err = lru.New(10240)
	if err != nil {
		panic("NewP2PBasePanic, err=" + err.Error())
	}
	mgr.ChainCfg = cfg
	mgr.p2pCfg = cfg.GetModuleConfig().P2P
	// set default p2p config
	if len(mgr.p2pCfg.Types) == 0 {
		//默认设置为老版本p2p协议
		mgr.p2pCfg.Types = append(mgr.p2pCfg.Types, defaultP2PType)
	}

	return mgr
}

// Wait wait p2p
func (mgr *P2PMgr) Wait() {}

// Close close p2p
func (mgr *P2PMgr) Close() {

	for _, p2p := range mgr.p2ps {
		p2p.CloseP2P()
		p2p = nil
	}
	mgr.p2ps = nil
	//mgr.PubSub.Unsub(mgr.subChan)
	if mgr.Client != nil {
		mgr.Client.Close()
	}
}

// SetQueueClient set the queue
func (mgr *P2PMgr) SetQueueClient(cli queue.Client) {
	var err error
	mgr.SysAPI, err = client.New(cli, nil)
	if err != nil {
		panic("SetQueueClient client.New err:" + err.Error())
	}
	mgr.Client = cli

	p2pCfg := mgr.ChainCfg.GetModuleConfig().P2P
	subCfg := mgr.ChainCfg.GetSubConfig().P2P

	for _, p2pTy := range p2pCfg.Types {

		newP2P := LoadP2PCreate(p2pTy)(mgr, subCfg[p2pTy])
		newP2P.StartP2P()
		mgr.p2ps = append(mgr.p2ps, newP2P)
	}

	go mgr.handleSysEvent()
	//go mgr.handleP2PSub()
}
