package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"

	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
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
//第二步，precreate token，预创建token，但是发起token precreate的人不是同一个人，需要是平台创建人
func TestRevokeCreateToken(t *testing.T) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		t.Error(err)
		return
	}
	//apporover's private key
	privateKeyhex := "0XF5E317BADDCA1E151EEE50FB7BBAD5F4D6098D5B109EF893161EBD64D815C973"
	privateKeyhexbytes, err := common.FromHex(privateKeyhex)
	if err != nil {
		t.Error(err.Error())
		return
	}
	priv, err := cr.PrivKeyFromBytes(privateKeyhexbytes)
	if err != nil {
		t.Error(err)
		return
	}

	tokenOwner := "1Lmmwzw6ywVa3UZpA4tHvCB7gR9ZKRwpom"
	t.Log(tokenOwner)

	v := &types.TokenAction_Tokenrevokecreate{&types.TokenRevokeCreate{
		"ABAB",
		tokenOwner}}

	tokenPrecreate := &types.TokenAction{v, types.TokenActionRevokeCreate}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx := &types.Transaction{Execer: []byte("token"), Payload: types.Encode(tokenPrecreate), Fee: 1e6, Nonce: random.Int63()}
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
