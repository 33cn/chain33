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
	q           *queue.Queue
	c           queue.IClient
	node        *Node
	addrBook    *AddrBook // known peers
	cli         *P2pCli
	taskCapcity int32
	taskFactory chan struct{}
	done        chan struct{}
	loopdone    chan struct{}
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
	<-network.loopdone
	log.Error("close", "loopdone", "done")
	close(network.taskFactory)
	network.c.Close()
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
	log.Error("ShowTaskCapcity", "Capcity", network.taskCapcity)
	defer ticker.Stop()
	for {

		select {
		case <-network.done:
			log.Error("ShowTaskCapcity", "Show", "will Done")
			return
		case <-ticker.C:
			log.Error("ShowTaskCapcity", "Capcity", atomic.AddInt32(&network.taskCapcity, -1)+1)
			atomic.AddInt32(&network.taskCapcity, 1)

		}
	}
}
func (network *P2p) subP2pMsg() {
	if network.c == nil {
		return
	}
	network.taskFactory = make(chan struct{}, 2000) // 2000 task
	network.taskCapcity = 2000
	network.c.Sub("p2p")
	go network.ShowTaskCapcity()
	go func() {
		//TODO channel
		for msg := range network.c.Recv() {
			if msg.Ty != types.EventBlockBroadcast {
				network.taskFactory <- struct{}{} //allocal task
				atomic.AddInt32(&network.taskCapcity, -1)
			}
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
				//log.Error("SubP2pMsg", "Ty", msg.Ty, "timestamp", time.Now().Second(), "taskCapcity", capcity)
				go network.cli.GetPeerInfo(msg)
			default:
				<-network.taskFactory
				atomic.AddInt32(&network.taskCapcity, -1)
				log.Warn("unknown msgtype", "msg", msg)
				msg.Reply(network.c.NewMessage("", msg.Ty, types.Reply{false, []byte("unknown msgtype")}))

				continue
			}
		}
		network.loopdone <- struct{}{}
	}()

}
