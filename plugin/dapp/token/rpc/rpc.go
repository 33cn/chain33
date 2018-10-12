package rpc

import (
	"context"
	"encoding/hex"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var log = log15.New("module", "token.rpc")

//TODO:和GetBalance进行泛化处理，同时LoadAccounts和LoadExecAccountQueue也需要进行泛化处理, added by hzj
func (c *channelClient) GetTokenBalance(in *tokenty.ReqTokenBalance) ([]*types.Account, error) {
	accountTokendb, err := account.NewAccountDB(types.TokenX, in.GetTokenSymbol(), nil)
	if err != nil {
		return nil, err
	}
	switch in.GetExecer() {
	case types.ExecName(types.TokenX):
		addrs := in.GetAddresses()
		var queryAddrs []string
		for _, addr := range addrs {
			if err := address.CheckAddress(addr); err != nil {
				addr = string(accountTokendb.AccountKey(addr))
			}
			queryAddrs = append(queryAddrs, addr)
		}

		accounts, err := accountTokendb.LoadAccounts(c.QueueProtocolAPI, queryAddrs)
		if err != nil {
			log.Error("GetTokenBalance", "err", err.Error(), "token symbol", in.GetTokenSymbol(), "address", queryAddrs)
			return nil, err
		}
		return accounts, nil

	default: //trade
		execaddress := address.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			acc, err := accountTokendb.LoadExecAccountQueue(c.QueueProtocolAPI, addr, execaddress)
			if err != nil {
				log.Error("GetTokenBalance for exector", "err", err.Error(), "token symbol", in.GetTokenSymbol(),
					"address", addr)
				continue
			}
			accounts = append(accounts, acc)
		}

		return accounts, nil
	}
}

func (c *channelClient) CreateRawTokenPreCreateTx(parm *tokenty.TokenPreCreateTx) ([]byte, error) {
	return callExecNewTx(types.ExecName(types.TokenX), "TokenPreCreate", parm)
}

func (c *channelClient) CreateRawTokenFinishTx(parm *tokenty.TokenFinishTx) ([]byte, error) {
	return callExecNewTx(types.ExecName(types.TokenX), "TokenFinish", parm)
}

func (c *channelClient) CreateRawTokenRevokeTx(parm *tokenty.TokenRevokeTx) ([]byte, error) {
	return callExecNewTx(types.ExecName(types.TokenX), "TokenRevoke", parm)
}

func (g *Grpc) GetTokenBalance(ctx context.Context, in *tokenty.ReqTokenBalance) (*pb.Accounts, error) {
	reply, err := g.cli.GetTokenBalance(in)
	if err != nil {
		return nil, err
	}
	return &pb.Accounts{Acc: reply}, nil
}

func (c *Chain33) GetTokenBalance(in tokenty.ReqTokenBalance, result *interface{}) error {

	balances, err := c.cli.GetTokenBalance(&in)
	if err != nil {
		return err
	}
	var accounts []*rpctypes.Account
	for _, balance := range balances {
		accounts = append(accounts, &rpctypes.Account{Addr: balance.GetAddr(),
			Balance:  balance.GetBalance(),
			Currency: balance.GetCurrency(),
			Frozen:   balance.GetFrozen()})
	}
	*result = accounts
	return nil
}

func (c *Chain33) CreateRawTokenPreCreateTx(in *tokenty.TokenPreCreateTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenPreCreateTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTokenFinishTx(in *tokenty.TokenFinishTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenFinishTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTokenRevokeTx(in *tokenty.TokenRevokeTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
