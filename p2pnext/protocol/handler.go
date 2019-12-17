package protocol

import (
	"github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/queue"
)

func HandleEvent(msg *queue.Message) {

	if eventHander, ok := types.GetEventHandler(msg.Ty); ok {
		eventHander(msg)
	}
}
