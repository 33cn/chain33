package p2p

import (
	"sync/atomic"
	"time"

	l "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/pubsub"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	log = l.New("module", "p2p")
	pub *pubsub.PubSub
)

type P2p struct {
	client       queue.Client
	node         *Node
	p2pCli       *P2pCli
	txCapcity    int32
	txFactory    chan struct{}
	otherFactory chan struct{}
	closed       int32
}

func New(cfg *types.P2P) *P2p {
	pub = pubsub.NewPubSub(int(cfg.GetMsgCacheSize()))
	node, err := NewNode(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	p2p := new(P2p)
	p2p.node = node
	p2p.p2pCli = NewP2pCli(p2p)
	p2p.txFactory = make(chan struct{}, 1000)    // 1000 task
	p2p.otherFactory = make(chan struct{}, 1000) //other task 1000
	return p2p
}

func (network *P2p) IsClose() bool {
	return atomic.LoadInt32(&network.closed) == 1
}

func (network *P2p) Close() {
	atomic.StoreInt32(&network.closed, 1)
	log.Debug("close", "network", "ShowTaskCapcity done")
	network.node.Close()
	log.Debug("close", "node", "done")
	if network.client != nil {
		network.client.Close()
	}
	close(network.txFactory)
	close(network.otherFactory)
	pub.Shutdown()

}

func (network *P2p) SetQueueClient(client queue.Client) {
	network.client = client
	network.node.SetQueueClient(client.Clone())
	go func() {
		network.node.Start()
		network.subP2pMsg()
		network.loadP2PPrivKeyToWallet()
	}()
}

func (network *P2p) ShowTaskCapcity() {
	ticker := time.NewTicker(time.Second * 5)
	log.Info("ShowTaskCapcity", "Capcity", network.txCapcity)
	defer ticker.Stop()
	for {
		if network.IsClose() {
			return
		}
		select {
		case <-ticker.C:
			log.Debug("ShowTaskCapcity", "Capcity", atomic.LoadInt32(&network.txCapcity))

		}
	}
}

func (network *P2p) loadP2PPrivKeyToWallet() error {

	for {

		msg := network.client.NewMessage("wallet", types.EventGetWalletStatus, nil)
		err := network.client.Send(msg, true)
		if err != nil {
			log.Error("GetWalletStatus", "Error", err.Error())
			time.Sleep(time.Second)
			continue
		}
		resp, err := network.client.Wait(msg)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		if resp.GetData().(*types.WalletStatus).GetIsWalletLock() { //上锁
			time.Sleep(time.Second)
			continue
		}

		if resp.GetData().(*types.WalletStatus).GetIsHasSeed() == false { //无种子
			time.Sleep(time.Second)
			continue
		}

		break
	}
	var parm types.ReqWalletImportPrivKey
	parm.Privkey, _ = network.node.nodeInfo.addrBook.GetPrivPubKey()
	parm.Label = "node award"
	msg := network.client.NewMessage("wallet", types.EventWalletImportprivkey, &parm)
	err := network.client.Send(msg, true)
	if err != nil {
		log.Error("ImportPrivkey", "Error", err.Error())
		return err
	}
	resp, err := network.client.Wait(msg)
	if err != nil {
		if err == types.ErrPrivkeyExist || err == types.ErrLabelHasUsed {
			return nil
		}
		log.Error("loadP2PPrivKeyToWallet", "err", err.Error())
		return err
	}

	log.Debug("loadP2PPrivKeyToWallet", "resp", resp.GetData())
	return nil

}

func (network *P2p) subP2pMsg() {
	if network.client == nil {
		return
	}
	var taskIndex int64
	network.txCapcity = 1000
	network.client.Sub("p2p")
	go network.ShowTaskCapcity()
	go func() {
		for msg := range network.client.Recv() {
			if network.IsClose() {
				return
			}
			taskIndex++
			log.Debug("p2p recv", "msg", types.GetEventName(int(msg.Ty)), "msg type", msg.Ty, "taskIndex", taskIndex)
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
				go network.p2pCli.BroadCastTx(msg, taskIndex)
			case types.EventBlockBroadcast: //广播block
				go network.p2pCli.BlockBroadcast(msg, taskIndex)
			case types.EventFetchBlocks:
				go network.p2pCli.GetBlocks(msg, taskIndex)
			case types.EventGetMempool:
				go network.p2pCli.GetMemPool(msg, taskIndex)
			case types.EventPeerInfo:
				go network.p2pCli.GetPeerInfo(msg, taskIndex)
			case types.EventFetchBlockHeaders:
				go network.p2pCli.GetHeaders(msg, taskIndex)
			default:
				log.Warn("unknown msgtype", "msg", msg)
				msg.Reply(network.client.NewMessage("", msg.Ty, types.Reply{false, []byte("unknown msgtype")}))
				<-network.otherFactory
				continue
			}
		}
		log.Info("subP2pMsg", "loop", "close")

	}()

}
