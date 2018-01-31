package raft

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"context"

	"code.aliyun.com/chain33/chain33/types"
	"github.com/coreos/etcd/etcdserver/stats"
	"github.com/coreos/etcd/pkg/fileutil"
	typec "github.com/coreos/etcd/pkg/types"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/rafthttp"
	"github.com/coreos/etcd/snap"
	"github.com/coreos/etcd/wal"
	"github.com/coreos/etcd/wal/walpb"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	//"github.com/pkg/errors"
)

//var once sync.Once

const (
	IsLeader   = "0"
	LeaderIsOK = "1"
)

var (
	//wasLeader               bool   = false
	defaultSnapCount        uint64 = 10000
	snapshotCatchUpEntriesN uint64 = 10000
	leaderCache             uint64
)

type raftNode struct {
	proposeC         <-chan *types.Block
	confChangeC      <-chan raftpb.ConfChange
	commitC          chan<- *types.Block
	errorC           chan<- error
	id               int
	peers            []string
	join             bool
	waldir           string
	snapdir          string
	getSnapshot      func() ([]byte, error)
	lastIndex        uint64
	confState        raftpb.ConfState
	snapshotIndex    uint64
	appliedIndex     uint64
	node             raft.Node
	raftStorage      *raft.MemoryStorage
	wal              *wal.WAL
	snapshotter      *snap.Snapshotter
	snapshotterReady chan *snap.Snapshotter
	snapCount        uint64
	transport        *rafthttp.Transport
	stopMu           sync.RWMutex
	stopc            chan struct{}
	httpstopc        chan struct{}
	httpdonec        chan struct{}
	//备用的leaderC用于watch leader节点的变更，暂时没用
	//leaderC    chan int
	validatorC chan map[string]bool
}

func NewRaftNode(id int, peers []string, getSnapshot func() ([]byte, error), proposeC <-chan *types.Block,
	confChangeC <-chan raftpb.ConfChange) (<-chan *types.Block, <-chan error, <-chan *snap.Snapshotter, <-chan map[string]bool) {

	log.Info("Enter consensus raft")
	// commit channel
	commitC := make(chan *types.Block)
	errorC := make(chan error)
	storage := raft.NewMemoryStorage()
	rc := &raftNode{
		proposeC:    proposeC,
		confChangeC: confChangeC,
		commitC:     commitC,
		errorC:      errorC,
		id:          id,
		peers:       peers,
		waldir:      fmt.Sprintf("chain33_raft-%d", id),
		snapdir:     fmt.Sprintf("chain33_raft-%d-snap", id),
		getSnapshot: getSnapshot,
		snapCount:   defaultSnapCount,
		stopc:       make(chan struct{}),
		httpstopc:   make(chan struct{}),
		httpdonec:   make(chan struct{}),
		raftStorage: storage,
		//leaderC: make(chan int),
		validatorC:       make(chan map[string]bool),
		snapshotterReady: make(chan *snap.Snapshotter, 1),
	}
	go rc.startRaft()

	return commitC, errorC, rc.snapshotterReady, rc.validatorC
}

//  启动raft节点
func (rc *raftNode) startRaft() {
	// 有snapshot就打开，没有则创建
	if !fileutil.Exist(rc.snapdir) {
		if err := os.Mkdir(rc.snapdir, 0750); err != nil {
			log.Error("chain33_raft: cannot create dir for snapshot (%v)", err)
		}
	}

	rc.snapshotter = snap.New(rc.snapdir)
	rc.snapshotterReady <- rc.snapshotter

	oldwal := wal.Exist(rc.waldir)
	rc.wal = rc.replayWAL()

	rpeers := make([]raft.Peer, len(rc.peers))
	for i := range rpeers {
		rpeers[i] = raft.Peer{ID: uint64(i + 1)}
	}
	c := &raft.Config{
		ID:              uint64(rc.id),
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         rc.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
		//设置成预投票，当节点重连时，可以快速地重新加入集群
		PreVote:     true,
		CheckQuorum: false,
	}

	if oldwal {
		rc.node = raft.RestartNode(c)
	} else {
		startPeers := rpeers
		if rc.join {
			startPeers = nil
		}
		rc.node = raft.StartNode(c, startPeers)
	}

	rc.transport = &rafthttp.Transport{
		ID:          typec.ID(rc.id),
		ClusterID:   0x1000,
		Raft:        rc,
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(strconv.Itoa(rc.id)),
		ErrorC:      make(chan error),
	}

	rc.transport.Start()
	for i := range rc.peers {
		if i+1 != rc.id {
			rc.transport.AddPeer(typec.ID(i+1), []string{rc.peers[i]})
		}
	}

	// 启动网络监听
	go rc.serveRaft()
	go rc.serveChannels()

	//定时轮询watch leader 状态是否改变，更新validator
	go rc.updateValidator()
}

