package rpc

import (
	"encoding/hex"

	evmtype "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
)

type Jrpc struct {
	cli channelClient
}

func (c *Jrpc) CreateRawEvmCreateCallTx(in *evmtype.CreateCallTx, result *interface{}) error {
	reply, err := c.cli.CreateRawEvmCreateCallTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
