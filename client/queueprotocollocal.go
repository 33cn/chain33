package client

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/types"
)

var accountdb = account.NewCoinsAccount()

func (q *QueueProtocol) GetAddrOverview(param *types.ReqAddr) (*types.AddrOverview, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetAddrOverview", "Error", err)
		return nil, err
	}
	msg, err := q.query(blockchainKey, types.EventGetAddrOverview, param)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.AddrOverview); ok {
		//获取地址账户的余额通过account模块
		addrs := make([]string, 1)
		addrs[0] = param.Addr
		accounts, err := accountdb.LoadAccounts(q.client, addrs)
		if err != nil {
			return nil, err
		}
		if len(accounts) != 0 {
			reply.Balance = accounts[0].Balance
		}
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueProtocol) GetBalance(param *types.ReqBalance) ([]*types.Account, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBalance", "Error", err)
		return nil, err
	}

	execer := param.GetExecer()
	switch execer {
	case types.AllowUserExec[types.ExecTypeCoins]:
		addrs := param.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := account.CheckAddress(addr); err != nil {
				addr = account.ExecAddress(addr).String()
			}
			exaddrs = append(exaddrs, addr)
		}

		accounts, err := accountdb.LoadAccounts(q.client, exaddrs)
		if err != nil {
			log.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := account.ExecAddress(execer)
		addrs := param.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			acc, err := accountdb.LoadExecAccountQueue(q.client, addr, execaddress.String())
			if err != nil {
				log.Error("GetBalance", "err", err.Error())
				continue
			}
			accounts = append(accounts, acc)
		}
		return accounts, nil
	}
}

func (q *QueueProtocol) GetTokenBalance(param *types.ReqTokenBalance) ([]*types.Account, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetTokenBalance", "Error", err)
		return nil, err
	}

	tokenSymbol := param.GetTokenSymbol()
	execer := param.GetExecer()
	accountTokendb := account.NewTokenAccountWithoutDB(tokenSymbol)
	switch execer {
	case types.AllowUserExec[types.ExecTypeToken]:
		addrs := param.GetAddresses()
		var queryAddrs []string
		for _, addr := range addrs {
			if err := account.CheckAddress(addr); err != nil {
				addr = account.ExecAddress(addr).String()
			}
			queryAddrs = append(queryAddrs, addr)
		}

		accounts, err := accountTokendb.LoadAccounts(q.client, queryAddrs)
		if err != nil {
			log.Error("GetTokenBalance", "err", err.Error(), "token symbol", tokenSymbol, "address", queryAddrs)
			return nil, err
		}
		return accounts, nil

	default: //trade
		execaddress := account.ExecAddress(execer)
		addrs := param.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			acc, err := accountTokendb.LoadExecAccountQueue(q.client, addr, execaddress.String())
			if err != nil {
				log.Error("GetTokenBalance for exector", "err", err.Error(), "token symbol", tokenSymbol,
					"address", addr)
				continue
			}
			accounts = append(accounts, acc)
		}

		return accounts, nil
	}
}
