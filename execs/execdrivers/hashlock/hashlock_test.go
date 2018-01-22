package coins

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

var conn *grpc.ClientConn
var random *rand.Rand

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

//test the account locked

func TestHashlockCase1(t *testing.T) {
	var wallet *types.WalletAccount
	//is it better to initalize this client in init()?
	c := types.NewGrpcserviceClient(conn)

	//all coins are generated from this private key
	priv := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")

	//just show or record the header
	_, err := getlastheader()
	if err != nil {
		t.Error(err)
		return
	}

	//generate a new address, and import to wallet
	addrto, privkey := genaddress()
	fmt.Println("privkey: ", common.ToHex(privkey.Bytes()))
	params := types.ReqWalletImportPrivKey{Privkey: common.ToHex(privkey.Bytes()), Label: "222"}
	wallet, err = c.ImportPrivKey(context.Background(), &params)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second)
		t.Error(err)
		return
	}
	//err
	if checkAccount(0, 0, wallet) {
		fmt.Println("Init Account check pass")
	} else {
		fmt.Println("Init Account check not pass")
		//err
	}
	if wallet != nil {
		showAccount(wallet)
		time.Sleep(5000 * time.Millisecond)
		err = sendtoaddress(priv, addrto, 1e10)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(5000 * time.Millisecond)
		showAccount(wallet)
		if checkAccount(100, 0, wallet) {
			fmt.Println("send Account check pass")
		} else {
			fmt.Println("send Account check not pass")
			//err
		}
	} else {
		fmt.Println("it's not right")
	}

	//frozen account.....
	//sendtolock()
	//if checkAccount(90, 10, wallet) {
	//fmt.Println("frozen Account check pass")
	//} else {
	//	fmt.Println("frozen Account check not pass")
	//err
	//t.Error(err)
	//return
	//}
}

func showAccount(wallet *types.WalletAccount) {
	balanceResult := strconv.FormatFloat(float64(wallet.Acc.Balance)/float64(1e8), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(wallet.Acc.Frozen)/float64(1e8), 'f', 4, 64)
	//Currency := wallet.Acc.Currency
	//Addr := wallet.Acc.Addr
	fmt.Printf("balanceResult:%s\n", balanceResult)
	fmt.Printf("frozenResult:%s\n", frozenResult)
	//fmt.Printf("currency:%d\n", Currency)
	//fmt.Printf("Addr:%s\n", Addr)
	//fmt.Println(wallet.Acc.Balance)
	//fmt.Printf("frozen:%d\n", wallet.Acc.Frozen)
	fmt.Printf("currency:%d\n", wallet.Acc.Currency)
	fmt.Printf("Addr:%s\n", wallet.Acc.Addr)
}

func checkAccount(balance int64, frozen int64, wallet *types.WalletAccount) bool {
	return ((balance == wallet.Acc.Balance) && (frozen == wallet.Acc.Frozen))
}

func TstGrpcSendToAddress(t *testing.T) {
	priv := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	keymap := make(map[string]crypto.PrivKey)
	N := 10
	header, err := getlastheader()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("before send...", header.Height)
	for i := 0; i < N; i++ {
		addrto, privkey := genaddress()
		err := sendtoaddress(priv, addrto, 1e10)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			continue
		}
		fmt.Println("privkey: ", common.ToHex(privkey.Bytes()))
		keymap[addrto] = privkey
	}
	header, err = getlastheader()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("after send...", header.Height)
	fmt.Println("wait for balance pack\n")
	time.Sleep(time.Second * 10)
	header, err = getlastheader()
	if err != nil {
		t.Error(err)
		return
	}
	var errcount int64
	fmt.Println("after sleep header...", header.Height)
	ch := make(chan struct{}, N)
	for _, value := range keymap {
		//闭包
		go func(pkey crypto.PrivKey) {
			for i := 0; i < N; {
				addrto, _ := genaddress()
				err := sendtoaddress(pkey, addrto, 10000)
				if err != nil {
					atomic.AddInt64(&errcount, 1)
					time.Sleep(time.Second)
					continue
				}
				fmt.Print("*")
				i++
			}
			fmt.Print("\n")
			ch <- struct{}{}
		}(value)
	}
	for i := 0; i < N; i++ {
		<-ch
	}
	fmt.Println("total err:", errcount)
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

func sendtoaddress(priv crypto.PrivKey, to string, amount int64) error {
	//defer conn.Close()
	//fmt.Println("sign key privkey: ", common.ToHex(priv.Bytes()))
	c := types.NewGrpcserviceClient(conn)
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = rand.Int63()
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
	return nil
}

func getAccounts() (*types.WalletAccounts, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetAccounts(context.Background(), v)
}

/*
func sendtolock() {
	c := types.NewGrpcserviceClient(conn)
	v := &types.HashlockAction_Hlock{&types.HashlockLock{Amount: amount, Time: time, Hash: hash, ToAddress: toaddress, ReturnAddress: rtadd}}
	transfer := &types.HashlockAction{Value: v, Ty: types.HashlockActionLock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = rand.Int63()
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
	return nil
}
*/
func getlastheader() (*types.Header, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
