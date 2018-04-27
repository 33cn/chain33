package client

import (
	"math/rand"
	"time"

	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/account"
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

func (q *QueueProtocol) GetTotalCoins(param *types.ReqGetTotalCoins) (*types.ReplyGetTotalCoins, error) {
	//获取地址账户的余额通过account模块
	resp, err := accountdb.GetTotalCoins(q.client, param)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (q *QueueProtocol) CreateRawTradeRevokeTx(param *TradeRevokeTx) (*types.Transaction, error) {
	if param == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeSell{Sellid: param.SellId}
	buy := &types.Trade{
		Ty:    types.TradeRevokeSell,
		Value: &types.Trade_Tokenrevokesell{v},
	}
	return &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     param.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}, nil
}

func (q *QueueProtocol) CreateRawTradeBuyTx(param *TradeBuyTx) (*types.Transaction, error) {
	if param == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForBuy{Sellid: param.SellId, Boardlotcnt: param.BoardlotCnt}
	buy := &types.Trade{
		Ty:    types.TradeBuy,
		Value: &types.Trade_Tokenbuy{v},
	}
	return &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     param.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}, nil
}

func (q *QueueProtocol) CreateRawTokenPreCreateTx(parm *TokenPreCreateTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TokenPreCreate{
		Name:         parm.Name,
		Symbol:       parm.Symbol,
		Introduction: parm.Introduction,
		Total:        parm.Total,
		Price:        parm.Price,
		Owner:        parm.OwnerAddr,
	}
	precreate := &types.TokenAction{
		Ty:    types.TokenActionPreCreate,
		Value: &types.TokenAction_Tokenprecreate{v},
	}
	return &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(precreate),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("token").String(),
	}, nil
}

func (q *QueueProtocol) CreateRawTokenFinishTx(param *TokenFinishTx) (*types.Transaction, error) {
	if param == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TokenFinishCreate{Symbol: param.Symbol, Owner: param.OwnerAddr}
	finish := &types.TokenAction{
		Ty:    types.TokenActionFinishCreate,
		Value: &types.TokenAction_Tokenfinishcreate{v},
	}
	return &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(finish),
		Fee:     param.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("token").String(),
	}, nil
}

func (q *QueueProtocol) CreateRawTokenRevokeTx(parm *TokenRevokeTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TokenRevokeCreate{Symbol: parm.Symbol, Owner: parm.OwnerAddr}
	revoke := &types.TokenAction{
		Ty:    types.TokenActionRevokeCreate,
		Value: &types.TokenAction_Tokenrevokecreate{v},
	}
	return &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(revoke),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("token").String(),
	}, nil
}

func (q *QueueProtocol) CreateRawTradeSellTx(parm *TradeSellTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForSell{
		Tokensymbol:       parm.TokenSymbol,
		Amountperboardlot: parm.AmountPerBoardlot,
		Minboardlot:       parm.MinBoardlot,
		Priceperboardlot:  parm.PricePerBoardlot,
		Totalboardlot:     parm.TotalBoardlot,
		Starttime:         0,
		Stoptime:          0,
		Crowdfund:         false,
	}
	sell := &types.Trade{
		Ty:    types.TradeSell,
		Value: &types.Trade_Tokensell{v},
	}
	return &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}, nil
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
