package grpcclient

import (
	"testing"
	"google.golang.org/grpc"
	"github.com/33cn/chain33/types"
	"context"
	"fmt"
	"time"
)

func TestConnect(t *testing.T) {
	url := MultiPleHostsBalancerPrefix+"127.0.0.1:8902,127.0.0.1:8802"

	fmt.Println(url)
	conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBalancerName(SwitchBalancer))
	if err != nil {
		panic(err)
	}
	grpcClient := types.NewChain33Client(conn)

	rep,err := grpcClient.GetLastHeader(context.Background(), &types.ReqNil{})
	if err != nil {
		panic(err)
	}
	fmt.Println(rep)
	time.Sleep(time.Second * time.Duration(5))

	rep,err = grpcClient.GetLastHeader(context.Background(), &types.ReqNil{})
	if err != nil {
		panic(err)
	}
	fmt.Println(rep)
	time.Sleep(time.Second * time.Duration(5))

	rep,err = grpcClient.GetLastHeader(context.Background(), &types.ReqNil{})
	if err != nil {
		panic(err)
	}
	fmt.Println(rep)
}
