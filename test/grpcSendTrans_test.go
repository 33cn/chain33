package main

import (
	"context"
	"log"
	"time"
    "testing"
	pb "code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

func Test_grpcSendTransaction() {
	conn, err := grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	//defer conn.Close()
	c := pb.NewGrpcserviceClient(conn)

	// Contact the server and print out its response.
	r, err := c.SendTransaction(context.Background(), &pb.Transaction{Account: []byte("bangzhu"), Payload: []byte("money"), Signature: []byte("42342342")})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("####### get server Greeting response: %s", string(r.Msg))
	time.Sleep(time.Second * 1)
	conn.Close()
	log.Printf("closed")
	time.Sleep(time.Second * 1)
}
