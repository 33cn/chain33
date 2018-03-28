package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

var height = flag.Int("height", 1, "blockheight")

func TestGetBlock(t *testing.T) {
	flag.Parse()
	fmt.Println("height:", *height)
	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"JRpcRequest.GetBlocks","params":[{"Start":"%d","End":"%d"}]}`, *height, *height)
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
