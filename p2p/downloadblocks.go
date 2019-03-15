// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"container/list"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// DownloadJob defines download job type
type DownloadJob struct {
	wg            sync.WaitGroup
	retryList     *list.List
	p2pcli        *Cli
	canceljob     int32
	mtx           sync.Mutex
	busyPeer      map[string]*peerJob
	downloadPeers []*Peer
}

type peerJob struct {
	//pbPeer *pb.Peer
	limit int32
}

// NewDownloadJob create a downloadjob object
func NewDownloadJob(p2pcli *Cli, peers []*Peer) *DownloadJob {
	job := new(DownloadJob)
	job.retryList = list.New()
	job.p2pcli = p2pcli
	job.busyPeer = make(map[string]*peerJob)
	job.downloadPeers = peers
	return job
}

func (d *DownloadJob) isBusyPeer(pid string) bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if pjob, ok := d.busyPeer[pid]; ok {
		return atomic.LoadInt32(&pjob.limit) >= 1 //每个节点最多同时接受10个下载任务
	}
	return false
}

func (d *DownloadJob) setBusyPeer(pid string) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if pjob, ok := d.busyPeer[pid]; ok {
		atomic.AddInt32(&pjob.limit, 1)
		d.busyPeer[pid] = pjob
		return
	}

	d.busyPeer[pid] = &peerJob{1}
}

func (d *DownloadJob) RemovePeer(pid string) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	for i, pr := range d.downloadPeers {
		if pr.GetPeerName() == pid {
			if i != len(d.downloadPeers)-1 {
				d.downloadPeers = append(d.downloadPeers[:i], d.downloadPeers[i+1:]...)
				return
			}

			d.downloadPeers = d.downloadPeers[:i]
			return
		}
	}
}

func (d *DownloadJob) AvalidPeersNum() int {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return len(d.downloadPeers)
}
func (d *DownloadJob) setFreePeer(pid string) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if pjob, ok := d.busyPeer[pid]; ok {
		if atomic.AddInt32(&pjob.limit, -1) <= 0 {
			delete(d.busyPeer, pid)
			return
		}
		d.busyPeer[pid] = pjob
	}
}

// GetFreePeer get free peer ,return peer
func (d *DownloadJob) GetFreePeer(joblimit int64) *Peer {
	_, infos := d.p2pcli.network.node.GetActivePeers()
	for _, peer := range d.downloadPeers {
		pbpeer, ok := infos[peer.Addr()]
		if ok {
			if len(peer.GetPeerName()) == 0 {
				peer.SetPeerName(pbpeer.GetName())
			}

			if pbpeer.GetHeader().GetHeight() >= joblimit {
				if d.isBusyPeer(pbpeer.GetName()) {
					continue
				}
				d.setBusyPeer(pbpeer.GetName())
				return peer
			}
		}
	}

	return nil
}

// CancelJob cancel the downloadjob object
func (d *DownloadJob) CancelJob() {
	atomic.StoreInt32(&d.canceljob, 1)
}

func (d *DownloadJob) IsCancel() bool {
	return atomic.LoadInt32(&d.canceljob) == 1
}

// DownloadBlock download the block
func (d *DownloadJob) DownloadBlock(invs []*pb.Inventory,
	bchan chan *pb.BlockPid) []*pb.Inventory {
	var errinvs []*pb.Inventory
	if d.IsCancel() {
		return errinvs
	}

	for _, inv := range invs { //让一个节点一次下载一个区块，下载失败区块，交给下一轮下载
		freePeer := d.GetFreePeer(inv.GetHeight())
		if freePeer == nil {
			//log.Error("DownloadBlock", "freepeer is null", inv.GetHeight())
			d.retryList.PushBack(inv)
			continue
		}

		d.wg.Add(1)
		go func(peer *Peer, inv *pb.Inventory) {
			defer d.wg.Done()
			err := d.syncDownloadBlock(peer, inv, bchan)
			if err != nil {
				d.RemovePeer(peer.GetPeerName())
				d.retryList.PushBack(inv) //失败的下载，放在下一轮ReDownload进行下载
			} else {
				d.setFreePeer(peer.GetPeerName())
			}

		}(freePeer, inv)

	}

	return d.restOfInvs(bchan)
}

func (d *DownloadJob) restOfInvs(bchan chan *pb.BlockPid) []*pb.Inventory {
	var errinvs []*pb.Inventory
	if d.IsCancel() {
		return errinvs
	}

	d.wg.Wait()
	if d.retryList.Len() == 0 {
		return errinvs
	}

	var invs []*pb.Inventory
	for e := d.retryList.Front(); e != nil; {
		if e.Value == nil {
			continue
		}
		log.Debug("resetofInvs", "inv", e.Value.(*pb.Inventory).GetHeight())
		invs = append(invs, e.Value.(*pb.Inventory)) //把下载遗漏的区块，重新组合进行下载
		next := e.Next()
		d.retryList.Remove(e)
		e = next
	}

	return invs
}

func (d *DownloadJob) syncDownloadBlock(peer *Peer, inv *pb.Inventory, bchan chan *pb.BlockPid) error {
	//每次下载一个高度的数据，通过bchan返回上层
	if peer == nil {
		return fmt.Errorf("peer is not exist")
	}

	if !peer.GetRunning() {
		return fmt.Errorf("peer not running")
	}
	var p2pdata pb.P2PGetData
	p2pdata.Version = d.p2pcli.network.node.nodeInfo.cfg.Version
	p2pdata.Invs = []*pb.Inventory{inv}
	resp, err := peer.mconn.gcli.GetData(context.Background(), &p2pdata, grpc.FailFast(true))
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		log.Error("syncDownloadBlock", "GetData err", err.Error())
		return err
	}
	defer resp.CloseSend()
	for {
		invdatas, err := resp.Recv()
		if err != nil {
			if err == io.EOF {
				log.Debug("download", "from", peer.Addr(), "block", inv.GetHeight())
				return nil
			}
			log.Error("download", "resp,Recv err", err.Error(), "download from", peer.Addr())
			return err
		}
		for _, item := range invdatas.Items {
			bchan <- &pb.BlockPid{Pid: peer.GetPeerName(), Block: item.GetBlock()} //下载完成后插入bchan

		}
	}
}
