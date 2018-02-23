package norm

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var nlog = log.New("module", "norm.db")

type NormDB struct {
	types.Norm
}

func NewNormDB(id []byte, blocktime int64, key string, value string) *NormDB {
	n := &NormDB{}
	n.NormId = id
	n.CreateTime = blocktime
	n.Key = key
	n.Value = value
	return n
}

func (n *NormDB) Save(db dbm.KVDB) {
	set := n.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func (n *NormDB) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&n.Norm)
	kvset = append(kvset, &types.KeyValue{NormKey(n.NormId), value})
	return kvset
}

func NormKey(id []byte) (key []byte) {
	key = append(key, []byte("mavl-norm-")...)
	key = append(key, []byte(id)...)
	return key
}

type NormAction struct {
	db        dbm.KVDB
	txhash    []byte
	blocktime int64
	height    int64
	execaddr  string
}

func NewNormAction(db dbm.KVDB, tx *types.Transaction, execaddr string, blocktime, height int64) *NormAction {
	hash := tx.Hash()
	return &NormAction{db, hash, blocktime, height, execaddr}
}

func (action *NormAction) Normput(nput *types.NormPut) (*types.Receipt, error) {
	var kv []*types.KeyValue

	n := NewNormDB(action.txhash, action.blocktime, nput.Key, nput.Value)
	n.Save(action.db)
	kv = append(kv, n.GetKVSet()...)
	kv = append(kv, &types.KeyValue{[]byte(n.Key), []byte(n.Value)})

	receipt := &types.Receipt{types.ExecOk, kv, nil}
	return receipt, nil
}
