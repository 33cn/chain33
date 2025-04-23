// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"unicode"

	"strings"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/queue"
	_ "github.com/33cn/chain33/system/address" //init address driver
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

func init() {
	rand.Seed(types.Now().UnixNano())
}

var ulog = log15.New("module", "util")

// GetParaExecName : 如果 name 没有 paraName 前缀，那么加上这个前缀
func GetParaExecName(paraName string, name string) string {
	if strings.HasPrefix(name, "user.p.") {
		return name
	}
	return paraName + name
}

// MakeStringToUpper : 将给定的in字符串从pos开始一共count个转换为大写字母
func MakeStringToUpper(in string, pos, count int) (out string, err error) {
	l := len(in)
	if pos < 0 || pos >= l || (pos+count) >= l || count <= 0 {
		err = fmt.Errorf("Invalid params. in=%s pos=%d count=%d", in, pos, count)
		return
	}
	tmp := []rune(in)
	for n := pos; n < pos+count; n++ {
		tmp[n] = unicode.ToUpper(tmp[n])
	}
	out = string(tmp)
	return
}

// MakeStringToLower : 将给定的in字符串从pos开始一共count个转换为小写字母
func MakeStringToLower(in string, pos, count int) (out string, err error) {
	l := len(in)
	if pos < 0 || pos >= l || (pos+count) >= l || count <= 0 {
		err = fmt.Errorf("Invalid params. in=%s pos=%d count=%d", in, pos, count)
		return
	}
	tmp := []rune(in)
	for n := pos; n < pos+count; n++ {
		tmp[n] = unicode.ToLower(tmp[n])
	}
	out = string(tmp)
	return
}

// GenNoneTxs : 创建一些 none 执行器的 交易列表，一般用于测试
func GenNoneTxs(cfg *types.Chain33Config, priv crypto.PrivKey, n int64) (txs []*types.Transaction) {
	for i := 0; i < int(n); i++ {
		txs = append(txs, CreateNoneTx(cfg, priv))
	}
	return txs
}

// GenCoinsTxs : generate txs to be executed on exector coin
func GenCoinsTxs(cfg *types.Chain33Config, priv crypto.PrivKey, n int64) (txs []*types.Transaction) {
	to, _ := Genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, CreateCoinsTx(cfg, priv, to, n+1))
	}
	return txs
}

// Genaddress : generate a address
func Genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.Load(types.GetSignName("", types.SECP256K1), -1)
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := address.PubKeyToAddr(address.DefaultID, privto.PubKey().Bytes())
	return addrto, privto
}

// CreateNoneTx : Create None Tx
func CreateNoneTx(cfg *types.Chain33Config, priv crypto.PrivKey) *types.Transaction {
	return CreateTxWithExecer(cfg, priv, "none")
}

// UpdateExpireWithTxHeight 设置txHeight类型交易过期
func UpdateExpireWithTxHeight(tx *types.Transaction, priv crypto.PrivKey, txHeight int64) {
	tx.Expire = txHeight + types.TxHeightFlag
	if priv != nil {
		tx.Sign(types.SECP256K1, priv)
	}
}

// CreateCoinsTxWithTxHeight 使用txHeight作为交易过期
func CreateCoinsTxWithTxHeight(cfg *types.Chain33Config, priv crypto.PrivKey, to string, amount, txHeight int64) *types.Transaction {

	tx := CreateCoinsTx(cfg, nil, to, amount)
	UpdateExpireWithTxHeight(tx, priv, txHeight)
	return tx
}

// CreateNoneTxWithTxHeight 使用txHeight作为交易过期
func CreateNoneTxWithTxHeight(cfg *types.Chain33Config, priv crypto.PrivKey, txHeight int64) *types.Transaction {

	tx := CreateNoneTx(cfg, nil)
	UpdateExpireWithTxHeight(tx, priv, txHeight)
	return tx
}

