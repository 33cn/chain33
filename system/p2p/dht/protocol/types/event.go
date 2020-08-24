// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"

	"github.com/33cn/chain33/queue"
)

// EventHandler handle chain33 event
type EventHandler func(*queue.Message)

var (
	eventHandlerMap = make(map[int64]EventHandler)
)

// RegisterEventHandler 注册消息处理函数
func RegisterEventHandler(eventID int64, handler EventHandler) {

	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	if _, dup := eventHandlerMap[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d", eventID))
	}
	eventHandlerMap[eventID] = handler
}

// GetEventHandler get event handler
func GetEventHandler(eventID int64) (EventHandler, bool) {
	handler, ok := eventHandlerMap[eventID]

	return handler, ok
}

// ClearEventHandler clear event handler map
func ClearEventHandler() {
	eventHandlerMap = make(map[int64]EventHandler)
}
