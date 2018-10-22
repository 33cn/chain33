package executor

import (
	"context"
	//crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var (
	conn        *grpc.ClientConn
	r           *rand.Rand
	c           types.Chain33Client
	ErrTest     = errors.New("ErrTest")
	addrexec    string
	delayLevel1 = minPeriod + 10
	delayLevel2 = 5*minPeriod + 10
)

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
	r = rand.New(rand.NewSource(types.Now().UnixNano()))
	c = types.NewChain33Client(conn)

	addrexec = address.ExecAddress("retrieve")
}

func estInitAccount(t *testing.T) {
	fmt.Println("\nTestInitAccount start")
	fmt.Println("*This case is used for initilizing accounts\n*Four accounts A/B/a/b will be set as 1e10/1e10/50*fee/50*fee, 50*fee is used for passing check in mempool")
	defer fmt.Println("TestInitAccount end")

	var label [accountMax]string
	var params types.ReqWalletImportPrivkey
	var hashes [][]byte

	privGenesis := getprivkey("11A61A97B3A89E614419BACF735DA585BB3F745A5DF05BF93A63AE6F24A3712E")
	for index := 0; index < accountMax; index++ {
		addr[index], privkey[index] = genaddress()
		//fmt.Println("privkey: ", common.ToHex(privkey[index].Bytes()))
		label[index] = strconv.Itoa(int(types.Now().UnixNano()))
		params = types.ReqWalletImportPrivkey{Privkey: common.ToHex(privkey[index].Bytes()), Label: label[index]}
		_, err := c.ImportPrivkey(context.Background(), &params)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
	}

	for index := 0; index <= accountindexB; index++ {
		txhash, err := sendtoaddress(c, privGenesis, addr[index], defaultAmount)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		hashes = append(hashes, txhash)
	}

	for index := accountindexa; index <= accountindexb; index++ {
		txhash, err := sendtoaddress(c, privGenesis, addr[index], 50*fee)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		hashes = append(hashes, txhash)
	}

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

	for index := 0; index <= accountindexB; index++ {
		if !showOrCheckAcc(c, addr[index], showandcheck, defaultAmount) {
			t.Error(ErrTest)
			return
		}
	}

	for index := accountindexa; index <= accountindexb; index++ {
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

func estRetrieveBackup(t *testing.T) {

	fmt.Println("\nTestRetrieveBackup start")
	fmt.Println("*This case is used for checking backup operation\n*Backup action is done with privkey of account A/B, Backup: A->a, A->b, B->b;\n*currentbalanceA = currentbalanceA- 2*1e8 - 3*fee. currentbalanceB = currentbalanceB - 1e8 - 2*fee")
	defer fmt.Println("TestRetrieveBackup end")

	var hashes [][]byte

	//1. step1 account A/B发送余额给合约
	txHash, err := sendtoaddress(c, privkey[accountindexA], addrexec, 2*retrieveAmount)
	if err != nil {
		panic(err)
	}
	hashes = append(hashes, txHash)

	txHash, err = sendtoaddress(c, privkey[accountindexB], addrexec, retrieveAmount)
	if err != nil {
		panic(err)
	}
	hashes = append(hashes, txHash)

	if !waitTxs(hashes) {
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

	//2. a，b为A做备份，b为B做备份
	txHash, err = backup(accountindexa, accountindexA, accountindexA, delayLevel1)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txHash)

	txHash, err = backup(accountindexb, accountindexA, accountindexA, delayLevel2)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txHash)

	txHash, err = backup(accountindexb, accountindexB, accountindexB, delayLevel2)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txHash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

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

func estRetrievePrepare(t *testing.T) {
	fmt.Println("\nTestRetrievePrepare start")
	fmt.Println("*This case is used for checking prepare operation\n*Prepare action is done with privkey of account a/b, currBalancea = currBalancea- fee,currBalanceb = currBalanceb -2*fee")
	defer fmt.Println("TestRetrievePrepare end")
	var hashes [][]byte

	txhash, err := prepare(accountindexa, accountindexA, accountindexa)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	txhash, err = prepare(accountindexb, accountindexA, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	txhash, err = prepare(accountindexb, accountindexB, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

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
}

func estRetrievePerform(t *testing.T) {
	fmt.Println("\nTestRetrievePerform start")
	fmt.Println("*This case is used for checking perform operation\n*perform action is done with privkey of account a/b, b can't withdraw balance as time is not enough\n*currBalancea = currBalancea +2*1e8 -2*fee, currBalanceb = currBalanceb -2*fee")
	defer fmt.Println("TestRetrievePerform end")
	var hashes [][]byte

	//not success as time not enough
	txhash, err := perform(accountindexb, accountindexB, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	time.Sleep(75 * time.Second)

	txhash, err = perform(accountindexa, accountindexA, accountindexa)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

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
	txhash, err = sendtoaddress(c, privkey[accountindexa], addrexec, -2*retrieveAmount)
	if err != nil {
		panic(err)
	}
	hashes = append(hashes, txhash)

	txhash, err = sendtoaddress(c, privkey[accountindexb], addrexec, -retrieveAmount)
	if err != nil {
		panic(err)
	}
	hashes = append(hashes, txhash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

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

func estRetrieveCancel(t *testing.T) {
	fmt.Println("\nTestRetrieveCancel start")
	fmt.Println("*This case is used for checking cancel operation\n*Cancel action is done with privkey of account A/B, although the cancel action for A could succeed, but the balance have been transfered by last action with backup a\n*currBalanceA = currBalanceA - 2*fee currBalanceB = currBalanceB + 1e8 - 2*fee")
	defer fmt.Println("TestRetrieveCancel end")
	var hashes [][]byte

	txhash, err := cancel(accountindexb, accountindexA, accountindexA)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	txhash, err = cancel(accountindexb, accountindexB, accountindexB)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

	if !checkexecAcc(c, addr[accountindexA], showandcheck, 0) {
		t.Error(ErrTest)
		return
	}
	if !checkexecAcc(c, addr[accountindexB], showandcheck, retrieveAmount) {
		t.Error(ErrTest)
		return
	}
	time.Sleep(5 * time.Second)

	txhash, err = sendtoaddress(c, privkey[accountindexA], addrexec, -retrieveAmount)
	if err != nil {
		panic(err)
	}
	hashes = append(hashes, txhash)

	txhash, err = sendtoaddress(c, privkey[accountindexB], addrexec, -retrieveAmount)
	if err != nil {
		panic(err)
	}
	hashes = append(hashes, txhash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

	currBalanceA = currBalanceA - 2*fee
	currBalanceB = currBalanceB + retrieveAmount - 2*fee

	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
}

func estRetrievePerformB(t *testing.T) {
	fmt.Println("\nTestRetrievePerformB start")
	fmt.Println("*This case is used for checking perform operation for B again\n*perform action is done with privkey of account b, b can't withdraw balance as it has been canceled before\n*currBalanceb = currBalanceb -2*fee")
	defer fmt.Println("TestRetrievePerformB end")
	var hashes [][]byte

	//failed as canceled before
	txhash, err := perform(accountindexb, accountindexB, accountindexb)
	if err != nil {
		t.Error(err)
		return
	}
	hashes = append(hashes, txhash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

	//canceled before, so there is no balance in contract
	if !checkexecAcc(c, addr[accountindexb], showandcheck, 0) {
		t.Error(ErrTest)
		return
	}

	txhash, err = sendtoaddress(c, privkey[accountindexb], addrexec, retrieveAmount)
	if err != nil {
		panic(err)
	}
	hashes = append(hashes, txhash)

	if !waitTxs(hashes) {
		t.Error(ErrTest)
		return
	}

	currBalanceb = currBalanceb - 2*fee
	if !showOrCheckAcc(c, addr[accountindexb], showandcheck, currBalanceb) {
		t.Error(ErrTest)
		return
	}
}

func backup(backupaddrindex int, defaultaddrindex int, privkeyindex int, delayperiod int64) ([]byte, error) {
	vbackup := &rt.RetrieveAction_Backup{&rt.BackupRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex], DelayPeriod: delayperiod}}
	//fmt.Println(vlock)
	transfer := &rt.RetrieveAction{Value: vbackup, Ty: rt.RetrieveBackup}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	// Contact the server and print out its response.
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func prepare(backupaddrindex int, defaultaddrindex int, privkeyindex int) ([]byte, error) {
	vprepare := &rt.RetrieveAction_Prepare{&rt.PrepareRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex]}}
	transfer := &rt.RetrieveAction{Value: vprepare, Ty: rt.RetrievePreapre}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func perform(backupaddrindex int, defaultaddrindex int, privkeyindex int) ([]byte, error) {
	vperform := &rt.RetrieveAction_Perform{&rt.PerformRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex]}}
	transfer := &rt.RetrieveAction{Value: vperform, Ty: rt.RetrievePerform}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func cancel(backupaddrindex int, defaultaddrindex int, privkeyindex int) ([]byte, error) {
	vcancel := &rt.RetrieveAction_Cancel{&rt.CancelRetrieve{BackupAddress: addr[backupaddrindex], DefaultAddress: addr[defaultaddrindex]}}
	transfer := &rt.RetrieveAction{Value: vcancel, Ty: rt.RetrieveCancel}
	tx := &types.Transaction{Execer: []byte("retrieve"), Payload: types.Encode(transfer), Fee: fee, To: addr[backupaddrindex]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[privkeyindex])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func checkexecAcc(c types.Chain33Client, addr string, sorc int, balance int64) bool {
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

func showOrCheckAcc(c types.Chain33Client, addr string, sorc int, balance int64) bool {
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
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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

func sendtoaddress(c types.Chain33Client, priv crypto.PrivKey, to string, amount int64) ([]byte, error) {
	//defer conn.Close()
	//fmt.Println("sign key privkey: ", common.ToHex(priv.Bytes()))
	if amount > 0 {
		v := &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: amount}}
		transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: fee, To: to}
		tx.Nonce = r.Int63()
		tx.Sign(types.SECP256K1, priv)
		// Contact the server and print out its response.
		reply, err := c.SendTransaction(context.Background(), tx)
		if err != nil {
			fmt.Println("err", err)
			return nil, err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return nil, errors.New(string(reply.GetMsg()))
		}
		return tx.Hash(), nil
	} else {
		v := &cty.CoinsAction_Withdraw{&types.AssetsWithdraw{Amount: -amount}}
		withdraw := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionWithdraw}
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(withdraw), Fee: fee, To: to}
		tx.Nonce = r.Int63()
		tx.Sign(types.SECP256K1, priv)
		// Contact the server and print out its response.
		reply, err := c.SendTransaction(context.Background(), tx)
		if err != nil {
			fmt.Println("err", err)
			return nil, err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return nil, errors.New(string(reply.GetMsg()))
		}
		return tx.Hash(), nil
	}
}

func getAccounts() (*types.WalletAccounts, error) {
	c := types.NewChain33Client(conn)
	v := &types.ReqNil{}
	return c.GetAccounts(context.Background(), v)
}

func getlastheader() (*types.Header, error) {
	c := types.NewChain33Client(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}

func waitTx(hash []byte) bool {
	i := 0
	for {
		i++
		if i%100 == 0 {
			fmt.Println("wait transaction timeout")
			return false
		}
		c := types.NewChain33Client(conn)
		var reqHash types.ReqHash
		reqHash.Hash = hash
		res, err := c.QueryTransaction(context.Background(), &reqHash)
		if err != nil {
			time.Sleep(time.Second)
		}
		if res != nil {
			return true
		}
	}
}

func waitTxs(hashes [][]byte) bool {
	for _, hash := range hashes {
		result := waitTx(hash)
		if !result {
			return false
		}
	}
	return true
}