// CreateTxWithExecer ： Create Tx With Execer
func CreateTxWithExecer(cfg *types.Chain33Config, priv crypto.PrivKey, execer string) *types.Transaction {
	if execer == cfg.GetCoinExec() {
		to, _ := Genaddress()
		return CreateCoinsTx(cfg, priv, to, cfg.GetCoinPrecision())
	}
	tx := &types.Transaction{Execer: []byte(execer), Payload: []byte("none")}
	tx.To = address.ExecAddress(execer)
	tx, err := types.FormatTx(cfg, execer, tx)
	if err != nil {
		return nil
	}
	if priv != nil {
		tx.Sign(types.SECP256K1, priv)
	}
	return tx
}

// TestingT 测试类型
type TestingT interface {
	Error(args ...interface{})
	Log(args ...interface{})
}

// JSONPrint : print in json format
func JSONPrint(t TestingT, input interface{}) {
	data, err := json.MarshalIndent(input, "", "\t")
	if err != nil {
		t.Error(err)
		return
	}
	if t == nil {
		fmt.Println(string(data))
	} else {
		t.Log(string(data))
	}
}

// CreateManageTx : Create Manage Tx
func CreateManageTx(cfg *types.Chain33Config, priv crypto.PrivKey, key, op, value string) *types.Transaction {
	v := &types.ModifyConfig{Key: key, Op: op, Value: value, Addr: ""}
	exec := types.LoadExecutorType("manage")
	if exec == nil {
		panic("manage exec is not init")
	}
	tx, err := exec.Create("Modify", v)
	if err != nil {
		panic(err)
	}
	tx, err = types.FormatTx(cfg, "manage", tx)
	if err != nil {
		return nil
	}
	tx.Sign(types.SECP256K1, priv)
	return tx
}

// CreateCoinsTx : Create Coins Tx
func CreateCoinsTx(cfg *types.Chain33Config, priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	tx := createCoinsTx(cfg, to, amount)
	if priv != nil {
		tx.Sign(types.SECP256K1, priv)
	}
	return tx
}

func createCoinsTx(cfg *types.Chain33Config, to string, amount int64) *types.Transaction {
	exec := types.LoadExecutorType(cfg.GetCoinExec())
	if exec == nil {
		panic("unknow driver coins")
	}
	tx, err := exec.AssertCreate(&types.CreateTx{
		To:     to,
		Amount: amount,
	})
	if err != nil {
		panic(err)
	}
	tx.To = to
	tx, err = types.FormatTx(cfg, cfg.GetCoinExec(), tx)
	if err != nil {
		return nil
	}
	return tx
}

// CreateTxWithTxHeight : Create Tx With Tx Height
func CreateTxWithTxHeight(cfg *types.Chain33Config, priv crypto.PrivKey, to string, amount, expire int64) *types.Transaction {
	tx := createCoinsTx(cfg, to, amount)
	tx.Expire = expire + types.TxHeightFlag
	tx.Sign(types.SECP256K1, priv)
	return tx
}

// GenTxsTxHeight : Gen Txs with Heigt
func GenTxsTxHeight(cfg *types.Chain33Config, priv crypto.PrivKey, n, height int64) (txs []*types.Transaction) {
	to, _ := Genaddress()
	for i := 0; i < int(n); i++ {
		tx := CreateTxWithTxHeight(cfg, priv, to, (n+1)*cfg.GetCoinPrecision(), height)
		txs = append(txs, tx)
	}
	return txs
}

var zeroHash [32]byte

// CreateNoneBlock : Create None Block
func CreateNoneBlock(cfg *types.Chain33Config, priv crypto.PrivKey, n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = 1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = GenNoneTxs(cfg, priv, n)
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
	return newblock
}

// CreateCoinsBlock : create coins block, n size
func CreateCoinsBlock(cfg *types.Chain33Config, priv crypto.PrivKey, n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = 1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = GenCoinsTxs(cfg, priv, n)
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
	return newblock
}

