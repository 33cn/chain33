package rpc

import (
	context "golang.org/x/net/context"

	bw "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *channelClient) Create(ctx context.Context, head *bw.BlackwhiteCreate) (*types.UnsignTx, error) {
	val := &bw.BlackwhiteAction{
		Ty:    bw.BlackwhiteActionCreate,
		Value: &bw.BlackwhiteAction_Create{head},
	}
	tx := &types.Transaction{
		Payload: types.Encode(val),
	}
	data, err := types.FormatTxEncode(string(bw.ExecerBlackwhite), tx)
	if err != nil {
		return nil, err
	}
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Show(ctx context.Context, head *bw.BlackwhiteShow) (*types.UnsignTx, error) {
	val := &bw.BlackwhiteAction{
		Ty:    bw.BlackwhiteActionShow,
		Value: &bw.BlackwhiteAction_Show{head},
	}
	tx := &types.Transaction{
		Payload: types.Encode(val),
	}
	data, err := types.FormatTxEncode(string(bw.ExecerBlackwhite), tx)
	if err != nil {
		return nil, err
	}
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Play(ctx context.Context, head *bw.BlackwhitePlay) (*types.UnsignTx, error) {
	val := &bw.BlackwhiteAction{
		Ty:    bw.BlackwhiteActionPlay,
		Value: &bw.BlackwhiteAction_Play{head},
	}
	tx := &types.Transaction{
		Payload: types.Encode(val),
	}
	data, err := types.FormatTxEncode(string(bw.ExecerBlackwhite), tx)
	if err != nil {
		return nil, err
	}
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) TimeoutDone(ctx context.Context, head *bw.BlackwhiteTimeoutDone) (*types.UnsignTx, error) {
	val := &bw.BlackwhiteAction{
		Ty:    bw.BlackwhiteActionTimeoutDone,
		Value: &bw.BlackwhiteAction_TimeoutDone{head},
	}
	tx := &types.Transaction{
		Payload: types.Encode(val),
	}
	data, err := types.FormatTxEncode(string(bw.ExecerBlackwhite), tx)
	if err != nil {
		return nil, err
	}
	return &types.UnsignTx{Data: data}, nil
}
