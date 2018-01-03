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

	// TODO:目前单节点测试用，需要改成从toml文件中读取配置
	urls := "http://127.0.0.1:9021"
	nodeId := 1
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
	commitC, errorC, snapshotterReady := NewRaft(nodeId, strings.Split(urls, ","), getSnapshot, proposeC, confChangeC)

	// 监听commit channel,取block
	b = NewBlockstore(<-snapshotterReady, proposeC, commitC, errorC)

	return b
}
