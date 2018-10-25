package rpc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util/testnode"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

func TestExecQuery(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	mock33.Listen()

	rpcCfg := mock33.GetCfg().Rpc
	jsonClient, err := jsonclient.NewJSONClient("http://" + rpcCfg.JrpcBindAddr + "/")
	assert.Nil(t, err)
	assert.NotNil(t, jsonClient)

	js, err := types.PBToJson(&ty.ReqRelayQryBTCHeadHeight{})
	assert.Nil(t, err)
	in := &rpctypes.Query4Jrpc{
		Execer:   "relay",
		FuncName: "GetBTCHeaderCurHeight",
		Payload:  js,
	}
	var reply ty.ReplayRelayQryBTCHeadHeight
	err = jsonClient.Call("Chain33.Query", in, &reply)
	assert.Nil(t, err)
	assert.Equal(t, int64(-1), reply.CurHeight)
}
