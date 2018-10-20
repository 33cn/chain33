package executor

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	hlt "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var (
	conn         *grpc.ClientConn
	r            *rand.Rand
	c            types.Chain33Client
	ErrTest      = errors.New("ErrTest")
	secret       []byte
	wrongsecret  []byte
	anothersec   []byte //used in send case
	addrexec     string
	locktime     = minLockTime + 10 // bigger than minLockTime defined in hashlock.go
	addr         [accountMax]string
	privkey      [accountMax]crypto.PrivKey
	currBalanceA int64
	currBalanceB int64
)

const (
	accountindexA = 0
	accountindexB = 1
	accountMax    = 2
)

const (
	defaultAmount = 1e10
	fee           = 1e6
	lockAmount    = 1e8
)

const (
	onlyshow     = 0
	onlycheck    = 1
	showandcheck = 2
)

const secretLen = 32

type TestHashlockquery struct {
	Time        int64
	Status      int32
	Amount      int64
	CreateTime  int64
	CurrentTime int64
}

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(types.Now().UnixNano()))
	c = types.NewChain33Client(conn)
	secret = make([]byte, secretLen)
	wrongsecret = make([]byte, secretLen)
	anothersec = make([]byte, secretLen)
	crand.Read(secret)
	crand.Read(wrongsecret)
	crand.Read(anothersec)
	addrexec = address.ExecAddress("hashlock")
}

func estInitAccount(t *testing.T) {
	fmt.Println("TestInitAccount start")
	defer fmt.Println("TestInitAccount end")

	var label [accountMax]string
	var params types.ReqWalletImportPrivkey

	privGenesis := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
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
		time.Sleep(10 * time.Second)
		if !showOrCheckAcc(c, addr[index], showandcheck, 0) {
			t.Error(ErrTest)
			return
		}
		time.Sleep(10 * time.Second)
	}

	for index := 0; index < accountMax; index++ {
		err := sendtoaddress(c, privGenesis, addr[index], defaultAmount)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(10 * time.Second)
		if !showOrCheckAcc(c, addr[index], showandcheck, defaultAmount) {
			t.Error(ErrTest)
			return
		}
	}
	currBalanceA = defaultAmount
	currBalanceB = defaultAmount
}

func estHashlock(t *testing.T) {

	fmt.Println("TestHashlock start")
	defer fmt.Println("TestHashlock end")

	//1. step1 发送余额给合约
	err := sendtoaddress(c, privkey[accountindexA], addrexec, lockAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Second)

	err = lock(secret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(10 * time.Second)

	currBalanceA -= lockAmount + 2*fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	fmt.Println("TestHashlockQuery start")
	defer fmt.Println("TestHashlockQuery end")
	var req types.ChainExecutor
	req.Driver = "hashlock"
	req.FuncName = "GetHashlocKById"
	req.Param = common.Sha256(secret)
	time.Sleep(15 * time.Second)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err =", reply.GetMsg())
		t.Error(errors.New(string(reply.GetMsg())))
		return
	}
	value := strings.TrimSpace(string(reply.Msg))
	fmt.Println("GetValue=", []byte(value))

	var hashlockquery hlt.Hashlockquery
	err = proto.Unmarshal([]byte(value), &hashlockquery)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("QueryHashlock =", hashlockquery)
}

