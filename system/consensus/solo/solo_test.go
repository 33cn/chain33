// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package solo

import (
	"testing"

	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"

	//加载系统内置store, 不要依赖plugin
	_ "github.com/33cn/chain33/system/dapp/init"
	_ "github.com/33cn/chain33/system/store/init"
)

// 执行： go test -cover
func TestSolo(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	txs := util.GenNoneTxs(mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		mock33.GetAPI().SendTx(txs[i])
	}
	mock33.WaitHeight(1)
	txs = util.GenNoneTxs(mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		mock33.GetAPI().SendTx(txs[i])
	}
	mock33.WaitHeight(2)
}
