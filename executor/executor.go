// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package executor 实现执行器模块基类功能
package executor

//store package store the world - state data
import (
	"strings"
	"sync"

	"github.com/33cn/chain33/client/api"
	dbm "github.com/33cn/chain33/common/db"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/rpc/grpcclient"
	drivers "github.com/33cn/chain33/system/dapp"

	// register drivers
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var elog = log.New("module", "execs")
var coinsAccount *dbm.DB

// SetLogLevel set log level
func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

// DisableLog disable log
func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

// Executor executor struct
type Executor struct {
	disableLocal bool
	client       queue.Client
	qclient      client.QueueProtocolAPI
	grpccli      types.Chain33Client
	pluginEnable map[string]bool
	alias        map[string]string
}

func execInit(sub map[string][]byte) {
	pluginmgr.InitExec(sub)
}

var runonce sync.Once

// New new executor
func New(cfg *types.Exec, sub map[string][]byte) *Executor {
	// init executor
	runonce.Do(func() {
		execInit(sub)
	})
	//设置区块链的MinFee，低于Mempool和Wallet设置的MinFee
	//在cfg.MinExecFee == 0 的情况下，必须 cfg.IsFree == true 才会起效果
	if cfg.MinExecFee == 0 && cfg.IsFree {
		elog.Warn("set executor to free fee")
		types.SetMinFee(0)
	}
	if cfg.MinExecFee > 0 {
		types.SetMinFee(cfg.MinExecFee)
	}
	exec := &Executor{}
	exec.pluginEnable = make(map[string]bool)
	exec.pluginEnable["stat"] = cfg.EnableStat
	exec.pluginEnable["mvcc"] = cfg.EnableMVCC
	exec.pluginEnable["addrindex"] = !cfg.DisableAddrIndex
	exec.pluginEnable["txindex"] = true
	exec.pluginEnable["fee"] = true

	exec.alias = make(map[string]string)
	for _, v := range cfg.Alias {
		data := strings.Split(v, ":")
		if len(data) != 2 {
			panic("exec.alias config error: " + v)
		}
		if _, ok := exec.alias[data[0]]; ok {
			panic("exec.alias repeat name: " + v)
		}
		if pluginmgr.HasExec(data[0]) {
			panic("exec.alias repeat name with system Exec: " + v)
		}
		exec.alias[data[0]] = data[1]
	}
	return exec
}

//Wait Executor ready
func (exec *Executor) Wait() {}

// SetQueueClient set client queue, for recv msg
func (exec *Executor) SetQueueClient(qcli queue.Client) {
	exec.client = qcli
	exec.client.Sub("execs")
	var err error
	exec.qclient, err = client.New(qcli, nil)
	if err != nil {
		panic(err)
	}
	if types.IsPara() {
		exec.grpccli, err = grpcclient.NewMainChainClient("")
		if err != nil {
			panic(err)
		}
	}
	//recv 消息的处理
	go func() {
		for msg := range exec.client.Recv() {
			elog.Debug("exec recv", "msg", msg)
			if msg.Ty == types.EventExecTxList {
				go exec.procExecTxList(msg)
			} else if msg.Ty == types.EventAddBlock {
				go exec.procExecAddBlock(msg)
			} else if msg.Ty == types.EventDelBlock {
				go exec.procExecDelBlock(msg)
			} else if msg.Ty == types.EventCheckTx {
				go exec.procExecCheckTx(msg)
			} else if msg.Ty == types.EventBlockChainQuery {
				go exec.procExecQuery(msg)
			}
		}
	}()
}

func (exec *Executor) procExecQuery(msg *queue.Message) {
	//panic 处理
	defer func() {
		if r := recover(); r != nil {
			elog.Error("panic error", "err", r)
			msg.Reply(exec.client.NewMessage("", types.EventReceipts, types.ErrExecPanic))
			return
		}
	}()
	header, err := exec.qclient.GetLastHeader()
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	data := msg.GetData().(*types.ChainExecutor)
	driver, err := drivers.LoadDriver(data.Driver, header.GetHeight())
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	if data.StateHash == nil {
		data.StateHash = header.StateHash
	}
	var localdb dbm.KVDB
	if !exec.disableLocal {
		localdb = NewLocalDB(exec.client)
		defer localdb.(*LocalDB).Close()
		driver.SetLocalDB(localdb)
	}
	opt := &StateDBOption{EnableMVCC: exec.pluginEnable["mvcc"], Height: header.GetHeight()}

	db := NewStateDB(exec.client, data.StateHash, localdb, opt)
	db.(*StateDB).enableMVCC(nil)
	driver.SetStateDB(db)
	driver.SetAPI(exec.qclient)
	driver.SetExecutorAPI(exec.qclient, exec.grpccli)
	driver.SetEnv(header.GetHeight(), header.GetBlockTime(), uint64(header.GetDifficulty()))
	//查询的情况下下，执行器不做严格校验，allow，尽可能的加载执行器，并且做查询

	ret, err := driver.Query(data.FuncName, data.Param)
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, err))
		return
	}
	msg.Reply(exec.client.NewMessage("", types.EventBlockChainQuery, ret))
}

