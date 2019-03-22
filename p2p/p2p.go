// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package p2p 实现了chain33网络协议
package p2p

import (
	"fmt"
	"sync/atomic"
	"time"

	l "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"

	// register gzip
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	log = l.New("module", "p2p")
)

// P2p interface
type P2p struct {
	client       queue.Client
	node         *Node
	p2pCli       EventInterface
	txCapcity    int32
	txFactory    chan struct{}
	otherFactory chan struct{}
	closed       int32
}

// New produce a p2p object
func New(cfg *types.P2P) *P2p {
	if cfg.Version == 0 {
		if types.IsTestNet() {
			cfg.Version = 119
			cfg.VerMin = 118
			cfg.VerMax = 128
		} else {
			cfg.Version = 10020
			cfg.VerMin = 10020
			cfg.VerMax = 11000
		}
	}
	if cfg.VerMin == 0 {
		cfg.VerMin = cfg.Version
	}

	if cfg.VerMax == 0 {
		cfg.VerMax = cfg.VerMin + 1
	}

	VERSION = cfg.Version
	log.Info("p2p", "Version", VERSION, "IsTest", types.IsTestNet())
	if cfg.InnerBounds == 0 {
		cfg.InnerBounds = 500
	}
	log.Info("p2p", "InnerBounds", cfg.InnerBounds)

	node, err := NewNode(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	p2p := new(P2p)
	p2p.node = node
	p2p.p2pCli = NewP2PCli(p2p)
	p2p.txFactory = make(chan struct{}, 1000)    // 1000 task
	p2p.otherFactory = make(chan struct{}, 1000) //other task 1000
	p2p.txCapcity = 1000
	return p2p
}

//Wait wait for ready
func (network *P2p) Wait() {}

func (network *P2p) isClose() bool {
	return atomic.LoadInt32(&network.closed) == 1
}

// Close network client
func (network *P2p) Close() {
	atomic.StoreInt32(&network.closed, 1)
	log.Debug("close", "network", "ShowTaskCapcity done")
	network.node.Close()
	log.Debug("close", "node", "done")
	if network.client != nil {
		network.client.Close()
	}
	network.node.pubsub.Shutdown()

}

// SetQueueClient set the queue
func (network *P2p) SetQueueClient(client queue.Client) {
	network.client = client
	network.node.SetQueueClient(client)
	go func() {
		log.Info("p2p", "setqueuecliet", "ok")
		network.node.Start()
		network.subP2pMsg()
		err := network.loadP2PPrivKeyToWallet()
		if err != nil {
			return
		}
	}()
}

func (network *P2p) showTaskCapcity() {
	ticker := time.NewTicker(time.Second * 5)
	log.Info("ShowTaskCapcity", "Capcity", atomic.LoadInt32(&network.txCapcity))
	defer ticker.Stop()
	for {
		if network.isClose() {
			log.Debug("ShowTaskCapcity", "loop", "done")
			return
		}
		<-ticker.C
		log.Debug("ShowTaskCapcity", "Capcity", atomic.LoadInt32(&network.txCapcity))
	}
}

func (network *P2p) loadP2PPrivKeyToWallet() error {

	for {
		if network.isClose() {
			return nil
		}
		msg := network.client.NewMessage("wallet", types.EventGetWalletStatus, nil)
		err := network.client.SendTimeout(msg, true, time.Minute)
		if err != nil {
			log.Error("GetWalletStatus", "Error", err.Error())
			time.Sleep(time.Second)
			continue
		}

		resp, err := network.client.WaitTimeout(msg, time.Minute)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if resp.GetData().(*types.WalletStatus).GetIsWalletLock() { //上锁
			time.Sleep(time.Second)
			continue
		}

		if !resp.GetData().(*types.WalletStatus).GetIsHasSeed() { //无种子
			time.Sleep(time.Second)
			continue
		}

		break
	}
	var parm types.ReqWalletImportPrivkey
	parm.Privkey, _ = network.node.nodeInfo.addrBook.GetPrivPubKey()
	parm.Label = "node award"
ReTry:
	msg := network.client.NewMessage("wallet", types.EventWalletImportPrivkey, &parm)
	err := network.client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("ImportPrivkey", "Error", err.Error())
		return err
	}
	resp, err := network.client.WaitTimeout(msg, time.Minute)
	if err != nil {
		if err == types.ErrPrivkeyExist {
			return nil

		}
		if err == types.ErrLabelHasUsed {
			//切换随机lable
			parm.Label = fmt.Sprintf("node award %v", P2pComm.RandStr(3))
			time.Sleep(time.Second)
			goto ReTry
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

	go network.showTaskCapcity()
	go func() {
		defer func() {
			close(network.otherFactory)
			close(network.txFactory)
		}()
		var taskIndex int64
		network.client.Sub("p2p")
		for msg := range network.client.Recv() {
			if network.isClose() {
				log.Debug("subP2pMsg", "loop", "done")
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
			case types.EventGetNetInfo:
				go network.p2pCli.GetNetInfo(msg, taskIndex)
			default:
				log.Warn("unknown msgtype", "msg", msg)
				msg.Reply(network.client.NewMessage("", msg.Ty, types.Reply{Msg: []byte("unknown msgtype")}))
				<-network.otherFactory
				continue
			}
		}
		log.Info("subP2pMsg", "loop", "close")

	}()

}
