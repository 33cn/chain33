package rpc

import (
	"log"

	"code.aliyun.com/chain33/chain33/types"
)

type JRpcRequest struct {
	jserver *jsonrpcServer
}

//发送交易信息到topic=channel 的queue 中
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
	iclient := req.jserver.c
	message := iclient.NewMessage("rpc", types.EventTx, &data)
	err := iclient.Send(message, true)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	reply, err := iclient.Wait(message)
	if err != nil {
		log.Println("wait err:", err.Error())
		return err
	}

	*result = string(reply.GetData().(types.Reply).Msg)
	return reply.Err()

}

type QueryParm struct {
	hash string
}

func (req JRpcRequest) QueryTransaction(in QueryParm, result *interface{}) error {
	var data types.RequestHash
	data.Hash = []byte(in.hash)
	iclient := req.jserver.c
	message := iclient.NewMessage("rpc", types.EventQueryTx, &data)
	err := iclient.Send(message, true)
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	reply, err := iclient.Wait(message)
	if err != nil {
		log.Println("wait err:", err.Error())
		return err
	}

	*result = reply.GetData().(*types.MerkleProof)
	return reply.Err()
}

type BlockParam struct {
	Start int64
	End   int64
}

func (req JRpcRequest) GetBlocks(in BlockParam, result *interface{}) error {
	var data types.RequestBlocks
	data.End = in.End
	data.Start = in.Start
	iclient := req.jserver.c
	message := iclient.NewMessage("rpc", types.EventGetBlocks, &data)
	err := iclient.Send(message, true)
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	reply, err := iclient.Wait(message)
	if err != nil {
		log.Fatal("wait err:", err.Error())
		return err
	}

	*result = reply.GetData().(*types.Blocks)
	return reply.Err()

}
