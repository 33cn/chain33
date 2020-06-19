package protocol

import (
	"github.com/33cn/chain33/common/log/log15"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol/types"
)

var (
	log = log15.New("module", "p2pnext.protocol")
)

// HandleEvent handle p2p event
func HandleEvent(msg *queue.Message) {

	if eventHander, ok := types.GetEventHandler(msg.Ty); ok {
		//log.Debug("HandleEvent", "msgTy", msg.Ty)
		eventHander(msg)
	} else if eventHandler := GetEventHandler(msg.Ty); eventHandler != nil {
		//log.Debug("HandleEvent2", "msgTy", msg.Ty)
		eventHandler(msg)
	} else {

		log.Error("HandleEvent", "unknown msgTy", msg.Ty)
	}
}
