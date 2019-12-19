package types

import (
	"fmt"
	"reflect"
	"time"

	"github.com/33cn/chain33/p2pnext/manage"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
)

var (
	protocolTypeMap = make(map[string]reflect.Type)
)

type IProtocol interface {
	InitProtocol(*GlobalData)
}

func RegisterProtocolType(typeName string, proto IProtocol) {

	if proto == nil {
		panic("RegisterProtocolType, protocol is nil, msgId=" + typeName)
	}
	if _, dup := protocolTypeMap[typeName]; dup {
		panic("RegisterProtocolType, protocol is nil, msgId=" + typeName)
	}
	protocolTypeMap[typeName] = reflect.TypeOf(proto)
}

func init() {
	RegisterProtocolType("BaseProtocol", &BaseProtocol{})
}

type ProtocolManager struct {
	protoMap map[string]IProtocol
}

type GlobalData struct {
	ChainCfg        *types.Chain33Config
	QueueClient     queue.Client
	Host            core.Host
	StreamManager   *manage.StreamManager
	PeerInfoManager *manage.PeerInfoManager
}

// BaseProtocol store public data
type BaseProtocol struct {
	*GlobalData
}

func (p *BaseProtocol) InitProtocol(data *GlobalData) {
	if p.GlobalData == nil {
		p.GlobalData = new(GlobalData)
	}
	p.GlobalData = data
}

func (p *ProtocolManager) Init(data *GlobalData) {
	p.protoMap = make(map[string]IProtocol)
	//每个P2P实例都重新分配相关的protocol结构
	for id, protocolType := range protocolTypeMap {
		fmt.Println("protoTy id", id)

		protocol := reflect.New(protocolType.Elem()).Interface().(IProtocol)
		protocol.InitProtocol(data)
		p.protoMap[id] = protocol

	}

	//每个P2P实例都重新分配相关的handler结构
	for id, handlerType := range streamHandlerTypeMap {
		fmt.Println("stream msg id", id)
		newHandler := reflect.New(handlerType.Elem()).Interface().(StreamHandler)
		protoID, msgID := decodeHandlerTypeID(id)
		protol := p.protoMap[protoID]
		newHandler.SetProtocol(protol)

		var baseHander BaseStreamHandler
		baseHander.child = newHandler
		baseHander.SetProtocol(p.protoMap[protoID])
		data.Host.SetStreamHandler(core.ProtocolID(msgID), baseHander.HandleStream)
	}

}

func (s *BaseProtocol) NewMessageCommon(messageId, pid string, nodePubkey []byte, gossip bool) *types.MessageComm {
	return &types.MessageComm{Version: "",
		NodeId:     pid,
		NodePubKey: nodePubkey,
		Timestamp:  time.Now().Unix(),
		Id:         messageId,
		Gossip:     gossip}

}

func (p *BaseProtocol) GetChainCfg() *types.Chain33Config {

	return p.ChainCfg

}

func (p *BaseProtocol) GetQueueClient() queue.Client {

	return p.QueueClient
}

func (p *BaseProtocol) GetHost() core.Host {

	return p.Host

}

func (p *BaseProtocol) GetStreamManager() *manage.StreamManager {
	return p.StreamManager

}

func (p *BaseProtocol) GetPeerInfoManager() *manage.PeerInfoManager {
	return p.PeerInfoManager
}
