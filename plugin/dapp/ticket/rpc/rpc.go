package rpc

import (
	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
	context "golang.org/x/net/context"
)

func bindMiner(param *ty.ReqBindMiner) (*ty.ReplyBindMiner, error) {
	tBind := &ty.TicketBind{
		MinerAddress:  param.BindAddr,
		ReturnAddress: param.OriginAddr,
	}
	data, err := types.CallCreateTx(types.ExecName(ty.TicketX), "Tbind", tBind)
	if err != nil {
		return nil, err
	}
	hex := common.ToHex(data)
	return &ty.ReplyBindMiner{TxHex: hex}, nil
}

// 创建绑定挖矿
func (g *channelClient) CreateBindMiner(ctx context.Context, in *ty.ReqBindMiner) (*ty.ReplyBindMiner, error) {
	if in.Amount%(10000*types.Coin) != 0 || in.Amount < 0 {
		return nil, types.ErrAmount
	}
	err := address.CheckAddress(in.BindAddr)
	if err != nil {
		return nil, err
	}
	err = address.CheckAddress(in.OriginAddr)
	if err != nil {
		return nil, err
	}

	if in.CheckBalance {
		getBalance := &types.ReqBalance{Addresses: []string{in.OriginAddr}, Execer: "coins"}
		balances, err := g.GetCoinsAccountDB().GetBalance(g, getBalance)
		if err != nil {
			return nil, err
		}
		if len(balances) == 0 {
			return nil, types.ErrInvalidParam
		}
		if balances[0].Balance < in.Amount+2*types.Coin {
			return nil, types.ErrNoBalance
		}
	}
	return bindMiner(in)
}

func (g *channelClient) SetAutoMining(ctx context.Context, in *ty.MinerFlag) (*types.Reply, error) {
	data, err := g.ExecWalletFunc(ty.TicketX, "WalletAutoMiner", in)
	if err != nil {
		return nil, err
	}
	return data.(*types.Reply), nil
}

func (g *channelClient) GetTicketCount(ctx context.Context, in *types.ReqNil) (*types.Int64, error) {
	data, err := g.QueryConsensusFunc(ty.TicketX, "GetTicketCount", &types.ReqNil{})
	if err != nil {
		return nil, err
	}
	return data.(*types.Int64), nil
}

func (g *channelClient) CloseTickets(ctx context.Context, in *types.ReqNil) (*types.ReplyHashes, error) {
	data, err := g.ExecWalletFunc(ty.TicketX, "CloseTickets", &types.ReqNil{})
	if err != nil {
		return nil, err
	}
	return data.(*types.ReplyHashes), nil
}

func (c *Jrpc) CreateBindMiner(in *ty.ReqBindMiner, result *interface{}) error {
	reply, err := c.cli.CreateBindMiner(context.Background(), in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Jrpc) GetTicketCount(in *types.ReqNil, result *int64) error {
	resp, err := c.cli.GetTicketCount(context.Background(), &types.ReqNil{})
	if err != nil {
		return err
	}
	*result = resp.GetData()
	return nil

}

func (c *Jrpc) CloseTickets(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.CloseTickets(context.Background(), &types.ReqNil{})
	if err != nil {
		return err
	}
	var hashes rpctypes.ReplyHashes
	for _, has := range resp.Hashes {
		hashes.Hashes = append(hashes.Hashes, hex.EncodeToString(has))
	}
	*result = &hashes
	return nil
}

func (c *Jrpc) SetAutoMining(in *ty.MinerFlag, result *rpctypes.Reply) error {
	resp, err := c.cli.SetAutoMining(context.Background(), in)
	if err != nil {
		return err
	}
	var reply rpctypes.Reply
	reply.IsOk = resp.GetIsOk()
	reply.Msg = string(resp.GetMsg())
	*result = reply
	return nil
}
