package protocol

import (
	"fmt"
	"strings"

	"github.com/33cn/chain33/queue"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
)

//TODO
func RegisterStreamHandler(h host.Host, p protocol.ID, handler network.StreamHandler) {
	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, protocol=%s", p))
	}
	//协议格式为形如Linux绝对路径的的字符串，最后一个子串为形如 1.0.0 格式的版本号，如：
	// /chain33/fetch-chunk/1.0.0
	//仅版本号不同的协议会匹配相同的handler，不同版本具体处理逻辑在handler里采用不同的逻辑分支
	match := func(protocol string) bool {
		expected := strings.Split(string(p), "/")
		actual := strings.Split(protocol, "/")
		if len(expected) != len(actual) {
			return false
		}
		for i := range expected[:len(expected)-1] {
			if expected[i] != actual[i] {
				return false
			}
		}
		return true
	}
	h.SetStreamHandlerMatch(p, match, HandlerWithClose(handler))
}

//Initializer is a initial function which any protocol should have.
type Initializer func(env *P2PEnv)

var (
	protocolInitializerArray []Initializer
)

//RegisterProtocolInitializer registers the initial function.
func RegisterProtocolInitializer(initializer Initializer) {
	protocolInitializerArray = append(protocolInitializerArray, initializer)
}

//InitAllProtocol initials all protocols.
func InitAllProtocol(env *P2PEnv) {
	for _, initializer := range protocolInitializerArray {
		initializer(env)
	}
}

// EventHandler handles chain33 event
type EventHandler func(*queue.Message)

var (
	eventHandlers = make(map[int64]EventHandler)
)

// RegisterEventHandler registers a handler with an event ID.
func RegisterEventHandler(eventID int64, handler EventHandler) {
	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	if _, dup := eventHandlers[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d, len=%d", eventID, len(eventHandlers)))
	}
	eventHandlers[eventID] = EventHandlerWithRecover(handler)
}

// GetEventHandler gets event handler by event ID.
func GetEventHandler(eventID int64) EventHandler {
	return eventHandlers[eventID]
}

// ClearEventHandler clear event handler map, plugin存在多个p2p实例测试，会导致重复注册，需要清除
func ClearEventHandler() {
	for k := range eventHandlers {
		delete(eventHandlers, k)
	}
}
