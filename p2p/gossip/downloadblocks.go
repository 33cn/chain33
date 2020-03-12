// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gossip

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

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
	p2pcli        *Cli
	canceljob     int32
	mtx           sync.Mutex
	busyPeer      map[string]*peerJob
	downloadPeers []*Peer
	MaxJob        int32
	retryItems    Invs
}

type peerJob struct {
	limit int32
}

// NewDownloadJob create a downloadjob object
func NewDownloadJob(p2pcli *Cli, peers []*Peer) *DownloadJob {
	job := new(DownloadJob)
	job.p2pcli = p2pcli
	job.busyPeer = make(map[string]*peerJob)
	job.downloadPeers = peers
	job.retryItems = make([]*pb.Inventory, 0)
	job.MaxJob = 5
	if len(peers) < 5 {
		job.MaxJob = 10
	}
	//job.okChan = make(chan *pb.Inventory, 512)
	return job
}

func (d *DownloadJob) getDownloadPeers() []*Peer {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	peers := make([]*Peer, len(d.downloadPeers))
	copy(peers, d.downloadPeers)
	return peers
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

//加入到重试数组
func (d *DownloadJob) appendRetryItem(item *pb.Inventory) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.retryItems = append(d.retryItems, item)
}

// GetFreePeer get free peer ,return peer
func (d *DownloadJob) GetFreePeer(blockHeight int64) *Peer {
	infos := d.p2pcli.network.node.nodeInfo.peerInfos.GetPeerInfos()
	var minJobNum int32 = 10
	var bestPeer *Peer
	//对download peer读取需要增加保护
	for _, peer := range d.getDownloadPeers() {

		peerName := peer.GetPeerName()
		if d.isBusyPeer(peerName) {
			continue
		}

		if jobNum := d.getJobNum(peerName); jobNum < minJobNum &&
			infos[peerName].GetHeader().GetHeight() >= blockHeight {
			minJobNum = jobNum
			bestPeer = peer
		}
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
	if d.isCancel() {
		return nil
	}

	for _, inv := range invs { //让一个节点一次下载一个区块，下载失败区块，交给下一轮下载

		//获取当前任务数最少的节点，相当于 下载速度最快的节点
		freePeer := d.GetFreePeer(inv.GetHeight())
		for freePeer == nil {
			log.Debug("no free peer")
			time.Sleep(time.Millisecond * 100)
			freePeer = d.GetFreePeer(inv.GetHeight())
		}
		d.setBusyPeer(freePeer.GetPeerName())
		d.wg.Add(1)
		go func(peer *Peer, inv *pb.Inventory) {
			defer d.wg.Done()
			err := d.syncDownloadBlock(peer, inv, bchan)
			if err != nil {
				d.removePeer(peer.GetPeerName())
				log.Error("DownloadBlock:syncDownloadBlock", "height", inv.GetHeight(), "peer", peer.GetPeerName(), "err", err)
				d.appendRetryItem(inv) //失败的下载，放在下一轮ReDownload进行下载

			} else {
				d.setFreePeer(peer.GetPeerName())
			}

		}(freePeer, inv)

	}

	//等待下载任务
	d.wg.Wait()
	retryInvs := d.retryItems
	//存在重试项
	if retryInvs.Len() > 0 {
		d.retryItems = make([]*pb.Inventory, 0)
		sort.Sort(retryInvs)
	}
	return retryInvs
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
	p2pdata.Version = d.p2pcli.network.node.nodeInfo.channelVersion
	p2pdata.Invs = []*pb.Inventory{inv}
	ctx, cancel := context.WithCancel(context.Background())
	//主动取消grpc流, 即时释放资源
	defer cancel()
	beg := pb.Now()
	resp, err := peer.mconn.gcli.GetData(ctx, &p2pdata, grpc.FailFast(true))
	P2pComm.CollectPeerStat(err, peer)
	if err != nil {
		log.Error("syncDownloadBlock", "GetData err", err.Error())
		return err
	}
	defer func() {
		log.Debug("download", "frompeer", peer.Addr(), "blockheight", inv.GetHeight(), "downloadcost", pb.Since(beg))
	}()

	invData, err := resp.Recv()
	if err != nil && err != io.EOF {
		log.Error("syncDownloadBlock", "RecvData err", err.Error())
		return err
	}
	//返回单个数据条目
	if invData == nil || len(invData.Items) != 1 {
		return fmt.Errorf("InvalidRecvData")
	}

	block := invData.Items[0].GetBlock()
	log.Debug("download", "frompeer", peer.Addr(), "blockheight", inv.GetHeight(), "blockSize", block.Size())
	bchan <- &pb.BlockPid{Pid: peer.GetPeerName(), Block: block} //加入到输出通道
	return nil
}