// 网络监听
func (rc *raftNode) serveRaft() {
	nodeURL, err := url.Parse(rc.peers[rc.id-1])
	if err != nil {
		log.Error("raft: Failed parsing URL (%v)", err)
	}

	ln, err := newStoppableListener(nodeURL.Host, rc.httpstopc)
	if err != nil {
		log.Error("raft: Failed to listen rafthttp (%v)", err)
	}

	err = (&http.Server{Handler: rc.transport.Handler()}).Serve(ln)
	select {
	case <-rc.httpstopc:
	default:
		log.Error("raft: Failed to serve rafthttp (%v)", err)
	}
	close(rc.httpdonec)
}

func (rc *raftNode) serveChannels() {
	snapShot, err := rc.raftStorage.Snapshot()
	if err != nil {
		panic(err)
	}
	rc.confState = snapShot.Metadata.ConfState
	rc.snapshotIndex = snapShot.Metadata.Index
	rc.appliedIndex = snapShot.Metadata.Index

	defer rc.wal.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		var confChangeCount uint64 = 0
		// 通过propose和proposeConfchange方法往RaftNode发通知
		for rc.proposeC != nil && rc.confChangeC != nil {
			select {
			case prop, ok := <-rc.proposeC:
				if !ok {
					rc.proposeC = nil
				} else {
					//fmt.Println(prop.String())
					out, err := proto.Marshal(prop)
					if err != nil {
						log.Error("failed to marshal block: ", err)
					}
					rc.node.Propose(context.TODO(), out)
				}

			case cc, ok := <-rc.confChangeC:
				if !ok {
					rc.confChangeC = nil
				} else {
					confChangeCount += 1
					cc.ID = confChangeCount
					rc.node.ProposeConfChange(context.TODO(), cc)
				}
			}
		}
		close(rc.stopc)
	}()

	// 从Ready()中接收数据
	for {
		select {
		case <-ticker.C:
			rc.node.Tick()
		case rd := <-rc.node.Ready():
			rc.wal.Save(rd.HardState, rd.Entries)
			if !raft.IsEmptySnap(rd.Snapshot) {
				rc.saveSnap(rd.Snapshot)
				rc.raftStorage.ApplySnapshot(rd.Snapshot)
				rc.publishSnapshot(rd.Snapshot)
			}
			rc.raftStorage.Append(rd.Entries)
			rc.transport.Send(rd.Messages)
			if ok := rc.publishEntries(rc.entriesToApply(rd.CommittedEntries)); !ok {
				rc.stop()
				return
			}
			rc.maybeTriggerSnapshot()

			rc.node.Advance()
		case err := <-rc.transport.ErrorC:
			rc.writeError(err)
			return

		case <-rc.stopc:
			rc.stop()
			return
		}
	}
}

func (rc *raftNode) updateValidator() {
	var validatorMap map[string]bool
	// TODO: Sould add tcp healthy check
	time.Sleep(15 * time.Second)
	for {
		validatorMap = make(map[string]bool)
		if rc.Leader() == raft.None {
			rlog.Info("==============Leader is not ready==============")
			validatorMap[LeaderIsOK] = false
		} else {

			// 获取到leader Id,选主成功
			validatorMap[LeaderIsOK] = true

			if rc.id == int(rc.Leader()) {
				validatorMap[IsLeader] = true
			} else {
				validatorMap[IsLeader] = false
			}
		}
		time.Sleep(time.Second)
		rc.validatorC <- validatorMap
	}
}

