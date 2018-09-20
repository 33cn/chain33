package rpc

import (
	"context"
	"math/rand"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/address"
	pb "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type channelClient struct {
	client.QueueProtocolAPI
}

func (c *channelClient) Init(q queue.Client) {
	c.QueueProtocolAPI, _ = client.New(q, nil)
}

func (c *channelClient) Start(ctx context.Context, head *pb.PBGameStart) (*types.UnsignTx, error) {
	val := &pb.PBGameAction{
		Ty:    pb.PBGameActionStart,
		Value: &pb.PBGameAction_Start{head},
	}
	tx := &types.Transaction{
		Execer:  pb.ExecerPokerBull,
		Payload: types.Encode(val),
		Fee:     0,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(pb.ExecerPokerBull)),
	}
	tx.SetRealFee(types.MinFee)
	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Continue(ctx context.Context, head *pb.PBGameContinue) (*types.UnsignTx, error) {
	val := &pb.PBGameAction{
		Ty:    pb.PBGameActionContinue,
		Value: &pb.PBGameAction_Continue{head},
	}
	tx := &types.Transaction{
		Execer:  pb.ExecerPokerBull,
		Payload: types.Encode(val),
		Fee:     0,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(pb.ExecerPokerBull)),
	}
	tx.SetRealFee(types.MinFee)
	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Quit(ctx context.Context, head *pb.PBGameQuit) (*types.UnsignTx, error) {
	val := &pb.PBGameAction{
		Ty:    pb.PBGameActionQuit,
		Value: &pb.PBGameAction_Quit{head},
	}
	tx := &types.Transaction{
		Execer:  pb.ExecerPokerBull,
		Payload: types.Encode(val),
		Fee:     0,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(pb.ExecerPokerBull,)),
	}
	tx.SetRealFee(types.MinFee)
	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (c *channelClient) Show(ctx context.Context, head *pb.PBGameQuery) (*types.UnsignTx, error) {
	val := &pb.PBGameAction{
		Ty:    pb.PBGameActionQuery,
		Value: &pb.PBGameAction_Query{head},
	}
	tx := &types.Transaction{
		Execer:  pb.ExecerPokerBull,
		Payload: types.Encode(val),
		Fee:     0,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(pb.ExecerPokerBull,)),
	}
	tx.SetRealFee(types.MinFee)
	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}
