// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package solo

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof" //
	"sync"
	"testing"
	"time"

	"github.com/33cn/chain33/common"
	log "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/rpc/grpcclient"
	"github.com/decred/base58"
	b58 "github.com/mr-tron/base58"

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
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	subcfg := cfg.GetSubConfig()
	cfg.GetModuleConfig().Exec.DisableAddrIndex = true
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 100)
	assert.Nil(b, err)
	solocfg, err = types.ModifySubConfig(solocfg, "benchMode", true)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	cfg.GetModuleConfig().RPC.JrpcBindAddr = "localhost:8801"
	cfg.GetModuleConfig().RPC.GrpcBindAddr = "localhost:8802"
	mock33 := testnode.NewWithRPC(cfg, nil)
	log.SetLogLevel("error")
	defer mock33.Close()
	priv := mock33.GetGenesisKey()
	b.ResetTimer()
	b.Run("SendTx-Internal", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, 0)
				mock33.GetAPI().SendTx(tx)
			}
		})
	})

	b.Run("SendTx-GRPC", func(b *testing.B) {
		gcli, _ := grpcclient.NewMainChainClient(cfg, "localhost:8802")
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, 0)
				gcli.SendTransaction(context.Background(), tx)
			}
		})
	})

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
	defer http.DefaultClient.CloseIdleConnections()
	b.Run("SendTx-JSONRPC", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, 0)
				poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.SendTransaction","params":[{"data":"%v"}]}`,
					common.ToHex(types.Encode(tx)))

				resp, _ := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
		})
	})
}

//测试每10000笔交易打包的并发时间  3500tx/s
func BenchmarkSoloNewBlock(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	cfg.GetModuleConfig().Exec.DisableAddrIndex = true
	cfg.GetModuleConfig().Mempool.DisableExecCheck = true
	subcfg := cfg.GetSubConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 100)
	assert.Nil(b, err)
	solocfg, err = types.ModifySubConfig(solocfg, "benchMode", true)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	mock33 := testnode.NewWithConfig(cfg, nil)
	defer mock33.Close()
	start := make(chan struct{})

	//pub := mock33.GetGenesisKey().PubKey().Bytes()
	for i := 0; i < 10; i++ {
		addr, _ := util.Genaddress()
		go func(addr string) {
			start <- struct{}{}
			for {
				tx := util.CreateNoneTxWithTxHeight(cfg, mock33.GetGenesisKey(), 0)
				//测试去签名情况
				//tx := util.CreateNoneTxWithTxHeight(cfg, nil, 0)
				//tx.Signature = &types.Signature{
				//	Ty: types.SECP256K1,
				//	Pubkey:pub,
				//}
				_, err := mock33.GetAPI().SendTx(tx)
				if err == types.ErrChannelClosed {
					return
				}
				if err != nil {
					time.Sleep(2 * time.Second)
					continue
				}
			}
		}(addr)
		<-start
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		err := mock33.WaitHeight(int64(i + 1))
		for err != nil {
			b.Log("SoloNewBlock", "waitblkerr", err)
			time.Sleep(time.Second / 10)
			err = mock33.WaitHeight(int64(i + 1))
		}
	}
}

// 交易签名性能测试  单核4k
func BenchmarkTxSign(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	txBenchNum := 10000
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, int64(txBenchNum))

	start := make(chan struct{})
	wait := make(chan struct{})
	result := make(chan interface{}, txBenchNum)
	//控制并发协程数量
	for i := 0; i < 8; i++ {
		go func() {
			wait <- struct{}{}
			index := 0
			<-start
			for {
				//txs[index%txBenchNum].Sign(types.SECP256K1, priv)
				result <- txs[index%txBenchNum].CheckSign()
				index++
			}
		}()
		<-wait
	}
	b.ResetTimer()
	close(start)
	for i := 0; i < b.N; i++ {
		<-result
	}
}

//消息队列发送性能, 80w /s
func BenchmarkMsgQueue(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
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

func BenchmarkTxHash(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txs[0].Hash()
	}
}

func BenchmarkEncode(b *testing.B) {

	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, 10000)

	block := &types.Block{}
	block.Txs = txs
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		types.Encode(block)
	}
}

func BenchmarkBase58(b *testing.B) {

	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	addr := "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
	buf := base58.Decode(addr)
	b.ResetTimer()
	b.Run("decred-base58", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			base58.Encode(buf)
		}
	})

	b.Run("mr-tron-base58", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b58.Encode(buf)
		}
	})
}
