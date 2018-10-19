package rpc

import (
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *channelClient) GetHeight(req *types.ReqString) (*pt.ParacrossStatus, error) {
	data, err := c.Query(pt.GetExecName(), "GetHeight", req)
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
	data, err := c.Query(pt.GetExecName(), "ListTitles", req)
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
	data, err := c.Query(pt.GetExecName(),"GetTitleHeight", req)
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
	data, err := c.Query(pt.GetExecName(), "GetAssetTxResult", req)
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
