package drivers

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"errors"
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var blog = log.New("module", "execs.base")

type Driver interface {
	SetDB(dbm.KVDB)
	GetCoinsAccount() *account.DB
	SetLocalDB(dbm.KVDB)
	SetQueryDB(dbm.DB)
	SetExecDriver(execDriver *ExecDrivers)
	GetExecDriver() *ExecDrivers
	GetName() string
	GetActionName(tx *types.Transaction) string
	SetEnv(height, blocktime int64)
	CheckTx(tx *types.Transaction, index int) error
	Exec(tx *types.Transaction, index int) (*types.Receipt, error)
	ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	Query(funcName string, params []byte) (types.Message, error)
	IsFree() bool
}

type DriverBase struct {
	db           dbm.KVDB
	localdb      dbm.KVDB
	querydb      dbm.DB
	height       int64
	blocktime    int64
	child        Driver
	coinsaccount *account.DB
	execDriver   *ExecDrivers
	isFree       bool
}

func (d *DriverBase) SetEnv(height, blocktime int64) {
	d.height = height
	d.blocktime = blocktime
}

func (d *DriverBase) SetIsFree(isFree bool) {
	d.isFree = isFree
}

func (d *DriverBase) IsFree() bool {
	return d.isFree
}

func (d *DriverBase) SetChild(e Driver) {
	d.child = e
}

func (d *DriverBase) SetExecDriver(execDriver *ExecDrivers) {
	d.execDriver = execDriver
}

func (d *DriverBase) GetExecDriver() *ExecDrivers {
	return d.execDriver
}

func (d *DriverBase) GetAddr() string {
	return ExecAddress(d.child.GetName())
}

func (d *DriverBase) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//保存：tx
	hash, result := d.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	//保存: from/to
	txindex := d.getTxIndex(tx, receipt, index)
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
func (d *DriverBase) GetTx(tx *types.Transaction, receipt *types.ReceiptData, index int) ([]byte, *types.TxResult) {
	txhash := tx.Hash()
	//构造txresult 信息保存到db中
	var txresult types.TxResult
	txresult.Height = d.GetHeight()
	txresult.Index = int32(index)
	txresult.Tx = tx
	txresult.Receiptdate = receipt
	txresult.Blocktime = d.GetBlockTime()
	txresult.ActionName = d.child.GetActionName(tx)
	return txhash, &txresult
}

type txIndex struct {
	from      string
	to        string
	heightstr string
	index     *types.ReplyTxInfo
}

//交易中 from/to 的索引
func (d *DriverBase) getTxIndex(tx *types.Transaction, receipt *types.ReceiptData, index int) *txIndex {
	var txIndexInfo txIndex
	var txinf types.ReplyTxInfo
	txinf.Hash = tx.Hash()
	txinf.Height = d.GetHeight()
	txinf.Index = int64(index)

	txIndexInfo.index = &txinf
	heightstr := fmt.Sprintf("%018d", d.GetHeight()*types.MaxTxsPerBlock+int64(index))
	txIndexInfo.heightstr = heightstr

	txIndexInfo.from = account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	txIndexInfo.to = tx.To
	return &txIndexInfo
}

func (d *DriverBase) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//del：tx
	hash, _ := d.GetTx(tx, receipt, index)
	//del: addr index
	txindex := d.getTxIndex(tx, receipt, index)
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

func (d *DriverBase) checkAddress(addr string) error {
	if _, ok := d.execDriver.ExecAddr2Name[addr]; ok {
		return nil
	}
	return account.CheckAddress(addr)
}

func (d *DriverBase) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	//检查ToAddr
	if err := d.checkAddress(tx.To); err != nil {
		return nil, err
	}
	//非coins 或token 模块的 ToAddr 指向合约
	exec := string(tx.Execer)
	if exec != "coins" && exec != "token" && ExecAddress(exec) != tx.To {
		return nil, types.ErrToAddrNotSameToExecAddr
	}
	return nil, nil
}

func (d *DriverBase) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func (d *DriverBase) Query(funcname string, params []byte) (types.Message, error) {
	return nil, types.ErrActionNotSupport
}

func (d *DriverBase) SetDB(db dbm.KVDB) {
	if d.coinsaccount == nil {
		d.coinsaccount = account.NewCoinsAccount()
	}
	d.db = db
	d.coinsaccount.SetDB(db)
}

func (d *DriverBase) GetCoinsAccount() *account.DB {
	if d.coinsaccount == nil {
		d.coinsaccount = account.NewCoinsAccount()
		d.coinsaccount.SetDB(d.db)
	}
	return d.coinsaccount
}

func (d *DriverBase) GetDB() dbm.KVDB {
	return d.db
}

func (d *DriverBase) SetLocalDB(db dbm.KVDB) {
	d.localdb = db
}

func (d *DriverBase) GetLocalDB() dbm.KVDB {
	return d.localdb
}

func (d *DriverBase) SetQueryDB(db dbm.DB) {
	d.querydb = db
}

func (d *DriverBase) GetQueryDB() dbm.DB {
	return d.querydb
}

func (d *DriverBase) GetHeight() int64 {
	return d.height
}

func (d *DriverBase) GetBlockTime() int64 {
	return d.blocktime
}

func (d *DriverBase) GetName() string {
	return "driver"
}

func (d *DriverBase) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

// 通过addr前缀查找本地址参与的所有交易
//查询交易默认放到：coins 中查询
func (d *DriverBase) GetTxsByAddr(addr *types.ReqAddr) (types.Message, error) {
	db := d.GetQueryDB()
	var prefix []byte
	var key []byte
	var txinfos [][]byte
	//取最新的交易hash列表
	if addr.Flag == 0 { //所有的交易hash列表
		prefix = CalcTxAddrHashKey(addr.GetAddr(), "")
	} else if addr.Flag > 0 { //from的交易hash列表
		prefix = CalcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, "")
	} else {
		return nil, errors.New("flag unknown")
	}
	blog.Error("GetTxsByAddr", "height", addr.GetHeight())
	if addr.GetHeight() == -1 {
		list := dbm.NewListHelper(db)
		txinfos = list.IteratorScanFromLast(prefix, addr.Count)
		if len(txinfos) == 0 {
			return nil, errors.New("tx does not exist")
		}
	} else { //翻页查找指定的txhash列表
		blockheight := addr.GetHeight()*types.MaxTxsPerBlock + int64(addr.GetIndex())
		heightstr := fmt.Sprintf("%018d", blockheight)
		if addr.Flag == 0 {
			key = CalcTxAddrHashKey(addr.GetAddr(), heightstr)
		} else if addr.Flag > 0 { //from的交易hash列表
			key = CalcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, heightstr)
		} else {
			return nil, errors.New("flag unknown")
		}
		list := dbm.NewListHelper(db)
		txinfos = list.IteratorScan(prefix, key, addr.Count, addr.Direction)
		if len(txinfos) == 0 {
			return nil, errors.New("tx does not exist")
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
