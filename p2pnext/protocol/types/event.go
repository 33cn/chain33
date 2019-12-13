package types

import (
	"fmt"
	"github.com/33cn/chain33/queue"
)


type EventHandler func(*queue.Message)

var (
	eventHandlerMap = make(map[int64]EventHandler)
)

func RegisterEventHandler(eventID int64, handler EventHandler) {

	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	if _, dup := eventHandlerMap[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d", eventID))
	}
	eventHandlerMap[eventID] = handler
}



func GetEventHandler(eventID int64) (EventHandler, bool) {
	handler, ok := eventHandlerMap[eventID]
	return handler, ok
}


