//这个包提供了平行链和主链的统一的访问接口

package api

import (
	"context"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/types"
	"google.golang.org/grpc"
)

/*
平行链可以访问的有两条链:
1. 通过 client.QueueProtocolAPI 事件访问 平行链本身
2. 通过 grpc 接口访问主链

通过 client.QueueProtocolAPI 和 grpc 都可能会产生 网络错误，或者超时的问题
这个时候，区块做异常处理的时候需要做一些特殊处理

接口函数:
1. GetBlockByHash
2. GetLastBlockHash
3. GetRandNum
*/

//ExecutorAPI 提供给执行器使用的接口
//因为合约是主链和平行链通用的，所以，主链和平行链都可以调用这套接口
type ExecutorAPI interface {
	GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error)
	GetRandNum(param *types.ReqRandHash) ([]byte, error)
}

type mainChainAPI struct {
	api client.QueueProtocolAPI
}

//New 新建接口
func New(api client.QueueProtocolAPI, grpcaddr string) ExecutorAPI {
	if types.IsPara() {
		return newParaChainAPI(api, grpcaddr)
	}
	return &mainChainAPI{api: api}
}

func (api *mainChainAPI) GetRandNum(param *types.ReqRandHash) ([]byte, error) {
	msg, err := api.api.Query(param.ExecName, "RandNumHash", param)
	if err != nil {
		return nil, err
	}
	reply, ok := msg.(*types.ReplyHash)
	if !ok {
		return nil, types.ErrTypeAsset
	}
	return reply.Hash, nil
}

func (api *mainChainAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	return api.api.GetBlockByHashes(param)
}

type paraChainAPI struct {
	api        client.QueueProtocolAPI
	grpcClient types.Chain33Client
}

func newParaChainAPI(api client.QueueProtocolAPI, grpcaddr string) ExecutorAPI {
	paraRemoteGrpcClient := types.Conf("config.consensus").GStr("ParaRemoteGrpcClient")
	if grpcaddr != "" {
		paraRemoteGrpcClient = grpcaddr
	}
	if paraRemoteGrpcClient == "" {
		paraRemoteGrpcClient = "127.0.0.1:8002"
	}
	conn, err := grpc.Dial(paraRemoteGrpcClient, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	grpcClient := types.NewChain33Client(conn)
	return &paraChainAPI{api: api, grpcClient: grpcClient}
}

func (api *paraChainAPI) GetRandNum(param *types.ReqRandHash) ([]byte, error) {
	reply, err := api.grpcClient.QueryRandNum(context.Background(), param)
	if err != nil {
		return nil, err
	}
	return reply.Hash, nil
}

func (api *paraChainAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	return api.grpcClient.GetBlockByHashes(context.Background(), param)
}
