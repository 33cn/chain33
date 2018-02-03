package execdrivers

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"fmt"
	"sync"

	"code.aliyun.com/chain33/chain33/account"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var blog = log.New("module", "execs.base")

type ExecBase struct {
	db        dbm.KVDB
	localdb   dbm.KVDB
	querydb   dbm.DB
	height    int64
	blocktime int64
	mu        sync.Mutex
	child     Executer
}

func NewExecBase() *ExecBase {
	return &ExecBase{}
}

func (n *ExecBase) SetEnv(height, blocktime int64) {
	n.height = height
	n.blocktime = blocktime
}

func (n *ExecBase) SetChild(e Executer) {
	n.child = e
}

func (n *ExecBase) GetAddr() string {
	return ExecAddress(n.child.GetName()).String()
}

func (n *ExecBase) ExecLocalCommon(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//保存：tx
	hash, result := n.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	//保存: from/to
	txindex := n.GetTxIndex(tx, receipt, index)
	txinfobyte := types.Encode(txindex.index)
	if len(txindex.from) != 0 {
		fromkey1 := calcTxAddrDirHashKey(txindex.from, 1, txindex.heightstr)
		fromkey2 := calcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{fromkey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{fromkey2, txinfobyte})
	}
	if len(txindex.to) != 0 {
		tokey1 := calcTxAddrDirHashKey(txindex.to, 2, txindex.heightstr)
		tokey2 := calcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{tokey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{tokey2, txinfobyte})
	}
	return &set, nil
}

//获取公共的信息
func (n *ExecBase) GetTx(tx *types.Transaction, receipt *types.ReceiptData, index int) ([]byte, *types.TxResult) {
	txhash := tx.Hash()
	//构造txresult 信息保存到db中
	var txresult types.TxResult
	txresult.Height = n.GetHeight()
	txresult.Index = int32(index)
	txresult.Tx = tx
	txresult.Receiptdate = receipt
	txresult.Blocktime = n.GetBlockTime()
	txresult.ActionName = n.child.GetActionName(tx)
	return txhash, &txresult
}

type txIndex struct {
	from      string
	to        string
	heightstr string
	index     *types.ReplyTxInfo
}

//交易中 from/to 的索引
func (n *ExecBase) GetTxIndex(tx *types.Transaction, receipt *types.ReceiptData, index int) *txIndex {
	var txIndexInfo txIndex
	var txinf types.ReplyTxInfo
	txinf.Hash = tx.Hash()
	txinf.Height = n.GetHeight()
	txinf.Index = int64(index)

	txIndexInfo.index = &txinf
	heightstr := fmt.Sprintf("%018d", n.GetHeight()*types.MaxTxsPerBlock+int64(index))
	txIndexInfo.heightstr = heightstr

	txIndexInfo.from = account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	txIndexInfo.to = tx.To
	return &txIndexInfo
}

func (n *ExecBase) ExecDelLocalCommon(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//del：tx
	hash, _ := n.GetTx(tx, receipt, index)
	//del: addr index
	txindex := n.GetTxIndex(tx, receipt, index)
	if len(txindex.from) != 0 {
		fromkey1 := calcTxAddrDirHashKey(txindex.from, 1, txindex.heightstr)
		fromkey2 := calcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{fromkey1, nil})
		set.KV = append(set.KV, &types.KeyValue{fromkey2, nil})
	}
	if len(txindex.to) != 0 {
		tokey1 := calcTxAddrDirHashKey(txindex.to, 2, txindex.heightstr)
		tokey2 := calcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{tokey1, nil})
		set.KV = append(set.KV, &types.KeyValue{tokey2, nil})
	}
	set.KV = append(set.KV, &types.KeyValue{hash, nil})
	return n.child.ExecDelLocal(tx, receipt, index)
}

func (n *ExecBase) ExecCommon(tx *types.Transaction, index int) (*types.Receipt, error) {
	//检查ToAddr
	if err := account.CheckAddress(tx.To); err != nil {
		return nil, err
	}
	//非coins 模块的 ToAddr 指向合约
	exec := string(tx.Execer)
	if exec != "coins" && ExecAddress(exec).String() != tx.To {
		return nil, types.ErrToAddrNotSameToExecAddr
	}
	return nil, nil
}

func (n *ExecBase) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func (n *ExecBase) Query(funcname string, params []byte) (types.Message, error) {
	return nil, types.ErrActionNotSupport
}

func (n *ExecBase) SetDB(db dbm.KVDB) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.db = db
}

func (n *ExecBase) GetDB() dbm.KVDB {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.db
}

func (n *ExecBase) SetLocalDB(db dbm.KVDB) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.localdb = db
}

func (n *ExecBase) GetLocalDB() dbm.KVDB {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.localdb
}

func (n *ExecBase) SetQueryDB(db dbm.DB) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.querydb = db
}

func (n *ExecBase) GetQueryDB() dbm.DB {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.querydb
}

func (n *ExecBase) GetHeight() int64 {
	return n.height
}

func (n *ExecBase) GetBlockTime() int64 {
	return n.blocktime
}
