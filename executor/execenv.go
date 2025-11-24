// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/client/api"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

// 执行器 -> db 环境
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
	//单个区块执行期间执行器缓存
	driverCache map[string]drivers.Driver
	//在单笔交易执行期间，将当前交易的执行driver缓存，避免多次load
	currTxIdx  int
	currExecTx *types.Transaction
	currDriver drivers.Driver
	cfg        *types.Chain33Config
	exec       *Executor
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
	types.AssertConfig(client)
	cfg := client.GetConfig()
	enableMVCC := exec.pluginEnable["mvcc"]
	opt := &StateDBOption{EnableMVCC: enableMVCC, Height: ctx.height}

	e := &executor{
		stateDB:      NewStateDB(client, ctx.stateHash, localdb, opt),
		localDB:      localdb,
		coinsAccount: account.NewCoinsAccount(cfg),
		height:       ctx.height,
		blocktime:    ctx.blocktime,
		difficulty:   ctx.difficulty,
		ctx:          ctx,
		txs:          txs,
		receipts:     receipts,
		api:          exec.qclient,
		gcli:         exec.grpccli,
		driverCache:  make(map[string]drivers.Driver),
		currTxIdx:    -1,
		cfg:          cfg,
		exec:         exec,
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

// 隐私交易费扣除规则：
// 1.公对私交易：直接从coin合约中扣除
// 2.私对私交易或者私对公交易：交易费的扣除从隐私合约账户在coin合约中的账户中扣除
func (e *executor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := tx.From()
	accFrom := e.coinsAccount.LoadAccount(from)
	if accFrom.GetBalance()-tx.Fee >= 0 {
		copyfrom := types.CloneAccount(accFrom)
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		receiptBalance := &types.ReceiptAccountTransfer{Prev: copyfrom, Current: accFrom}
		set := e.coinsAccount.GetKVSet(accFrom)
		e.coinsAccount.SaveKVSet(set)
		return e.cutFeeReceipt(set, receiptBalance), nil
	}
	return nil, types.ErrNoBalance
}

func (e *executor) cutFeeReceipt(kvset []*types.KeyValue, receiptBalance proto.Message) *types.Receipt {
	feelog := &types.ReceiptLog{Ty: types.TyLogFee, Log: types.Encode(receiptBalance)}
	return &types.Receipt{
		Ty:   types.ExecPack,
		KV:   kvset,
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

	// 转发的交易由主链验证, 平行链忽略基础检查
	if e.cfg.IsPara() && types.IsForward2MainChainTx(e.cfg, tx) {
		return nil
	}
	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.cfg, e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := tx.Check(e.cfg, e.height, e.cfg.GetMinTxFeeRate(), e.cfg.GetMaxTxFee(e.height)); err != nil {
		return err
	}
	//允许重写的情况
	//看重写的名字 name, 是否被允许执行
	if !types.IsAllowExecName(e.getRealExecName(tx, index), tx.Execer) {
		elog.Error("checkTx execNameNotAllow", "realname", string(e.getRealExecName(tx, index)), "exec", string(tx.Execer))
		return types.ErrExecNameNotAllow
	}
	return nil
}

func (e *executor) setEnv(exec drivers.Driver) {
	exec.SetAPI(e.api)
	//执行器共用一个coins account对象
	exec.SetCoinsAccount(e.coinsAccount)
	exec.SetStateDB(e.stateDB)
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime, e.difficulty)
	exec.SetBlockInfo(e.ctx.parentHash, e.ctx.mainHash, e.ctx.mainHeight)
	exec.SetExecutorAPI(e.api, e.gcli)
	e.execapi = exec.GetExecutorAPI()
	exec.SetTxs(e.txs)
	exec.SetReceipt(e.receipts)
}

func (e *executor) checkTxGroup(txgroup *types.Transactions, index int) error {

	if e.height > 0 && e.blocktime > 0 && txgroup.IsExpire(e.cfg, e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := txgroup.Check(e.cfg, e.height, e.cfg.GetMinTxFeeRate(), e.cfg.GetMaxTxFee(e.height)); err != nil {
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
	if err := address.CheckAddress(tx.To, e.height); err != nil {
		return err
	}
	var exec drivers.Driver

	//暂时只对none driver做了缓存处理 TODO: 增加其他执行器pool缓存
	if types.Bytes2Str(tx.Execer) == "none" {
		exec = e.getNoneDriver()
		defer e.freeNoneDriver(exec)
	} else {
		exec = e.loadDriver(tx, index)
	}
	//手续费检查
	if !exec.IsFree() && e.cfg.GetMinTxFeeRate() > 0 {
		from := tx.From()
		accFrom := e.coinsAccount.LoadAccount(from)

		//余额少于手续费时直接返回错误
		if accFrom.GetBalance() < tx.GetTxFee() {
			elog.Error("execCheckTx", "ispara", e.cfg.IsPara(), "exec", string(tx.Execer), "account", accFrom.String(), "Balance", accFrom.GetBalance(), "TxFee", tx.GetTxFee())
			return types.ErrNoBalance
		}

		if accFrom.GetBalance() < e.cfg.GInt("MinBalanceTransfer") {
			elog.Error("execCheckTx", "ispara", e.cfg.IsPara(), "exec", string(tx.Execer), "nonce", tx.Nonce, "Balance", accFrom.GetBalance())
			return types.ErrBalanceLessThanTenTimesFee
		}

	}

	return exec.CheckTx(tx, index)
}

// Exec base exec func
func (e *executor) Exec(tx *types.Transaction, index int) (receipt *types.Receipt, err error) {
	//针对一个交易执行阶段panic的处理，防止链停止，返回TyLogErr
	defer func() {
		if r := recover(); r != nil {
			receipt = nil
			err = types.ErrExecPanic
			elog.Error("execTx.Exec", "index", index, "hash", common.ToHex(tx.Hash()), "err", r, "stack", GetStack())
			return
		}
	}()

	exec := e.loadDriver(tx, index)
	//to 必须是一个地址
	if err := drivers.CheckAddress(e.cfg, tx.GetRealToAddr(), e.height); err != nil {
		return nil, err
	}
	if e.localDB != nil && e.cfg.IsFork(e.height, "ForkLocalDBAccess") {
		e.localDB.(*LocalDB).DisableWrite()
		if exec.ExecutorOrder() != drivers.ExecLocalSameTime {
			e.localDB.(*LocalDB).DisableRead()
		}
		defer func() {
			e.localDB.(*LocalDB).EnableWrite()
			if exec.ExecutorOrder() != drivers.ExecLocalSameTime {
				e.localDB.(*LocalDB).EnableRead()
			}
		}()
	}
	//第一步先检查 CheckTx
	if err := exec.CheckTx(tx, index); err != nil {
		return nil, err
	}
	r, err := exec.Exec(tx, index)
	return r, err
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecDelLocal(tx, r, index)
}

func (e *executor) getNoneDriver() drivers.Driver {
	none := e.exec.noneDriverPool.Get().(drivers.Driver)
	e.setEnv(none)
	return none
}

func (e *executor) freeNoneDriver(none drivers.Driver) {
	e.exec.noneDriverPool.Put(none)
}

// 加载none执行器
func (e *executor) loadNoneDriver() drivers.Driver {
	none, ok := e.driverCache["none"]
	var err error
	if !ok {
		none, err = drivers.LoadDriverWithClient(e.api, "none", 0)
		if err != nil {
			panic(err)
		}
		e.driverCache["none"] = none
	}
	return none
}

// loadDriver 加载执行器
// 对单笔交易执行期间的执行器做了缓存，避免多次加载
// 只有参数的tx，index，和记录的当前交易，当前索引，均相等时，才返回缓存的当前交易执行器
func (e *executor) loadDriver(tx *types.Transaction, index int) (c drivers.Driver) {

	// 交易和index都相等时，返回已缓存的当前交易执行器
	if e.currExecTx == tx && e.currTxIdx == index {
		return e.currDriver
	}
	var err error
	name := types.Bytes2Str(tx.Execer)
	driver, ok := e.driverCache[name]
	isFork := e.cfg.IsFork(e.height, "ForkCacheDriver")

	if !ok {
		driver, err = drivers.LoadDriverWithClient(e.api, name, e.height)
		if err != nil {
			driver = e.loadNoneDriver()
		}
		e.driverCache[name] = driver
	}

	//fork之前，多笔相同执行器的交易只有第一笔会进行Allow判定，从缓存中获取的执行器不需要进行allow判定
	//fork之后，所有的交易均需要单独执行Allow判定
	//Allow判定主要目的是在主链中, 对平行链的交易只做存证, 而不进行实际的执行逻辑
	//Allow判定失败, 加载none执行器, 即平行链交易被认为是存证交易类型
	if !ok || isFork {
		driver.SetEnv(e.height, 0, 0)
		err = driver.Allow(tx, index)
	}

	// allow不通过时，统一加载none执行器
	if err != nil {
		driver = e.loadNoneDriver()
		//fork之前，cache中存放的是经过allow判定后，实际用于执行的执行器，比如主链执行平行链交易的执行器对应的是none对象
		//fork之后，cache中存放的是和Execer名称对应的driver对象, 如user.p.para.coins => coins
		//fork之前的问题在于cache缓存错乱，不应该缓存实际用于执行的，即缓存包含了allow的逻辑，导致错乱
		//正确逻辑是，cache中的执行器对象和名称是一一对应的，保证了driver对象复用，但同时不同交易的allow需要重新判定
		if !isFork {
			e.driverCache[name] = driver
		}
	} else {
		driver.SetName(types.Bytes2Str(types.GetRealExecName(tx.Execer)))
		driver.SetCurrentExecName(name)
	}
	e.setEnv(driver)

	//均不相等时，表明当前交易已更新，需要同步更新缓存，并记录当前交易及其index
	if e.currExecTx != tx && e.currTxIdx != index {
		e.currExecTx = tx
		e.currTxIdx = index
		e.currDriver = driver
	}
	return driver
}

func (e *executor) execTxGroup(txs []*types.Transaction, index int) ([]*types.Receipt, error) {
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
	e.begin()
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
		if e.cfg.IsFork(e.height, "ForkExecRollback") {
			e.rollback()
		}
		return receipts, nil
	}
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
			if e.cfg.IsFork(e.height, "ForkResetTx0") {
				receipts[0] = rollbackLog
			}
			//撤销所有的数据库更新
			e.rollback()
			return receipts, nil
		}
	}
	err = e.commit()
	if err != nil {
		return nil, err
	}
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
	// 非平行连情况下,手续费为0,直接返回
	if !e.cfg.IsPara() && e.cfg.GetMinTxFeeRate() == 0 {
		return feelog, nil
	}
	execer := string(tx.Execer)
	ex := e.loadDriver(tx, index)
	//执行器名称 和  pubkey 相同，费用从内置的执行器中扣除,但是checkTx 中要过
	//默认checkTx 中对这样的交易会返回
	if bytes.Equal(address.ExecPubKey(execer), tx.GetSignature().GetPubkey()) {
		err := ex.CheckTx(tx, index)
		if err != nil {
			return nil, err
		}
	}
	var err error
	//平行链不收取手续费
	if !e.cfg.IsPara() && e.cfg.GetMinTxFeeRate() > 0 && !ex.IsFree() {
		feelog, err = e.processFee(tx)
		if err != nil {
			return nil, err
		}
	}
	return feelog, nil
}

