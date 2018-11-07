package rpc

import (
	"context"

	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *channelClient) Backup(ctx context.Context, v *rt.BackupRetrieve) (*types.UnsignTx, error) {
	backup := &rt.RetrieveAction{
		Ty:    rt.RetrieveActionBackup,
		Value: &rt.RetrieveAction_Backup{v},
	}
	tx, err := types.CreateFormatTx(string(rt.ExecerRetrieve), types.Encode(backup))
	if err != nil {
		return nil, err
	}
	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Prepare(ctx context.Context, v *rt.PrepareRetrieve) (*types.UnsignTx, error) {
	prepare := &rt.RetrieveAction{
		Ty:    rt.RetrieveActionPrepare,
		Value: &rt.RetrieveAction_Prepare{v},
	}
	tx, err := types.CreateFormatTx(string(rt.ExecerRetrieve), types.Encode(prepare))
	if err != nil {
		return nil, err
	}

	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Perform(ctx context.Context, v *rt.PerformRetrieve) (*types.UnsignTx, error) {
	perform := &rt.RetrieveAction{
		Ty:    rt.RetrieveActionPerform,
		Value: &rt.RetrieveAction_Perform{v},
	}
	tx, err := types.CreateFormatTx(string(rt.ExecerRetrieve), types.Encode(perform))
	if err != nil {
		return nil, err
	}
	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Cancel(ctx context.Context, v *rt.CancelRetrieve) (*types.UnsignTx, error) {
	cancel := &rt.RetrieveAction{
		Ty:    rt.RetrieveActionCancel,
		Value: &rt.RetrieveAction_Cancel{v},
	}
	tx, err := types.CreateFormatTx(string(rt.ExecerRetrieve), types.Encode(cancel))
	if err != nil {
		return nil, err
	}
	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}
