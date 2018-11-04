package ticket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/init"
	_ "gitlab.33.cn/chain33/chain33/plugin/store/init"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/util/testnode"
)

// 执行： go test -cover
func TestTicket(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	cfg.Consensus.Name = "ticket"
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	defer mock33.Close()
	err := mock33.WaitHeight(100)
	assert.Nil(t, err)
}
