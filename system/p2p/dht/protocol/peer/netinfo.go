package peer

import (
	"strings"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (p *peerInfoProtol) netinfoHandleEvent(msg *queue.Message) {
	log.Debug("peerInfoProtol", "net info", msg)
	insize, outsize := p.ConnManager.BoundSize()
	var netinfo types.NodeNetInfo

	netinfo.Externaladdr = p.getExternalAddr()
	netinfo.Localaddr = strings.Split(p.GetHost().Addrs()[0].String(), "/")[2]
	if netinfo.Localaddr == "127.0.0.1" {
		addrs := p.GetHost().Addrs()
		if len(addrs) > 0 {
			netinfo.Localaddr = strings.Split(p.GetHost().Addrs()[len(addrs)-1].String(), "/")[2]
		}

	}
	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	netinfo.Service = false
	if netinfo.Inbounds != 0 {
		netinfo.Service = true
	}
	msg.Reply(p.GetQueueClient().NewMessage("rpc", types.EventReplyNetInfo, &netinfo))

}
