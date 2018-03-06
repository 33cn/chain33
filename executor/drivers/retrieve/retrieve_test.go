package retrieve

import (
	"context"
	//crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

var conn *grpc.ClientConn
var r *rand.Rand
var c types.GrpcserviceClient
var ErrTest = errors.New("ErrTest")

var addrexec *account.Address

var delayLevel1 int64 = minPeriod + 10
var delayLevel2 int64 = 2*minPeriod + 10

//var delayPeriod = minPeriod + 10

//account:b will be used as backup for both acc A & B in this test
const (
	accountindexA = 0
	accountindexB = 1
	accountindexa = 2
	accountindexb = 3
	accountMax    = 4
)

const (
	defaultAmount  = 1e10
	fee            = 1e6
	retrieveAmount = 1e8
)

const (
	onlyshow     = 0
	onlycheck    = 1
	showandcheck = 2
)

var addr [accountMax]string
var privkey [accountMax]crypto.PrivKey

var currBalanceA int64
var currBalanceB int64
var currBalancea int64
var currBalanceb int64

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	c = types.NewGrpcserviceClient(conn)

	addrexec = account.ExecAddress("retrieve")
}

func TestInitAccount(t *testing.T) {
	fmt.Println("\nTestInitAccount start")
	fmt.Println("*This case is used for initilizing accounts\n*Four accounts A/B/a/b will be set as 1e10/1e10/50*fee/50*fee, 50*fee is used for passing check in mempool\n")
	defer fmt.Println("TestInitAccount end\n")

	var label [accountMax]string
	var params types.ReqWalletImportPrivKey

	privGenesis := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	for index := 0; index < accountMax; index++ {
		addr[index], privkey[index] = genaddress()
		//fmt.Println("privkey: ", common.ToHex(privkey[index].Bytes()))
		label[index] = strconv.Itoa(int(time.Now().UnixNano()))
		params = types.ReqWalletImportPrivKey{Privkey: common.ToHex(privkey[index].Bytes()), Label: label[index]}
		_, err := c.ImportPrivKey(context.Background(), &params)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
		if !showOrCheckAcc(c, addr[index], showandcheck, 0) {
			t.Error(ErrTest)
			return
		}
		time.Sleep(5 * time.Second)
	}

	for index := 0; index <= accountindexB; index++ {
		err := sendtoaddress(c, privGenesis, addr[index], defaultAmount)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
		if !showOrCheckAcc(c, addr[index], showandcheck, defaultAmount) {
			t.Error(ErrTest)
			return
		}
	}

	for index := accountindexa; index <= accountindexb; index++ {
		err := sendtoaddress(c, privGenesis, addr[index], 50*fee)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
		if !showOrCheckAcc(c, addr[index], showandcheck, 50*fee) {
			t.Error(ErrTest)
			return
		}
	}
	currBalanceA = defaultAmount
	currBalanceB = defaultAmount
	currBalancea = 50 * fee
	currBalanceb = 50 * fee
}

