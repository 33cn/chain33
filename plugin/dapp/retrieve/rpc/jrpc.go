package rpc

import (
	"context"
	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
)

func (c *Jrpc) CreateRawRetrieveBackupTx(in *RetrieveBackupTx, result *interface{}) error {
	head := &types.BackupRetrieve{
		BackupAddress:  in.BackupAddr,
		DefaultAddress: in.DefaultAddr,
		DelayPeriod:    in.DelayPeriod,
	}

	reply, err := c.cli.Backup(context.Background(), head)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (c *Jrpc) CreateRawRetrievePrepareTx(in *RetrievePrepareTx, result *interface{}) error {
	head := &types.PrepareRetrieve{
		BackupAddress:  in.BackupAddr,
		DefaultAddress: in.DefaultAddr,
	}

	reply, err := c.cli.Prepare(context.Background(), head)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (c *Jrpc) CreateRawRetrievePerformTx(in *RetrievePerformTx, result *interface{}) error {
	head := &types.PerformRetrieve{
		BackupAddress:  in.BackupAddr,
		DefaultAddress: in.DefaultAddr,
	}
	reply, err := c.cli.Perform(context.Background(), head)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (c *Jrpc) CreateRawRetrieveCancelTx(in *RetrieveCancelTx, result *interface{}) error {
	head := &types.CancelRetrieve{
		BackupAddress:  in.BackupAddr,
		DefaultAddress: in.DefaultAddr,
	}

	reply, err := c.cli.Cancel(context.Background(), head)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply.Data)
	return nil
}
