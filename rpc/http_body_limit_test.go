package rpc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/33cn/chain33/client/mocks"
	qmocks "github.com/33cn/chain33/queue/mocks"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func TestJSONRPCBodyLimit(t *testing.T) {
	rpcCfg = new(types.RPC)
	rpcCfg.JrpcBindAddr = "127.0.0.1:0"
	rpcCfg.Whitelist = []string{"127.0.0.1"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	InitCfg(rpcCfg)

	api := new(mocks.QueueProtocolAPI)
	api.On("GetConfig").Return(types.NewChain33Config(types.GetDefaultCfgstring()))
	api.On("Close").Return()
	qm := &qmocks.Client{}
	qm.On("GetConfig").Return(types.NewChain33Config(types.GetDefaultCfgstring()))

	server := NewJSONRPCServer(qm, api)
	require.NotNil(t, server)
	defer server.Close()

	port, err := server.Listen()
	require.NoError(t, err)
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	post := func(body io.Reader) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodPost, url, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return http.DefaultClient.Do(req)
	}

	// 超过上限：必须返回明确错误，而不是和断连/超时共用的 Can't get request body!
	oversized := bytes.Repeat([]byte("a"), maxJSONRPCBodyBytes+1)
	resp, err := post(bytes.NewReader(oversized))
	require.NoError(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Contains(t, string(body), "request body too large")

	// 小 body：仍应进入 JSON 解析，说明上限没有误伤正常请求。
	resp, err = post(strings.NewReader("not-json"))
	require.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Contains(t, string(body), "invalid json request err")
}
