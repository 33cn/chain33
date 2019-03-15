//这个包提供了平行链和主链的统一的访问接口

package api

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

//ErrAPIEnv api的执行环境出问题，区块执行的时候，遇到这一个的错误需要retry
var errAPIEnv = errors.New("ErrAPIEnv")

//ExecutorAPI 提供给执行器使用的接口
//因为合约是主链和平行链通用的，所以，主链和平行链都可以调用这套接口
type ExecutorAPI interface {
	GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error)
	GetRandNum(param *types.ReqRandHash) ([]byte, error)
	QueryTx(param *types.ReqHash) (*types.TransactionDetail, error)
	IsErr() bool
}

type mainChainAPI struct {
	api     client.QueueProtocolAPI
	errflag int32
}

//New 新建接口
func New(api client.QueueProtocolAPI, grpcClient types.Chain33Client) ExecutorAPI {
	if types.IsPara() {
		return newParaChainAPI(api, grpcClient)
	}
	return &mainChainAPI{api: api}
}

func (api *mainChainAPI) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	data, err := api.api.QueryTx(param)
	return data, seterr(err, &api.errflag)
}

func (api *mainChainAPI) IsErr() bool {
	return atomic.LoadInt32(&api.errflag) == 1
}

func (api *mainChainAPI) GetRandNum(param *types.ReqRandHash) ([]byte, error) {
	msg, err := api.api.Query(param.ExecName, "RandNumHash", param)
	if err != nil {
		return nil, seterr(err, &api.errflag)
	}
	reply, ok := msg.(*types.ReplyHash)
	if !ok {
		return nil, types.ErrTypeAsset
	}
	return reply.Hash, nil
}

func (api *mainChainAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	data, err := api.api.GetBlockByHashes(param)
	return data, seterr(err, &api.errflag)
}

type paraChainAPI struct {
	api        client.QueueProtocolAPI
	grpcClient types.Chain33Client
	errflag    int32
}

func newParaChainAPI(api client.QueueProtocolAPI, grpcClient types.Chain33Client) ExecutorAPI {
	return &paraChainAPI{api: api, grpcClient: grpcClient}
}

func (api *paraChainAPI) IsErr() bool {
	return atomic.LoadInt32(&api.errflag) == 1
}

func (api *paraChainAPI) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	data, err := api.grpcClient.QueryTransaction(context.Background(), param)
	if err != nil {
		err = errAPIEnv
	}
	return data, seterr(err, &api.errflag)
}

func (api *paraChainAPI) GetRandNum(param *types.ReqRandHash) ([]byte, error) {
	reply, err := api.grpcClient.QueryRandNum(context.Background(), param)
	if err != nil {
		err = errAPIEnv
		return nil, seterr(err, &api.errflag)
	}
	return reply.Hash, nil
}

func (api *paraChainAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	data, err := api.grpcClient.GetBlockByHashes(context.Background(), param)
	if err != nil {
		err = errAPIEnv
	}
	return data, seterr(err, &api.errflag)
}

func seterr(err error, flag *int32) error {
	if IsGrpcError(err) || IsQueueError(err) {
		atomic.StoreInt32(flag, 1)
	}
	return err
}

//IsGrpcError 判断系统api 错误，还是 rpc 本身的错误
func IsGrpcError(err error) bool {
	if err == nil {
		return false
	}
	if err == errAPIEnv {
		return true
	}
	if grpc.Code(err) == codes.Unknown {
		return false
	}
	return true
}

//IsQueueError 是否是队列错误
func IsQueueError(err error) bool {
	if err == nil {
		return false
	}
	if err == errAPIEnv {
		return true
	}
	if err == queue.ErrQueueTimeout ||
		err == queue.ErrQueueChannelFull ||
		err == queue.ErrIsQueueClosed {
		return true
	}
	return false
}

//IsAPIEnvError 是否是api执行环境的错误
func IsAPIEnvError(err error) bool {
	return IsGrpcError(err) || IsQueueError(err)
}
