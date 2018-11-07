package pbft

import (
	"strings"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/queue"
	pb "gitlab.33.cn/chain33/chain33/types"
)

var (
	plog             = log.New("module", "Pbft")
	genesis          string
	genesisBlockTime int64
	clientAddr       string
)

type subConfig struct {
	Genesis          string `json:"genesis"`
	GenesisBlockTime int64  `json:"genesisBlockTime"`
	NodeId           int64  `json:"nodeId"`
	PeersURL         string `json:"peersURL"`
	ClientAddr       string `json:"clientAddr"`
}

func NewPbft(cfg *pb.Consensus, sub []byte) queue.Module {
	plog.Info("start to creat pbft node")
	var subcfg subConfig
	if sub != nil {
		pb.MustDecode(sub, &subcfg)
	}

	if subcfg.Genesis != "" {
		genesis = subcfg.Genesis
	}
	if subcfg.GenesisBlockTime > 0 {
		genesisBlockTime = subcfg.GenesisBlockTime
	}
	if int(subcfg.NodeId) == 0 || strings.Compare(subcfg.PeersURL, "") == 0 || strings.Compare(subcfg.ClientAddr, "") == 0 {
		plog.Error("The nodeId, peersURL or clientAddr is empty!")
		return nil
	}
	clientAddr = subcfg.ClientAddr

	var c *PbftClient
	replyChan, requestChan, isPrimary := NewReplica(uint32(subcfg.NodeId), subcfg.PeersURL, subcfg.ClientAddr)
	c = NewBlockstore(cfg, replyChan, requestChan, isPrimary)
	return c
}
