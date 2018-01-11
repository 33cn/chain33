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
	cli      *P2pCli
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
	p2p.cli = NewP2pCli(p2p)
	return p2p
}
func (network *P2p) Stop() {
	network.done <- struct{}{}
}

func (network *P2p) Close() {
	network.Stop()
	log.Debug("close", "network", "done")
	network.cli.Close()
	log.Debug("close", "msg", "done")
	network.node.Close()
	log.Debug("close", "node", "done")
}

func (network *P2p) SetQueue(q *queue.Queue) {
	network.c = q.GetClient()
	network.q = q
	network.node.setQueue(q)
	network.node.Start()
	network.subP2pMsg()
	network.cli.monitorPeerInfo()

}

func (network *P2p) subP2pMsg() {
	if network.c == nil {
		return
	}

	network.c.Sub("p2p")
	go func() {
		var EventQueryTask int64 = 666 //TODO
		for msg := range network.c.Recv() {
			log.Debug("SubP2pMsg", "Ty", msg.Ty)

			switch msg.Ty {
			case types.EventTxBroadcast: //广播tx
				go network.cli.BroadCastTx(msg)
			case types.EventBlockBroadcast: //广播block
				go network.cli.BlockBroadcast(msg)
			case types.EventFetchBlocks:
				go network.cli.GetBlocks(msg)
			case types.EventGetMempool:
				go network.cli.GetMemPool(msg)
			case types.EventPeerInfo:
				go network.cli.GetPeerInfo(msg)
			case EventQueryTask:
				go network.cli.GetTaskInfo(msg)
			default:
				log.Warn("unknown msgtype", "Ty", msg.Ty)
				msg.Reply(network.c.NewMessage("", msg.Ty, types.Reply{false, []byte("unknown msgtype")}))

				continue
			}
		}
	}()

}
