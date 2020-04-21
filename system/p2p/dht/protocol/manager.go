package protocol

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/queue"
)

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
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d", eventID))
	}
	eventHandlerMap[eventID] = handler
}

// GetEventHandler gets event handler by event ID.
func GetEventHandler(eventID int64) EventHandler {
	eventHandlerMutex.RLock()
	defer eventHandlerMutex.RUnlock()
	return eventHandlerMap[eventID]
}
