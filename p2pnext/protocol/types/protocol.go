// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package types protocol and stream register	`
package types

import (
	"reflect"
	"time"

	p2pmanage "github.com/33cn/chain33/p2p/manage"

	"github.com/33cn/chain33/p2pnext/dht"
	"github.com/33cn/chain33/p2pnext/manage"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
)

var (
	protocolTypeMap = make(map[string]reflect.Type)
)

type IProtocol interface {
	InitProtocol(*P2PEnv)
}

// RegisterProtocolType 注册协议类型
func RegisterProtocolType(typeName string, proto IProtocol) {

	if proto == nil {
		panic("RegisterProtocolType, protocol is nil, msgId=" + typeName)
	}
	if _, dup := protocolTypeMap[typeName]; dup {
		panic("RegisterProtocolType, protocol is nil, msgId=" + typeName)
	}

	protoType := reflect.TypeOf(proto)

	if protoType.Kind() == reflect.Ptr {
		protoType = protoType.Elem()
	}
	protocolTypeMap[typeName] = protoType
}

func init() {
	RegisterProtocolType("BaseProtocol", &BaseProtocol{})
}

// ProtocolManager 协议管理
type ProtocolManager struct {
	protoMap map[string]IProtocol
}

// P2PEnv p2p全局公共变量
type P2PEnv struct {
	ChainCfg        *types.Chain33Config
	QueueClient     queue.Client
	Host            core.Host
	ConnManager     *manage.ConnManager
	PeerInfoManager *manage.PeerInfoManager
	Discovery       *dht.Discovery
	P2PManager      *p2pmanage.P2PMgr
	SubConfig       *p2pty.P2PSubConfig
}

// BaseProtocol store public data
type BaseProtocol struct {
	*P2PEnv
}

// Init 初始化
func (p *ProtocolManager) Init(env *P2PEnv) {
	p.protoMap = make(map[string]IProtocol)
	//每个P2P实例都重新分配相关的protocol结构
	for id, protocolType := range protocolTypeMap {
		log.Debug("InitProtocolManager", "protoTy id", id)
		protoVal := reflect.New(protocolType)
		baseValue := protoVal.Elem().FieldByName("BaseProtocol")
		//指针形式继承,需要初始化BaseProtocol结构
		if baseValue != reflect.ValueOf(nil) && baseValue.Kind() == reflect.Ptr {
			baseValue.Set(reflect.ValueOf(&BaseProtocol{})) //对baseprotocal进行初始化
		}

		protocol := protoVal.Interface().(IProtocol)
		protocol.InitProtocol(env)
		p.protoMap[id] = protocol

	}

	//每个P2P实例都重新分配相关的handler结构
	for id, handlerType := range streamHandlerTypeMap {
		log.Debug("InitProtocolManager", "stream handler id", id)
		handlerValue := reflect.New(handlerType)
		baseValue := handlerValue.Elem().FieldByName("BaseStreamHandler")
		//指针形式继承,需要初始化BaseStreamHandler结构
		if baseValue != reflect.ValueOf(nil) && baseValue.Kind() == reflect.Ptr {
			baseValue.Set(reflect.ValueOf(&BaseStreamHandler{}))
		}

		newHandler := handlerValue.Interface().(StreamHandler)
		protoID, msgID := decodeHandlerTypeID(id)
		protol := p.protoMap[protoID]
		newHandler.SetProtocol(protol)
		var baseHander BaseStreamHandler
		baseHander.child = newHandler
		baseHander.SetProtocol(p.protoMap[protoID])
		env.Host.SetStreamHandler(core.ProtocolID(msgID), baseHander.HandleStream)
	}

}

// InitProtocol 初始化协议
func (base *BaseProtocol) InitProtocol(data *P2PEnv) {

	base.P2PEnv = data
}

// NewMessageCommon new msg common struct
func (base *BaseProtocol) NewMessageCommon(msgID, pid string, nodePubkey []byte, gossip bool) *types.MessageComm {
	return &types.MessageComm{Version: "",
		NodeId:     pid,
		NodePubKey: nodePubkey,
		Timestamp:  time.Now().Unix(),
		Id:         msgID,
		Gossip:     gossip}

}

// GetChainCfg get chain cfg
func (base *BaseProtocol) GetChainCfg() *types.Chain33Config {

	return base.ChainCfg

}

// GetQueueClient get chain33 msg queue client
func (base *BaseProtocol) GetQueueClient() queue.Client {

	return base.QueueClient
}

// GetHost get local host
func (base *BaseProtocol) GetHost() core.Host {

	return base.Host

}

// GetConnsManager get connection manager
func (base *BaseProtocol) GetConnsManager() *manage.ConnManager {
	return base.ConnManager

}

// GetPeerInfoManager get peer info manager
func (base *BaseProtocol) GetPeerInfoManager() *manage.PeerInfoManager {
	return base.PeerInfoManager
}
