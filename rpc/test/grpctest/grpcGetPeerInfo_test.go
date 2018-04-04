package main

import (
	"context"
	"log"
	"testing"

	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
)

func TestGrpcGetPeerInfo(t *testing.T) {
	var err error
	conn, err := grpc.Dial("localhost:8802", grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	if err != nil {
		panic(err)
	}
	c := types.NewGrpcserviceClient(conn)
	resp, err := c.GetPeerInfo(context.Background(), &types.ReqNil{})
	if err != nil {
		log.Fatalln("err:", err.Error())
	}
	t.Log(types.Size(resp))
	t.Log("peers:", resp.GetPeers())
}

var errstr = `grpc: Decompressor is not installed for grpc-encoding "gzip"`

func TestCompressSupport(t *testing.T) {
	conn, err := grpc.Dial("localhost:8802", grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	if err != nil {
		panic(err)
	}
	c := types.NewGrpcserviceClient(conn)
	_, err = c.IsSync(context.Background(), &types.ReqNil{})
	if err != nil {
		if grpc.Code(err) == codes.Unimplemented && grpc.ErrorDesc(err) == errstr {
			t.Log("compress not support")
		}
	} else {
		t.Log("compress support")
	}
}
