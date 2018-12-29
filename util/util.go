// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

func init() {
	rand.Seed(types.Now().UnixNano())
}

var chainlog = log15.New("module", "util")

//GetParaExecName : 如果 name 没有 paraName 前缀，那么加上这个前缀
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

//GenNoneTxs : 创建一些 none 执行器的 交易列表，一般用于测试
func GenNoneTxs(priv crypto.PrivKey, n int64) (txs []*types.Transaction) {
	for i := 0; i < int(n); i++ {
		txs = append(txs, CreateNoneTx(priv))
	}
	return txs
}

//GenCoinsTxs : generate txs to be executed on exector coin
func GenCoinsTxs(priv crypto.PrivKey, n int64) (txs []*types.Transaction) {
	to, _ := Genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, CreateCoinsTx(priv, to, n+1))
	}
	return txs
}

//Genaddress : generate a address
func Genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

// CreateNoneTx : Create None Tx
func CreateNoneTx(priv crypto.PrivKey) *types.Transaction {
	return CreateTxWithExecer(priv, "none")
}

// CreateTxWithExecer ： Create Tx With Execer
func CreateTxWithExecer(priv crypto.PrivKey, execer string) *types.Transaction {
	if execer == "coins" {
		to, _ := Genaddress()
		return CreateCoinsTx(priv, to, types.Coin)
	}
	tx := &types.Transaction{Execer: []byte(execer), Payload: []byte("none")}
	tx.To = address.ExecAddress(execer)
	tx, _ = types.FormatTx(execer, tx)
	if priv != nil {
		tx.Sign(types.SECP256K1, priv)
	}
	return tx
}

// JSONPrint : print in json format
func JSONPrint(t *testing.T, input interface{}) {
	data, err := json.MarshalIndent(input, "", "\t")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(data))
}

// CreateManageTx : Create Manage Tx
func CreateManageTx(priv crypto.PrivKey, key, op, value string) *types.Transaction {
	v := &types.ModifyConfig{Key: key, Op: op, Value: value, Addr: ""}
	exec := types.LoadExecutorType("manage")
	if exec == nil {
		panic("manage exec is not init")
	}
	tx, err := exec.Create("Modify", v)
	if err != nil {
		panic(err)
	}
	tx, _ = types.FormatTx("manage", tx)
	tx.Sign(types.SECP256K1, priv)
	return tx
}

// CreateCoinsTx : Create Coins Tx
func CreateCoinsTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	tx := createCoinsTx(to, amount)
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func createCoinsTx(to string, amount int64) *types.Transaction {
	exec := types.LoadExecutorType("coins")
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
	tx, _ = types.FormatTx("coins", tx)
	return tx
}

//CreateTxWithTxHeight : Create Tx With Tx Height
func CreateTxWithTxHeight(priv crypto.PrivKey, to string, amount, expire int64) *types.Transaction {
	tx := createCoinsTx(to, amount)
	tx.Expire = expire + types.TxHeightFlag
	tx.Sign(types.SECP256K1, priv)
	return tx
}

// GenTxsTxHeigt : Gen Txs with Heigt
func GenTxsTxHeigt(priv crypto.PrivKey, n, height int64) (txs []*types.Transaction) {
	to, _ := Genaddress()
	for i := 0; i < int(n); i++ {
		tx := CreateTxWithTxHeight(priv, to, types.Coin*(n+1), 20+height)
		txs = append(txs, tx)
	}
	return txs
}

var zeroHash [32]byte