// ExecBlock : just exec block
func ExecBlock(client queue.Client, prevStateRoot []byte, block *types.Block, errReturn, sync, checkblock bool) (*types.BlockDetail, []*types.Transaction, error) {
	ulog.Debug("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
	beg := types.Now()
	defer func() {
		ulog.Info("ExecBlock", "height", block.Height, "ntx", len(block.Txs), "writebatchsync", sync, "cost", types.Since(beg))
	}()

	detail, deltx, err := PreExecBlock(client, prevStateRoot, block, errReturn, sync, checkblock)
	if err != nil {
		return nil, nil, err
	}
	// 写数据库失败时需要及时返回错误，防止错误数据被写入localdb中CHAIN33-567
	err = ExecKVSetCommit(client, block.StateHash, false)
	if err != nil {
		return nil, nil, err
	}
	return detail, deltx, nil
}

// PreExecBlock : pre exec block
func PreExecBlock(client queue.Client, prevStateRoot []byte, block *types.Block, errReturn, sync, checkblock bool) (*types.BlockDetail, []*types.Transaction, error) {
	//发送执行交易给execs模块
	//通过consensus module 再次检查
	config := client.GetConfig()
	dupErrChan := make(chan error, 1)
	cacheTxs := types.TxsToCache(block.Txs)
	beg := types.Now()
	//区块验签，只验证非本节点打包的区块
	if errReturn && block.Height > 0 {

		//首先向mempool模块查询是否存在该交易，避免重复验签，同步历史区块时方案无效
		checkReq := &types.ReqCheckTxsExist{TxHashes: make([][]byte, len(block.Txs))}
		for i, tx := range cacheTxs {
			checkReq.TxHashes[i] = tx.Hash()
		}
		checkReqMsg := client.NewMessage("mempool", types.EventCheckTxsExist, checkReq)
		err := client.Send(checkReqMsg, true)
		if err != nil {
			ulog.Error("PreExecBlock", "send mempool check txs exist err", err)
			return nil, nil, err
		}
		reply, err := client.Wait(checkReqMsg)
		if err != nil {
			ulog.Error("PreExecBlock", "wait mempool check txs exist reply err", err)
			return nil, nil, err
		}
		replyData := reply.GetData().(*types.ReplyCheckTxsExist)
		unverifiedTxs := block.Txs
		//区块中交易在mempool中已有存在情况，重新构造需要验签的交易列表
		if replyData.ExistCount > 0 {
			unverifiedTxs = make([]*types.Transaction, 0, len(block.Txs)-int(replyData.ExistCount))
			for index, exist := range replyData.ExistFlags {
				//只需要对mempool中不存在的交易验签
				if !exist {
					unverifiedTxs = append(unverifiedTxs, block.Txs[index])
				}
			}
		}
		signOK := types.VerifySignature(config, block, unverifiedTxs)
		ulog.Info("PreExecBlock", "height", block.GetHeight(), "checkCount", len(unverifiedTxs), "CheckSign", types.Since(beg))
		if !signOK {
			return nil, nil, types.ErrSign
		}
	}

	//check dup routine
	go func() {
		beg := types.Now()
		defer func() {
			ulog.Debug("PreExecBlock", "height", block.GetHeight(), "CheckTxDup", types.Since(beg))
		}()
		//check tx Duplicate
		var err error
		cacheTxs, err = CheckTxDup(client, cacheTxs, block.Height)
		if err != nil {
			dupErrChan <- err
			return
		}
		if len(block.Txs) != len(cacheTxs) {
			ulog.Error("PreExecBlock", "prevtx", len(block.Txs), "newtx", len(cacheTxs))
			if errReturn {
				err = types.ErrTxDup
			}
		}
		dupErrChan <- err
	}()

	// exec tx routine
	beg = types.Now()
	//对区块的正确性保持乐观，交易查重和执行并行处理，提高效率
	receipts, err := ExecTx(client, prevStateRoot, block)
	ulog.Info("PreExecBlock", "height", block.GetHeight(), "ExecTx", types.Since(beg))
	beg = types.Now()

	//检查交易查重结果
	if dupErr := <-dupErrChan; dupErr != nil {
		ulog.Error("PreExecBlock", "height", block.GetHeight(), "CheckDupErr", dupErr)
		return nil, nil, dupErr
	}
	//如果有重复交易， 需要重新赋值交易内容并执行
	if len(block.Txs) != len(cacheTxs) {
		block.Txs = types.CacheToTxs(cacheTxs)
		receipts, err = ExecTx(client, prevStateRoot, block)
	}
	ulog.Info("PreExecBlock", "height", block.GetHeight(), "WaitDupCheck", types.Since(beg))
	if err != nil {
		return nil, nil, err
	}

	beg = types.Now()
	kvset := make([]*types.KeyValue, 0, len(receipts.GetReceipts()))
	rdata := make([]*types.ReceiptData, 0, len(receipts.GetReceipts())) //save to db receipt log
	//删除无效的交易
	var deltxs []*types.Transaction
	index := 0
	for i, receipt := range receipts.Receipts {
		if receipt.Ty == types.ExecErr {
			errTx := block.Txs[i]
			ulog.Error("exec tx err", "err", receipt, "txhash", common.ToHex(errTx.Hash()))
			if errReturn { //认为这个是一个错误的区块
				return nil, nil, types.ErrBlockExec
			}
			deltxs = append(deltxs, errTx)
			continue
		}
		block.Txs[index] = block.Txs[i]
		cacheTxs[index] = cacheTxs[i]
		index++
		rdata = append(rdata, &types.ReceiptData{Ty: receipt.Ty, Logs: receipt.Logs})
		kvset = append(kvset, receipt.KV...)
	}
	block.Txs = block.Txs[:index]
	cacheTxs = cacheTxs[:index]

	height := block.Height
	//此时需要区分主链和平行链
	if config.IsPara() {
		height = block.MainHeight
	}

	var txHash []byte
	if !config.IsFork(height, "ForkRootHash") {
		txHash = merkle.CalcMerkleRootCache(cacheTxs)
	} else {
		txHash = merkle.CalcMerkleRoot(config, height, types.TransactionSort(block.Txs))
	}
	// 非本节点产生的区块, 校验TxHash, 其他节点errReturn = true
	if errReturn && !bytes.Equal(txHash, block.TxHash) {
		ulog.Error("PreExecBlock", "height", block.GetHeight(), "blkHash", hex.EncodeToString(block.Hash(config)),
			"txHash", hex.EncodeToString(txHash), "errTxHash", hex.EncodeToString(block.TxHash),
			"blkData", hex.EncodeToString(types.Encode(block)))
		return nil, nil, types.ErrCheckTxHash
	}
	// 存在错误交易, 或共识生成区块时未设置TxHash, 需要重新赋值
	block.TxHash = txHash
	ulog.Debug("PreExecBlock", "CalcMerkleRootCache", types.Since(beg))
	beg = types.Now()
	kvset = DelDupKey(kvset)
	stateHash, err := ExecKVMemSet(client, prevStateRoot, block.Height, kvset, sync, false)
	if err != nil {
		return nil, nil, err
	}

	if errReturn && !bytes.Equal(block.StateHash, stateHash) {
		err = ExecKVSetRollback(client, stateHash)
		if err != nil {
			ulog.Error("PreExecBlock-->ExecKVSetRollback", "err", err)
		}
		if len(rdata) > 0 {
			for i, rd := range rdata {
				rd.OutputReceiptDetails(block.Txs[i].Execer, ulog)
			}
		}
		ulog.Error("PreExecBlock ErrCheckStateHash", "height", block.Height,
			"calcHash", common.ToHex(stateHash), "recvHash", common.ToHex(block.StateHash))
		return nil, nil, types.ErrCheckStateHash
	}
	block.StateHash = stateHash
	var detail types.BlockDetail
	detail.Block = block
	detail.Receipts = rdata
	if detail.Block.Height > 0 && checkblock {
		err := CheckBlock(client, &detail)
		if err != nil {
			ulog.Error("PreExecBlock", "height", block.GetHeight(), "checkBlockErr", err)
			return nil, nil, err
		}
	}
	ulog.Debug("PreExecBlock", "CheckBlock", types.Since(beg))

	if len(kvset) > 0 {
		detail.KV = kvset
	}
	detail.PrevStatusHash = prevStateRoot
	return &detail, deltxs, nil
}

// ExecBlockUpgrade : just exec block
func ExecBlockUpgrade(client queue.Client, prevStateRoot []byte, block *types.Block, sync bool) error {
	//发送执行交易给execs模块
	//通过consensus module 再次检查
	ulog.Debug("ExecBlockUpgrade", "height------->", block.Height, "ntx", len(block.Txs))
	beg := types.Now()
	beg1 := beg
	defer func() {
		ulog.Info("ExecBlockUpgrade", "height", block.Height, "ntx", len(block.Txs), "writebatchsync", sync, "cost", types.Since(beg1))
	}()

	var err error
	//println("1")
	receipts, err := ExecTx(client, prevStateRoot, block)
	if err != nil {
		return err
	}
	ulog.Debug("ExecBlockUpgrade", "ExecTx", types.Since(beg))
	beg = types.Now()
	var kvset []*types.KeyValue
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		kvset = append(kvset, receipt.KV...)
	}
	kvset = DelDupKey(kvset)
	calcHash, err := ExecKVMemSet(client, prevStateRoot, block.Height, kvset, sync, true)
	if err != nil {
		return err
	}
	//println("2")
	if !bytes.Equal(block.StateHash, calcHash) {
		return types.ErrCheckStateHash
	}
	ulog.Debug("ExecBlockUpgrade", "CheckBlock", types.Since(beg))
	// 写数据库失败时需要及时返回错误，防止错误数据被写入localdb中CHAIN33-567
	err = ExecKVSetCommit(client, calcHash, true)
	return err
}

