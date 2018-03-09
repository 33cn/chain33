package p2p

import (
	"sync/atomic"
	"time"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	l "github.com/inconshreveable/log15"
)

var (
	log = l.New("module", "p2p")
	pub *common.PubSub
)

type P2p struct {
	q            *queue.Queue
	c            queue.Client
	node         *Node
	addrBook     *AddrBook // known peers
	cli          *P2pCli
	txCapcity    int32
	txFactory    chan struct{}
	otherFactory chan struct{}
	loopdone     chan struct{}
}

func New(cfg *types.P2P) *P2p {

	pub = common.NewPubSub(int(cfg.GetMsgCacheSize()))
	node, err := NewNode(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	p2p := new(P2p)
	p2p.node = node
	p2p.loopdone = make(chan struct{}, 1)
	p2p.cli = NewP2pCli(p2p)
	p2p.txFactory = make(chan struct{}, 1000)    // 1000 task
	p2p.otherFactory = make(chan struct{}, 1000) //other task 1000
	return p2p
}
func (network *P2p) Stop() {
	network.loopdone <- struct{}{}
}

func (network *P2p) Close() {
	network.Stop()
	log.Info("close", "network", "ShowTaskCapcity done")
	network.cli.Close()
	log.Info("close", "msg", "done")
	network.node.Close()
	log.Info("close", "node", "done")
	if network.c != nil {
		network.c.Close()
	}

	close(network.txFactory)
	close(network.otherFactory)
	pub.Shutdown()
}

func (network *P2p) SetQueue(q *queue.Queue) {
	network.c = q.NewClient()
	network.q = q
	network.node.SetQueue(q)
	go func() {
		network.node.Start()
		network.cli.monitorPeerInfo()
		network.subP2pMsg()
	}()
}

func (network *P2p) ShowTaskCapcity() {
	ticker := time.NewTicker(time.Second * 5)
	log.Info("ShowTaskCapcity", "Capcity", network.txCapcity)
	defer ticker.Stop()
	for {

		select {
		case <-network.loopdone:
			log.Debug("ShowTaskCapcity", "Show", "will Done")
			return
		case <-ticker.C:
			log.Debug("ShowTaskCapcity", "Capcity", atomic.LoadInt32(&network.txCapcity))

		}
	}
}

func (network *P2p) subP2pMsg() {
	if network.c == nil {
		return
	}

	network.txCapcity = 1000
	network.c.Sub("p2p")
	go network.ShowTaskCapcity()
	go func() {
		for msg := range network.c.Recv() {

			log.Debug("p2p recv", "msg", types.GetEventName(int(msg.Ty)), "txCap", len(network.txFactory), "othercap", len(network.otherFactory))
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
			case types.EventFetchBlockHeaders:
				go network.cli.GetHeaders(msg)
			default:
				log.Warn("unknown msgtype", "msg", msg)
				msg.Reply(network.c.NewMessage("", msg.Ty, types.Reply{false, []byte("unknown msgtype")}))
				<-network.otherFactory
				continue
			}
		}
		log.Info("subP2pMsg", "loop", "close")

	}()

}
