package raft

import (
	"strings"

	"github.com/coreos/etcd/raft/raftpb"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var genesisAddr = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

var rlog = log.New("module", "raft")

var (
	defaultSnapCount        uint64 = 1000
	snapshotCatchUpEntriesN uint64 = 1000
	isLeader                bool   = false
	confChangeC             chan raftpb.ConfChange
)

func NewRaftCluster(cfg *types.Consensus) *RaftClient {
	if cfg.Genesis != "" {
		genesisAddr = cfg.Genesis
	}
	rlog.Info("Start to create raft cluster")

	if int(cfg.NodeId) == 0 || strings.Compare(cfg.PeersURL, "") == 0 {
		rlog.Error("Please check whether the configuration of nodeId and peersURL is empty!")
		//TODO 当传入的参数异常时，返回给主函数的是个nil,这时候需要做异常处理
		return nil
	}
	// 默认1000个Entry打一个snapshot
	if cfg.DefaultSnapCount <= 0 {
		defaultSnapCount = 1000
		snapshotCatchUpEntriesN = 1000
	} else {
		defaultSnapCount = uint64(cfg.DefaultSnapCount)
		snapshotCatchUpEntriesN = uint64(cfg.DefaultSnapCount)
	}

	// propose channel
	proposeC := make(chan *types.Block)
	confChangeC = make(chan raftpb.ConfChange)

	var b *RaftClient
	getSnapshot := func() ([]byte, error) { return b.getSnapshot() }
	// raft集群的建立,1. 初始化两条channel： propose channel用于客户端和raft底层交互, commit channel用于获取commit消息
	// 2. raft集群中的节点之间建立http连接
	peers := strings.Split(cfg.PeersURL, ",")
	if len(peers) == 1 && peers[0] == "" {
		peers = []string{}
	}
	readOnlyPeers := strings.Split(cfg.ReadOnlyPeersURL, ",")
	if len(readOnlyPeers) == 1 && readOnlyPeers[0] == "" {
		readOnlyPeers = []string{}
	}
	addPeers := strings.Split(cfg.AddPeersURL, ",")
	if len(addPeers) == 1 && addPeers[0] == "" {
		addPeers = []string{}
	}
	commitC, errorC, snapshotterReady, validatorC := NewRaftNode(int(cfg.NodeId), cfg.IsNewJoinNode, peers, readOnlyPeers, addPeers, getSnapshot, proposeC, confChangeC)
	//启动raft删除节点操作监听
	go serveHttpRaftAPI(int(cfg.RaftApiPort), confChangeC, errorC)
	// 监听commit channel,取block
	b = NewBlockstore(cfg, <-snapshotterReady, proposeC, commitC, errorC, validatorC)
	return b
}
