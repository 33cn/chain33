package rpc

import (
	"encoding/json"
	"reflect"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/client"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type channelClient struct {
	client.QueueProtocolAPI
}

func (c *channelClient) Init(q queue.Client) {
	c.QueueProtocolAPI, _ = client.New(q, nil)
}

func (c *channelClient) CreateRawEvmCreateCallTx(parm *evmtypes.CreateCallTx) ([]byte, error) {
	return callExecNewTx(types.ExecName(types.EvmX), "CreateCall", parm)
}

func callExecNewTx(execName, action string, param interface{}) ([]byte, error) {
	exec := types.LoadExecutorType(execName)
	if exec == nil {
		log15.Error("callExecNewTx", "Error", "exec not found")
		return nil, types.ErrNotSupport
	}

	// param is interface{type, var-nil}, check with nil always fail
	if reflect.ValueOf(param).IsNil() {
		log15.Error("callExecNewTx", "Error", "param in nil")
		return nil, types.ErrInvalidParam
	}

	jsonStr, err := json.Marshal(param)
	if err != nil {
		log15.Error("callExecNewTx", "Error", err)
		return nil, err
	}

	tx, err := exec.CreateTx(action, json.RawMessage(jsonStr))
	if err != nil {
		log15.Error("callExecNewTx", "Error", err)
		return nil, err
	}

	txHex := types.Encode(tx)
	return txHex, nil
}
