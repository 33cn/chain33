package commands

import (
	"encoding/json"
	"fmt"
	"os"

	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
)

// TODO: SetPostRunCb()
type RpcCtx struct {
	Addr   string
	Method string
	Params interface{}
	Res    interface{}

	cb Callback
}

type Callback func(res interface{}) (interface{}, error)

func NewRpcCtx(laddr, methed string, params, res interface{}) *RpcCtx {
	return &RpcCtx{
		Addr:   laddr,
		Method: methed,
		Params: params,
		Res:    res,
	}
}

func (c *RpcCtx) SetResultCb(cb Callback) {
	c.cb = cb
}

func (c *RpcCtx) Run() {
	rpc, err := jsonrpc.NewJSONClient(c.Addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = rpc.Call(c.Method, c.Params, c.Res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// maybe format rpc result
	var result interface{}
	if c.cb != nil {
		result, err = c.cb(c.Res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	} else {
		result = c.Res
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}
