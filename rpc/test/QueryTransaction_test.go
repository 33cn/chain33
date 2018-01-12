package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

var hash = flag.String("txhash", "", "txhash")

func TestQueryTransaction(t *testing.T) {
	flag.Parse()
	fmt.Println("transaction hash:", *hash)
	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"JRpcRequest.QueryTransaction","params":[{"hash":"%s"}]}`, *hash)
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
