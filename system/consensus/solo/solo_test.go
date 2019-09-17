// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package solo

import (
	"sync"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"

	//加载系统内置store, 不要依赖plugin
	_ "github.com/33cn/chain33/system/dapp/init"
	_ "github.com/33cn/chain33/system/mempool/init"
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

func BenchmarkSolo(b *testing.B) {
	cfg, subcfg := testnode.GetDefaultConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 1000)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	mock33 := testnode.NewWithConfig(cfg, subcfg, nil)
	defer mock33.Close()
	txs := util.GenCoinsTxs(mock33.GetGenesisKey(), int64(b.N))
	var last []byte
	var mu sync.Mutex
	b.ResetTimer()
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			for n := index; n < b.N; n += 10 {
				reply, err := mock33.GetAPI().SendTx(txs[n])
				if err != nil {
					assert.Nil(b, err)
				}
				mu.Lock()
				last = reply.GetMsg()
				mu.Unlock()
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	mock33.WaitTx(last)
}
