package rpctest

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"testing"
	"time"
)

func SendTransaction(account, payload, signature string) {
	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"JRpcRequest.SendTransaction","params":[{"account":"%s","payload":"%s","signature":"%s"}]}`, account, payload, signature)
	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("returned JSON: %s\n", string(b))

}

func Test_RandSendTransaction(t *testing.T) {

	var accounts = []string{"xiaoming", "xiaowang", "xiaoli", "xiaosi", "zhangsan", "wang2", "xiaohe", "liangliang", "bobo", "shuoshuo"}

	for {
		bigi, err := rand.Int(rand.Reader, big.NewInt(int64(len(accounts))))
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		index := bigi.Int64()
		fmt.Println("account:", accounts[index])
		var payload uint32
		var paybytes [4]byte
		binary.Read(bytes.NewBuffer(paybytes[:]), binary.LittleEndian, &payload)
		SendTransaction(accounts[index], fmt.Sprintf("%s", payload), fmt.Sprintf("%s", index))
		time.Sleep(time.Second * 1)
	}

}

