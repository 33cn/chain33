// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package p2p 实现了chain33网络协议
package p2p

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/client"
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
	api          client.QueueProtocolAPI
	client       queue.Client
	node         *Node
	p2pCli       EventInterface
	txCapcity    int32
	txFactory    chan struct{}
	otherFactory chan struct{}
	waitRestart  chan struct{}

	closed  int32
	restart int32
	cfg     *types.P2P
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
	p2p.waitRestart = make(chan struct{}, 1)
	p2p.txCapcity = 1000
	p2p.cfg = cfg
	return p2p
}

//Wait wait for ready
func (network *P2p) Wait() {}

func (network *P2p) isClose() bool {
	return atomic.LoadInt32(&network.closed) == 1
}

func (network *P2p) isRestart() bool {
	return atomic.LoadInt32(&network.restart) == 1
}

// Close network client
func (network *P2p) Close() {
	log.Info("p2p network start shutdown")
	atomic.StoreInt32(&network.closed, 1)
	network.node.Close()
	if network.client != nil {
		if !network.isRestart() {
			network.client.Close()

		}

	}

	network.node.pubsub.Shutdown()

}

// SetQueueClient set the queue
func (network *P2p) SetQueueClient(cli queue.Client) {
	var err error
	if network.client == nil {
		network.client = cli

	}
	network.api, err = client.New(cli, nil)
	if err != nil {
		panic("SetQueueClient client.New err")
	}
	network.node.SetQueueClient(cli)

	go func(p2p *P2p) {

		if p2p.isRestart() {
			p2p.node.Start()
			atomic.StoreInt32(&p2p.closed, 0)
			atomic.StoreInt32(&p2p.restart, 0)
			network.waitRestart <- struct{}{}
			return
		}

		p2p.subP2pMsg()
		key, pub := p2p.node.nodeInfo.addrBook.GetPrivPubKey()
		log.Debug("key pub:", pub, "")
		if key == "" {
			if p2p.cfg.WaitPid { //key为空，则为初始钱包，阻塞模式，一直等到钱包导入助记词，解锁
				if p2p.genAirDropKeyFromWallet() != nil {
					return
				}
			} else {
				//创建随机Pid,会同时出现node award ,airdropaddr
				p2p.node.nodeInfo.addrBook.ResetPeerkey(key, pub)
				go p2p.genAirDropKeyFromWallet()
			}

		} else {
			//key 有两种可能，老版本的随机key,也有可能是seed的key, 非阻塞模式
			go p2p.genAirDropKeyFromWallet()

		}
		p2p.node.Start()
		log.Debug("SetQueueClient gorountine ret")

	}(network)

}

func (network *P2p) loadP2PPrivKeyToWallet() error {

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

func (network *P2p) genAirDropKeyFromWallet() error {
	_, savePub := network.node.nodeInfo.addrBook.GetPrivPubKey()
	for {
		if network.isClose() {
			log.Error("genAirDropKeyFromWallet", "p2p closed", "")
			return fmt.Errorf("p2p closed")
		}
		msg := network.client.NewMessage("wallet", types.EventGetWalletStatus, nil)
		err := network.client.SendTimeout(msg, true, time.Minute)
		if err != nil {
			log.Error("genAirDropKeyFromWallet", "Error", err.Error())
			time.Sleep(time.Second)
			continue
		}

		resp, err := network.client.WaitTimeout(msg, time.Minute)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if resp.GetData().(*types.WalletStatus).GetIsWalletLock() { //上锁
			if savePub == "" {
				log.Warn("P2P Stuck ! Wallet must be unlock and save with mnemonics")

			}
			time.Sleep(time.Second)
			continue
		}

		if !resp.GetData().(*types.WalletStatus).GetIsHasSeed() { //无种子
			if savePub == "" {
				log.Warn("P2P Stuck ! Wallet must be imported with mnemonics")

			}
			time.Sleep(time.Second * 5)
			continue
		}

		break
	}

	r := rand.New(rand.NewSource(types.Now().Unix()))
	var minIndex int32 = 100000000
	randIndex := minIndex + r.Int31n(1000000)
	reqIndex := &types.Int32{Data: randIndex}
	msg, err := network.api.ExecWalletFunc("wallet", "NewAccountByIndex", reqIndex)
	if err != nil {
		log.Error("genAirDropKeyFromWallet", "err", err)
		return err
	}
	var hexPrivkey string
	if reply, ok := msg.(*types.ReplyString); !ok {
		log.Error("genAirDropKeyFromWallet", "wrong format data", "")
		panic(err)

	} else {
		hexPrivkey = reply.GetData()
	}
	if hexPrivkey[:2] == "0x" {
		hexPrivkey = hexPrivkey[2:]
	}

	hexPubkey, err := P2pComm.Pubkey(hexPrivkey)
	if err != nil {
		log.Error("genAirDropKeyFromWallet", "gen pub error", err)
		panic(err)
	}

	log.Info("genAirDropKeyFromWallet", "pubkey", hexPubkey)

	if savePub == hexPubkey {
		return nil
	}

	if savePub != "" {
		//priv,pub是随机公私钥对，兼容老版本，先对其进行导入钱包处理
		err = network.loadP2PPrivKeyToWallet()
		if err != nil {
			log.Error("genAirDropKeyFromWallet", "loadP2PPrivKeyToWallet error", err)
			panic(err)
		}
		network.node.nodeInfo.addrBook.ResetPeerkey(hexPrivkey, hexPubkey)
		//重启p2p模块
		log.Info("genAirDropKeyFromWallet", "p2p will Restart....")
		network.ReStart()
		return nil
	}
	//覆盖addrbook 中的公私钥对
	network.node.nodeInfo.addrBook.ResetPeerkey(hexPrivkey, hexPubkey)

	return nil
}

//ReStart p2p
func (network *P2p) ReStart() {
	atomic.StoreInt32(&network.restart, 1)
	network.Close()
	node, err := NewNode(network.cfg) //创建新的node节点
	if err != nil {
		panic(err.Error())
	}

	network.node = node
	network.SetQueueClient(network.client)

}

func (network *P2p) subP2pMsg() {
	if network.client == nil {
		return
	}

	go network.showTaskCapcity()
	go func() {

		var taskIndex int64
		network.client.Sub("p2p")
		for msg := range network.client.Recv() {

			if network.isRestart() {

				//wait for restart
				log.Info("waitp2p restart....")
				<-network.waitRestart
				log.Info("p2p restart ok....")
			}

			if network.isClose() {
				log.Debug("subP2pMsg", "loop", "done")
				close(network.otherFactory)
				close(network.txFactory)
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
