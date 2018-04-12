package hashlock

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	//. "github.com/smartystreets/goconvey/convey"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	db "gitlab.33.cn/chain33/chain33/common/db"
	//"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var random *rand.Rand

var toAddr string
var returnAddr string
var toPriv crypto.PrivKey
var returnPriv crypto.PrivKey

var testNormErr error
var hashlock *Hashlock

func TestInit(t *testing.T) {
	toAddr, toPriv = genaddress()
	returnAddr, returnPriv = genaddress()
	testNormErr = errors.New("Err")
	hashlock = constructHashlockInstance()
	secret = make([]byte, secretLen)
	crand.Read(secret)
	addrexec = account.ExecAddress("hashlock")
}

func TestExecHashlock(t *testing.T) {

	var targetReceipt types.Receipt
	var targetErr error
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructLockTx()

	acc1 := hashlock.GetCoinsAccount().LoadExecAccount(returnAddr, addrexec.String())
	acc1.Balance = 100
	hashlock.GetCoinsAccount().SaveExecAccount(addrexec.String(), acc1)

	receipt, err = hashlock.Exec(tx, 0)

	if !CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
	return
}

//timelimit
func TestExecHashunlock(t *testing.T) {

	var targetReceipt types.Receipt
	var targetErr = types.ErrTime
	var receipt *types.Receipt
	var err error
	targetReceipt.Ty = 2
	tx := ConstructUnlockTx()

	receipt, err = hashlock.Exec(tx, 0)

	if CompareRetrieveExecResult(receipt, err, &targetReceipt, targetErr) {
		t.Error(testNormErr)
	}
	return
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
	return
}

func constructHashlockInstance() *Hashlock {
	h := newHashlock()
	h.SetDB(NewTestDB())
	return h
}

func ConstructLockTx() *types.Transaction {

	var lockAmount int64 = 90
	var locktime int64 = 70
	var fee int64 = 1e6

	vlock := &types.HashlockAction_Hlock{&types.HashlockLock{Amount: lockAmount, Time: locktime, Hash: common.Sha256(secret), ToAddress: toAddr, ReturnAddress: returnAddr}}
	transfer := &types.HashlockAction{Value: vlock, Ty: types.HashlockActionLock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: toAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, returnPriv)

	return tx
}

func ConstructUnlockTx() *types.Transaction {

	var fee int64 = 1e6

	vunlock := &types.HashlockAction_Hunlock{&types.HashlockUnlock{Secret: secret}}
	transfer := &types.HashlockAction{Value: vunlock, Ty: types.HashlockActionUnlock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: toAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, returnPriv)
	return tx
}

func ConstructSendTx() *types.Transaction {

	var fee int64 = 1e6

	vsend := &types.HashlockAction_Hsend{&types.HashlockSend{Secret: secret}}
	transfer := &types.HashlockAction{Value: vsend, Ty: types.HashlockActionSend}
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
}

func NewTestDB() *TestDB {
	return &TestDB{make(map[string][]byte)}
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

func (e *TestDB) Iterator(prefix []byte, reserver bool) db.Iterator {
	return nil
}
