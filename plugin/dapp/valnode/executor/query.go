package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (val *ValNode) Query_GetValNodeByHeight(in *pty.ReqNodeInfo) (types.Message, error) {
	height := in.GetHeight()

	if height <= 0 {
		return nil, types.ErrInvalidParam
	}
	key := CalcValNodeUpdateHeightKey(height)
	values, err := val.GetLocalDB().List(key, nil, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, types.ErrNotFound
	}

	reply := &pty.ValNodes{}
	for _, valnodeByte := range values {
		var valnode pty.ValNode
		err := types.Decode(valnodeByte, &valnode)
		if err != nil {
			return nil, err
		}
		reply.Nodes = append(reply.Nodes, &valnode)
	}
	return reply, nil
}

func (val *ValNode) Query_GetBlockInfoByHeight(in *pty.ReqBlockInfo) (types.Message, error) {
	height := in.GetHeight()

	if height <= 0 {
		return nil, types.ErrInvalidParam
	}
	key := CalcValNodeBlockInfoHeightKey(height)
	value, err := val.GetLocalDB().Get(key)
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, types.ErrNotFound
	}

	reply := &pty.TendermintBlockInfo{}
	err = types.Decode(value, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}
