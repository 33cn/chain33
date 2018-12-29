package p2p

import (
	"fmt"
	"testing"
	"time"

	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func mTestGrpcConns(t *testing.T) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 30; i++ {
		fmt.Println("index:", i)
		conn, err = grpc.Dial("localhost:13802", grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		if err != nil {
			fmt.Println("grpc DialCon", "did not connect", err)
			return
		}

		cli := pb.NewP2PgserviceClient(conn)

		resp, err := cli.GetHeaders(context.Background(), &pb.P2PGetHeaders{StartHeight: 0, EndHeight: 0, Version: 1002}, grpc.FailFast(true))

		if err != nil {
			fmt.Println("GetHeaders error:", err.Error())
			continue

		}
		fmt.Println(resp)

	}

	time.Sleep(time.Second * 5)
	return
}

func TestGrpcStreamConns(t *testing.T) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 30; i++ {
		fmt.Println("index:", i)
		conn, err = grpc.Dial("localhost:13802", grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		if err != nil {
			fmt.Println("grpc DialCon", "did not connect", err)
			return
		}

		cli := pb.NewP2PgserviceClient(conn)
		//流测试
		//		ping := &pb.P2PPing{Nonce: int64(123456), Addr: "192.168.1.1:12345", Port: 12345}
		//		_, err := P2pComm.Signature("a7769f8ca43b0b694105e165b9c94eda71f374bafb6979f923c6f4593fea10b9", ping)
		//		if err != nil {
		//			log.Error("Signature", "Error", err.Error())
		//			return
		//		}
		var p2pdata pb.P2PGetData
		//ctx, _ := context.WithCancel(context.Background(), &p2pdata)

		resp, err := cli.GetData(context.Background(), &p2pdata)
		if err != nil {
			log.Error("GetData", "err:", err.Error())
			continue
		}
		_, err = resp.Recv()
		if err != nil {
			log.Error("ServerStreamSend", "err:", err.Error())
			continue
		}

	}
}
