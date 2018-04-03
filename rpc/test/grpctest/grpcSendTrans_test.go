package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var (
	conn   *grpc.ClientConn
	random *rand.Rand
)

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestGrpcSendToAddress(t *testing.T) {
	priv := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	keymap := make(map[string]crypto.PrivKey)
	N := 95
	header, err := getlastheader()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("before send...", header.Height)
	for i := 0; i < N; i++ {
		addrto, privkey := genaddress()
		err := sendtoaddress(priv, addrto, 1e7)
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
	fmt.Println("wait for balance pack")
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
		go func(pkey crypto.PrivKey) {
			for i := 0; i < N; {
				addrto, _ := genaddress()
				err := sendtoaddress(pkey, addrto, 1e7)
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

func getlastheader() (*types.Header, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
