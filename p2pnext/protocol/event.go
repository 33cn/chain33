package protocol

import (
	"fmt"
	"github.com/33cn/chain33/queue"
)


type eventHandler func(*queue.Message)

var (
	eventHandlerMap = make(map[int64]eventHandler)
)

func RegisterEventHandler(eventID int64, handler eventHandler) {

	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	if _, dup := eventHandlerMap[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d", eventID))
	}
	eventHandlerMap[eventID] = handler
}


func ProcessEvent(msg *queue.Message) {

	if handler, ok := eventHandlerMap[msg.Ty]; ok {
		handler(msg)
	}
}