func copyReceipt(feelog *types.Receipt) *types.Receipt {

	receipt := types.Receipt{}
	receipt.Ty = feelog.Ty
	receipt.KV = make([]*types.KeyValue, len(feelog.KV))
	copy(receipt.KV, feelog.KV)
	receipt.Logs = make([]*types.ReceiptLog, len(feelog.Logs))
	copy(receipt.Logs, feelog.Logs)
	return &receipt
}

func (e *executor) execTxOne(feelog *types.Receipt, tx *types.Transaction, index int) (*types.Receipt, error) {
	//只有到pack级别的，才会增加index
	e.startTx()
	receipt, err := e.Exec(tx, index)
	if err != nil {
		elog.Error("execTxOne", "exec tx error", err, "exec", string(tx.Execer), "action", tx.ActionName())
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
	err = e.checkKV(memkvset, receipt.GetKV())
	if err != nil {
		errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	feelog, err = e.checkKeyAllow(feelog, tx, index, receipt.GetKV())
	if err != nil {
		return feelog, err
	}
	err = e.execLocalSameTime(tx, receipt, index)
	if err != nil {
		elog.Error("execLocalSameTime", "err", err)
		errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	if receipt != nil {
		feelog.KV = append(feelog.KV, receipt.KV...)
		feelog.Logs = append(feelog.Logs, receipt.Logs...)
		feelog.Ty = receipt.Ty
	}
	if e.cfg.IsFork(e.height, "ForkStateDBSet") {
		for _, v := range feelog.KV {
			if err := e.stateDB.Set(v.Key, v.Value); err != nil {
				panic(err)
			}
		}
	}
	return feelog, nil
}

func (e *executor) checkKV(memset []string, kvs []*types.KeyValue) error {
	keys := make(map[string]struct{}, len(kvs))
	for _, kv := range kvs {
		k := kv.GetKey()
		keys[string(k)] = struct{}{}
	}
	for _, key := range memset {
		if _, ok := keys[key]; !ok {
			elog.Error("err memset key", "key", key)
			//非法的receipt，交易执行失败
			return types.ErrNotAllowMemSetKey
		}
	}
	return nil
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

func (e *executor) begin() {
	if e.cfg.IsFork(e.height, "ForkExecRollback") {
		if e.stateDB != nil {
			e.stateDB.Begin()
		}
		if e.localDB != nil {
			e.localDB.Begin()
		}
	}
}

func (e *executor) commit() error {
	if e.cfg.IsFork(e.height, "ForkExecRollback") {
		if e.stateDB != nil {
			if err := e.stateDB.Commit(); err != nil {
				return err
			}
		}
		if e.localDB != nil {
			if err := e.localDB.Commit(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *executor) startTx() {
	if e.stateDB != nil {
		e.stateDB.(*StateDB).StartTx()
	}
	if e.localDB != nil {
		e.localDB.(*LocalDB).StartTx()
	}
}

func (e *executor) rollback() {
	if e.cfg.IsFork(e.height, "ForkExecRollback") {
		if e.stateDB != nil {
			e.stateDB.Rollback()
		}
		if e.localDB != nil {
			e.localDB.Rollback()
		}
	}
}

func (e *executor) getCurrentNonce(addr string) int64 {

	nonceLocalKey := secp256k1eth.CaculCoinsEvmAccountKey(addr)
	evmNonce := &types.EvmAccountNonce{}
	nonceV, err := e.localDB.Get(nonceLocalKey)
	if err == nil {
		_ = types.Decode(nonceV, evmNonce)
	}

	return evmNonce.GetNonce()

}

func (e *executor) proxyGetRealTx(tx *types.Transaction) (*types.Transaction, error) {

	if string(types.GetRealExecName(tx.GetExecer())) != "evm" {
		return nil, fmt.Errorf("execName %s not allowd", string(types.GetRealExecName(tx.GetExecer())))
	}

	var actionData types.EVMContractAction4Chain33
	err := types.Decode(tx.GetPayload(), &actionData)
	if err != nil {
		return nil, err
	}
	if len(actionData.GetPara()) == 0 {
		return nil, errors.New("empty tx")
	}
	var realTx types.Transaction
	err = types.Decode(actionData.Para, &realTx)
	if err != nil {
		return nil, err
	}

	return &realTx, nil

}

func (e *executor) checkProxyExecTx(tx *types.Transaction) bool {
	if e.cfg.IsFork(e.height, "ForkProxyExec") && tx.To == e.cfg.GetModuleConfig().Exec.ProxyExecAddress && types.IsEthSignID(tx.Signature.Ty) {
		if string(types.GetRealExecName(tx.GetExecer())) == "evm" {
			return true
		}

	}
	return false
}
func (e *executor) proxyExecTx(tx *types.Transaction) (*types.Transaction, error) {

	realTx, err := e.proxyGetRealTx(tx)
	if err != nil {
		return realTx, err
	}
	realTx.Signature = tx.Signature

	return realTx, nil
}

func (e *executor) execTx(exec *Executor, tx *types.Transaction, index int) (*types.Receipt, error) {
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
	var feelog *types.Receipt

	var err error
	//代理执行 EVM-->txpayload-->chain33 tx
	if e.checkProxyExecTx(tx) {
		defer func(tx *types.Transaction) {
			cloneTx := tx.Clone()
			//执行evm execlocal 数据，主要是nonce++
			//此处执行execlocal 是为了连续多笔同地址下的交易，关系上下文用，否则下一笔evm交易的nonce 将会报错
			elog.Info("proxyExec", "isSameTmeExecLocal", e.isExecLocalSameTime(tx, index))
			err = e.execLocalSameTime(cloneTx, feelog, index)
			if err != nil {
				elog.Error("proxyExec ReExecLocal", " execLocalSameTime", err.Error())
			}
		}(tx)

		////由于代理执行交易并不会检查tx.nonce的正确性，所以在此处检查
		currentNonce := e.getCurrentNonce(tx.From())
		if currentNonce != tx.GetNonce() {
			return nil, fmt.Errorf("proxyExec nonce missmatch,tx.nonce:%v,localnonce:%v", tx.Nonce, currentNonce)
		}
		tx, err = e.proxyExecTx(tx)
		if err != nil {
			return nil, err
		}

	}

	//交易检查规则：
	//1. mempool 检查区块，尽量检查更多的错误
	//2. 打包的时候，尽量打包更多的交易，只要基本的签名，以及格式没有问题
	err = e.checkTx(tx, index)
	if err != nil {
		elog.Error("execTx.checkTx ", "txhash", common.ToHex(tx.Hash()), "err", err)
		if e.cfg.IsPara() {
			panic(err)
		}
		return nil, err
	}
	//处理交易手续费(先把手续费收了)
	//如果收了手续费，表示receipt 至少是pack 级别
	//收不了手续费的交易才是 error 级别
	feelog, err = e.execFee(tx, index)
	if err != nil {
		return nil, err
	}
	//ignore err
	e.begin()
	feelog, err = e.execTxOne(feelog, tx, index)
	if err != nil {
		e.rollback()
		elog.Error("execTx", "index", index, "execer", string(tx.Execer), "err", err)
	} else {
		err := e.commit()
		if err != nil {
			return nil, err
		}
	}

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
	return isAllowKeyWrite(e, key, realExecer, tx, index)
}

func (e *executor) isExecLocalSameTime(tx *types.Transaction, index int) bool {
	exec := e.loadDriver(tx, index)
	return exec.ExecutorOrder() == drivers.ExecLocalSameTime
}

func (e *executor) checkPrefix(execer []byte, kvs []*types.KeyValue) error {
	for i := 0; i < len(kvs); i++ {
		err := isAllowLocalKey(e.cfg, execer, kvs[i].Key)
		if err != nil {
			//测试的情况下，先panic，实际情况下会删除返回错误
			panic(err)
			//return err
		}
	}
	return nil
}

func (e *executor) execLocalSameTime(tx *types.Transaction, receipt *types.Receipt, index int) error {
	if e.isExecLocalSameTime(tx, index) {
		var r = &types.ReceiptData{}
		if receipt != nil {
			r.Ty = receipt.Ty
			r.Logs = receipt.Logs
		}

		_, err := e.execLocalTx(tx, r, index)
		return err
	}
	return nil
}

func (e *executor) execLocalTx(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := e.execLocal(tx, r, index)

	if err == types.ErrActionNotSupport {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	memkvset := e.localDB.(*LocalDB).GetSetKeys()
	if kv != nil && kv.KV != nil {
		err := e.checkKV(memkvset, kv.KV)
		if err != nil {
			return nil, types.ErrNotAllowMemSetLocalKey
		}
		err = e.checkPrefix(tx.Execer, kv.KV)
		if err != nil {
			return nil, err
		}
		for _, kv := range kv.KV {
			err = e.localDB.Set(kv.Key, kv.Value)
			if err != nil {
				panic(err)
			}
		}
	} else {
		if len(memkvset) > 0 {
			return nil, types.ErrNotAllowMemSetLocalKey
		}
	}
	return kv, nil
}
