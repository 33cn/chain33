package executor

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"testing"

	//. "github.com/smartystreets/goconvey/convey"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var toAddr string
var returnAddr string
var toPriv crypto.PrivKey
var returnPriv crypto.PrivKey

var testNormErr error
var hashlock drivers.Driver

func TestInit(t *testing.T) {
	toAddr, toPriv = genaddress()
	returnAddr, returnPriv = genaddress()
	testNormErr = errors.New("Err")
	hashlock = constructHashlockInstance()
	secret = make([]byte, secretLen)
	crand.Read(secret)
	addrexec = address.ExecAddress("hashlock")
}

func TestExecHashlock(t *testing.T) {

	var targetReceipt types.Receipt
	var targetErr error
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructLockTx()

	acc1 := hashlock.GetCoinsAccount().LoadExecAccount(returnAddr, addrexec)
	acc1.Balance = 100
	hashlock.GetCoinsAccount().SaveExecAccount(addrexec, acc1)

	receipt, err = hashlock.Exec(tx, 0)

	if !CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
}

//timelimit
func TestExecHashunlock(t *testing.T) {

	var targetReceipt types.Receipt
	var targetErr = pty.ErrTime
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructUnlockTx()

	receipt, err = hashlock.Exec(tx, 0)

	if CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
}

func TestExecHashsend(t *testing.T) {

	var targetReceipt types.Receipt
	var targetErr error
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructSendTx()

	receipt, err = hashlock.Exec(tx, 0)

	if !CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
}

func constructHashlockInstance() drivers.Driver {
	h := newHashlock()
	h.SetStateDB(NewTestDB())
	return h
}

func ConstructLockTx() *types.Transaction {

	var lockAmount int64 = 90
	var locktime int64 = 70
	var fee int64 = 1e6

	vlock := &pty.HashlockAction_Hlock{&pty.HashlockLock{Amount: lockAmount, Time: locktime, Hash: common.Sha256(secret), ToAddress: toAddr, ReturnAddress: returnAddr}}
	transfer := &pty.HashlockAction{Value: vlock, Ty: pty.HashlockActionLock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: toAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, returnPriv)

	return tx
}

func ConstructUnlockTx() *types.Transaction {

	var fee int64 = 1e6

	vunlock := &pty.HashlockAction_Hunlock{&pty.HashlockUnlock{Secret: secret}}
	transfer := &pty.HashlockAction{Value: vunlock, Ty: pty.HashlockActionUnlock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: toAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, returnPriv)
	return tx
}

func ConstructSendTx() *types.Transaction {

	var fee int64 = 1e6

	vsend := &pty.HashlockAction_Hsend{&pty.HashlockSend{Secret: secret}}
	transfer := &pty.HashlockAction{Value: vsend, Ty: pty.HashlockActionSend}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: toAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, toPriv)
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
	cache map[string][]byte
	db.TransactionDB
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
