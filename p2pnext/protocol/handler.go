package protocol

import (
	"fmt"

	"github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/queue"
)

// HandleEvent handle p2p event
func HandleEvent(msg *queue.Message) {

	if eventHander, ok := types.GetEventHandler(msg.Ty); ok {
		fmt.Println("HandleEvent", msg.Ty)
		eventHander(msg)
	} else {

		fmt.Println("------------------unknown msgtype", msg.Ty)

	}
}
