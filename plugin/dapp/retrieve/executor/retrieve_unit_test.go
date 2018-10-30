package executor

import (
	"errors"
	"fmt"
	"testing"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	backupAddr  string
	defaultAddr string
	backupPriv  crypto.PrivKey
	defaultPriv crypto.PrivKey
	testNormErr error
	retrieve    drivers.Driver
)

func init() {
	backupAddr, backupPriv = genaddress()
	defaultAddr, defaultPriv = genaddress()
	testNormErr = errors.New("Err")
	retrieve = constructRetrieveInstance()
}

func TestExecBackup(t *testing.T) {
	var targetReceipt types.Receipt
	var targetErr error
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructBackupTx()
	receipt, err = retrieve.Exec(tx, 0)

	if !CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
}

func TestExecPrepare(t *testing.T) {
	var targetReceipt types.Receipt
	var targetErr error
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructPrepareTx()
	receipt, err = retrieve.Exec(tx, 0)

	if !CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
}

//timelimit
func TestExecPerform(t *testing.T) {
	var targetReceipt types.Receipt
	var targetErr = rt.ErrRetrievePeriodLimit
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructPerformTx()
	receipt, err = retrieve.Exec(tx, 0)

	if CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
}

func constructRetrieveInstance() drivers.Driver {
	r := newRetrieve()
	r.SetStateDB(NewTestDB())
	return r
}

func ConstructBackupTx() *types.Transaction {

	var delayPeriod int64 = 70
	var fee int64 = 1e6

	vbackup := &rt.RetrieveAction_Backup{&rt.BackupRetrieve{BackupAddress: backupAddr, DefaultAddress: defaultAddr, DelayPeriod: delayPeriod}}
	//fmt.Println(vlock)
	transfer := &rt.RetrieveAction{Value: vbackup, Ty: rt.RetrieveBackup}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: backupAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, defaultPriv)
	return tx
}

func ConstructPrepareTx() *types.Transaction {
	var fee int64 = 1e6
	vprepare := &rt.RetrieveAction_Prepare{&rt.PrepareRetrieve{BackupAddress: backupAddr, DefaultAddress: defaultAddr}}
	transfer := &rt.RetrieveAction{Value: vprepare, Ty: rt.RetrievePreapre}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: backupAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, backupPriv)
	//tx.Sign(types.SECP256K1, defaultPriv)
	return tx
}

func ConstructPerformTx() *types.Transaction {
	var fee int64 = 1e6

	vperform := &rt.RetrieveAction_Perform{&rt.PerformRetrieve{BackupAddress: backupAddr, DefaultAddress: defaultAddr}}
	transfer := &rt.RetrieveAction{Value: vperform, Ty: rt.RetrievePerform}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: backupAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, backupPriv)

	return tx
}

func CompareRetrieveExecResult(rec1 *types.Receipt, err1 error, rec2 *types.Receipt, err2 error) bool {
	if err1 != err2 {
		fmt.Println(err1, err2)
		return false
	}
	if (rec1 == nil) != (rec2 == nil) {
		return false
	}
	if rec1.Ty != rec2.Ty {
		fmt.Println(rec1.Ty, rec2.Ty)
		return false
	}
	return true
}

type TestDB struct {
	db.TransactionDB
	cache map[string][]byte
}

func NewTestDB() *TestDB {
	return &TestDB{cache: make(map[string][]byte)}
}

func (e *TestDB) Get(key []byte) (value []byte, err error) {
	if value, ok := e.cache[string(key)]; ok {
		//elog.Error("getkey", "key", string(key), "value", string(value))
		return value, nil
	}
	return nil, types.ErrNotFound
}

func (e *TestDB) Set(key []byte, value []byte) error {
	//elog.Error("setkey", "key", string(key), "value", string(value))
	e.cache[string(key)] = value
	return nil
}

func (db *TestDB) BatchGet(keys [][]byte) (values [][]byte, err error) {
	return nil, types.ErrNotFound
}

//从数据库中查询数据列表，set 中的cache 更新不会影响这个list
func (l *TestDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	return nil, types.ErrNotFound
}