func (exec *Executor) procExecCheckTx(msg *queue.Message) {
	//panic 处理
	defer func() {
		if r := recover(); r != nil {
			elog.Error("panic error", "err", r)
			msg.Reply(exec.client.NewMessage("", types.EventReceipts, types.ErrExecPanic))
			return
		}
	}()
	datas := msg.GetData().(*types.ExecTxList)
	ctx := &executorCtx{
		stateHash:  datas.StateHash,
		height:     datas.Height,
		blocktime:  datas.BlockTime,
		difficulty: datas.Difficulty,
		mainHash:   datas.MainHash,
		mainHeight: datas.MainHeight,
		parentHash: datas.ParentHash,
	}
	var localdb dbm.KVDB
	if !exec.disableLocal {
		localdb = NewLocalDB(exec.client)
		defer localdb.(*LocalDB).Close()
	}
	execute := newExecutor(ctx, exec, localdb, datas.Txs, nil)
	execute.enableMVCC(nil)
	//返回一个列表表示成功还是失败
	result := &types.ReceiptCheckTxList{}
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		index := i
		if datas.IsMempool {
			index = -1
		}
		err := execute.execCheckTx(tx, index)
		if err != nil {
			result.Errs = append(result.Errs, err.Error())
		} else {
			result.Errs = append(result.Errs, "")
		}
	}
	msg.Reply(exec.client.NewMessage("", types.EventReceiptCheckTx, result))
}

func (exec *Executor) procExecTxList(msg *queue.Message) {
	//panic 处理
	defer func() {
		if r := recover(); r != nil {
			elog.Error("panic error", "err", r)
			msg.Reply(exec.client.NewMessage("", types.EventReceipts, types.ErrExecPanic))
			return
		}
	}()
	datas := msg.GetData().(*types.ExecTxList)
	ctx := &executorCtx{
		stateHash:  datas.StateHash,
		height:     datas.Height,
		blocktime:  datas.BlockTime,
		difficulty: datas.Difficulty,
		mainHash:   datas.MainHash,
		mainHeight: datas.MainHeight,
		parentHash: datas.ParentHash,
	}
	var localdb dbm.KVDB
	if !exec.disableLocal {
		localdb = NewLocalDB(exec.client)
		defer localdb.(*LocalDB).Close()
	}
	execute := newExecutor(ctx, exec, localdb, datas.Txs, nil)
	execute.enableMVCC(nil)
	var receipts []*types.Receipt
	index := 0
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		//检查groupcount
		if tx.GroupCount < 0 || tx.GroupCount == 1 || tx.GroupCount > 20 {
			receipts = append(receipts, types.NewErrReceipt(types.ErrTxGroupCount))
			continue
		}
		if tx.GroupCount == 0 {
			receipt, err := execute.execTx(exec, tx, index)
			if api.IsAPIEnvError(err) {
				msg.Reply(exec.client.NewMessage("", types.EventReceipts, err))
				return
			}
			if err != nil {
				receipts = append(receipts, types.NewErrReceipt(err))
				continue
			}
			//update local
			receipts = append(receipts, receipt)
			index++
			continue
		}
		//所有tx.GroupCount > 0 的交易都是错误的交易
		if !types.IsFork(datas.Height, "ForkTxGroup") {
			receipts = append(receipts, types.NewErrReceipt(types.ErrTxGroupNotSupport))
			continue
		}
		//判断GroupCount 是否会产生越界
		if i+int(tx.GroupCount) > len(datas.Txs) {
			receipts = append(receipts, types.NewErrReceipt(types.ErrTxGroupCount))
			continue
		}
		receiptlist, err := execute.execTxGroup(datas.Txs[i:i+int(tx.GroupCount)], index)
		i = i + int(tx.GroupCount) - 1
		if len(receiptlist) > 0 && len(receiptlist) != int(tx.GroupCount) {
			panic("len(receiptlist) must be equal tx.GroupCount")
		}
		if err != nil {
			if api.IsAPIEnvError(err) {
				msg.Reply(exec.client.NewMessage("", types.EventReceipts, err))
				return
			}
			for n := 0; n < int(tx.GroupCount); n++ {
				receipts = append(receipts, types.NewErrReceipt(err))
			}
			continue
		}
		receipts = append(receipts, receiptlist...)
		index += int(tx.GroupCount)
	}
	msg.Reply(exec.client.NewMessage("", types.EventReceipts,
		&types.Receipts{Receipts: receipts}))
}