// CreateNewBlock : Create a New Block
func CreateNewBlock(cfg *types.Chain33Config, parent *types.Block, txs []*types.Transaction) *types.Block {
	newblock := &types.Block{}
	newblock.Height = parent.Height + 1
	newblock.BlockTime = parent.BlockTime + 1
	newblock.ParentHash = parent.Hash(cfg)
	newblock.Txs = append(newblock.Txs, txs...)

	//需要首先对交易进行排序然后再计算TxHash
	if cfg.IsFork(newblock.GetHeight(), "ForkRootHash") {
		newblock.Txs = types.TransactionSort(newblock.Txs)
	}
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
	return newblock
}

// ExecAndCheckBlock ...
func ExecAndCheckBlock(qclient queue.Client, block *types.Block, txs []*types.Transaction, result []int) (*types.Block, error) {
	return ExecAndCheckBlockCB(qclient, block, txs, func(index int, receipt *types.ReceiptData) error {
		if len(result) <= index {
			return errors.New("txs num and status len not equal")
		}
		status := result[index]
		if status == 0 && receipt != nil {
			return errors.New("must failed, but index = " + fmt.Sprint(index))
		}
		if status > 0 && receipt == nil {
			return errors.New("must not faild, but index = " + fmt.Sprint(index))
		}
		if status > 0 && receipt.Ty != int32(status) {
			return errors.New("status must equal, but index = " + fmt.Sprint(index))
		}
		return nil
	})
}

