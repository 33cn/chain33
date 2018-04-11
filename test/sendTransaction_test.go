package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestSendTransaction(t *testing.T) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		t.Error(err)
		return
	}
	hex := "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	hexbytes, err := common.FromHex(hex)
	if err != nil {
		t.Error(err.Error())
		return
	}
	priv, err := cr.PrivKeyFromBytes(hexbytes)
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
	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.SendTransaction","params":[{"data":"%v"}]}`,
		common.ToHex(types.Encode(tx)))

	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Printf("returned JSON: %s\n", string(b))

}
