package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

func CreateRawTx() string {
	//CreateRawTransaction
	client := http.DefaultClient

	postdata := fmt.Sprintf(`{"id":1,"method":"Chain33.CreateRawTransaction","params":[{"to":"1ALB6hHJCayUqH5kfPHU3pz8aCUMw1QiT3","amount":%v,"fee":%v,"note":"for test"}]}`, int64(1*1e4), int64(2*1e6))
	req, err := http.NewRequest("post", "http://localhost:8801/", strings.NewReader(postdata))
	if err != nil {
		fmt.Println("err:", err.Error())
		return ""
	}
	fmt.Printf("postdata:%v\n", postdata)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("err:", err.Error())
		return ""
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("err:", err.Error())
		return ""
	}
	var txdata = make(map[string]interface{})
	fmt.Println("resp:", string(data))
	json.Unmarshal(data, &txdata)
	if hextx, ok := txdata["result"]; ok {
		return hextx.(string)
	}
	return ""
}

func TestSendRawTx(t *testing.T) {
	//unsign Tx 一个构造好的但未签名的交易
	unsigntx := CreateRawTx()
	unsignedTx, err := hex.DecodeString(unsigntx)
	if err != nil {
		fmt.Println("hex.Decode", err.Error())
		return
	}

	prikeybyte, err := common.FromHex("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	if err != nil || len(prikeybyte) == 0 {
		fmt.Println("ProcSendToAddress", "FromHex err", err)
		return
	}

	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		fmt.Println("ProcSendToAddress", "err", err)
		return
	}
	priv, err := cr.PrivKeyFromBytes(prikeybyte)
	if err != nil {
		fmt.Println("ProcSendToAddress", "PrivKeyFromBytes err", err)
		return
	}

	signedTx := hex.EncodeToString(priv.Sign(unsignedTx).Bytes())
	postdata := fmt.Sprintf(`{"id":2,"method":"Chain33.SendRawTransaction","params":[{"unsigntx":"%v","signedtx":"%v","pubkey":"%v","ty":%v}]}`, unsigntx, signedTx, hex.EncodeToString(priv.PubKey().Bytes()), 1)
	client := http.DefaultClient
	req, err := http.NewRequest("post", "http://localhost:8801/", strings.NewReader(postdata))
	if err != nil {
		fmt.Println("newRequest error", err.Error())
		return
	}
	fmt.Println("post data:", postdata)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Do error", err.Error())
		return
	}

	defer resp.Body.Close()
	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll error", err.Error())
		return
	}

	fmt.Println("response:", string(bodyData))

}
