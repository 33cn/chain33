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
	//外网地址只有一个，显示太多，意义不大
	netinfo.Externaladdr = p.GetExternalAddr()
	netinfo.Localaddr = strings.Split(p.GetHost().Addrs()[0].String(), "/")[2]
	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	netinfo.Service = false
	if netinfo.Inbounds != 0 {
		netinfo.Service = true
	}
	msg.Reply(p.GetQueueClient().NewMessage("rpc", types.EventReplyNetInfo, &netinfo))

}
