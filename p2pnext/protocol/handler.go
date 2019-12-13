package protocol

import (
	"github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/queue"
	net "github.com/libp2p/go-libp2p-core/network"
	"io/ioutil"

)

func HandleEvent(msg *queue.Message) {

	if handler, ok := types.GetEventHandler(msg.Ty); ok {
		handler(msg)
	}
}



func HandleStream(stream net.Stream) {

	msgID := stream.Protocol()
	for {

		buf, err := ioutil.ReadAll(stream)
		if err != nil {
			stream.Reset()
			logger.Error(err)
			continue
		}

		if handler, ok := types.GetStreamHandler(string(msgID)); ok {

			if !handler.VerifyRequest(buf) {
				//invalid request
				continue
			}

			resp, err := handler.Handle(buf, stream)
			if err != nil {
				continue
			}
			if resp.Msg != nil {
				//TODO, send response message
			}
		}
	}
}

