// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"bytes"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/client/api"
	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

//执行器 -> db 环境
type executor struct {
	stateDB      dbm.KV
	localDB      dbm.KVDB
	coinsAccount *account.DB
	ctx          *executorCtx
	height       int64
	blocktime    int64
	// 增加区块的难度值，后面的执行器逻辑需要这些属性
	difficulty uint64
	txs        []*types.Transaction
	api        client.QueueProtocolAPI
	gcli       types.Chain33Client
	execapi    api.ExecutorAPI
	receipts   []*types.ReceiptData
}

type executorCtx struct {
	stateHash  []byte
	height     int64
	blocktime  int64
	difficulty uint64
	parentHash []byte
	mainHash   []byte
	mainHeight int64
}

func newExecutor(ctx *executorCtx, exec *Executor, localdb dbm.KVDB, txs []*types.Transaction, receipts []*types.ReceiptData) *executor {
	client := exec.client
	enableMVCC := exec.pluginEnable["mvcc"]
	opt := &StateDBOption{EnableMVCC: enableMVCC, Height: ctx.height}
	e := &executor{
		stateDB:      NewStateDB(client, ctx.stateHash, localdb, opt),
		localDB:      localdb,
		coinsAccount: account.NewCoinsAccount(),
		height:       ctx.height,
		blocktime:    ctx.blocktime,
		difficulty:   ctx.difficulty,
		ctx:          ctx,
		txs:          txs,
		receipts:     receipts,
		api:          exec.qclient,
		gcli:         exec.grpccli,
	}
	e.coinsAccount.SetDB(e.stateDB)
	return e
}

func (e *executor) enableMVCC(hash []byte) {
	e.stateDB.(*StateDB).enableMVCC(hash)
}

// AddMVCC convert key value to mvcc kv data
func AddMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	kvs := detail.KV
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	//检查版本号是否是连续的
	kvlist, err := mvcc.AddMVCC(kvs, hash, detail.PrevStatusHash, detail.Block.Height)
	if err != nil {
		panic(err)
	}
	return kvlist
}

// DelMVCC convert key value to mvcc kv data
func DelMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	kvlist, err := mvcc.DelMVCC(hash, detail.Block.Height, true)
	if err != nil {
		panic(err)
	}
	return kvlist
}

//隐私交易费扣除规则：
//1.公对私交易：直接从coin合约中扣除
//2.私对私交易或者私对公交易：交易费的扣除从隐私合约账户在coin合约中的账户中扣除
func (e *executor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := tx.From()
	accFrom := e.coinsAccount.LoadAccount(from)
	if accFrom.GetBalance()-tx.Fee >= 0 {
		copyfrom := *accFrom
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		receiptBalance := &types.ReceiptAccountTransfer{Prev: &copyfrom, Current: accFrom}
		e.coinsAccount.SaveAccount(accFrom)
		return e.cutFeeReceipt(accFrom, receiptBalance), nil
	}
	return nil, types.ErrNoBalance
}

func (e *executor) cutFeeReceipt(acc *types.Account, receiptBalance proto.Message) *types.Receipt {
	feelog := &types.ReceiptLog{Ty: types.TyLogFee, Log: types.Encode(receiptBalance)}
	return &types.Receipt{
		Ty:   types.ExecPack,
		KV:   e.coinsAccount.GetKVSet(acc),
		Logs: append([]*types.ReceiptLog{}, feelog),
	}
}

func (e *executor) getRealExecName(tx *types.Transaction, index int) []byte {
	exec := e.loadDriver(tx, index)
	realexec := exec.GetDriverName()
	var execer []byte
	if realexec != "none" {
		execer = []byte(realexec)
	} else {
		execer = tx.Execer
	}
	return execer
}

func (e *executor) checkTx(tx *types.Transaction, index int) error {
	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := tx.Check(e.height, types.GInt("MinFee")); err != nil {
		return err
	}
	//允许重写的情况
	//看重写的名字 name, 是否被允许执行
	if !types.IsAllowExecName(e.getRealExecName(tx, index), tx.Execer) {
		return types.ErrExecNameNotAllow
	}
	return nil
}

func (e *executor) setEnv(exec drivers.Driver) {
	exec.SetStateDB(e.stateDB)
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime, e.difficulty)
	exec.SetBlockInfo(e.ctx.parentHash, e.ctx.mainHash, e.ctx.mainHeight)
	exec.SetAPI(e.api)
	exec.SetExecutorAPI(e.api, e.gcli)
	e.execapi = exec.GetExecutorAPI()
	exec.SetTxs(e.txs)
	exec.SetReceipt(e.receipts)
}

