package rpc

import (
	"context"
	"encoding/hex"

	bw "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Jrpc) BlackwhiteCreateTx(parm *bw.BlackwhiteCreateTxReq, result *interface{}) error {
	if parm == nil {
		return types.ErrInvalidParam
	}
	head := &bw.BlackwhiteCreate{
		PlayAmount:  parm.PlayAmount,
		PlayerCount: parm.PlayerCount,
		Timeout:     parm.Timeout,
		GameName:    parm.GameName,
	}
	reply, err := c.cli.Create(context.Background(), head)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (c *Jrpc) BlackwhiteShowTx(parm *BlackwhiteShowTx, result *interface{}) error {
	if parm == nil {
		return types.ErrInvalidParam
	}
	head := &bw.BlackwhiteShow{
		GameID: parm.GameID,
		Secret: parm.Secret,
	}
	reply, err := c.cli.Show(context.Background(), head)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (c *Jrpc) BlackwhitePlayTx(parm *BlackwhitePlayTx, result *interface{}) error {
	if parm == nil {
		return types.ErrInvalidParam
	}

	head := &bw.BlackwhitePlay{
		GameID:     parm.GameID,
		Amount:     parm.Amount,
		HashValues: parm.HashValues,
	}
	reply, err := c.cli.Play(context.Background(), head)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (c *Jrpc) BlackwhiteTimeoutDoneTx(parm *BlackwhiteTimeoutDoneTx, result *interface{}) error {
	if parm == nil {
		return types.ErrInvalidParam
	}
	head := &bw.BlackwhiteTimeoutDone{
		GameID: parm.GameID,
	}
	reply, err := c.cli.TimeoutDone(context.Background(), head)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply.Data)
	return nil
}
