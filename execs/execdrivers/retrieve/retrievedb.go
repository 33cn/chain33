package retrieve

import (
	//"bytes"

	"code.aliyun.com/chain33/chain33/account"
	//"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	//log "github.com/inconshreveable/log15"
)

const (
	Retrieve_Prepared  = 1
	Retrieve_Performed = 2
)

type RetrieveDB struct {
	types.Retrieve
}

func NewRetrieveDB(address string, blocktime int64, amount int64, time int64) *RetrieveDB {
	r := &RetrieveDB{}
	r.Address = address
	r.CreateTime = blocktime
	r.Amount = amount
	r.Frozentime = time
	r.Status = Retrieve_Prepared
	return r
}

func (r *RetrieveDB) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&r.Retrieve)
	kvset = append(kvset, &types.KeyValue{RetrieveKey(r.Address), value})
	return kvset
}

func (r *RetrieveDB) Save(db dbm.KVDB) {
	set := r.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func RetrieveKey(address string) (key []byte) {
	key = append(key, []byte("mavl-retrieve-")...)
	key = append(key, address...)
	return key
}

type RetrieveAction struct {
	db        dbm.KVDB
	txhash    []byte
	fromaddr  string
	blocktime int64
	height    int64
	execaddr  string
}

func NewRetrieveAcction(db dbm.KVDB, tx *types.Transaction, execaddr string, blocktime, height int64) *RetrieveAction {
	hash := tx.Hash()
	fromaddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &RetrieveAction{db, hash, fromaddr, blocktime, height, execaddr}
}

func (action *RetrieveAction) RetrievePrepare(preRet *types.PreRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var r *RetrieveDB

	//一个地址，在prepare状态，不允许再次prepare
	retrieve, err := readRetrieve(action.db, preRet.Address)
	if err != nil && err != types.ErrNotFound {
		rlog.Error("RetrievePrepare", "readRetrieve err")
		return nil, err
	} else if err == types.ErrNotFound {
		rlog.Debug("RetrievePrepare", "newAddress", preRet.Address)
	} else if retrieve != nil {
		if retrieve.Status != Retrieve_Prepared {
			rlog.Debug("RetrievePrepare", "reusedAddress", preRet.Address)
		} else {
			rlog.Error("RetrievePrepare", "address repeated", preRet.Address)
			return nil, types.ErrRetrieveRepeatAddress
		}
	}

	if retrieve != nil {
		retrieve.CreateTime = action.blocktime
		retrieve.Amount = preRet.Amount
		retrieve.Frozentime = preRet.BackoffPeriod
		retrieve.Status = Retrieve_Prepared
		r = &RetrieveDB{*retrieve}
	} else {
		r = NewRetrieveDB(preRet.Address, action.blocktime, preRet.Amount, preRet.BackoffPeriod)
	}

	receipt, err = account.ExecFrozen(action.db, action.fromaddr, action.execaddr, preRet.Amount)

	if err != nil {
		rlog.Error("RetrievePrepare.Frozen", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", preRet.Amount)
		return nil, err
	}

	r.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, h.GetReceiptLog())
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *RetrieveAction) RetrievePerform(perfRet *types.PerformRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt

	retrieve, err := readRetrieve(action.db, perfRet.Address)
	if err != nil {
		rlog.Error("RetrievePerform", "perfRet.Address", perfRet.Address)
		return nil, err
	}

	//也许还没有这么严重
	//if retrieve == nil {
	//	panic("RetrievePerform")
	//}

	if retrieve.Status != Retrieve_Prepared {
		rlog.Error("RetrievePerform", "address repeated", perfRet.Address)
		return nil, types.ErrRetrieveRepeatAddress
	}

	if perfRet.Factor > 0 {
		if action.blocktime-retrieve.GetCreateTime() < perfRet.Factor*retrieve.Frozentime {
			rlog.Error("RetrievePerform", "action.blocktime-retrieve.GetCreateTime", action.blocktime-retrieve.GetCreateTime())
			return nil, types.ErrTime
		}
	}

	r := &RetrieveDB{*retrieve}

	receipt, err = account.ExecTransferFrozen(action.db, action.fromaddr, r.Address, action.execaddr, r.Amount)
	if err != nil {
		return nil, err
	}
	r.Status = Retrieve_Performed
	r.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, t.GetReceiptLog())
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func readRetrieve(db dbm.KVDB, address string) (*types.Retrieve, error) {
	data, err := db.Get(RetrieveKey(address))
	if err != nil {
		return nil, err
	}
	var retrieve types.Retrieve
	//decode
	err = types.Decode(data, &retrieve)
	if err != nil {
		return nil, err
	}
	return &retrieve, nil
}
