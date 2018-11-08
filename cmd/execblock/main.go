package main

//这个软件包的主要目的是执行已经同步好的区块链的某个区块
import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os/user"
	"path/filepath"

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	log "gitlab.33.cn/chain33/chain33/common/log/log15"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

var height = flag.Int64("height", 1, "exec block height")
var datadir = flag.String("datadir", "", "data dir of chain33, include logs and datas")
var configPath = flag.String("f", "chain33.toml", "configfile")

func resetDatadir(cfg *types.Config, datadir string) {
	// Check in case of paths like "/something/~/something/"
	if datadir[:2] == "~/" {
		usr, _ := user.Current()
		dir := usr.HomeDir
		datadir = filepath.Join(dir, datadir[2:])
	}
	log.Info("current user data dir is ", "dir", datadir)
	cfg.Log.LogFile = filepath.Join(datadir, cfg.Log.LogFile)
	cfg.BlockChain.DbPath = filepath.Join(datadir, cfg.BlockChain.DbPath)
	cfg.P2P.DbPath = filepath.Join(datadir, cfg.P2P.DbPath)
	cfg.Wallet.DbPath = filepath.Join(datadir, cfg.Wallet.DbPath)
	cfg.Store.DbPath = filepath.Join(datadir, cfg.Store.DbPath)
}

func initEnv() (queue.Queue, queue.Module, queue.Module) {
	var q = queue.New("channel")
	cfg, sub := types.InitCfg(*configPath)
	if *datadir != "" {
		resetDatadir(cfg, *datadir)
	}
	cfg.Consensus.Minerstart = false
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())
	exec := executor.New(cfg.Exec, sub.Exec)
	exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	s := store.New(cfg.Store, sub.Store)
	s.SetQueueClient(q.Client())
	return q, chain, s
}

func main() {
	clog.SetLogLevel("info")
	flag.Parse()
	q, chain, s := initEnv()
	defer s.Close()
	defer chain.Close()
	defer q.Close()
	qclient, err := client.New(q.Client(), nil)
	if err != nil {
		panic(err)
	}
	req := &types.ReqBlocks{Start: *height - 1, End: *height}
	blocks, err := qclient.GetBlocks(req)
	if err != nil {
		panic(err)
	}
	log.Info("execblock", "block height", *height)
	prevState := blocks.Items[0].Block.StateHash
	block := blocks.Items[1].Block
	receipt := util.ExecTx(q.Client(), prevState, block)
	for i, r := range receipt.GetReceipts() {
		println("=======================")
		println("tx index ", i)
		for j, kv := range r.GetKV() {
			fmt.Println("\tKV:", j, kv)
		}
		for k, l := range r.GetLogs() {
			logType := types.LoadLog(block.Txs[i].Execer, int64(l.Ty))
			lTy := "unkownType"
			var logIns interface{}
			if logType != nil {
				logIns, err = logType.Decode(l.GetLog())
				if err != nil {
					panic(err)
				}
				lTy = logType.Name()
			}
			fmt.Printf("\tLog:%d %s->%v\n", k, lTy, logIns)
		}
	}
}