func (e *executor) checkTxGroup(txgroup *types.Transactions, index int) error {
	if e.height > 0 && e.blocktime > 0 && txgroup.IsExpire(e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := txgroup.Check(e.height, types.GInt("MinFee")); err != nil {
		return err
	}
	return nil
}

func (e *executor) execCheckTx(tx *types.Transaction, index int) error {
	//基本检查
	err := e.checkTx(tx, index)
	if err != nil {
		return err
	}
	//检查地址的有效性
	if err := address.CheckAddress(tx.To); err != nil {
		return err
	}
	//checkInExec
	exec := e.loadDriver(tx, index)
	//手续费检查
	if !exec.IsFree() && types.GInt("MinFee") > 0 {
		from := tx.From()
		accFrom := e.coinsAccount.LoadAccount(from)
		if accFrom.GetBalance() < types.GInt("MinBalanceTransfer") {
			elog.Error("execCheckTx", "ispara", types.IsPara(), "exec", string(tx.Execer), "nonce", tx.Nonce)
			return types.ErrBalanceLessThanTenTimesFee
		}
	}
	e.setEnv(exec)
	return exec.CheckTx(tx, index)
}

// Exec base exec func
func (e *executor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec := e.loadDriver(tx, index)
	//to 必须是一个地址
	if err := drivers.CheckAddress(tx.GetRealToAddr(), e.height); err != nil {
		return nil, err
	}
	//第一步先检查 CheckTx
	if err := exec.CheckTx(tx, index); err != nil {
		return nil, err
	}
	return exec.Exec(tx, index)
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecDelLocal(tx, r, index)
}

func (e *executor) loadDriver(tx *types.Transaction, index int) (c drivers.Driver) {
	exec := drivers.LoadDriverAllow(tx, index, e.height)
	e.setEnv(exec)
	return exec
}

func (e *executor) execTxGroup(exec *Executor, txs []*types.Transaction, index int) ([]*types.Receipt, error) {
	txgroup := &types.Transactions{Txs: txs}
	err := e.checkTxGroup(txgroup, index)
	if err != nil {
		return nil, err
	}
	feelog, err := e.execFee(txs[0], index)
	if err != nil {
		return nil, err
	}
	//开启内存事务处理，假设系统只有一个thread 执行
	//如果系统执行失败，回滚到这个状态
	rollbackLog := copyReceipt(feelog)
	e.stateDB.Begin()
	e.localDB.Begin()
	receipts := make([]*types.Receipt, len(txs))
	for i := 1; i < len(txs); i++ {
		receipts[i] = &types.Receipt{Ty: types.ExecPack}
	}
	receipts[0], err = e.execTxOne(feelog, txs[0], index)
	if err != nil {
		//接口临时错误，取消执行
		if api.IsAPIEnvError(err) {
			return nil, err
		}
		//状态数据库回滚
		if types.IsFork(e.height, "ForkExecRollback") {
			e.stateDB.Rollback()
		}
		e.localDB.Rollback()
		return receipts, nil
	}
	exec.execLocalSameTime(e, txs[0], receipts[0], index)
	for i := 1; i < len(txs); i++ {
		//如果有一笔执行失败了，那么全部回滚
		receipts[i], err = e.execTxOne(receipts[i], txs[i], index+i)
		if err != nil {
			//reset other exec , and break!
			if api.IsAPIEnvError(err) {
				return nil, err
			}
			for k := 1; k < i; k++ {
				receipts[k] = &types.Receipt{Ty: types.ExecPack}
			}
			//撤销txs[0]的交易
			if types.IsFork(e.height, "ForkResetTx0") {
				receipts[0] = rollbackLog
			}
			//撤销所有的数据库更新
			e.stateDB.Rollback()
			e.localDB.Rollback()
			return receipts, nil
		}
		exec.execLocalSameTime(e, txs[i], receipts[i], index+i)
	}
	e.stateDB.Commit()
	e.localDB.Commit()
	return receipts, nil
}

func (e *executor) loadFlag(key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := e.localDB.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound {
		return 0, nil
	}
	return 0, err
}

func (e *executor) execFee(tx *types.Transaction, index int) (*types.Receipt, error) {
	feelog := &types.Receipt{Ty: types.ExecPack}
	execer := string(tx.Execer)
	ex := e.loadDriver(tx, index)
	e.setEnv(ex)
	//执行器名称 和  pubkey 相同，费用从内置的执行器中扣除,但是checkTx 中要过
	//默认checkTx 中对这样的交易会返回
	if bytes.Equal(address.ExecPubkey(execer), tx.GetSignature().GetPubkey()) {
		err := ex.CheckTx(tx, index)
		if err != nil {
			return nil, err
		}
	}
	var err error
	//公链不允许手续费为0
	if !types.IsPara() && types.GInt("MinFee") > 0 && !ex.IsFree() {
		feelog, err = e.processFee(tx)
		if err != nil {
			return nil, err
		}
	}
	return feelog, nil
}

func copyReceipt(feelog *types.Receipt) *types.Receipt {
	receipt := types.Receipt{}
	receipt = *feelog
	receipt.KV = make([]*types.KeyValue, len(feelog.KV))
	copy(receipt.KV, feelog.KV)
	receipt.Logs = make([]*types.ReceiptLog, len(feelog.Logs))
	copy(receipt.Logs, feelog.Logs)
	return &receipt
}

func (e *executor) execTxOne(feelog *types.Receipt, tx *types.Transaction, index int) (*types.Receipt, error) {
	//只有到pack级别的，才会增加index
	e.stateDB.(*StateDB).StartTx()
	receipt, err := e.Exec(tx, index)
	if err != nil {
		elog.Error("exec tx error = ", "err", err, "exec", string(tx.Execer), "action", tx.ActionName())
		//add error log
		errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	//合并两个receipt，如果执行不返回错误，那么就认为成功
	//需要检查两个东西:
	//1. statedb 中 Set的 key 必须是 在 receipt.GetKV() 这个集合中
	//2. receipt.GetKV() 中的 key, 必须符合权限控制要求
	memkvset := e.stateDB.(*StateDB).GetSetKeys()
	feelog, err = e.checkKV(feelog, memkvset, receipt.GetKV())
	if err != nil {
		return feelog, err
	}
	feelog, err = e.checkKeyAllow(feelog, tx, index, receipt.GetKV())
	if err != nil {
		return feelog, err
	}
	if receipt != nil {
		feelog.KV = append(feelog.KV, receipt.KV...)
		feelog.Logs = append(feelog.Logs, receipt.Logs...)
		feelog.Ty = receipt.Ty
	}
	return feelog, nil
}

func (e *executor) checkKV(feelog *types.Receipt, memset []string, kvs []*types.KeyValue) (*types.Receipt, error) {
	keys := make(map[string]bool)
	for _, kv := range kvs {
		k := kv.GetKey()
		keys[string(k)] = true
	}
	for _, key := range memset {
		if _, ok := keys[key]; !ok {
			elog.Error("err memset key", "key", key)
			//非法的receipt，交易执行失败
			errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(types.ErrNotAllowMemSetKey.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
			return feelog, types.ErrNotAllowMemSetKey
		}
	}
	return feelog, nil
}

func (e *executor) checkKeyAllow(feelog *types.Receipt, tx *types.Transaction, index int, kvs []*types.KeyValue) (*types.Receipt, error) {
	for _, kv := range kvs {
		k := kv.GetKey()
		if !e.isAllowExec(k, tx, index) {
			elog.Error("err receipt key", "key", string(k), "tx.exec", string(tx.GetExecer()),
				"tx.action", tx.ActionName())
			//非法的receipt，交易执行失败
			errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(types.ErrNotAllowKey.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
			return feelog, types.ErrNotAllowKey
		}
	}
	return feelog, nil
}

func (e *executor) execTx(tx *types.Transaction, index int) (*types.Receipt, error) {
	if e.height == 0 { //genesis block 不检查手续费
		receipt, err := e.Exec(tx, index)
		if err != nil {
			panic(err)
		}
		if err == nil && receipt == nil {
			panic("genesis block: executor not exist")
		}
		return receipt, nil
	}
	//交易检查规则：
	//1. mempool 检查区块，尽量检查更多的错误
	//2. 打包的时候，尽量打包更多的交易，只要基本的签名，以及格式没有问题
	err := e.checkTx(tx, index)
	if err != nil {
		return nil, err
	}
	//处理交易手续费(先把手续费收了)
	//如果收了手续费，表示receipt 至少是pack 级别
	//收不了手续费的交易才是 error 级别
	feelog, err := e.execFee(tx, index)
	if err != nil {
		return nil, err
	}
	//ignore err
	matchfork := types.IsFork(e.height, "ForkExecRollback")
	if matchfork {
		e.stateDB.Begin()
	}
	feelog, err = e.execTxOne(feelog, tx, index)
	if err != nil {
		if matchfork {
			e.stateDB.Rollback()
		}
	} else {
		if matchfork {
			e.stateDB.Commit()
		}
	}
	elog.Debug("exec tx = ", "index", index, "execer", string(tx.Execer), "err", err)
	if api.IsAPIEnvError(err) {
		return nil, err
	}
	return feelog, nil
}

//allowExec key 行为判断放入 执行器
/*
权限控制规则:
1. 默认行为:
执行器只能修改执行器下面的 key
或者能修改其他执行器 exec key 下面的数据

2. friend 合约行为, 合约可以定义其他合约 可以修改的 key的内容
*/
func (e *executor) isAllowExec(key []byte, tx *types.Transaction, index int) bool {
	realExecer := e.getRealExecName(tx, index)
	height := e.height
	return isAllowKeyWrite(key, realExecer, tx, height)
}
