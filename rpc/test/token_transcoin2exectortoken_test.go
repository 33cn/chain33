package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"time"
)

//{ owner:
//addr:1Lmmwzw6ywVa3UZpA4tHvCB7gR9ZKRwpom
//privateKey: 0X3C47A5183E11A4D9F39E730939159EF352AC2B3FBD5D3FC79424A355C2A83447
//}
//
//{approver:
//addr:1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S
//privateKey: 0XF5E317BADDCA1E151EEE50FB7BBAD5F4D6098D5B109EF893161EBD64D815C973
//}
//第一步，发送100个bty到合约地址
func TestTransfer2ExecAddrToken(t *testing.T) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		t.Error(err)
		return
	}
	//owner's private key
	hex := "0X3C47A5183E11A4D9F39E730939159EF352AC2B3FBD5D3FC79424A355C2A83447"
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

	addrfrom := account.PubKeyToAddress(priv.PubKey().Bytes())
	addrto := account.ExecAddress("token")
	amount := int64(100 * 1e8)
	t.Log("addrfrom", addrfrom)
	t.Log("addrto", addrto)
	t.Log("amount", amount)

	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, Nonce: random.Int63(), To: addrto.String()}
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
