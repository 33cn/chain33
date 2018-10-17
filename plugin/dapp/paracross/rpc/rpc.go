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

func (c *channelClient) ListTitles(req *types.ReqNil) (*pt.RespParacrossTitles, error) {
	in := &types.Query{
		Execer: []byte(pt.GetExecName()),
		FuncName: "ListTitles",
		Payload: nil,
	}
	data, err := c.Query(in)
	if err != nil {
		return nil, err
	}
	if resp, ok := data.(*pt.RespParacrossTitles); ok {
		return resp, nil
	}
	return nil, types.ErrDecode
}

func (c *Jrpc) ListTitles(req *types.ReqNil, result *interface{}) error {
	data, err := c.cli.ListTitles(req)
	*result = data
	return err
}

func (c *channelClient) GetTitleHeight(req *pt.ReqParacrossTitleHeight) (*pt.RespParacrossTitles, error) {
	in := &types.Query{
		Execer: []byte(pt.GetExecName()),
		FuncName: "GetTitleHeight",
		Payload: nil,
	}
	data, err := c.Query(in)
	if err != nil {
		return nil, err
	}
	if resp, ok := data.(*pt.RespParacrossTitles); ok {
		return resp, nil
	}
	return nil, types.ErrDecode
}

func (c *Jrpc) GetTitleHeight(req *pt.ReqParacrossTitleHeight, result *interface{}) error {
	if req == nil {
		return types.ErrInvalidParam
	}
	data, err := c.cli.GetTitleHeight(req)
	*result = data
	return err
}

func (c *channelClient) GetAssetTxResult(req *types.ReqHash) (*pt.ParacrossAsset, error) {
	in := &types.Query{
		Execer: []byte(pt.GetExecName()),
		FuncName: "GetAssetTxResult",
		Payload: nil,
	}
	data, err := c.Query(in)
	if err != nil {
		return nil, err
	}
	if resp, ok := data.(*pt.ParacrossAsset); ok {
		return resp, nil
	}
	return nil, types.ErrDecode
}

func (c *Jrpc) GetAssetTxResult(req *types.ReqHash, result *interface{}) error {
	if req == nil {
		return types.ErrInvalidParam
	}
	data, err := c.cli.GetAssetTxResult(req)
	*result = data
	return err
}

