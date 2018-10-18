package ticket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/init"
	_ "gitlab.33.cn/chain33/chain33/plugin/store/init"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/util/testnode"
)

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	log.SetLogLevel("info")
}

// 执行： go test -cover
func TestTicket(t *testing.T) {
	mock33 := testnode.New("testdata/chain33.test.toml", nil)
	err := mock33.WaitHeight(2)
	assert.Nil(t, err)
}
