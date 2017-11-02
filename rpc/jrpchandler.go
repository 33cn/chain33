package rpc

import (
	"code.aliyun.com/chain33/chain33/types"
)

type JRpcRequest struct {
	jserver *jsonrpcServer
}

//发送交易信息到topic=rpc 的queue 中
type JTransparm struct {
	Accout    string
	Payload   string
	Signature string
}

func (req JRpcRequest) SendTransaction(in JTransparm, result *interface{}) error {

	var data types.Transaction
	data.Account = []byte(in.Accout)
	data.Payload = []byte(in.Payload)
	data.Signature = []byte(in.Signature)
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply := cli.SendTx(&data)
	*result = string(reply.GetData().(*types.Reply).Msg)
	return reply.Err()

}

type QueryParm struct {
	hash string
}

func (req JRpcRequest) QueryTransaction(in QueryParm, result *interface{}) error {
	var data types.RequestHash
	data.Hash = []byte(in.hash)

	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.QueryTx(data.Hash)
	if err != nil {
		return err
	}
	*result = reply
	return nil

}

type BlockParam struct {
	Start int64
	End   int64
}

func (req JRpcRequest) GetBlocks(in BlockParam, result *interface{}) error {
	var data types.RequestBlocks
	data.End = in.End
	data.Start = in.Start

	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.GetBlocks(data.Start, data.End)
	if err != nil {
		return err
	}
	*result = reply
	return nil

}
