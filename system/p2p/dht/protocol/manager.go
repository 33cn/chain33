package protocol

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/queue"
)

//TODO
type Initializer func(env *P2PEnv)

var (
	protocolInitializerArray []Initializer
)

func RegisterProtocolInitializer(initializer Initializer) {
	protocolInitializerArray = append(protocolInitializerArray, initializer)
}

func InitAllProtocol(env *P2PEnv) {
	for _, initializer := range protocolInitializerArray {
		initializer(env)
	}
}

// EventHandler handle chain33 event
type EventHandler func(*queue.Message)

var (
	eventHandlerMap   = make(map[int64]EventHandler)
	eventHandlerMutex sync.RWMutex
)

// RegisterEventHandler registers a handler with an event ID.
func RegisterEventHandler(eventID int64, handler EventHandler) {
	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	eventHandlerMutex.Lock()
	defer eventHandlerMutex.Unlock()
	if _, dup := eventHandlerMap[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d, len=%d", eventID, len(eventHandlerMap)))
	}
	eventHandlerMap[eventID] = handler
}

// GetEventHandler gets event handler by event ID.
func GetEventHandler(eventID int64) EventHandler {
	eventHandlerMutex.RLock()
	defer eventHandlerMutex.RUnlock()
	return eventHandlerMap[eventID]
}

// ClearEventHandler clear event handler map, plugin存在多个p2p实例测试，会导致重复注册，需要清除
func ClearEventHandler() {
	for k := range eventHandlerMap {
		delete(eventHandlerMap, k)
	}
}
