// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"container/list"
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	//"time"

	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Invs datastruct
type Invs []*pb.Inventory

//Len size of the Invs data
func (i Invs) Len() int {
	return len(i)
}

//Less Sort from low to high
func (i Invs) Less(a, b int) bool {
	return i[a].GetHeight() < i[b].GetHeight()
}

//Swap  the param
func (i Invs) Swap(a, b int) {
	i[a], i[b] = i[b], i[a]
}

// DownloadJob defines download job type
type DownloadJob struct {
	wg            sync.WaitGroup
	retryList     *list.List
	p2pcli        *Cli
	canceljob     int32
	mtx           sync.Mutex
	busyPeer      map[string]*peerJob
	downloadPeers []*Peer
	MaxJob        int32
}

type peerJob struct {
	limit int32
}

// NewDownloadJob create a downloadjob object
func NewDownloadJob(p2pcli *Cli, peers []*Peer) *DownloadJob {
	job := new(DownloadJob)
	job.retryList = list.New()
	job.p2pcli = p2pcli
	job.busyPeer = make(map[string]*peerJob)
	job.downloadPeers = peers

	job.MaxJob = 5
	if len(peers) < 5 {
		job.MaxJob = 10
	}
	//job.okChan = make(chan *pb.Inventory, 512)
	return job
}

func (d *DownloadJob) isBusyPeer(pid string) bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if pjob, ok := d.busyPeer[pid]; ok {
		return atomic.LoadInt32(&pjob.limit) >= d.MaxJob //每个节点最多同时接受10个下载任务
	}
	return false
}

func (d *DownloadJob) getJobNum(pid string) int32 {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if pjob, ok := d.busyPeer[pid]; ok {
		return atomic.LoadInt32(&pjob.limit)
	}

	return 0
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

func (d *DownloadJob) removePeer(pid string) {
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

// ResetDownloadPeers reset download peers
func (d *DownloadJob) ResetDownloadPeers(peers []*Peer) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	copy(d.downloadPeers, peers)
}

func (d *DownloadJob) avalidPeersNum() int {
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
func (d *DownloadJob) GetFreePeer(blockHeight int64) *Peer {
	_, infos := d.p2pcli.network.node.GetActivePeers()
	var jobNum int32 = 10
	var bestPeer *Peer
	for _, peer := range d.downloadPeers {
		pbpeer, ok := infos[peer.Addr()]
		if ok {
			if len(peer.GetPeerName()) == 0 {
				peer.SetPeerName(pbpeer.GetName())
			}

			if pbpeer.GetHeader().GetHeight() >= blockHeight {
				if d.isBusyPeer(pbpeer.GetName()) {
					continue
				}
				peerJopNum := d.getJobNum(pbpeer.GetName())
				if jobNum > peerJopNum {
					jobNum = peerJopNum
					bestPeer = peer
				}
			}
		}
	}

	if bestPeer != nil {
		d.setBusyPeer(bestPeer.GetPeerName())

	}
	return bestPeer
}

// CancelJob cancel the downloadjob object
func (d *DownloadJob) CancelJob() {
	atomic.StoreInt32(&d.canceljob, 1)
}

func (d *DownloadJob) isCancel() bool {
	return atomic.LoadInt32(&d.canceljob) == 1
}

// DownloadBlock download the block
func (d *DownloadJob) DownloadBlock(invs []*pb.Inventory,
	bchan chan *pb.BlockPid) []*pb.Inventory {
	var errinvs []*pb.Inventory
	if d.isCancel() {
		return errinvs
	}

	for _, inv := range invs { //让一个节点一次下载一个区块，下载失败区块，交给下一轮下载
	REGET:
		freePeer := d.GetFreePeer(inv.GetHeight()) //获取当前任务数最少的节点，相当于 下载速度最快的节点
		if freePeer == nil {
			time.Sleep(time.Millisecond * 100)
			goto REGET
		}

		d.wg.Add(1)
		go func(peer *Peer, inv *pb.Inventory) {
			defer d.wg.Done()
			err := d.syncDownloadBlock(peer, inv, bchan)
			if err != nil {
				d.removePeer(peer.GetPeerName())
				log.Error("DownloadBlock:syncDownloadBlock", "height", inv.GetHeight(), "peer", peer.GetPeerName(), "err", err)
				d.retryList.PushFront(inv) //失败的下载，放在下一轮ReDownload进行下载

			} else {
				d.setFreePeer(peer.GetPeerName())
			}

		}(freePeer, inv)

	}

	return d.restOfInvs(bchan)
}

func (d *DownloadJob) restOfInvs(bchan chan *pb.BlockPid) []*pb.Inventory {
	var errinvs []*pb.Inventory
	if d.isCancel() {
		return errinvs
	}

	d.wg.Wait()
	if d.retryList.Len() == 0 {
		return errinvs
	}

	var invsArr Invs
	for e := d.retryList.Front(); e != nil; {
		if e.Value == nil {
			continue
		}
		log.Debug("resetofInvs", "inv", e.Value.(*pb.Inventory).GetHeight())
		invsArr = append(invsArr, e.Value.(*pb.Inventory)) //把下载遗漏的区块，重新组合进行下载
		next := e.Next()
		d.retryList.Remove(e)
		e = next
	}
	//Sort
	sort.Sort(invsArr)
	//log.Info("resetOfInvs", "sorted:", invs)
	return invsArr
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
	beg := pb.Now()
	resp, err := peer.mconn.gcli.GetData(context.Background(), &p2pdata, grpc.FailFast(true))
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		log.Error("syncDownloadBlock", "GetData err", err.Error())
		return err
	}
	defer func() {
		log.Debug("download", "frompeer", peer.Addr(), "blockheight", inv.GetHeight(), "downloadcost", pb.Since(beg))
	}()
	defer resp.CloseSend()
	for {
		invdatas, err := resp.Recv()
		if err != nil {
			if err == io.EOF {
				if invdatas == nil {
					return nil
				}
				goto RECV
			}
			log.Error("download", "resp,Recv err", err.Error(), "download from", peer.Addr())
			return err
		}
	RECV:
		for _, item := range invdatas.Items {
			bchan <- &pb.BlockPid{Pid: peer.GetPeerName(), Block: item.GetBlock()} //下载完成后插入bchan
			log.Debug("download", "frompeer", peer.Addr(), "blockheight", inv.GetHeight(), "Blocksize", item.GetBlock().Size())
		}
	}
}
