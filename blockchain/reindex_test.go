package blockchain_test

import (
	"testing"

	"gitlab.33.cn/chain33/chain33/util"
	"gitlab.33.cn/chain33/chain33/util/testnode"
)

func TestReindex(t *testing.T) {
	node := testnode.New("", nil)
	//发送交易
	util.CreateCoinsTx(node.GetGenesisKey())
}
