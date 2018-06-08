package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestNewAccount(t *testing.T) {
	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.NewAccount","params":[{"label":"test"}]}`)

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
