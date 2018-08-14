package drivers

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"errors"
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/address"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var blog = log.New("module", "execs.base")

type Driver interface {
	SetStateDB(dbm.KV)
	GetCoinsAccount() *account.DB
	SetLocalDB(dbm.KVDB)
	GetName() string
	GetActionName(tx *types.Transaction) string
	SetEnv(height, blocktime int64, difficulty uint64)
	CheckTx(tx *types.Transaction, index int) error
	Exec(tx *types.Transaction, index int) (*types.Receipt, error)
	ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	Query(funcName string, params []byte) (types.Message, error)
	IsFree() bool
	SetApi(client.QueueProtocolAPI)
}

type DriverBase struct {
	statedb      dbm.KV
	localdb      dbm.KVDB
	coinsaccount *account.DB
	height       int64
	blocktime    int64
	child        Driver
	isFree       bool
	difficulty   uint64
	api          client.QueueProtocolAPI
}

func (d *DriverBase) SetApi(api client.QueueProtocolAPI) {
	d.api = api
}

func (d *DriverBase) GetApi() client.QueueProtocolAPI {
	return d.api
}

func (d *DriverBase) SetEnv(height, blocktime int64, difficulty uint64) {
	d.height = height
	d.blocktime = blocktime
	d.difficulty = difficulty
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

func (d *DriverBase) GetAddr() string {
	return ExecAddress(d.child.GetName())
}

func (d *DriverBase) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//保存：tx
	kv := d.GetTx(tx, receipt, index)
	set.KV = append(set.KV, kv...)
	//保存: from/to
	txindex := d.getTxIndex(tx, receipt, index)
	txinfobyte := types.Encode(txindex.index)
	if len(txindex.from) != 0 {
		fromkey1 := CalcTxAddrDirHashKey(txindex.from, TxIndexFrom, txindex.heightstr)
		fromkey2 := CalcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{fromkey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{fromkey2, txinfobyte})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.from, 1, true)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}
	if len(txindex.to) != 0 {
		tokey1 := CalcTxAddrDirHashKey(txindex.to, TxIndexTo, txindex.heightstr)
		tokey2 := CalcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{tokey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{tokey2, txinfobyte})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.to, 1, true)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}

	return &set, nil
}

//获取公共的信息
func (d *DriverBase) GetTx(tx *types.Transaction, receipt *types.ReceiptData, index int) []*types.KeyValue {
	txhash := tx.Hash()
	//构造txresult 信息保存到db中
	var txresult types.TxResult
	txresult.Height = d.GetHeight()
	txresult.Index = int32(index)
	txresult.Tx = tx
	txresult.Receiptdate = receipt
	txresult.Blocktime = d.GetBlockTime()
	txresult.ActionName = d.child.GetActionName(tx)
	var kvlist []*types.KeyValue
	kvlist = append(kvlist, &types.KeyValue{Key: types.CalcTxKey(txhash), Value: types.Encode(&txresult)})
	if types.IsEnable("quickIndex") {
		kvlist = append(kvlist, &types.KeyValue{Key: types.CalcTxShortKey(txhash), Value: []byte("1")})
	}
	return kvlist
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

	txIndexInfo.from = address.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	txIndexInfo.to = tx.GetRealToAddr()
	return &txIndexInfo
}

func (d *DriverBase) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//del：tx
	kvdel := d.GetTx(tx, receipt, index)
	for k := range kvdel {
		kvdel[k].Value = nil
	}
	//del: addr index
	txindex := d.getTxIndex(tx, receipt, index)
	if len(txindex.from) != 0 {
		fromkey1 := CalcTxAddrDirHashKey(txindex.from, TxIndexFrom, txindex.heightstr)
		fromkey2 := CalcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{Key: fromkey1, Value: nil})
		set.KV = append(set.KV, &types.KeyValue{Key: fromkey2, Value: nil})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.from, 1, false)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}
	if len(txindex.to) != 0 {
		tokey1 := CalcTxAddrDirHashKey(txindex.to, TxIndexTo, txindex.heightstr)
		tokey2 := CalcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{Key: tokey1, Value: nil})
		set.KV = append(set.KV, &types.KeyValue{Key: tokey2, Value: nil})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.to, 1, false)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}
	set.KV = append(set.KV, kvdel...)
	return &set, nil
}

func (d *DriverBase) checkAddress(addr string) error {
	if IsDriverAddress(addr, d.height) {
		return nil
	}
	return address.CheckAddress(addr)
}

