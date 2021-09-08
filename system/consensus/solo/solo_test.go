// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package solo

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof" //
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/33cn/chain33/system/crypto/none"
	"github.com/33cn/chain33/system/crypto/secp256k1"

	"github.com/33cn/chain33/common/log/log15"
	"google.golang.org/grpc"

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

var (
	tlog = log15.New("module", "test solo")
)

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
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, types.LowAllowPackHeight/2)
				mock33.GetAPI().SendTx(tx)
			}
		})
	})

	b.Run("SendTx-GRPC", func(b *testing.B) {
		gcli, _ := grpcclient.NewMainChainClient(cfg, "localhost:8802")
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, types.LowAllowPackHeight/2)
				_, err := gcli.SendTransaction(context.Background(), tx)
				if err != nil {
					tlog.Error("sendtx grpc", "err", err)
				}
			}
		})
	})

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
	defer http.DefaultClient.CloseIdleConnections()
	b.Run("SendTx-JSONRPC", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, types.LowAllowPackHeight/2)
				poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.SendTransaction","params":[{"data":"%v"}]}`,
					common.ToHex(types.Encode(tx)))

				resp, _ := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
		})
	})
}

func sendTxGrpc(cfg *types.Chain33Config, recvChan <-chan *types.Transaction) {
	grpcAddr := cfg.GetModuleConfig().RPC.GrpcBindAddr
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()
	gcli := types.NewChain33Client(conn)
	for {
		tx, ok := <-recvChan
		if !ok {
			return
		}
		_, err := gcli.SendTransaction(context.Background(), tx)
		if err != nil {
			if strings.Contains(err.Error(), "ErrChannelClosed") {
				return
			}
			tlog.Error("sendtx", "err", err.Error())
			time.Sleep(time.Second)
			continue
		}
	}
}

func sendTxDirect(mock33 *testnode.Chain33Mock, recvChan <-chan *types.Transaction) {

	for {
		tx, ok := <-recvChan
		if !ok {
			return
		}
		_, err := mock33.GetAPI().SendTx(tx)
		if err != nil {
			if strings.Contains(err.Error(), "ErrChannelClosed") {
				return
			}
			tlog.Error("sendtx", "err", err.Error())
			time.Sleep(time.Second)
			continue
		}
	}
}

var (
	enablesign      *bool
	sendtxgrpc      *bool
	enabletxfee     *bool
	disabledupcheck *bool
	enabletxindex   *bool
	enableexeccheck *bool
	maxtxnum        *int64
)

func init() {
	enablesign = flag.Bool("enablesign", false, "enable tx sign")
	sendtxgrpc = flag.Bool("sendtxgrpc", false, "send tx with grpc")
	enabletxfee = flag.Bool("enabletxfee", false, "enable tx sign")
	disabledupcheck = flag.Bool("disabledupcheck", false, "diabledupcheck")
	enabletxindex = flag.Bool("enabletxindex", false, "enabletxindex")
	enableexeccheck = flag.Bool("enableexeccheck", false, "enabletxindex")
	maxtxnum = flag.Int64("maxtxnum", 10000, "max tx num in block")
	testing.Init()
	flag.Parse()

}

//测试solo并发
func BenchmarkSolo(b *testing.B) {

	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfgStr := types.GetDefaultCfgstring()
	if *maxtxnum > 10000 {
		str := fmt.Sprintf("maxTxNumber = %d", *maxtxnum)
		cfgStr = strings.Replace(cfgStr, "maxTxNumber = 10000", str, -1)
	}
	cfg := types.NewChain33Config(cfgStr)
	cfg.GetModuleConfig().Exec.DisableAddrIndex = true
	cfg.GetModuleConfig().Exec.DisableFeeIndex = true
	cfg.GetModuleConfig().Exec.DisableTxIndex = !*enabletxindex
	cfg.GetModuleConfig().Exec.DisableTxDupCheck = *disabledupcheck
	cfg.GetModuleConfig().Mempool.DisableExecCheck = !*enableexeccheck
	if !*enabletxfee {
		cfg.GetModuleConfig().Mempool.MinTxFeeRate = 0
		cfg.SetMinFee(0)
	}
	cfg.GetModuleConfig().RPC.GrpcBindAddr = "localhost:8802"
	cfg.GetModuleConfig().Crypto.EnableTypes = []string{secp256k1.Name, none.Name}
	subcfg := cfg.GetSubConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 100)
	assert.Nil(b, err)
	solocfg, err = types.ModifySubConfig(solocfg, "benchMode", true)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	cpuNum := runtime.NumCPU()
	mock33 := testnode.NewWithRPC(cfg, nil)
	defer mock33.Close()
	go func() {
		_ = http.ListenAndServe(":6060", nil)
	}()

	txChan := make(chan *types.Transaction, 10000)
	for i := 0; i < cpuNum*3; i++ {
		if *sendtxgrpc {
			go sendTxGrpc(cfg, txChan)
		} else {
			go sendTxDirect(mock33, txChan)
		}
	}

	start := make(chan struct{})
	var height int64
	for i := 0; i < cpuNum*2; i++ {
		go func() {
			start <- struct{}{}
			pub := mock33.GetGenesisKey().PubKey().Bytes()
			var tx *types.Transaction
			for {
				txHeight := atomic.LoadInt64(&height) + types.LowAllowPackHeight/2
				if *enablesign {
					tx = util.CreateNoneTxWithTxHeight(cfg, mock33.GetGenesisKey(), txHeight)
				} else {
					//测试去签名情况
					tx = util.CreateNoneTxWithTxHeight(cfg, nil, txHeight)
					tx.Signature = &types.Signature{
						Ty:     none.ID,
						Pubkey: pub,
					}
				}
				txChan <- tx
			}
		}()
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
		atomic.StoreInt64(&height, int64(i+1))
	}
}

// 交易签名性能测试  单核4k
func BenchmarkCheckSign(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	txBenchNum := 10000
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, int64(txBenchNum))

	start := make(chan struct{})
	wait := make(chan struct{})
	result := make(chan interface{}, 0)
	//控制并发协程数量
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			wait <- struct{}{}
			index := 0
			<-start
			for {
				//txs[index%txBenchNum].Sign(types.SECP256K1, priv)
				result <- txs[index%txBenchNum].CheckSign(0)
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
