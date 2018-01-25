package coins

import (
	"context"
	crand "crypto/rand"
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

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

//init account A, init account B
//test the account locked
//case1: try to lock A
//case2: try to unlock A
//case3: try to active A
//case4: try to lock A
//case5: try to exec to B with error secret
//case6: try to exec to B with right secret
//case6: try to unlock A

func TestHashlockCase1(t *testing.T) {
	c := types.NewGrpcserviceClient(conn)
	//all coins are generated from this private key
	priv := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	//just show or record the header
	_, err := getlastheader()
	if err != nil {
		t.Error(err)
		return
	}
	label := strconv.Itoa(int(time.Now().UnixNano()))
	//generate a new address, and import to wallet
	addrto, privkey := genaddress()
	fmt.Println("privkey: ", common.ToHex(privkey.Bytes()))
	params := types.ReqWalletImportPrivKey{Privkey: common.ToHex(privkey.Bytes()), Label: label}
	_, err = c.ImportPrivKey(context.Background(), &params)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second)
		t.Error(err)
		return
	}
	//another account
	addrto_b, privkey_b := genaddress()
	time.Sleep(time.Second)
	params = types.ReqWalletImportPrivKey{Privkey: common.ToHex(privkey_b.Bytes()), Label: strconv.Itoa(int(time.Now().UnixNano()))}
	_, err = c.ImportPrivKey(context.Background(), &params)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second)
		t.Error(err)
		return
	}
	time.Sleep(1000 * time.Millisecond)
	err = sendtoaddress(c, priv, addrto, 1e10)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second)
		t.Error(err)
		return
	}
	err = sendtoaddress(c, priv, addrto_b, 1e10)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second)
		t.Error(err)
		return
	}
	time.Sleep(1000 * time.Millisecond)
	showAccount(c, addrto)
	showAccount(c, addrto_b)

	var amount int64 = 1e8
	var time int64 = 70
	len := 32
	secret := make([]byte, len)
	crand.Read(secret)
	fmt.Println(common.ToHex(secret))
	sendtolock(c, privkey, amount, time, secret, common.Sha256(secret), addrto_b, addrto, privkey_b)
}

func showAccount(c types.GrpcserviceClient, addr string) {
	req := &types.ReqNil{}
	accs, err := c.GetAccounts(context.Background(), req)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(accs.Wallets); i++ {
		wallet := accs.Wallets[i]
		if wallet.Acc.Addr == addr {
			fmt.Println(wallet)
			break
		}
	}
}

func checkAccount(balance int64, frozen int64, wallet *types.WalletAccount) bool {
	return ((balance == wallet.Acc.Balance) && (frozen == wallet.Acc.Frozen))
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
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
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

func getAccounts() (*types.WalletAccounts, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetAccounts(context.Background(), v)
}

func sendtolock(c types.GrpcserviceClient, priv crypto.PrivKey, amount int64, timelock int64,
	secret []byte, hash []byte, toaddress string, rtadd string, priv_b crypto.PrivKey) error {

	//1. step1 发送余额给合约
	addr := account.ExecAddress("hashlock")
	err := sendtoaddress(c, priv, addr.String(), amount)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second)

	//2. step2,show balance
	showAccount(c, account.PubKeyToAddress(priv.PubKey().Bytes()).String())
	time.Sleep(time.Second)

	//3. 执行lock
	v := &types.HashlockAction_Hlock{&types.HashlockLock{Amount: amount, Time: timelock, Hash: hash, ToAddress: toaddress, ReturnAddress: rtadd}}
	transfer := &types.HashlockAction{Value: v, Ty: types.HashlockActionLock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: 1e6, To: toaddress}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, priv)
	// Contact the server and print out its response.
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}

	//4. 执行解锁（因为时间没有到失败）
	/*
		time.Sleep(80 * time.Second)
		vunlock := &types.HashlockAction_Hunlock{&types.HashlockUnlock{Secret: secret}}
		unlocktransfer := &types.HashlockAction{Value: vunlock, Ty: types.HashlockActionUnlock}
		tx = &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(unlocktransfer), Fee: 1e6, To: toaddress}
		tx.Nonce = r.Int63()
		tx.Sign(types.SECP256K1, priv)
		reply, err = c.SendTransaction(context.Background(), tx)
		if err != nil {
			return err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return errors.New(string(reply.GetMsg()))
		}
		time.Sleep(time.Second)
		err = sendtoaddress(c, priv, addr.String(), 0-amount)
		if err != nil {
			fmt.Println("err")
		}
		time.Sleep(time.Second)
		showAccount(c, account.PubKeyToAddress(priv.PubKey().Bytes()).String())
	*/
	//5. 通过s 执行转账

	//6. 真正的取款操作（通过toaddress 对应的私钥）

	showAccount(c, account.PubKeyToAddress(priv_b.PubKey().Bytes()).String())
	time.Sleep(30 * time.Second)
	vsend := &types.HashlockAction_Hsend{&types.HashlockSend{Secret: secret}}
	sendtransfer := &types.HashlockAction{Value: vsend, Ty: types.HashlockActionSend}
	tx = &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(sendtransfer), Fee: 1e6, To: toaddress}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, priv)
	reply, err = c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}

	time.Sleep(10 * time.Second)
	err = sendtoaddress(c, priv_b, addr.String(), 0-amount)
	if err != nil {
		fmt.Println("err")
	}
	time.Sleep(time.Second)
	showAccount(c, account.PubKeyToAddress(priv_b.PubKey().Bytes()).String())
	//sleep locktime

	//7. 执行解锁(因为已经执行过转账操作，就算过期也不能取回钱了)
	return nil
}

func getlastheader() (*types.Header, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