func (exec *Executor) procExecAddBlock(msg *queue.Message) {
	//panic 处理
	defer func() {
		if r := recover(); r != nil {
			elog.Error("panic error", "err", r)
			msg.Reply(exec.client.NewMessage("", types.EventReceipts, types.ErrExecPanic))
			return
		}
	}()
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	ctx := &executorCtx{
		stateHash:  b.StateHash,
		height:     b.Height,
		blocktime:  b.BlockTime,
		difficulty: uint64(b.Difficulty),
		mainHash:   b.MainHash,
		mainHeight: b.MainHeight,
		parentHash: b.ParentHash,
	}
	var localdb dbm.KVDB
	if !exec.disableLocal {
		localdb = NewLocalDB(exec.client)
		defer localdb.(*LocalDB).Close()
	}
	execute := newExecutor(ctx, exec, localdb, b.Txs, datas.Receipts)
	//因为mvcc 还没有写入，所以目前的mvcc版本是前一个区块的版本
	execute.enableMVCC(datas.PrevStatusHash)
	var kvset types.LocalDBSet
	for _, kv := range datas.KV {
		err := execute.stateDB.Set(kv.Key, kv.Value)
		if err != nil {
			panic(err)
		}
	}
	for name, plugin := range globalPlugins {
		kvs, ok, err := plugin.CheckEnable(execute, exec.pluginEnable[name])
		if err != nil {
			panic(err)
		}
		if !ok {
			continue
		}
		if len(kvs) > 0 {
			kvset.KV = append(kvset.KV, kvs...)
		}
		kvs, err = plugin.ExecLocal(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		if len(kvs) > 0 {
			kvset.KV = append(kvset.KV, kvs...)
			for _, kv := range kvs {
				err := execute.localDB.Set(kv.Key, kv.Value)
				if err != nil {
					panic(err)
				}
			}
		}
	}
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		execute.localDB.(*LocalDB).StartTx()
		kv, err := execute.execLocalTx(tx, datas.Receipts[i], i)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		if kv != nil && kv.KV != nil {
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	msg.Reply(exec.client.NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Executor) procExecDelBlock(msg *queue.Message) {
	//panic 处理
	defer func() {
		if r := recover(); r != nil {
			elog.Error("panic error", "err", r)
			msg.Reply(exec.client.NewMessage("", types.EventReceipts, types.ErrExecPanic))
			return
		}
	}()
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	ctx := &executorCtx{
		stateHash:  b.StateHash,
		height:     b.Height,
		blocktime:  b.BlockTime,
		difficulty: uint64(b.Difficulty),
		mainHash:   b.MainHash,
		mainHeight: b.MainHeight,
		parentHash: b.ParentHash,
	}
	var localdb dbm.KVDB
	if !exec.disableLocal {
		localdb = NewLocalDB(exec.client)
		defer localdb.(*LocalDB).Close()
	}
	execute := newExecutor(ctx, exec, localdb, b.Txs, nil)
	execute.enableMVCC(nil)
	var kvset types.LocalDBSet
	for _, kv := range datas.KV {
		err := execute.stateDB.Set(kv.Key, kv.Value)
		if err != nil {
			panic(err)
		}
	}
	for name, plugin := range globalPlugins {
		kvs, ok, err := plugin.CheckEnable(execute, exec.pluginEnable[name])
		if err != nil {
			panic(err)
		}
		if !ok {
			continue
		}
		if len(kvs) > 0 {
			kvset.KV = append(kvset.KV, kvs...)
		}
		kvs, err = plugin.ExecDelLocal(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		if len(kvs) > 0 {
			kvset.KV = append(kvset.KV, kvs...)
		}
	}
	for i := len(b.Txs) - 1; i >= 0; i-- {
		tx := b.Txs[i]
		kv, err := execute.execDelLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
			return
		}
		if kv != nil && kv.KV != nil {
			err := execute.checkPrefix(tx.Execer, kv.KV)
			if err != nil {
				msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	msg.Reply(exec.client.NewMessage("", types.EventDelBlock, &kvset))
}

// Close close executor
func (exec *Executor) Close() {
	elog.Info("exec module closed")
	if exec.client != nil {
		exec.client.Close()
	}
}
