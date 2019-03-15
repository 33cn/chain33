package api

import (
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	qmocks "github.com/33cn/chain33/queue/mocks"
	"github.com/33cn/chain33/rpc"
	"github.com/33cn/chain33/rpc/grpcclient"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/status"
)

func TestAPI(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	gapi, err := grpcclient.NewMainChainClient("")
	assert.Nil(t, err)
	eapi := New(api, gapi)
	param := &types.ReqHashes{
		Hashes: [][]byte{[]byte("hello")},
	}
	api.On("GetBlockByHashes", mock.Anything).Return(&types.BlockDetails{}, nil)
	detail, err := eapi.GetBlockByHashes(param)
	assert.Nil(t, err)
	assert.Equal(t, detail, &types.BlockDetails{})
	param2 := &types.ReqRandHash{
		ExecName: "ticket",
		BlockNum: 5,
		Hash:     []byte("hello"),
	}
	api.On("Query", "ticket", "RandNumHash", mock.Anything).Return(&types.ReplyHash{Hash: []byte("hello")}, nil)
	randhash, err := eapi.GetRandNum(param2)
	assert.Nil(t, err)
	assert.Equal(t, randhash, []byte("hello"))
	assert.Equal(t, false, eapi.IsErr())
	api.On("QueryTx", mock.Anything).Return(&types.TransactionDetail{Height: 1}, nil)
	param3 := &types.ReqHash{Hash: []byte("hash")}
	txdetail, err := eapi.QueryTx(param3)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), txdetail.Height)
	types.SetTitleOnlyForTest("user.p.wzw.")
	//testnode setup
	rpcCfg := new(types.RPC)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8003"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8004"
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	rpc.InitCfg(rpcCfg)
	server := rpc.NewGRpcServer(&qmocks.Client{}, api)
	assert.NotNil(t, server)
	go server.Listen()
	time.Sleep(time.Second)

	eapi = New(api, gapi)
	_, err = eapi.GetBlockByHashes(param)
	assert.Equal(t, true, IsGrpcError(err))
	assert.Equal(t, true, IsGrpcError(status.New(codes.Aborted, "operation is abort").Err()))
	assert.Equal(t, false, IsGrpcError(nil))
	assert.Equal(t, false, IsGrpcError(errors.New("xxxx")))
	assert.Equal(t, true, eapi.IsErr())

	gapi2, err := grpcclient.NewMainChainClient("127.0.0.1:8003")
	assert.Nil(t, err)
	eapi = New(api, gapi2)
	detail, err = eapi.GetBlockByHashes(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, detail, &types.BlockDetails{})
	randhash, err = eapi.GetRandNum(param2)
	assert.Nil(t, err)
	assert.Equal(t, randhash, []byte("hello"))
	assert.Equal(t, false, eapi.IsErr())
	param3 = &types.ReqHash{Hash: []byte("hash")}
	txdetail, err = eapi.QueryTx(param3)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), txdetail.Height)
	//queue err
	assert.Equal(t, false, IsQueueError(nil))
	assert.Equal(t, false, IsQueueError(errors.New("xxxx")))
	assert.Equal(t, true, IsQueueError(queue.ErrQueueTimeout))
	assert.Equal(t, true, IsQueueError(queue.ErrIsQueueClosed))
	assert.Equal(t, false, IsQueueError(errors.New("ErrIsQueueClosed")))
}
