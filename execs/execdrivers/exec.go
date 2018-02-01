package execdrivers

//store package store the world - state data
import (
	"fmt"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var elog = log.New("module", "execs")
var zeroHash [32]byte

func ExecAddress(name string) *account.Address {
	return account.ExecAddress(name)
}

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Executer interface {
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
	Query(funcName string, params types.Message) (types.Message, error)
}

var (
	drivers     = make(map[string]Executer)
	execaddress = make(map[string]string)
)

func Register(name string, driver Executer) {
	if driver == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func LoadExecute(name string) (c Executer, err error) {
	c, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}
	return c, nil
}

func RegisterAddress(name string) {
	if len(name) == 0 {
		panic("empty name string")
	}
	if _, dup := execaddress[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	execaddress[ExecAddress(name).String()] = name
}

func IsExecAddress(addr string) bool {
	_, ok := execaddress[addr]
	return ok
}

type StateDB struct {
	cache map[string][]byte
	db    *DataBase
}

func NewStateDB(q *queue.Queue, stateHash []byte) *StateDB {
	return &StateDB{make(map[string][]byte), NewDataBase(q, stateHash)}
}

func (e *StateDB) Get(key []byte) (value []byte, err error) {
	if value, ok := e.cache[string(key)]; ok {
		return value, nil
	}
	value, err = e.db.Get(key)
	if err != nil {
		return nil, err
	}
	e.cache[string(key)] = value
	return value, nil
}

func (e *StateDB) Set(key []byte, value []byte) error {
	e.cache[string(key)] = value
	return nil
}

type LocalDB struct {
	cache map[string][]byte
	db    *DataBaseLocal
}

func NewLocalDB(q *queue.Queue) *LocalDB {
	return &LocalDB{make(map[string][]byte), NewDataBaseLocal(q)}
}

func (e *LocalDB) Get(key []byte) (value []byte, err error) {
	if value, ok := e.cache[string(key)]; ok {
		return value, nil
	}
	value, err = e.db.Get(key)
	if err != nil {
		return nil, err
	}
	e.cache[string(key)] = value
	return value, nil
}

func (e *LocalDB) Set(key []byte, value []byte) error {
	e.cache[string(key)] = value
	return nil
}

type DataBase struct {
	qclient   queue.IClient
	stateHash []byte
}

func NewDataBase(q *queue.Queue, stateHash []byte) *DataBase {
	return &DataBase{q.GetClient(), stateHash}
}

func (db *DataBase) Get(key []byte) (value []byte, err error) {
	query := &types.StoreGet{db.stateHash, [][]byte{key}}
	msg := db.qclient.NewMessage("store", types.EventStoreGet, query)
	db.qclient.Send(msg, true)
	resp, err := db.qclient.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		return nil, types.ErrNotFound
	}
	return value, nil
}

type DataBaseLocal struct {
	qclient queue.IClient
}

func NewDataBaseLocal(q *queue.Queue) *DataBaseLocal {
	return &DataBaseLocal{q.GetClient()}
}

func (db *DataBaseLocal) Get(key []byte) (value []byte, err error) {
	query := &types.LocalDBGet{[][]byte{key}}
	msg := db.qclient.NewMessage("blockchain", types.EventLocalGet, query)
	db.qclient.Send(msg, true)
	resp, err := db.qclient.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.LocalReplyValue).Values[0]
	if value == nil {
		return nil, types.ErrNotFound
	}
	return value, nil
}