func TestRetrieveBackup(t *testing.T) {

	fmt.Println("\nTestRetrieveBackup start")
	fmt.Println("*This case is used for checking backup operation\n*Backup action is done with privkey of account A/B, Backup: A->a, A->b, B->b;\n*currentbalanceA = currentbalanceA- 2*1e8 - 3*fee. currentbalanceB = currentbalanceB - 1e8 - 2*fee\n")
	defer fmt.Println("TestRetrieveBackup end\n")

	//1. step1 account A/B发送余额给合约
	err := sendtoaddress(c, privkey[accountindexA], addrexec.String(), 2*retrieveAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	err = sendtoaddress(c, privkey[accountindexB], addrexec.String(), retrieveAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	if !checkexecAcc(c, addr[accountindexA], showandcheck, 2*retrieveAmount) {
		t.Error(ErrTest)
		return
	}
	if !checkexecAcc(c, addr[accountindexB], showandcheck, retrieveAmount) {
		t.Error(ErrTest)
		return
	}

	//2. a，b为A做备份，b为B做备份
	err = backup(accountindexa, accountindexA, accountindexA, delayLevel1)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	err = backup(accountindexb, accountindexA, accountindexA, delayLevel2)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	err = backup(accountindexb, accountindexB, accountindexB, delayLevel1)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	currBalanceA -= 2*retrieveAmount + 3*fee
	currBalanceB -= retrieveAmount + 2*fee

	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	if !showOrCheckAcc(c, addr[accountindexB], showandcheck, currBalanceB) {
		t.Error(ErrTest)
		return
	}

}

func TestRetrievePrepare(t *testing.T) {
	fmt.Println("\nTestRetrievePrepare start")
	fmt.Println("*This case is used for checking prepare operation\n*Prepare action is done with privkey of account a/b, currBalancea = currBalancea- fee,currBalanceb = currBalanceb -2*fee\n")
	defer fmt.Println("TestRetrievePrepare end\n")

	err := prepare(accountindexa, accountindexA, accountindexa)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	err = prepare(accountindexb, accountindexA, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	err = prepare(accountindexb, accountindexB, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	currBalancea -= fee
	currBalanceb -= 2 * fee

	if !showOrCheckAcc(c, addr[accountindexa], showandcheck, currBalancea) {
		t.Error(ErrTest)
		return
	}
	if !showOrCheckAcc(c, addr[accountindexb], showandcheck, currBalanceb) {
		t.Error(ErrTest)
		return
	}
	if !checkexecAcc(c, addr[accountindexA], showandcheck, 2*retrieveAmount) {
		t.Error(ErrTest)
		return
	}
	if !checkexecAcc(c, addr[accountindexB], showandcheck, retrieveAmount) {
		t.Error(ErrTest)
		return
	}
	return
}

func TestRetrievePerform(t *testing.T) {
	fmt.Println("\nTestRetrievePerform start")
	fmt.Println("*This case is used for checking perform operation\n*perform action is done with privkey of account a/b, b can't withdraw balance as time is not enough\n*currBalancea = currBalancea +2*1e8 -2*fee, currBalanceb = currBalanceb -2*fee\n")
	defer fmt.Println("TestRetrievePerform end\n")

	//not success as time not enough
	err := perform(accountindexb, accountindexB, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	time.Sleep(75 * time.Second)

	err = perform(accountindexa, accountindexA, accountindexa)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	if !checkexecAcc(c, addr[accountindexa], showandcheck, 2*retrieveAmount) {
		t.Error(ErrTest)
		return
	}
	if !checkexecAcc(c, addr[accountindexB], showandcheck, retrieveAmount) {
		t.Error(ErrTest)
		return
	}
	time.Sleep(5 * time.Second)
	//取钱
	err = sendtoaddress(c, privkey[accountindexa], addrexec.String(), -2*retrieveAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	err = sendtoaddress(c, privkey[accountindexb], addrexec.String(), -retrieveAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	currBalancea = currBalancea + 2*retrieveAmount - 2*fee
	currBalanceb = currBalanceb - 2*fee

	if !showOrCheckAcc(c, addr[accountindexa], showandcheck, currBalancea) {
		t.Error(ErrTest)
		return
	}
	if !showOrCheckAcc(c, addr[accountindexb], showandcheck, currBalanceb) {
		t.Error(ErrTest)
		return
	}
}

func TestRetrieveCancel(t *testing.T) {
	fmt.Println("\nTestRetrieveCancel start")
	fmt.Println("*This case is used for checking cancel operation\n*Cancel action is done with privkey of account A/B, although the cancel action for A could succeed, but the balance have been transfered by last action with backup a\n*currBalanceA = currBalanceA - 2*fee currBalanceB = currBalanceB + 1e8 - 2*fee\n")
	defer fmt.Println("TestRetrieveCancel end\n")
	err := cancel(accountindexb, accountindexA, accountindexA)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	err = cancel(accountindexb, accountindexB, accountindexB)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	if !checkexecAcc(c, addr[accountindexA], showandcheck, 0) {
		t.Error(ErrTest)
		return
	}
	if !checkexecAcc(c, addr[accountindexB], showandcheck, retrieveAmount) {
		t.Error(ErrTest)
		return
	}
	time.Sleep(5 * time.Second)

	err = sendtoaddress(c, privkey[accountindexA], addrexec.String(), -retrieveAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexB], addrexec.String(), -retrieveAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	currBalanceA = currBalanceA - 2*fee
	currBalanceB = currBalanceB + retrieveAmount - 2*fee

	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
}

func TestRetrievePerformB(t *testing.T) {
	fmt.Println("\nTestRetrievePerformB start")
	fmt.Println("*This case is used for checking perform operation for B again\n*perform action is done with privkey of account b, b can't withdraw balance as it has been canceled before\n*currBalanceb = currBalanceb -2*fee\n")
	defer fmt.Println("TestRetrievePerformB end\n")

	//failed as canceled before
	err := perform(accountindexb, accountindexB, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	//canceled before, so there is no balance in contract
	if !checkexecAcc(c, addr[accountindexb], showandcheck, 0) {
		t.Error(ErrTest)
		return
	}

	err = sendtoaddress(c, privkey[accountindexb], addrexec.String(), retrieveAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	currBalanceb = currBalanceb - 2*fee
	if !showOrCheckAcc(c, addr[accountindexb], showandcheck, currBalanceb) {
		t.Error(ErrTest)
		return
	}
}

func backup(backupaddrindex int, defaultaddrindex int, privkeyindex int, delayperiod int64) error {
	vbackup := &types.RetrieveAction_Backup{&types.BackupRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex], DelayPeriod: delayperiod}}
	//fmt.Println(vlock)
	transfer := &types.RetrieveAction{Value: vbackup, Ty: types.RetrieveBackup}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	// Contact the server and print out its response.
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func prepare(backupaddrindex int, defaultaddrindex int, privkeyindex int) error {

	vprepare := &types.RetrieveAction_PreRet{&types.PreRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex]}}
	transfer := &types.RetrieveAction{Value: vprepare, Ty: types.RetrievePre}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func perform(backupaddrindex int, defaultaddrindex int, privkeyindex int) error {

	vperform := &types.RetrieveAction_PerfRet{&types.PerformRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex]}}
	transfer := &types.RetrieveAction{Value: vperform, Ty: types.RetrievePerf}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func cancel(backupaddrindex int, defaultaddrindex int, privkeyindex int) error {
	vcancel := &types.RetrieveAction_Cancel{&types.CancelRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex]}}
	transfer := &types.RetrieveAction{Value: vcancel, Ty: types.RetrieveCancel}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func checkexecAcc(c types.GrpcserviceClient, addr string, sorc int, balance int64) bool {
	var addrs []string
	addrs = append(addrs, addr)
	reqbalance := &types.ReqBalance{Addresses: addrs, Execer: "retrieve"}

	accs, err := c.GetBalance(context.Background(), reqbalance)
	if err != nil {
		return false
	}

	//only one input, so we try to find the first one
	if len(accs.Acc) == 0 {
		fmt.Printf("No acc")
		return false
	}

	if sorc != onlycheck {
		fmt.Println("exec acc:", accs.Acc[0])
	}
	if sorc != onlyshow {
		if accs.Acc[0].Balance != balance {
			return false
		}
	}
	/*
		for i := 0; i < len(accs.Acc); i++ {
			acc := accs.Acc[i]
			if acc.Addr
			if sorc != onlycheck {
				fmt.Println(acc)
			}
		}
	*/

	return true
}

func showOrCheckAcc(c types.GrpcserviceClient, addr string, sorc int, balance int64) bool {
	req := &types.ReqNil{}
	accs, err := c.GetAccounts(context.Background(), req)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(accs.Wallets); i++ {
		wallet := accs.Wallets[i]
		if wallet.Acc.Addr == addr {
			if sorc != onlycheck {
				fmt.Println(wallet)
			}
			if sorc != onlyshow {
				if balance != wallet.Acc.Balance {
					fmt.Println(balance, wallet.Acc.Balance)
					return false
				}
			}
			return true
		}
	}
	if sorc != onlyshow {
		return false
	} else {
		return true
	}
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func sendtoaddress(c types.GrpcserviceClient, priv crypto.PrivKey, to string, amount int64) error {
	//defer conn.Close()
	//fmt.Println("sign key privkey: ", common.ToHex(priv.Bytes()))
	if amount > 0 {
		v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
		transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: fee, To: to}
		tx.Nonce = r.Int63()
		tx.Sign(types.SECP256K1, priv)
		// Contact the server and print out its response.
		reply, err := c.SendTransaction(context.Background(), tx)
		if err != nil {
			fmt.Println("err", err)
			return err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return errors.New(string(reply.GetMsg()))
		}
		return nil
	} else {
		v := &types.CoinsAction_Withdraw{&types.CoinsWithdraw{Amount: -amount}}
		withdraw := &types.CoinsAction{Value: v, Ty: types.CoinsActionWithdraw}
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(withdraw), Fee: fee, To: to}
		tx.Nonce = r.Int63()
		tx.Sign(types.SECP256K1, priv)
		// Contact the server and print out its response.
		reply, err := c.SendTransaction(context.Background(), tx)
		if err != nil {
			fmt.Println("err", err)
			return err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return errors.New(string(reply.GetMsg()))
		}
		return nil
	}
}

func getAccounts() (*types.WalletAccounts, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetAccounts(context.Background(), v)
}

func getlastheader() (*types.Header, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
