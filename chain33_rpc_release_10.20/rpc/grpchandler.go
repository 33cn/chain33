package rpc

import (
	"context"
	"log"

	pb "code.aliyun.com/chain33/chain33/types"
)

type Grpc struct {
	gserver *grpcServer
}

func (req *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {

	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)

	reply := cli.SendTx(in)
	return &pb.Reply{IsOk: true, Msg: reply.GetData().(pb.Reply).Msg}, nil

}

func (req *Grpc) QueryTransaction(ctx context.Context, in *pb.RequestHash) (*pb.Reply, error) {
	iclient := req.gserver.c
	message := iclient.NewMessage("rpc", pb.EventQueryTx, in)
	err := iclient.Send(message, true)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	reply, err := iclient.Wait(message)
	if err != nil {
		log.Println("wait err:", err.Error())
		return nil, err
	}

	return &pb.Reply{IsOk: true, Msg: reply.GetData().(pb.Reply).Msg}, nil
}

func (req *Grpc) GetBlocks(ctx context.Context, in *pb.Block) (*pb.Reply, error) {
	iclient := req.gserver.c
	message := iclient.NewMessage("rpc", pb.EventGetBlocks, in)
	err := iclient.Send(message, true)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	reply, err := iclient.Wait(message)
	if err != nil {
		log.Println("wait err:", err.Error())
		return nil, err
	}

	return &pb.Reply{IsOk: true, Msg: reply.GetData().(pb.Reply).Msg}, nil
}
