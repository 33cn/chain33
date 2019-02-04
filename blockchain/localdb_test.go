package blockchain_test

import (
	"testing"

	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/util/testnode"
)

func TestLocalDB(t *testing.T) {
	log.SetLogLevel("crit")
	mock33 := testnode.New("", nil)
	defer mock33.Close()

	//测试localdb
	api := mock33.GetAPI()
	api.
}
