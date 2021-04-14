package blockchain

import (
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func TestCheckClockDrift(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	q := queue.New("channel")
	q.SetConfig(cfg)

	blockchain := &BlockChain{}
	blockchain.client = q.Client()
	blockchain.checkClockDrift()

	cfg.GetModuleConfig().NtpHosts = append(cfg.GetModuleConfig().NtpHosts, types.NtpHosts...)
	blockchain.checkClockDrift()
}
