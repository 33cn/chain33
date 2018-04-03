package main

import (
	"context"
	"log"
	"testing"

	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

func TestGrpcGetPeerInfo(t *testing.T) {
	var err error
	conn, err := grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	c := types.NewGrpcserviceClient(conn)
	resp, err := c.GetPeerInfo(context.Background(), &types.ReqNil{})
	if err != nil {
		log.Fatalln("err:", err.Error())
	}

	log.Println("peers:", resp.GetPeers())
}
