package main

import (
	"context"
	"testing"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

func TestGrpcSendToAddress(t *testing.T) {
	conn, err := grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		t.Error(err)
		return
	}
	//defer conn.Close()
	c := types.NewGrpcserviceClient(conn)

	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		t.Error(err)
		return
	}
	hex := "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	priv, err := cr.PrivKeyFromBytes(common.FromHex(hex))
	if err != nil {
		t.Error(err)
		return
	}

	privto, err := cr.GenKey()
	if err != nil {
		t.Error(err)
		return
	}

	addrfrom := account.PubKeyToAddress(priv.PubKey().Bytes())
	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	amount := int64(1e8)
	t.Log(addrfrom)
	t.Log(addrto)
	t.Log(amount)

	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}

	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: addrto.String()}
	tx.Sign(types.SECP256K1, priv)
	// Contact the server and print out its response.
	_, err = c.SendTransaction(context.Background(), tx)
	if err != nil {
		t.Error(err)
	}
	conn.Close()
}
