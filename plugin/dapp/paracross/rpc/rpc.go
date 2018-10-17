package rpc

import (
	"gitlab.33.cn/chain33/chain33/types"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
)


func (c *channelClient) GetHeight(req *types.ReqString) (*pt.ParacrossStatus, error) {
	in := &types.Query{
		Execer: []byte(pt.GetExecName()),
		FuncName: "GetHeight",
		Payload: types.Encode(req),
	}
	data, err := c.Query(in)
	if err != nil {
		return nil, err
	}
	if resp, ok := data.(*pt.ParacrossStatus); ok {
		return resp, nil
	}
	return nil, types.ErrDecode
}
func (c *Jrpc) GetHeight(req *types.ReqString, result *interface{}) error {
	if req == nil {
		return types.ErrInvalidParam
	}
	data, err := c.cli.GetHeight(req)
	*result = data
	return err
}

// TODO
func (c *Jrpc) ListTitles() {

}

func (c *Jrpc) GetTitleHeight() {

}

func (c *Jrpc) GetAssetTxResult() {

}

