package p2p

import (
	"code.aliyun.com/chain33/chain33/types"

	"code.aliyun.com/chain33/chain33/queue"
	l "github.com/inconshreveable/log15"
)

var log = l.New("module", "p2p")

type P2p struct {
	q        *queue.Queue
	c        queue.IClient
	node     *Node
	addrBook *AddrBook // known peers
	msg      *Msg
	done     chan struct{}
}

func New(cfg *types.P2P) *P2p {
	node, err := newNode(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	p2p := new(P2p)
	p2p.node = node
	p2p.done = make(chan struct{}, 1)
	p2p.msg = NewMsg(p2p)
	return p2p
}
func (network *P2p) Stop() {
	network.done <- struct{}{}
}

func (network *P2p) Close() {
	network.Stop()
	log.Debug("close", "network", "done")
	network.msg.Stop()
	log.Debug("close", "msg", "done")
	network.node.Stop()
	log.Debug("close", "node", "done")
}

func (network *P2p) SetQueue(q *queue.Queue) {
	network.c = q.GetClient()
	network.q = q
	network.node.setQueue(q)
	network.node.Start()
	network.subP2pMsg()
	network.msg.monitorPeerInfo()

}

func (network *P2p) subP2pMsg() {
	if network.c == nil {
		return
	}

	network.c.Sub("p2p")
	go func() {

		for msg := range network.c.Recv() {
			log.Debug("SubP2pMsg", "Ty", msg.Ty)

			switch msg.Ty {
			case types.EventTxBroadcast: //广播tx
				log.Debug("subP2pMsg", " EventTxBroadcast", "txmsg")
				go network.msg.TransToBroadCast(msg)
			case types.EventBlockBroadcast: //广播block
				go network.msg.BlockBroadcast(msg)
			case types.EventFetchBlocks:
				tempIntrans := NewMsg(network)
				go tempIntrans.GetBlocks(msg)
			case types.EventGetMempool:
				go network.msg.GetMemPool(msg)
			case types.EventPeerInfo:
				go network.msg.GetPeerInfo(msg)

			default:
				log.Error("unknown msgtype:", msg.Ty)
				msg.Reply(network.c.NewMessage("", msg.Ty, types.Reply{false, []byte("unknown msgtype")}))

				continue
			}
		}
	}()

}