// isLeader checks if we are the leader or not, without the protection of lock
func (rc *raftNode) leader() uint64 {
	return rc.node.Status().Lead
}

// IsLeader checks if we are the leader or not, with the protection of lock
func (rc *raftNode) Leader() uint64 {
	rc.stopMu.RLock()
	defer rc.stopMu.RUnlock()
	return rc.leader()
}
func (rc *raftNode) replayWAL() *wal.WAL {
	log.Info("replaying WAL of member %d", rc.id)
	snapshot := rc.loadSnapshot()
	w := rc.openWAL(snapshot)
	_, st, ents, err := w.ReadAll()
	if err != nil {
		log.Error("chain33_raft: failed to read WAL (%v)", err)
	}
	rc.raftStorage = raft.NewMemoryStorage()
	if snapshot != nil {
		rc.raftStorage.ApplySnapshot(*snapshot)
	}
	rc.raftStorage.SetHardState(st)

	// append to storage so raft starts at the right place in log
	rc.raftStorage.Append(ents)
	// send nil once lastIndex is published so client knows commit channel is current
	if len(ents) > 0 {
		rc.lastIndex = ents[len(ents)-1].Index
	} else {
		rc.commitC <- nil
	}
	return w
}

func (rc *raftNode) loadSnapshot() *raftpb.Snapshot {
	snapshot, err := rc.snapshotter.Load()
	if err != nil && err != snap.ErrNoSnapshot {
		log.Error("chain33_raft: error loading snapshot (%v)", err)
	}
	return snapshot
}

func (rc *raftNode) saveSnap(snap raftpb.Snapshot) error {
	walSnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term:  snap.Metadata.Term,
	}
	if err := rc.wal.SaveSnapshot(walSnap); err != nil {
		return err
	}
	if err := rc.snapshotter.SaveSnap(snap); err != nil {
		return err
	}
	return rc.wal.ReleaseLockTo(snap.Metadata.Index)
}

func (rc *raftNode) maybeTriggerSnapshot() {
	if rc.appliedIndex-rc.snapshotIndex <= rc.snapCount {
		return
	}

	log.Info("start snapshot [applied index: %d | last snapshot index: %d]", rc.appliedIndex, rc.snapshotIndex)
	data, err := rc.getSnapshot()
	if err != nil {
		log.Error("Err happened when get snapshot")
	}
	snapShot, err := rc.raftStorage.CreateSnapshot(rc.appliedIndex, &rc.confState, data)
	if err != nil {
		panic(err)
	}
	if err := rc.saveSnap(snapShot); err != nil {
		panic(err)
	}

	compactIndex := uint64(1)
	if rc.appliedIndex > snapshotCatchUpEntriesN {
		compactIndex = rc.appliedIndex - snapshotCatchUpEntriesN
	}
	if err := rc.raftStorage.Compact(compactIndex); err != nil {
		panic(err)
	}

	log.Info("compacted log at index %d", compactIndex)
	rc.snapshotIndex = rc.appliedIndex
}

func (rc *raftNode) publishSnapshot(snapshotToSave raftpb.Snapshot) {
	if raft.IsEmptySnap(snapshotToSave) {
		return
	}

	log.Info("publishing snapshot at index %d", rc.snapshotIndex)
	defer log.Info("finished publishing snapshot at index %d", rc.snapshotIndex)

	if snapshotToSave.Metadata.Index <= rc.appliedIndex {
		log.Error("snapshot index [%d] should > progress.appliedIndex [%d] + 1", snapshotToSave.Metadata.Index, rc.appliedIndex)
	}
	rc.commitC <- nil // trigger kvstore to load snapshot

	rc.confState = snapshotToSave.Metadata.ConfState
	rc.snapshotIndex = snapshotToSave.Metadata.Index
	rc.appliedIndex = snapshotToSave.Metadata.Index
}

