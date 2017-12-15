package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	jsonrpc "code.aliyun.com/chain33/chain33/rpc"
)

func TestGetLastHeader(t *testing.T) {
	rpc, err := jsonrpc.NewJsonClient("http://114.55.101.159:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res jsonrpc.Header
	err = rpc.Call("Chain33.GetLastHeader", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	data, err := json.MarshalIndent(res, "", "   ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("result:\n" + string(data))
}
