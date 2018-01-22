package p2p

import (
	"sync/atomic"
	"time"

	"code.aliyun.com/chain33/chain33/types"

	"code.aliyun.com/chain33/chain33/queue"
	l "github.com/inconshreveable/log15"
)

var log = l.New("module", "p2p")

type P2p struct {
	q            *queue.Queue
	c            queue.IClient
	node         *Node
	addrBook     *AddrBook // known peers
	cli          *P2pCli
	txCapcity    int32
	txFactory    chan struct{}
	otherFactory chan struct{}
	done         chan struct{}
	loopdone     chan struct{}
}

func New(cfg *types.P2P) *P2p {
	node, err := NewNode(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	p2p := new(P2p)
	p2p.node = node
	p2p.done = make(chan struct{}, 1)
	p2p.loopdone = make(chan struct{}, 1)
	p2p.cli = NewP2pCli(p2p)
	return p2p
}
func (network *P2p) Stop() {
	network.done <- struct{}{}
}

func (network *P2p) Close() {
	network.Stop()
	log.Error("close", "network", "ShowTaskCapcity done")
	network.cli.Close()
	log.Error("close", "msg", "done")
	network.node.Close()
	log.Error("close", "node", "done")
	network.c.Close()
	<-network.loopdone
	log.Error("close", "loopdone", "done")
	close(network.txFactory)
	close(network.otherFactory)
}

func (network *P2p) SetQueue(q *queue.Queue) {
	network.c = q.GetClient()
	network.q = q
	network.node.setQueue(q)
	network.node.Start()
	network.cli.monitorPeerInfo()
	network.subP2pMsg()

}
func (network *P2p) ShowTaskCapcity() {
	ticker := time.NewTicker(time.Second * 5)
	log.Error("ShowTaskCapcity", "Capcity", network.txCapcity)
	defer ticker.Stop()
	for {

		select {
		case <-network.done:
			log.Error("ShowTaskCapcity", "Show", "will Done")
			return
		case <-ticker.C:
			log.Error("ShowTaskCapcity", "Capcity", atomic.LoadInt32(&network.txCapcity))

		}
	}
}
func (network *P2p) subP2pMsg() {
	if network.c == nil {
		return
	}
	network.txFactory = make(chan struct{}, 1000)    // 1000 task
	network.otherFactory = make(chan struct{}, 1000) //other task 1000
	network.txCapcity = 1000
	network.c.Sub("p2p")
	go network.ShowTaskCapcity()
	go func() {
		//TODO channel
		for msg := range network.c.Recv() {

			log.Debug("Recv", "Ty", msg.Ty)
			if msg.Ty == types.EventTxBroadcast {
				network.txFactory <- struct{}{} //allocal task
				atomic.AddInt32(&network.txCapcity, -1)
			} else {
				if msg.Ty != types.EventPeerInfo {
					network.otherFactory <- struct{}{}
				}

			}
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
			default:
				log.Warn("unknown msgtype", "msg", msg)
				msg.Reply(network.c.NewMessage("", msg.Ty, types.Reply{false, []byte("unknown msgtype")}))
				<-network.otherFactory
				continue
			}
		}
		network.loopdone <- struct{}{}
	}()

}
