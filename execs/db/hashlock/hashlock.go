package hashlock

//database opeartion for execs hashlock
import (
	"bytes"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var hlog = log.New("module", "hashlock.db")

type Hashlock struct {
	types.Hashlock
}

func NewHashlock(id string, returnWallet string, blocktime int64, isGenesis bool, amount int64, time int64) *Hashlock {
	h := &Hashlock{}
	h.HashlockId = id
	h.ReturnAddress = returnWallet
	h.CreateTime = blocktime
	h.Status = 1
	h.IsGenesis = isGenesis
	h.Amount = amount
	h.Frozentime = time
	return h
}

func (h *Hashlock) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&h.Hashlock)
	//if t.Status == 3 {
	//	value = nil //delete ticket
	//}
	kvset = append(kvset, &types.KeyValue{HashlockKey(h.HashlockId), value})
	return kvset
}

func (h *Hashlock) Save(db dbm.KVDB) {
	set := h.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func HashlockKey(id string) (key []byte) {
	key = append(key, []byte("mavl-hashlock-")...)
	key = append(key, []byte(id)...)
	return key
}

type HashlockAction struct {
	db        dbm.KVDB
	txhash    []byte
	fromaddr  string
	blocktime int64
	height    int64
	execaddr  string
}

func NewHashlockAction(db dbm.KVDB, tx *types.Transaction, execaddr string, blocktime, height int64) *HashlockAction {
	hash := tx.Hash()
	fromaddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &HashlockAction{db, hash, fromaddr, blocktime, height, execaddr}
}

func (action *HashlockAction) Hashlocklock(hlock *types.HashlockLock) (*types.Receipt, error) {

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	id := common.ToHex(action.txhash)
	h := NewHashlock(id, action.fromaddr, action.blocktime, false, hlock.Amount, hlock.Time)
	//冻结子账户资金
	receipt, err := account.ExecFrozen(action.db, action.fromaddr, action.execaddr, hlock.Amount)
	if err != nil {
		hlog.Error("Hashlocklock.Frozen", "addr", action.fromaddr, "execaddr", action.execaddr)
		return nil, err
	}
	h.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, h.GetReceiptLog())
	kv = append(kv, h.GetKVSet()...)

	//receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *HashlockAction) Hashlockunlock(unlock *types.HashlockUnlock) (*types.Receipt, error) {

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var valid bool = false

	hash, err := readHashlock(action.db, unlock.UnlockId)
	if err != nil {
		hlog.Error("Hashlocklock.Frozen", "unlockId", unlock.UnlockId)
		return nil, err
	}

	if !hash.IsGenesis {
		if checksecret(unlock.Secret, unlock.Hash) {
			valid = true
		} else if action.blocktime-hash.GetCreateTime() > hash.Frozentime {
			valid = true
		}
	}

	if !valid {
		return nil, types.ErrTime
	}

	//different with typedef in C
	h := &Hashlock{*hash}
	receipt, _ := account.ExecActive(action.db, h.ReturnAddress, action.execaddr, h.Amount)
	h.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, t.GetReceiptLog())
	kv = append(kv, h.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func readHashlock(db dbm.KVDB, id string) (*types.Hashlock, error) {
	data, err := db.Get(HashlockKey(id))
	if err != nil {
		return nil, err
	}
	var hashlock types.Hashlock
	//decode
	err = types.Decode(data, &hashlock)
	if err != nil {
		return nil, err
	}
	return &hashlock, nil
}

func checksecret(secret []byte, hashresult []byte) bool {
	return bytes.Equal(common.Sha256(secret), hashresult)
}
