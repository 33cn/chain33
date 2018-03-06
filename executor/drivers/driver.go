package drivers

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"errors"
	"fmt"
	"sync"

	"code.aliyun.com/chain33/chain33/account"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var blog = log.New("module", "execs.base")

type Driver interface {
	SetDB(dbm.KVDB)
	SetLocalDB(dbm.KVDB)
	SetQueryDB(dbm.DB)
	GetName() string
	GetActionName(tx *types.Transaction) string
	SetEnv(height, blocktime int64)
	CheckTx(tx *types.Transaction, index int) error
	Exec(tx *types.Transaction, index int) (*types.Receipt, error)
	ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	Query(funcName string, params []byte) (types.Message, error)
}

type DriverBase struct {
	db        dbm.KVDB
	localdb   dbm.KVDB
	querydb   dbm.DB
	height    int64
	blocktime int64
	mu        sync.Mutex
	child     Driver
}

func (n *DriverBase) SetEnv(height, blocktime int64) {
	n.height = height
	n.blocktime = blocktime
}

func (n *DriverBase) SetChild(e Driver) {
	n.child = e
}

func (n *DriverBase) GetAddr() string {
	return ExecAddress(n.child.GetName()).String()
}

func (n *DriverBase) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//保存：tx
	hash, result := n.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	//保存: from/to
	txindex := n.GetTxIndex(tx, receipt, index)
	txinfobyte := types.Encode(txindex.index)
	if len(txindex.from) != 0 {
		fromkey1 := CalcTxAddrDirHashKey(txindex.from, 1, txindex.heightstr)
		fromkey2 := CalcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{fromkey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{fromkey2, txinfobyte})
	}
	if len(txindex.to) != 0 {
		tokey1 := CalcTxAddrDirHashKey(txindex.to, 2, txindex.heightstr)
		tokey2 := CalcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{tokey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{tokey2, txinfobyte})
	}
	return &set, nil
}

//获取公共的信息
func (n *DriverBase) GetTx(tx *types.Transaction, receipt *types.ReceiptData, index int) ([]byte, *types.TxResult) {
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
func (n *DriverBase) GetTxIndex(tx *types.Transaction, receipt *types.ReceiptData, index int) *txIndex {
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

func (n *DriverBase) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//del：tx
	hash, _ := n.GetTx(tx, receipt, index)
	//del: addr index
	txindex := n.GetTxIndex(tx, receipt, index)
	if len(txindex.from) != 0 {
		fromkey1 := CalcTxAddrDirHashKey(txindex.from, 1, txindex.heightstr)
		fromkey2 := CalcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{fromkey1, nil})
		set.KV = append(set.KV, &types.KeyValue{fromkey2, nil})
	}
	if len(txindex.to) != 0 {
		tokey1 := CalcTxAddrDirHashKey(txindex.to, 2, txindex.heightstr)
		tokey2 := CalcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{tokey1, nil})
		set.KV = append(set.KV, &types.KeyValue{tokey2, nil})
	}
	set.KV = append(set.KV, &types.KeyValue{hash, nil})
	return &set, nil
}

func (n *DriverBase) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
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

func (n *DriverBase) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func (n *DriverBase) Query(funcname string, params []byte) (types.Message, error) {
	return nil, types.ErrActionNotSupport
}

func (n *DriverBase) SetDB(db dbm.KVDB) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.db = db
}

func (n *DriverBase) GetDB() dbm.KVDB {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.db
}

func (n *DriverBase) SetLocalDB(db dbm.KVDB) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.localdb = db
}

func (n *DriverBase) GetLocalDB() dbm.KVDB {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.localdb
}

func (n *DriverBase) SetQueryDB(db dbm.DB) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.querydb = db
}

func (n *DriverBase) GetQueryDB() dbm.DB {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.querydb
}

func (n *DriverBase) GetHeight() int64 {
	return n.height
}

func (n *DriverBase) GetBlockTime() int64 {
	return n.blocktime
}

func (n *DriverBase) GetName() string {
	return "driver"
}

func (n *DriverBase) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

// 通过addr前缀查找本地址参与的所有交易
//查询交易默认放到：coins 中查询
func (n *DriverBase) GetTxsByAddr(addr *types.ReqAddr) (types.Message, error) {
	db := n.GetQueryDB()
	var prefix []byte
	var key []byte
	var txinfos [][]byte
	//取最新的交易hash列表
	if addr.Flag == 0 { //所有的交易hash列表
		prefix = CalcTxAddrHashKey(addr.GetAddr(), "")
	} else if addr.Flag > 0 { //from的交易hash列表
		prefix = CalcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, "")
	} else {
		return nil, errors.New("Flag unknow!")
	}
	blog.Error("GetTxsByAddr", "height", addr.GetHeight())
	if addr.GetHeight() == -1 {
		txinfos = db.IteratorScanFromLast(prefix, addr.Count, addr.Direction)
		if len(txinfos) == 0 {
			return nil, errors.New("does not exist tx!")
		}
	} else { //翻页查找指定的txhash列表
		blockheight := addr.GetHeight()*types.MaxTxsPerBlock + int64(addr.GetIndex())
		heightstr := fmt.Sprintf("%018d", blockheight)
		if addr.Flag == 0 {
			key = CalcTxAddrHashKey(addr.GetAddr(), heightstr)
		} else if addr.Flag > 0 { //from的交易hash列表
			key = CalcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, heightstr)
		} else {
			return nil, errors.New("Flag unknow!")
		}
		txinfos = db.IteratorScan(prefix, key, addr.Count, addr.Direction)
		if len(txinfos) == 0 {
			return nil, errors.New("does not exist tx!")
		}
	}
	var replyTxInfos types.ReplyTxInfos
	replyTxInfos.TxInfos = make([]*types.ReplyTxInfo, len(txinfos))
	for index, txinfobyte := range txinfos {
		var replyTxInfo types.ReplyTxInfo
		err := types.Decode(txinfobyte, &replyTxInfo)
		if err != nil {
			blog.Error("GetTxsByAddr proto.Unmarshal!", "err:", err)
			return nil, err
		}
		replyTxInfos.TxInfos[index] = &replyTxInfo
	}
	return &replyTxInfos, nil
}
