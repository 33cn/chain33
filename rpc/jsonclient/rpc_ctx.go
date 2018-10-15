package jsonclient

import (
	"encoding/json"
	"fmt"
	"os"
)

// TODO: SetPostRunCb()
type RpcCtx struct {
	Addr   string
	Method string
	Params interface{}
	Res    interface{}
	cb     Callback
}

type Callback func(res interface{}) (interface{}, error)

func NewRpcCtx(laddr, method string, params, res interface{}) *RpcCtx {
	return &RpcCtx{
		Addr:   laddr,
		Method: method,
		Params: params,
		Res:    res,
	}
}

func (c *RpcCtx) SetResultCb(cb Callback) {
	c.cb = cb
}

func (c *RpcCtx) RunResult() (interface{}, error) {
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

func (c *RpcCtx) Run() {
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

func (c *RpcCtx) RunWithoutMarshal() {
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
