package hashlock

//database opeartion for execs hashlock
import (
	"bytes"
    "encoding/json"
	"fmt"
	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var hlog = log.New("module", "hashlock.db")

const (
	Hashlock_Locked   = 1
	Hashlock_Unlocked = 2
	Hashlock_Sent     = 3
)

type HashlockDB struct {
	types.Hashlock
}

func NewHashlockDB(id []byte, returnWallet string, toAddress string, blocktime int64, amount int64, time int64) *HashlockDB {
	h := &HashlockDB{}
	h.HashlockId = id
	h.ReturnAddress = returnWallet
	h.ToAddress = toAddress
	h.CreateTime = blocktime
	h.Status = Hashlock_Locked
	h.Amount = amount
	h.Frozentime = time
	return h
}

func (h *HashlockDB) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&h.Hashlock)

	kvset = append(kvset, &types.KeyValue{HashlockKey(h.HashlockId), value})
	return kvset
}

func (h *HashlockDB) Save(db dbm.KVDB) {
	set := h.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func HashlockKey(id []byte) (key []byte) {
	key = append(key, []byte("mavl-hashlock-")...)
	key = append(key, id...)
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

	//不存在相同的hashlock，假定采用sha256
	_, err := readHashlock(action.db, hlock.Hash)
	if err != types.ErrNotFound {
		hlog.Error("Hashlocklock", "hlock.Hash repeated", hlock.Hash)
		return nil, types.ErrHashlockReapeathash
	}

	h := NewHashlockDB(hlock.Hash, action.fromaddr, hlock.ToAddress, action.blocktime, hlock.Amount, hlock.Time)
	//冻结子账户资金
	receipt, err := account.ExecFrozen(action.db, action.fromaddr, action.execaddr, hlock.Amount)

	if err != nil {
		hlog.Error("Hashlocklock.Frozen", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", hlock.Amount)
		return nil, err
	}

	h.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, h.GetReceiptLog())
	kv = append(kv, h.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *HashlockAction) Hashlockunlock(unlock *types.HashlockUnlock) (*types.Receipt, error) {

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	hash, err := readHashlock(action.db, common.Sha256(unlock.Secret))
	if err != nil {
		hlog.Error("Hashlockunlock", "unlock.Secret", unlock.Secret)
		return nil, err
	}

	if hash.ReturnAddress != action.fromaddr {
		hlog.Error("Hashlockunlock.Frozen", "action.fromaddr", action.fromaddr)
		return nil, types.ErrHashlockReturnAddrss
	}

	if hash.Status != Hashlock_Locked {
		hlog.Error("Hashlockunlock", "hash.Status", hash.Status)
		return nil, types.ErrHashlockStatus
	}

	if action.blocktime-hash.GetCreateTime() < hash.Frozentime {
		hlog.Error("Hashlockunlock", "action.blocktime-hash.GetCreateTime", action.blocktime-hash.GetCreateTime())
		return nil, types.ErrTime
	}

	//different with typedef in C
	h := &HashlockDB{*hash}
	receipt, errR := account.ExecActive(action.db, h.ReturnAddress, action.execaddr, h.Amount)
	if errR != nil {
		hlog.Error("ExecActive error", "ReturnAddress", h.ReturnAddress, "execaddr", action.execaddr, "amount", h.Amount)
		return nil, errR
	}

	h.Status = Hashlock_Unlocked
	h.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, t.GetReceiptLog())
	kv = append(kv, h.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *HashlockAction) Hashlocksend(send *types.HashlockSend) (*types.Receipt, error) {

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	hash, err := readHashlock(action.db, common.Sha256(send.Secret))
	if err != nil {
		hlog.Error("Hashlocksend", "send.Secret", send.Secret)
		return nil, err
	}

	if hash.Status != Hashlock_Locked {
		hlog.Error("Hashlocksend", "hash.Status", hash.Status)
		return nil, types.ErrHashlockStatus
	}

	if action.blocktime-hash.GetCreateTime() > hash.Frozentime {
		hlog.Error("Hashlocksend", "action.blocktime-hash.GetCreateTime", action.blocktime-hash.GetCreateTime())
		return nil, types.ErrTime
	}

	//different with typedef in C
	h := &HashlockDB{*hash}
	receipt, errR := account.ExecTransferFrozen(action.db, h.ReturnAddress, h.ToAddress, action.execaddr, h.Amount)
	if errR != nil {
		hlog.Error("ExecTransferFrozen error", "ReturnAddress", h.ReturnAddress, "ToAddress", h.ToAddress, "execaddr", action.execaddr, "amount", h.Amount)
		return nil, errR
	}
	h.Status = Hashlock_Sent
	h.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, t.GetReceiptLog())
	kv = append(kv, h.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func readHashlock(db dbm.KVDB, id []byte) (*types.Hashlock, error) {
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

type HashRecv struct {
	HashlockId []byte
	Infomation Hashlockquery
}

type Hashlockquery struct {
	Time   int64
	Status int32
	Amount int64
}

//将Infomation转换成byte类型，使输出为kv模式
func GeHashReciverKV(HashlockId []byte, Infomation Hashlockquery) *types.KeyValue {
	infomation := Hashlockquery{time: Infomation.Time, status: Infomation.Status, amount: Infomation.Amount}
	reciver, err := json.Marsha1(infomation)
	if err == nil {
		fmt.Println("成功转换为json格式")
	} else {
		fmt.Println(err)
	}
	kv := &types.KeyValue{HashlockId, reciver}
	return kv
}

//从db里面根据key获取value,期间需要进行解码
func GetHashReciver(db dbm.KVDB, HashlockId []byte) (*execs.Hashlockquery, error) {
	//reciver := types.Int64{}
	reciver := &execs.Hashlockquery{}
	hashReciver, err := db.Get(HashlockId)
	if err != nil && err != types.ErrNotFound {
		return 0, err
	}
	if len(hashReciver) == 0 {
		return 0, nil
	}
	err = proto.Unmarshal(hashReciver, &reciver)
	if err != nil {
		return 0, err
	}
	return reciver, nil
}

//将HashlockId和Infomation都以key和value形式存入db
func SetHashReciver(db dbm.KVDB, HashlockId []byte, Infomation Hashlockquery) error {
	kv := GeHashReciverKV(HashlockId, Infomation)
	return db.Set(kv.Key, kv.Value)
}

//根据状态值对db中存入的数据进行更改

func UpdateHashReciver(cachedb dbm.KVDB, HashlockId []byte, Infomation Hashlockquery) (*types.KeyValue, error) {
	recv, err := GetHashReciver(cachedb, HashlockId)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	var action types.HashlockAction
	if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		recv.Time = Infomation.Time
		recv.Status = HashlockActionLock
		recv.Amount += Infomation.Amount
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		recv.Time = Infomation.Time
		recv.Status = HashlockActionUnlock
		recv.Amount -= Infomation.Amount
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		recv.Time = Infomation.Time
		recv.Status = HashlockActionSend
		recv.Amount += Infomation.Amount
	}
	SetHashReciver(cachedb, HashlockId, recv)
	//keyvalue
	return GeHashReciverKV(HashlockId, recv), nil
}