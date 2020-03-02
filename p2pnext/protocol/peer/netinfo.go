package peer

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (p *PeerInfoProtol) netinfoHandleEvent(msg *queue.Message) {
	log.Info("PeerInfoProtol", "net info", msg)
	insize, outsize := p.ConnManager.BoundSize()

	var netinfo types.NodeNetInfo
	netinfo.Externaladdr = externalAddr
	netinfo.Localaddr = p.GetHost().Addrs()[0].String()
	netinfo.Service = true
	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	msg.Reply(p.GetQueueClient().NewMessage("rpc", types.EventReplyNetInfo, &netinfo))

}
