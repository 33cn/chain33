package rpc_test

//only load all plugin and system
import (
	"testing"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/util/testnode"
)

func TestNewTicket(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	cfg.Consensus.Name = "ticket"
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	defer mock33.Close()
	mock33.WaitHeight(5)
}