// ExecAndCheckBlockCB :
func ExecAndCheckBlockCB(qclient queue.Client, block *types.Block, txs []*types.Transaction, cb func(int, *types.ReceiptData) error) (*types.Block, error) {
	block2 := CreateNewBlock(qclient.GetConfig(), block, txs)
	detail, deltx, err := ExecBlock(qclient, block.StateHash, block2, false, true, false)
	if err != nil {
		return nil, err
	}
	for _, v := range deltx {
		s, err := types.PBToJSON(v)
		if err != nil {
			return nil, err
		}
		println(string(s))
	}
	var getIndex = func(hash []byte, txlist []*types.Transaction) int {
		for i := 0; i < len(txlist); i++ {
			if bytes.Equal(hash, txlist[i].Hash()) {
				return i
			}
		}
		return -1
	}
	for i := 0; i < len(txs); i++ {
		if getIndex(txs[i].Hash(), deltx) >= 0 {
			if err := cb(i, nil); err != nil {
				return nil, err
			}
		} else if index := getIndex(txs[i].Hash(), detail.Block.Txs); index >= 0 {
			if err := cb(i, detail.Receipts[index]); err != nil {
				return nil, err
			}
		}
	}
	return detail.Block, nil
}

// ResetDatadir 重写datadir
func ResetDatadir(cfg *types.Config, datadir string) string {
	// Check in case of paths like "/something/~/something/"
	if len(datadir) >= 2 && datadir[:2] == "~/" {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		dir := usr.HomeDir
		datadir = filepath.Join(dir, datadir[2:])
	}
	if len(datadir) >= 6 && datadir[:6] == "$TEMP/" {
		dir, err := ioutil.TempDir("", "chain33datadir-")
		if err != nil {
			panic(err)
		}
		datadir = filepath.Join(dir, datadir[6:])
	}
	ulog.Info("current user data dir is ", "dir", datadir)
	cfg.Log.LogFile = filepath.Join(datadir, cfg.Log.LogFile)
	cfg.BlockChain.DbPath = filepath.Join(datadir, cfg.BlockChain.DbPath)
	cfg.P2P.DbPath = filepath.Join(datadir, cfg.P2P.DbPath)
	cfg.Wallet.DbPath = filepath.Join(datadir, cfg.Wallet.DbPath)
	cfg.Store.DbPath = filepath.Join(datadir, cfg.Store.DbPath)
	return datadir
}