func estHashunlock(t *testing.T) {
	fmt.Println("TestHashunlock start")
	defer fmt.Println("TestHashunlock end")
	//not sucess as time not enough
	time.Sleep(5 * time.Second)
	err := unlock(secret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	//尝试取钱
	err = sendtoaddress(c, privkey[accountindexA], addrexec, 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= 2 * fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	//not success as secret is not right
	time.Sleep(70 * time.Second)
	err = unlock(wrongsecret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexA], addrexec, 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= 2 * fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	//success
	time.Sleep(5 * time.Second)
	err = unlock(secret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexA], addrexec, 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA = currBalanceA - 2*fee + lockAmount
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	fmt.Println("TestHashunlockQuery start")
	defer fmt.Println("TestHashlockQuery end")
	var req types.ChainExecutor
	req.Driver = "hashlock"
	req.FuncName = "GetHashlocKById"
	req.Param = common.Sha256(secret)
	time.Sleep(15 * time.Second)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err =", reply.GetMsg())
		t.Error(errors.New(string(reply.GetMsg())))
		return
	}
	value := strings.TrimSpace(string(reply.Msg))
	fmt.Println("GetValue=", []byte(value))

	var hashlockquery hlt.Hashlockquery
	err = proto.Unmarshal([]byte(value), &hashlockquery)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("QueryHashlock =", hashlockquery)
}

func estHashsend(t *testing.T) {
	fmt.Println("TestHashsend start")
	defer fmt.Println("TestHashsend end")
	//lock it again &send failed as secret is not right
	//send failed as secret is not right
	err := sendtoaddress(c, privkey[accountindexA], addrexec, lockAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	err = lock(anothersec)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= lockAmount + 2*fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}

	time.Sleep(5 * time.Second)

	err = send(wrongsecret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexB], addrexec, 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	time.Sleep(5 * time.Second)
	//currBalanceA -= fee
	currBalanceB -= 2 * fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	if !showOrCheckAcc(c, addr[accountindexB], showandcheck, currBalanceB) {
		t.Error(ErrTest)
		return
	}

	//success
	time.Sleep(5 * time.Second)
	err = send(anothersec)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexB], addrexec, 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	time.Sleep(5 * time.Second)
	//currBalanceA -= fee
	currBalanceB = currBalanceB + lockAmount - 2*fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	if !showOrCheckAcc(c, addr[accountindexB], showandcheck, currBalanceB) {
		t.Error(ErrTest)
		return
	}
	//lock it again & failed as overtime
	fmt.Println("TestHashsendQuery start")
	defer fmt.Println("TestHashsendQuery end")
	var req types.ChainExecutor
	req.Driver = "hashlock"
	req.FuncName = "GetHashlocKById"
	req.Param = common.Sha256(anothersec)
	time.Sleep(15 * time.Second)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err =", reply.GetMsg())
		t.Error(errors.New(string(reply.GetMsg())))
		return
	}
	value := strings.TrimSpace(string(reply.Msg))
	fmt.Println("GetValue=", []byte(value))

	var hashlockquery hlt.Hashlockquery
	err = proto.Unmarshal([]byte(value), &hashlockquery)
	if err != nil {
		//		clog.Error("TestHashlockQuery Unmarshal")
		t.Error(err)
		return
	}
	fmt.Println("QueryHashlock =", hashlockquery)

}

func lock(secret []byte) error {
	vlock := &hlt.HashlockAction_Hlock{&hlt.HashlockLock{Amount: lockAmount, Time: int64(locktime), Hash: common.Sha256(secret), ToAddress: addr[accountindexB], ReturnAddress: addr[accountindexA]}}
	//fmt.Println(vlock)
	transfer := &hlt.HashlockAction{Value: vlock, Ty: hlt.HashlockActionLock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: addr[accountindexB]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[accountindexA])
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
func unlock(secret []byte) error {

	vunlock := &hlt.HashlockAction_Hunlock{&hlt.HashlockUnlock{Secret: secret}}
	transfer := &hlt.HashlockAction{Value: vunlock, Ty: hlt.HashlockActionUnlock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: addr[accountindexB]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[accountindexA])
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

func send(secret []byte) error {

	vsend := &hlt.HashlockAction_Hsend{&hlt.HashlockSend{Secret: secret}}
	transfer := &hlt.HashlockAction{Value: vsend, Ty: hlt.HashlockActionSend}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: addr[accountindexB]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[accountindexB])
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

func showAccount(c types.Chain33Client, addr string) {
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

func sendtoaddress(c types.Chain33Client, priv crypto.PrivKey, to string, amount int64) error {
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
			return err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return errors.New(string(reply.GetMsg()))
		}
		return nil
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
	c := types.NewChain33Client(conn)
	v := &types.ReqNil{}
	return c.GetAccounts(context.Background(), v)
}

func getlastheader() (*types.Header, error) {
	c := types.NewChain33Client(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