// CreateNoneBlock : Create None Block
func CreateNoneBlock(priv crypto.PrivKey, n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = 1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = GenNoneTxs(priv, n)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

//CreateCoinsBlock : create coins block, n size
func CreateCoinsBlock(priv crypto.PrivKey, n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = 1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = GenCoinsTxs(priv, n)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

// ExecBlock : just exec block
func ExecBlock(client queue.Client, prevStateRoot []byte, block *types.Block, errReturn bool, sync bool) (*types.BlockDetail, []*types.Transaction, error) {
	//发送执行交易给execs模块
	//通过consensus module 再次检查
	ulog := chainlog
	ulog.Debug("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
	beg := types.Now()
	defer func() {
		ulog.Info("ExecBlock", "height", block.Height, "ntx", len(block.Txs), "writebatchsync", sync, "cost", types.Since(beg))
	}()
	if errReturn && block.Height > 0 && !block.CheckSign() {
		//block的来源不是自己的mempool，而是别人的区块
		return nil, nil, types.ErrSign
	}
	//tx交易去重处理, 这个地方要查询数据库，需要一个更快的办法
	cacheTxs := types.TxsToCache(block.Txs)
	oldtxscount := len(cacheTxs)
	var err error
	cacheTxs, err = CheckTxDup(client, cacheTxs, block.Height)
	if err != nil {
		return nil, nil, err
	}
	newtxscount := len(cacheTxs)
	if oldtxscount != newtxscount && errReturn {
		return nil, nil, types.ErrTxDup
	}
	ulog.Debug("ExecBlock", "prevtx", oldtxscount, "newtx", newtxscount)
	block.TxHash = merkle.CalcMerkleRootCache(cacheTxs)
	block.Txs = types.CacheToTxs(cacheTxs)

	receipts := ExecTx(client, prevStateRoot, block)
	var kvset []*types.KeyValue
	var deltxlist = make(map[int]bool)
	var rdata []*types.ReceiptData //save to db receipt log
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		if receipt.Ty == types.ExecErr {
			ulog.Error("exec tx err", "err", receipt)
			if errReturn { //认为这个是一个错误的区块
				return nil, nil, types.ErrBlockExec
			}
			deltxlist[i] = true
			continue
		}
		rdata = append(rdata, &types.ReceiptData{Ty: receipt.Ty, Logs: receipt.Logs})
		kvset = append(kvset, receipt.KV...)
	}
	kvset = DelDupKey(kvset)
	//check TxHash
	calcHash := merkle.CalcMerkleRoot(block.Txs)
	if errReturn && !bytes.Equal(calcHash, block.TxHash) {
		return nil, nil, types.ErrCheckTxHash
	}
	block.TxHash = calcHash
	//删除无效的交易
	var deltx []*types.Transaction
	if len(deltxlist) > 0 {
		var newtx []*types.Transaction
		for i := 0; i < len(block.Txs); i++ {
			if deltxlist[i] {
				deltx = append(deltx, block.Txs[i])
			} else {
				newtx = append(newtx, block.Txs[i])
			}
		}
		block.Txs = newtx
		block.TxHash = merkle.CalcMerkleRoot(block.Txs)
	}

	var detail types.BlockDetail
	calcHash = ExecKVMemSet(client, prevStateRoot, block.Height, kvset, sync)
	if errReturn && !bytes.Equal(block.StateHash, calcHash) {
		ExecKVSetRollback(client, calcHash)
		if len(rdata) > 0 {
			for i, rd := range rdata {
				rd.OutputReceiptDetails(block.Txs[i].Execer, ulog)
			}
		}
		return nil, nil, types.ErrCheckStateHash
	}
	block.StateHash = calcHash
	detail.Block = block
	detail.Receipts = rdata
	ExecKVSetCommit(client, block.StateHash)
	return &detail, deltx, nil
}

//CreateNewBlock : Create a New Block
func CreateNewBlock(parent *types.Block, txs []*types.Transaction) *types.Block {
	newblock := &types.Block{}
	newblock.Height = parent.Height + 1
	newblock.BlockTime = parent.BlockTime + 1
	newblock.ParentHash = parent.Hash()
	newblock.Txs = append(newblock.Txs, txs...)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

//ExecAndCheckBlock : Exec and Check Block
func ExecAndCheckBlock(qclient queue.Client, block *types.Block, txs []*types.Transaction, status int) (*types.Block, error) {
	return ExecAndCheckBlockCB(qclient, block, txs, func(index int, receipt *types.ReceiptData) error {
		if status == 0 && receipt != nil {
			return errors.New("all must failed index = " + fmt.Sprint(index))
		}
		if status > 0 && receipt == nil {
			return errors.New("all must not faild, but index = " + fmt.Sprint(index))
		}
		if status > 0 && receipt.Ty != int32(status) {
			return errors.New("status not equal, but index = " + fmt.Sprint(index))
		}
		return nil
	})
}

// ExecAndCheckBlock2 :
func ExecAndCheckBlock2(qclient queue.Client, block *types.Block, txs []*types.Transaction, result []int) (*types.Block, error) {
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

//ExecAndCheckBlockCB :
func ExecAndCheckBlockCB(qclient queue.Client, block *types.Block, txs []*types.Transaction, cb func(int, *types.ReceiptData) error) (*types.Block, error) {
	block2 := CreateNewBlock(block, txs)
	detail, deltx, err := ExecBlock(qclient, block.StateHash, block2, false, true)
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
		} else {
			panic("never happen")
		}
	}
	return detail.Block, nil
}

//ResetDatadir 重写datadir
func ResetDatadir(cfg *types.Config, datadir string) string {
	// Check in case of paths like "/something/~/something/"
	if datadir[:2] == "~/" {
		usr, _ := user.Current()
		dir := usr.HomeDir
		datadir = filepath.Join(dir, datadir[2:])
	}
	if datadir[:6] == "$TEMP/" {
		dir, err := ioutil.TempDir("", "chain33datadir-")
		if err != nil {
			panic(err)
		}
		datadir = filepath.Join(dir, datadir[6:])
	}
	chainlog.Info("current user data dir is ", "dir", datadir)
	cfg.Log.LogFile = filepath.Join(datadir, cfg.Log.LogFile)
	cfg.BlockChain.DbPath = filepath.Join(datadir, cfg.BlockChain.DbPath)
	cfg.P2P.DbPath = filepath.Join(datadir, cfg.P2P.DbPath)
	cfg.Wallet.DbPath = filepath.Join(datadir, cfg.Wallet.DbPath)
	cfg.Store.DbPath = filepath.Join(datadir, cfg.Store.DbPath)
	return datadir
}
