package rpc

import (
	"code.aliyun.com/chain33/chain33/types"
)

type JRpcRequest struct {
	jserver *jsonrpcServer
}

//发送交易信息到topic=rpc 的queue 中

//{"execer":"xxx","palyload":"xx","signature":{"ty":1,"pubkey":"xx","signature":"xxx"},"Fee":12}
type JTransparm struct {
	Execer    string
	Payload   string
	Signature *Signature
	Fee       int64
}
type Signature struct {
	Ty        int32
	Pubkey    string
	Signature string
}

func (req JRpcRequest) SendTransaction(in JTransparm, result *interface{}) error {

	var data types.Transaction
	data.Execer = []byte(in.Execer)
	data.Payload = []byte(in.Payload)
	data.Signature.Signature = []byte(in.Signature.Signature)
	data.Signature.Ty = in.Signature.Ty
	data.Signature.Pubkey = []byte(in.Signature.Pubkey)
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
	reply, err := cli.GetBlocks(data.Start, data.End, false)
	if err != nil {
		return err
	}
	*result = reply
	return nil

}
