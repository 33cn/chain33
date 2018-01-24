package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
)

func TestCreateRawTx(t *testing.T) {
	//CreateRawTransaction
	client := http.DefaultClient

	postdata := fmt.Sprintf(`{"id":1,"method":"Chain33.CreateRawTransaction","params":[{"to":"1ALB6hHJCayUqH5kfPHU3pz8aCUMw1QiT3","amount":%v,"fee":%v,"note":"for test"}]}`, int64(1*1e4), int64(2*1e6))
	req, err := http.NewRequest("post", "http://localhost:8801/", strings.NewReader(postdata))
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}
	fmt.Println("postdata:%v", postdata)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("resp:", string(data))
}

func TestSendRawTx(t *testing.T) {
	//unsign Tx
	unsignedTx, err := hex.DecodeString("0a05636f696e73121118010a0d10904e1a08666f7220746573742080897a309dfabda9e8dffbce383a2231414c423668484a436179557148356b6650485533707a386143554d773151695433")
	if err != nil {
		fmt.Println("hex.Decode", err.Error())
		return
	}
	var tx types.Transaction
	err = types.Decode(unsignedTx, &tx)
	if err != nil {
		fmt.Println("type.Decode:", err.Error())
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

	tx.Sign(types.SECP256K1, priv)
	signedTx := hex.EncodeToString(types.Encode(&tx))

	postdata := fmt.Sprintf(`{"id":2,"method":"Chain33.SendRawTransaction","params":[{"unsigntx":"0a05636f696e73121118010a0d10904e1a08666f7220746573742080897a309dfabda9e8dffbce383a2231414c423668484a436179557148356b6650485533707a386143554d773151695433",
	"signedtx":"%v","pubkey":"%v","ty":%v}]}`, signedTx, hex.EncodeToString(priv.PubKey().Bytes()), 1)
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