func (rc *raftNode) openWAL(snapshot *raftpb.Snapshot) *wal.WAL {
	if !wal.Exist(rc.waldir) {
		if err := os.Mkdir(rc.waldir, 0750); err != nil {
			log.Error("chain33_raft: cannot create dir for wal (%v)", err)
		}

		w, err := wal.Create(rc.waldir, nil)
		if err != nil {
			log.Error("chain33_raft: create wal error (%v)", err)
		}
		w.Close()
	}

	walsnap := walpb.Snapshot{}
	if snapshot != nil {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}
	log.Info("loading WAL at term %d and index %d", walsnap.Term, walsnap.Index)
	w, err := wal.Open(rc.waldir, walsnap)
	if err != nil {
		log.Error("chain33_raft: error loading wal (%v)", err)
	}

	return w
}

// 关闭http连接和channel
func (rc *raftNode) stop() {
	rc.stopHTTP()
	close(rc.commitC)
	close(rc.errorC)
	//close(rc.validatorC)
	rc.node.Stop()
}

func (rc *raftNode) stopHTTP() {
	rc.transport.Stop()
	close(rc.httpstopc)
	<-rc.httpdonec
}

func (rc *raftNode) writeError(err error) {
	rc.stopHTTP()
	close(rc.commitC)
	rc.errorC <- err
	close(rc.errorC)
	//close(rc.validatorC)
	rc.node.Stop()
}

// 往commit channel中写入commit log
func (rc *raftNode) publishEntries(ents []raftpb.Entry) bool {
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				break
			}
			// 解码
			block := &types.Block{}
			if err := proto.Unmarshal(ents[i].Data, block); err != nil {
				log.Error("failed to unmarshal: ", err)
			}
			select {
			case rc.commitC <- block:
			case <-rc.stopc:
				return false
			}

		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			cc.Unmarshal(ents[i].Data)
			rc.confState = *rc.node.ApplyConfChange(cc)
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				if len(cc.Context) > 0 {
					rc.transport.AddPeer(typec.ID(cc.NodeID), []string{string(cc.Context)})
				}
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == uint64(rc.id) {
					log.Info("I've been removed from the cluster! Shutting down.")
					return false
				}
				rc.transport.RemovePeer(typec.ID(cc.NodeID))
			}
		}

		rc.appliedIndex = ents[i].Index

		if ents[i].Index == rc.lastIndex {
			select {
			case rc.commitC <- nil:
			case <-rc.stopc:
				return false
			}
		}
	}
	return true
}

func (rc *raftNode) entriesToApply(ents []raftpb.Entry) (nents []raftpb.Entry) {
	if len(ents) == 0 {
		return
	}
	fmt.Println(len(ents))
	firstIdx := ents[0].Index
	if firstIdx > rc.appliedIndex+1 {
		log.Error("first index of committed entry[%d] should <= progress.appliedIndex[%d] 1", firstIdx, rc.appliedIndex)
	}
	if rc.appliedIndex-firstIdx+1 < uint64(len(ents)) {
		nents = ents[rc.appliedIndex-firstIdx+1:]
	}
	return
}

func (rc *raftNode) Process(ctx context.Context, m raftpb.Message) error {
	return rc.node.Step(ctx, m)
}
func (rc *raftNode) IsIDRemoved(id uint64) bool                           { return false }
func (rc *raftNode) ReportUnreachable(id uint64)                          {}
func (rc *raftNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {}

// 等待集群中leader节点的选举结果，并返回leadId
//func (rc *raftNode) WaitForLeader() (uint64, error) {
//	leadId := rc.node.Status().Lead
//	if leadId != raft.None {
//		return leadId, nil
//	}
//	ticker := time.NewTicker(50 * time.Millisecond)
//	defer ticker.Stop()
//	for leadId == raft.None {
//		select {
//		case <-ticker.C:
//		}
//		leadId := rc.node.Status().Lead
//		log.Info("chain33_raft==========leaderId===========:"+strconv.FormatUint(leadId,10))
//		if leadId != raft.None {
//			log.Info("=====chain-33-raft has elected cluster leader====.")
//			return leadId, nil
//		}
//	}
//	return leadId, errors.New("raft: no elected cluster leader")
//}
