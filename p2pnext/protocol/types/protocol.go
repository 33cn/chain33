package types

import (
	"reflect"

	"github.com/33cn/chain33/p2pnext/manage"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
)

var (
	protocolTypeMap map[string]reflect.Type
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
	p.GlobalData = data
}

func (p *ProtocolManager) Init(data *GlobalData) {

	//  每个P2P实例都重新分配相关的protocol结构
	for id, protocolType := range protocolTypeMap {
		protocol := reflect.New(protocolType).Interface().(IProtocol)
		protocol.InitProtocol(data)
		p.protoMap[id] = protocol
	}

	//  每个P2P实例都重新分配相关的handler结构
	for id, handlerType := range streamHandlerTypeMap {
		newHandler := reflect.New(handlerType).Interface().(StreamHandler)
		protoID, msgID := decodeHandlerTypeID(id)
		newHandler.SetProtocol(p.protoMap[protoID])
		data.Host.SetStreamHandler(core.ProtocolID(msgID), BaseStreamHandler{child: newHandler}.HandleStream)

	}

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
