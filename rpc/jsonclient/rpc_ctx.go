// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonclient

import (
	"encoding/json"
	"fmt"
	"os"
)

// RPCCtx rpc ctx interface
// TODO: SetPostRunCb()
type RPCCtx struct {
	Addr   string
	Method string
	Params interface{}
	Res    interface{}
	cb     Callback
}

// Callback a callback function
type Callback func(res interface{}) (interface{}, error)

// NewRPCCtx produce a object of rpcctx
func NewRPCCtx(laddr, method string, params, res interface{}) *RPCCtx {
	return &RPCCtx{
		Addr:   laddr,
		Method: method,
		Params: params,
		Res:    res,
	}
}

// SetResultCb rpcctx callback
func (c *RPCCtx) SetResultCb(cb Callback) {
	c.cb = cb
}

// RunResult  format rpc result
func (c *RPCCtx) RunResult() (interface{}, error) {
	rpc, err := NewJSONClient(c.Addr)
	if err != nil {
		return nil, err
	}

	err = rpc.Call(c.Method, c.Params, c.Res)
	if err != nil {
		return nil, err
	}
	// maybe format rpc result
	var result interface{}
	if c.cb != nil {
		result, err = c.cb(c.Res)
		if err != nil {
			return nil, err
		}
	} else {
		result = c.Res
	}
	return result, nil
}

// Run rpcctx to runresult
func (c *RPCCtx) Run() {
	result, err := c.RunResult()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(string(data))
}

// RunWithoutMarshal return source result of string
func (c *RPCCtx) RunWithoutMarshal() {
	var res string
	rpc, err := NewJSONClient(c.Addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = rpc.Call(c.Method, c.Params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(res)
}
