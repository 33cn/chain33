package peer

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (p *PeerInfoProtol) netinfoHandleEvent(msg *queue.Message) {
	log.Info("PeerInfoProtol", "net info", msg)
	insize, outsize := p.ConnManager.BoundSize()

	var netinfo types.NodeNetInfo
	for i, addr := range p.GetHost().Addrs() {
		if i == 0 {
			netinfo.Localaddr = addr.String()
		}else {
			netinfo.Externaladdr += ( addr.String() + " " )
		}
	}
	//netinfo.Externaladdr = externalAddr
	//netinfo.Localaddr = p.GetHost().Addrs()[0].String()
	netinfo.Service = true
	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	msg.Reply(p.GetQueueClient().NewMessage("rpc", types.EventReplyNetInfo, &netinfo))

}
