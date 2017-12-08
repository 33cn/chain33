package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

//{"jsonrpc":"2.0","id":2,"execer":"xxx","palyload":"xx","signature":{"ty":1,"pubkey":"xx","signature":"xxx"},"Fee":12}
func Test_SendTransaction(t *testing.T) {
	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":2,"method":"JRpcRequest.SendTransaction","params":[{"execer":"xxx","palyload":"xx","signature":{"ty":1,"pubkey":"xx","signature":"xxx"},"Fee":12}]}`,
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
