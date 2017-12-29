package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

func TestGrpcSendToAddress(t *testing.T) {
	priv := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	keymap := make(map[string]crypto.PrivKey)
	N := 100
	for i := 0; i < N; i++ {
		addrto, privkey := genaddress()
		err := sendtoaddress(priv, addrto, 1e9)
		if err != nil {
			t.Log(err)
			time.Sleep(time.Second)
			continue
		}
		fmt.Print(".")
		keymap[addrto] = privkey
	}
	fmt.Print("\n")
	ch := make(chan struct{}, N)
	for _, value := range keymap {
		go func(priv crypto.PrivKey) {
			for i := 0; i < N*10; {
				addrto, _ := genaddress()
				err := sendtoaddress(priv, addrto, 10000)
				if err != nil {
					t.Log(err)
					time.Sleep(time.Second)
					continue
				}
				fmt.Print(".")
				i++
			}
			fmt.Print("\n")
			ch <- struct{}{}
		}(value)
	}
	for i := 0; i < N; i++ {
		<-ch
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

func sendtoaddress(priv crypto.PrivKey, to string, amount int64) error {
	conn, err := grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		return err
	}
	//defer conn.Close()
	c := types.NewGrpcserviceClient(conn)
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}

	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}

	tx.Sign(types.SECP256K1, priv)
	// Contact the server and print out its response.
	_, err = c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