//调用子类的CheckTx, 也可以不调用，实现自己的CheckTx
func (d *DriverBase) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	//to 必须是一个地址
	if err := d.checkAddress(tx.GetRealToAddr()); err != nil {
		return nil, err
	}
	err := d.child.CheckTx(tx, index)
	return nil, err
}

//默认情况下，tx.To 地址指向合约地址
func (d *DriverBase) CheckTx(tx *types.Transaction, index int) error {
	execer := string(tx.Execer)
	if ExecAddress(execer) != tx.To {
		return types.ErrToAddrNotSameToExecAddr
	}
	return nil
}

func (d *DriverBase) Query(funcname string, params []byte) (types.Message, error) {
	return nil, types.ErrActionNotSupport
}

func (d *DriverBase) SetStateDB(db dbm.KV) {
	if d.coinsaccount == nil {
		//log.Error("new CoinsAccount")
		d.coinsaccount = account.NewCoinsAccount()
	}
	d.statedb = db
	d.coinsaccount.SetDB(db)
}

func (d *DriverBase) GetCoinsAccount() *account.DB {
	if d.coinsaccount == nil {
		//log.Error("new CoinsAccount")
		d.coinsaccount = account.NewCoinsAccount()
		d.coinsaccount.SetDB(d.statedb)
	}
	return d.coinsaccount
}

func (d *DriverBase) GetStateDB() dbm.KV {
	return d.statedb
}

func (d *DriverBase) SetLocalDB(db dbm.KVDB) {
	d.localdb = db
}

func (d *DriverBase) GetLocalDB() dbm.KVDB {
	return d.localdb
}

func (d *DriverBase) GetHeight() int64 {
	return d.height
}

func (d *DriverBase) GetBlockTime() int64 {
	return d.blocktime
}

func (d *DriverBase) GetDifficulty() uint64 {
	return d.difficulty
}

func (d *DriverBase) GetName() string {
	return "driver"
}

func (d *DriverBase) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

func (d *DriverBase) CheckSignatureData(tx *types.Transaction, index int) bool {
	return true
}

//通过addr前缀查找本地址参与的所有交易
//查询交易默认放到：coins 中查询
func (d *DriverBase) GetTxsByAddr(addr *types.ReqAddr) (types.Message, error) {
	db := d.GetLocalDB()
	var prefix []byte
	var key []byte
	var txinfos [][]byte
	var err error
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
		txinfos, err = db.List(prefix, nil, addr.Count, addr.GetDirection())
		if err != nil {
			return nil, err
		}
		if len(txinfos) == 0 {
			return nil, errors.New("tx does not exist")
		}
	} else { //翻页查找指定的txhash列表
		blockheight := addr.GetHeight()*types.MaxTxsPerBlock + addr.GetIndex()
		heightstr := fmt.Sprintf("%018d", blockheight)
		if addr.Flag == 0 {
			key = CalcTxAddrHashKey(addr.GetAddr(), heightstr)
		} else if addr.Flag > 0 { //from的交易hash列表
			key = CalcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, heightstr)
		} else {
			return nil, errors.New("flag unknown")
		}
		txinfos, err = db.List(prefix, key, addr.Count, addr.Direction)
		if err != nil {
			return nil, err
		}
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

//查询指定prefix的key数量，用于统计
func (d *DriverBase) GetPrefixCount(Prefix *types.ReqKey) (types.Message, error) {
	var counts types.Int64
	db := d.GetLocalDB()
	counts.Data = db.PrefixCount(Prefix.Key)
	return &counts, nil
}

//查询指定地址参与的交易计数，用于统计
func (d *DriverBase) GetAddrTxsCount(reqkey *types.ReqKey) (types.Message, error) {
	var counts types.Int64
	db := d.GetLocalDB()
	TxsCount, err := db.Get(reqkey.Key)
	if err != nil && err != types.ErrNotFound {
		blog.Error("GetAddrTxsCount!", "err:", err)
		counts.Data = 0
		return &counts, nil
	}
	if len(TxsCount) == 0 {
		blog.Error("GetAddrTxsCount TxsCount is nil!")
		counts.Data = 0
		return &counts, nil
	}
	err = types.Decode(TxsCount, &counts)
	if err != nil {
		blog.Error("GetAddrTxsCount!", "err:", err)
		counts.Data = 0
		return &counts, nil
	}
	return &counts, nil
}
