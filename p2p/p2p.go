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
	done     chan struct{}
}

func New(cfg *types.P2P) *P2p {
	node, err := newNode(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return &P2p{
		node: node,
		done: make(chan struct{}, 1),
	}
}

func (network *P2p) Close() {
	network.node.Stop()
	network.done <- struct{}{}
}

func (network *P2p) SetQueue(q *queue.Queue) {
	network.c = q.GetClient()
	network.q = q
	network.node.setQueue(q)
	network.node.Start()
	network.subP2pMsg()

}

func (network *P2p) subP2pMsg() {
	if network.c == nil {
		return
	}

	network.c.Sub("p2p")
	go func() {
		intrans := NewInTrans(network)
		go intrans.monitorPeerInfo()
		for msg := range network.c.Recv() {
			log.Debug("SubP2pMsg", "Ty", msg.Ty)

			switch msg.Ty {
			case types.EventTxBroadcast: //广播tx
				log.Debug("QUEUE P2P EventTxBroadcast", "Recv from mempool message EventTxBroadcast will broadcast outnet")
				go intrans.TransToBroadCast(msg)
			case types.EventBlockBroadcast: //广播block
				go intrans.BlockBroadcast(msg)
			case types.EventFetchBlocks:
				tempIntrans := NewInTrans(network)
				go tempIntrans.GetBlocks(msg)
			case types.EventGetMempool:
				go intrans.GetMemPool(msg)
			case types.EventPeerInfo:
				go intrans.GetPeerInfo(msg)

			default:
				log.Error("unknown msgtype:", msg.Ty)
				msg.Reply(network.c.NewMessage("", msg.Ty, types.Reply{false, []byte("unknown msgtype")}))

				continue
			}
		}
	}()

}
