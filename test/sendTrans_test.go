package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func Test_SendTransaction(t *testing.T) {
	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":2,"method":"JRpcRequest.SendTransaction","params":[{"account":"bangzhu","payload":"tongtong","signature":"9999"}]}`,
	))
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
