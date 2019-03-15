package grpcclient

import (
	"sync"
	"time"

	"github.com/33cn/chain33/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// paraChainGrpcRecSize 平行链receive最大100M
const paraChainGrpcRecSize = 100 * 1024 * 1024

var mu sync.Mutex

var defaultClient types.Chain33Client

//NewMainChainClient 创建一个平行链的 主链 grpc chain33 客户端
func NewMainChainClient(grpcaddr string) (types.Chain33Client, error) {
	mu.Lock()
	defer mu.Unlock()
	if grpcaddr == "" && defaultClient != nil {
		return defaultClient, nil
	}
	paraRemoteGrpcClient := types.Conf("config.consensus.sub.para").GStr("ParaRemoteGrpcClient")
	if grpcaddr != "" {
		paraRemoteGrpcClient = grpcaddr
	}
	if paraRemoteGrpcClient == "" {
		paraRemoteGrpcClient = "127.0.0.1:8802"
	}
	kp := keepalive.ClientParameters{
		Time:                time.Second * 5,
		Timeout:             time.Second * 20,
		PermitWithoutStream: true,
	}
	conn, err := grpc.Dial(NewMultipleURL(paraRemoteGrpcClient), grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(paraChainGrpcRecSize)),
		grpc.WithKeepaliveParams(kp))
	if err != nil {
		return nil, err
	}
	grpcClient := types.NewChain33Client(conn)
	if grpcaddr == "" {
		defaultClient = grpcClient
	}
	return grpcClient, nil
}
