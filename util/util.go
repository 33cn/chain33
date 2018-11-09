package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/log/log15"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	rand.Seed(types.Now().UnixNano())
}

var chainlog = log15.New("module", "testnode")

func GetRealExecName(paraName string, name string) string {
	if strings.HasPrefix(name, "user.p.") {
		return name
	}
	return paraName + name
}

// MakeStringToUpper 将给定的in字符串从pos开始一共count个转换为大写字母
func MakeStringToUpper(in string, pos, count int) (out string, err error) {
	l := len(in)
	if pos < 0 || pos >= l || (pos+count) >= l {
		err = errors.New(fmt.Sprintf("Invalid params. in=%s pos=%d count=%d", in, pos, count))
		return
	}
	tmp := []rune(in)
	for n := pos; n < pos+count; n++ {
		tmp[n] = unicode.ToUpper(tmp[n])
	}
	out = string(tmp)
	return
}

// MakeStringToLower 将给定的in字符串从pos开始一共count个转换为小写字母
func MakeStringToLower(in string, pos, count int) (out string, err error) {
	l := len(in)
	if pos < 0 || pos >= l || (pos+count) >= l {
		err = errors.New(fmt.Sprintf("Invalid params. in=%s pos=%d count=%d", in, pos, count))
		return
	}
	tmp := []rune(in)
	for n := pos; n < pos+count; n++ {
		tmp[n] = unicode.ToLower(tmp[n])
	}
	out = string(tmp)
	return
}

func GenNoneTxs(priv crypto.PrivKey, n int64) (txs []*types.Transaction) {
	for i := 0; i < int(n); i++ {
		txs = append(txs, CreateNoneTx(priv))
	}
	return txs
}

func GenCoinsTxs(priv crypto.PrivKey, n int64) (txs []*types.Transaction) {
	to, _ := Genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, CreateCoinsTx(priv, to, types.Coin*(n+1)))
	}
	return txs
}

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

func CreateNoneTx(priv crypto.PrivKey) *types.Transaction {
	return CreateTxWithExecer(priv, "none")
}

func CreateTxWithExecer(priv crypto.PrivKey, execer string) *types.Transaction {
	if execer == "coins" {
		to, _ := Genaddress()
		return CreateCoinsTx(priv, to, types.Coin)
	}
	tx := &types.Transaction{Execer: []byte(execer), Payload: []byte("none")}
	tx.To = address.ExecAddress(execer)
	tx, _ = types.FormatTx(execer, tx)
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func JsonPrint(t *testing.T, input interface{}) {
	data, err := json.MarshalIndent(input, "", "\t")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(data))
}

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

func CreateTxWithTxHeight(priv crypto.PrivKey, to string, amount, expire int64) *types.Transaction {
	tx := createCoinsTx(to, amount)
	tx.Expire = expire + types.TxHeightFlag
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func GenTxsTxHeigt(priv crypto.PrivKey, n, height int64) (txs []*types.Transaction) {
	to, _ := Genaddress()
	for i := 0; i < int(n); i++ {
		tx := CreateTxWithTxHeight(priv, to, types.Coin*(n+1), 20+height)
		txs = append(txs, tx)
	}
	return txs
}

var zeroHash [32]byte

func CreateNoneBlock(priv crypto.PrivKey, n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = -1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = GenNoneTxs(priv, n)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

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
	var maplist = make(map[string]*types.KeyValue)
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
		rdata = append(rdata, &types.ReceiptData{receipt.Ty, receipt.Logs})
		//处理KV
		kvs := receipt.KV
		for _, kv := range kvs {
			if item, ok := maplist[string(kv.Key)]; ok {
				item.Value = kv.Value //更新item 的value
			} else {
				maplist[string(kv.Key)] = kv
				kvset = append(kvset, kv)
			}
		}
	}
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
	//if kvset == nil {
	//	calcHash = prevStateRoot
	//} else {
	calcHash = ExecKVMemSet(client, prevStateRoot, block.Height, kvset, sync)
	//}
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
	//save to db
	//if kvset != nil {
	ExecKVSetCommit(client, block.StateHash)
	//}
	return &detail, deltx, nil
}

func CreateNewBlock(parent *types.Block, txs []*types.Transaction) *types.Block {
	newblock := &types.Block{}
	newblock.Height = parent.Height + 1
	newblock.BlockTime = parent.BlockTime + 1
	newblock.ParentHash = parent.Hash()
	newblock.Txs = append(newblock.Txs, txs...)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

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

func ExecAndCheckBlockCB(qclient queue.Client, block *types.Block, txs []*types.Transaction, cb func(int, *types.ReceiptData) error) (*types.Block, error) {
	block2 := CreateNewBlock(block, txs)
	detail, deltx, err := ExecBlock(qclient, block.StateHash, block2, false, true)
	if err != nil {
		return nil, err
	}
	for _, v := range deltx {
		s, err := types.PBToJson(v)
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
