package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

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
	case "GetBlocks":
		reply, err := rpc.GetBlocks(context.Background(), c.Params.(*types.ReqBlocks))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	case "GetLastHeader":
		reply, err := rpc.GetLastHeader(context.Background(), c.Params.(*types.ReqNil))
		if err == nil {
			*c.Res.(*types.Header) = *reply
		}
	case "CreateRawTransaction":
		reply, err := rpc.CreateRawTransaction(context.Background(), c.Params.(*types.CreateTx))
		if err == nil {
			*c.Res.(*types.UnsignTx) = *reply
		}
	case "SendRawTransaction":
		reply, err := rpc.SendRawTransaction(context.Background(), c.Params.(*types.SignedTx))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	case "QueryTransaction":
		reply, err := rpc.QueryTransaction(context.Background(), c.Params.(*types.ReqHash))
		if err == nil {
			*c.Res.(*types.TransactionDetail) = *reply
		}
	case "SendTransaction":
		reply, err := rpc.SendTransaction(context.Background(), c.Params.(*types.Transaction))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	case "GetTransactionByAddr":
		reply, err := rpc.GetTransactionByAddr(context.Background(), c.Params.(*types.ReqAddr))
		if err == nil {
			*c.Res.(*types.ReplyTxInfos) = *reply
		}
	case "GetTransactionByHashes":
		reply, err := rpc.GetTransactionByHashes(context.Background(), c.Params.(*types.ReqHashes))
		if err == nil {
			*c.Res.(*types.TransactionDetails) = *reply
		}
	case "GetMemPool":
		reply, err := rpc.GetMemPool(context.Background(), c.Params.(*types.ReqNil))
		if err == nil {
			*c.Res.(*types.ReplyTxList) = *reply
		}
	case "GetAccounts":
		reply, err := rpc.GetAccounts(context.Background(), c.Params.(*types.ReqNil))
		if err == nil {
			*c.Res.(*types.WalletAccounts) = *reply
		}
	case "NewAccount":
		reply, err := rpc.NewAccount(context.Background(), c.Params.(*types.ReqNewAccount))
		if err == nil {
			*c.Res.(*types.WalletAccount) = *reply
		}
	case "WalletTransactionList":
		reply, err := rpc.WalletTransactionList(context.Background(), c.Params.(*types.ReqWalletTransactionList))
		if err == nil {
			*c.Res.(*types.WalletTxDetails) = *reply
		}
	case "ImportPrivKey":
		reply, err := rpc.ImportPrivKey(context.Background(), c.Params.(*types.ReqWalletImportPrivKey))
		if err == nil {
			*c.Res.(*types.WalletAccount) = *reply
		}
	case "SendToAddress":
		reply, err := rpc.SendToAddress(context.Background(), c.Params.(*types.ReqWalletSendToAddress))
		if err == nil {
			*c.Res.(*types.ReplyHash) = *reply
		}
	case "SetTxFee":
		reply, err := rpc.SetTxFee(context.Background(), c.Params.(*types.ReqWalletSetFee))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	case "SetLabl":
		reply, err := rpc.SetLabl(context.Background(), c.Params.(*types.ReqWalletSetLabel))
		if err == nil {
			*c.Res.(*types.WalletAccount) = *reply
		}
	case "MergeBalance":
		reply, err := rpc.MergeBalance(context.Background(), c.Params.(*types.ReqWalletMergeBalance))
		if err == nil {
			*c.Res.(*types.ReplyHashes) = *reply
		}
	case "SetPasswd":
		reply, err := rpc.SetPasswd(context.Background(), c.Params.(*types.ReqWalletSetPasswd))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	case "Lock":
		reply, err := rpc.Lock(context.Background(), c.Params.(*types.ReqNil))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	case "UnLock":
		reply, err := rpc.UnLock(context.Background(), c.Params.(*types.WalletUnLock))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}

	case "Version":
		reply, err := rpc.Version(context.Background(), c.Params.(*types.ReqNil))
		if err == nil {
			*c.Res.(*types.Reply) = *reply
		}
	default:
		err = fmt.Errorf("Unsupport method ", c.Method)
	}
	return err
}