// CreateTestDB 创建一个测试数据库
func CreateTestDB() (string, db.DB, db.KVDB) {
	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	leveldb, err := db.NewGoLevelDB("goleveldb", dir, 128)
	if err != nil {
		panic(err)
	}
	return dir, leveldb, db.NewKVDB(leveldb)
}

// CloseTestDB 创建一个测试数据库
func CloseTestDB(dir string, dbm db.DB) {
	err := os.RemoveAll(dir)
	if err != nil {
		ulog.Info("RemoveAll ", "dir", dir, "err", err)
	}
	dbm.Close()
}

// SaveKVList 保存kvs to database
func SaveKVList(kvdb db.DB, kvs []*types.KeyValue) {
	//printKV(kvs)
	batch := kvdb.NewBatch(true)
	for i := 0; i < len(kvs); i++ {
		if kvs[i].Value == nil {
			batch.Delete(kvs[i].Key)
			continue
		}
		batch.Set(kvs[i].Key, kvs[i].Value)
	}
	err := batch.Write()
	if err != nil {
		panic(err)
	}
}

// PrintKV 打印KVList
func PrintKV(kvs []*types.KeyValue) {
	for i := 0; i < len(kvs); i++ {
		fmt.Printf("KV %d %s(%s)\n", i, string(kvs[i].Key), common.ToHex(kvs[i].Value))
	}
}

// MockModule struct
type MockModule struct {
	Key string
}

// SetQueueClient method
func (m *MockModule) SetQueueClient(client queue.Client) {
	go func() {
		client.Sub(m.Key)
		for msg := range client.Recv() {
			msg.Reply(client.NewMessage(m.Key, types.EventReply, &types.Reply{IsOk: false,
				Msg: []byte(fmt.Sprintf("mock %s module not handle message %v", m.Key, msg.Ty))}))
		}
	}()
}

// Wait for ready
func (m *MockModule) Wait() {}

// Close method
func (m *MockModule) Close() {}
