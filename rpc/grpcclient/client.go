package grpcclient

import (
	"github.com/33cn/chain33/types"
	"google.golang.org/grpc"
)

//NewMainChainClient 创建一个平行链的 主链 grpc chain33 客户端
func NewMainChainClient(grpcaddr string) (types.Chain33Client, error) {
	paraRemoteGrpcClient := types.Conf("config.consensus.sub.para").GStr("ParaRemoteGrpcClient")
	if grpcaddr != "" {
		paraRemoteGrpcClient = grpcaddr
	}
	if paraRemoteGrpcClient == "" {
		paraRemoteGrpcClient = "127.0.0.1:8002"
	}
	conn, err := grpc.Dial(NewMultipleURL(paraRemoteGrpcClient), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	grpcClient := types.NewChain33Client(conn)
	return grpcClient, nil
}
