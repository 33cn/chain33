// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package solo

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/33cn/chain33/queue"

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
	cfg := mock33.GetClient().GetConfig()
	txs := util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		mock33.GetAPI().SendTx(txs[i])
	}
	mock33.WaitHeight(1)
	txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		mock33.GetAPI().SendTx(txs[i])
	}
	mock33.WaitHeight(2)
}

func BenchmarkSolo(b *testing.B) {
	cfg := testnode.GetDefaultConfig()
	subcfg := cfg.GetSubConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 1000)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	mock33 := testnode.NewWithConfig(cfg, nil)
	defer mock33.Close()
	txs := util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), int64(b.N))
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

//mempool发送交易 10000tx/s
func BenchmarkSendTx(b *testing.B) {
	cfg := testnode.GetDefaultConfig()
	subcfg := cfg.GetSubConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 100)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	mock33 := testnode.NewWithConfig(cfg, nil)
	defer mock33.Close()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		amount := int64(1)
		addr, _ := util.Genaddress()
		for pb.Next() {
			tx := util.CreateCoinsTx(cfg, mock33.GetGenesisKey(), addr, amount)
			amount++
			mock33.GetAPI().SendTx(tx)
		}
	})
}

//测试每10000笔交易打包的并发时间  3500tx/s
func BenchmarkSoloNewBlock(b *testing.B) {
	cfg := testnode.GetDefaultConfig()
	subcfg := cfg.GetSubConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 100)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	mock33 := testnode.NewWithConfig(cfg, nil)
	defer mock33.Close()
	sendRoutine := 10
	startChan := make(chan struct{})
	txBenchCount := int64(10000)
	waitTxChan := make(chan []byte, 1)
	totalTx := int64(0)
	mutex := &sync.Mutex{}
	for i := 0; i < sendRoutine; i++ {

		go func() {
			addr, _ := util.Genaddress()
			startChan <- struct{}{}
			amount := int64(1)
			for {

				tx := util.CreateCoinsTx(cfg, mock33.GetGenesisKey(), addr, amount)
				reply, err := mock33.GetAPI().SendTx(tx)
				if err == types.ErrChannelClosed {
					return
				}

				if err != nil {
					time.Sleep(time.Second * 10)
					continue
				}
				amount++
				if amount > 1000 {
					addr, _ = util.Genaddress()
					amount = 1
					mutex.Lock()
					totalTx += 1000
					waitFlag := totalTx%txBenchCount == 0
					mutex.Unlock()

					if waitFlag {
						//同时控制了交易发送，否则mempool会占满状态
						waitTxChan <- reply.GetMsg()
					}
				}
			}
		}()
		<-startChan
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := <-waitTxChan
		param := &types.ReqHash{Hash: hash}
		for {
			_, err := mock33.GetAPI().QueryTx(param)
			if err == nil {
				break
			}
			time.Sleep(time.Second / 10)
		}
	}
}

// 交易签名性能测试  20000/s
func BenchmarkCheckTxSign(b *testing.B) {
	cfg := testnode.GetDefaultConfig()
	txBenchNum := 10000
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, int64(txBenchNum))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			txs[i%txBenchNum].CheckSign()
			i++
		}
	})
}

//消息队列发送性能, 200w /s
func BenchmarkMsgQueue(b *testing.B) {
	cfg := testnode.GetDefaultConfig()
	q := queue.New("channel")
	q.SetConfig(cfg)
	topicNum := 10
	topics := make([]string, topicNum)
	start := make(chan struct{})
	for i := 0; i < topicNum; i++ {
		topics[i] = fmt.Sprintf("bench-%d", i)
		go func(topic string) {
			start <- struct{}{}
			client := q.Client()
			client.Sub(topic)
			for range client.Recv() {
			}
		}(topics[i])
		<-start
	}
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, 1)
	sendCli := q.Client()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			msg := sendCli.NewMessage(topics[i%topicNum], int64(i), txs[0])
			err := sendCli.Send(msg, false)
			assert.Nil(b, err)
			i++
		}
	})
}
