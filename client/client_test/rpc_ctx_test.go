package client_test

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"context"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

// TODO: SetPostRunCb()
type JsonRpcCtx struct {
	Addr   string
	Method string
	Params interface{}
	Res    interface{}

	cb Callback
}

type Callback func(res interface{}) (interface{}, error)

func NewJsonRpcCtx(methed string, params, res interface{}) *JsonRpcCtx {
	time.Sleep(5 * time.Millisecond)
	return &JsonRpcCtx{
		Addr:   "http://localhost:8801",
		Method: methed,
		Params: params,
		Res:    res,
	}
}

func (c *JsonRpcCtx) SetResultCb(cb Callback) {
	c.cb = cb
}

func (c *JsonRpcCtx) Run() (err error) {
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
	return
}

type GrpcCtx struct {
	Method string
	Params interface{}
	Res    interface{}
}

func NewGRpcCtx(method string, params, res interface{}) *GrpcCtx {
	time.Sleep(5 * time.Millisecond)
	return &GrpcCtx{
		Method: method,
		Params: params,
		Res:    res,
	}
}

func (c *GrpcCtx) Run() (err error) {
	conn, err := grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	rpc := types.NewGrpcserviceClient(conn)
	switch c.Method {
	case "Version":
		reply, err := rpc.Version(context.Background(), &types.ReqNil{})
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	default:
		err = fmt.Errorf("Unsupport method ", c.Method)
	}
	return err
}
