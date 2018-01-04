package raft

import (
	"strings"

	"code.aliyun.com/chain33/chain33/types"
	"github.com/coreos/etcd/raft/raftpb"
	log "github.com/inconshreveable/log15"
)

var genesisAddr = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

var rlog = log.New("module", "raft")

var (
	isValidator bool = false
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
	// validator节点（客户端），此节点用来处理区块打包等工作，并把区块发往raft集群来做共识
	// 相对于raft集群来说，发送区块的是客户端，客户端不用考虑raft的leader节点是哪个
	// 就算发给follower也会自动移交给leader。

	// propose channel
	proposeC := make(chan *types.Block)
	confChangeC := make(chan raftpb.ConfChange)

	var b *RaftClient
	getSnapshot := func() ([]byte, error) { return b.getSnapshot() }

	// raft集群的建立,1. 初始化两条channel： propose channel用于客户端和raft底层交互, commit channel用于获取commit消息
	// 2. raft集群中的节点之间建立http连接
	commitC, errorC, snapshotterReady := NewRaft(int(cfg.NodeId), strings.Split(cfg.PeersURL, ","), getSnapshot, proposeC, confChangeC)

	// 监听commit channel,取block
	b = NewBlockstore(<-snapshotterReady, proposeC, commitC, errorC)

	return b
}
