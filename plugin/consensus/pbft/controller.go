package pbft

import (
	"strings"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/queue"
	pb "gitlab.33.cn/chain33/chain33/types"
)

var plog = log.New("module", "Pbft")

func NewPbft(cfg *pb.Consensus, sub []byte) queue.Module {
	plog.Info("start to creat pbft node")
	if int(cfg.NodeId) == 0 || strings.Compare(cfg.PeersURL, "") == 0 {
		plog.Error("nodeid or ip error")
	}
	var c *PbftClient
	replyChan, requestChan, isPrimary := NewReplica(uint32(cfg.NodeId), cfg.PeersURL, cfg.ClientAddr)
	c = NewBlockstore(cfg, replyChan, requestChan, isPrimary)
	return c
}
