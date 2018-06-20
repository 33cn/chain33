package rpc

import (
	"encoding/hex"
	"math/rand"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

//提供系统rpc接口

type channelClient struct {
	client.QueueProtocolAPI
	accountdb *account.DB
}

func (c *channelClient) Init(q queue.Client) {
	c.QueueProtocolAPI, _ = client.New(q, nil)
	c.accountdb = account.NewCoinsAccount()
}

func (c *channelClient) CreateRawTransaction(param *types.CreateTx) ([]byte, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("CreateRawTransaction", "Error", err)
		return nil, err
	}

	if param.ExecName != "" && !types.IsAllowExecName(param.ExecName) {
		log.Error("CreateRawTransaction", "Error", types.ErrExecNameNotMatch)
		return nil, types.ErrExecNameNotMatch
	}
	//to地址要么是普通用户地址，要么就是执行器地址，不能为空
	if param.To == "" {
		return nil, types.ErrAddrNotExist
	}

	var tx *types.Transaction
	if param.Amount < 0 {
		return nil, types.ErrAmount
	}
	if param.IsToken {
		tx = createTokenTransfer(param)
	} else {
		tx = createCoinsTransfer(param)
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	txHex := types.Encode(tx)

	return txHex, nil
}

func createCoinsTransfer(param *types.CreateTx) *types.Transaction {
	transfer := &types.CoinsAction{}
	if !param.IsWithdraw {
		if param.ExecName != "" {
			v := &types.CoinsAction_TransferToExec{TransferToExec: &types.CoinsTransferToExec{
				Amount: param.Amount, Note: param.GetNote(), ExecName: param.GetExecName()}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionTransferToExec
		} else {
			v := &types.CoinsAction_Transfer{Transfer: &types.CoinsTransfer{
				Amount: param.Amount, Note: param.GetNote()}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionTransfer
		}
	} else {
		v := &types.CoinsAction_Withdraw{Withdraw: &types.CoinsWithdraw{
			Amount: param.Amount, Note: param.GetNote()}}
		transfer.Value = v
		transfer.Ty = types.CoinsActionWithdraw
	}
	return &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), To: param.GetTo()}
}

func createTokenTransfer(param *types.CreateTx) *types.Transaction {
	transfer := &types.TokenAction{}
	if !param.IsWithdraw {
		v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{
			Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote()}}
		transfer.Value = v
		transfer.Ty = types.ActionTransfer
	} else {
		v := &types.TokenAction_Withdraw{Withdraw: &types.CoinsWithdraw{
			Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote()}}
		transfer.Value = v
		transfer.Ty = types.ActionWithdraw
	}
	return &types.Transaction{Execer: []byte("token"), Payload: types.Encode(transfer), To: param.GetTo()}
}

func (c *channelClient) SendRawTransaction(param *types.SignedTx) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("SendRawTransaction", "Error", err)
		return nil, err
	}
	var tx types.Transaction
	err := types.Decode(param.GetUnsign(), &tx)
	if err == nil {
		tx.Signature = &types.Signature{param.GetTy(), param.GetPubkey(), param.GetSign()}
		return c.SendTx(&tx)
	}
	return nil, err
}

func (c *channelClient) GetAddrOverview(parm *types.ReqAddr) (*types.AddrOverview, error) {
	err := address.CheckAddress(parm.Addr)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}
	reply, err := c.QueueProtocolAPI.GetAddrOverview(parm)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	//获取地址账户的余额通过account模块
	addrs := make([]string, 1)
	addrs[0] = parm.Addr
	accounts, err := c.accountdb.LoadAccounts(c.QueueProtocolAPI, addrs)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	if len(accounts) != 0 {
		reply.Balance = accounts[0].Balance
	}
	return reply, nil
}

func (c *channelClient) GetBalance(in *types.ReqBalance) ([]*types.Account, error) {

	switch in.GetExecer() {
	case "coins":
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := address.CheckAddress(addr); err != nil {
				addr = address.ExecAddress(addr)

			}
			exaddrs = append(exaddrs, addr)
		}
		var accounts []*types.Account
		var err error
		if len(in.StateHash) == 0 {
			accounts, err = c.accountdb.LoadAccounts(c.QueueProtocolAPI, exaddrs)
		} else {
			hash, err := common.FromHex(in.StateHash)
			if err != nil {
				return nil, err
			}
			accounts, err = c.accountdb.LoadAccountsHistory(c.QueueProtocolAPI, exaddrs, hash)
		}
		if err != nil {
			log.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := address.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {

			var acc *types.Account
			var err error
			if len(in.StateHash) == 0 {
				acc, err = c.accountdb.LoadExecAccountQueue(c.QueueProtocolAPI, addr, execaddress)
			} else {
				hash, err := common.FromHex(in.StateHash)
				if err != nil {
					return nil, err
				}
				acc, err = c.accountdb.LoadExecAccountHistoryQueue(c.QueueProtocolAPI, addr, execaddress, hash)
			}
			if err != nil {
				log.Error("GetBalance", "err", err.Error())
				continue
			}
			accounts = append(accounts, acc)
		}

		return accounts, nil
	}
}

//TODO:和GetBalance进行泛化处理，同时LoadAccounts和LoadExecAccountQueue也需要进行泛化处理, added by hzj
func (c *channelClient) GetTokenBalance(in *types.ReqTokenBalance) ([]*types.Account, error) {
	accountTokendb, err := account.NewAccountDB("token", in.GetTokenSymbol(), nil)
	if err != nil {
		return nil, err
	}
	switch in.GetExecer() {
	case "token":
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

func (c *channelClient) GetTotalCoins(in *types.ReqGetTotalCoins) (*types.ReplyGetTotalCoins, error) {
	//获取地址账户的余额通过account模块
	resp, err := c.accountdb.GetTotalCoins(c.QueueProtocolAPI, in)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *channelClient) CreateRawTokenPreCreateTx(parm *TokenPreCreateTx) ([]byte, error) {
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
	tx := &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(precreate),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("token"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTokenFinishTx(parm *TokenFinishTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TokenFinishCreate{Symbol: parm.Symbol, Owner: parm.OwnerAddr}
	finish := &types.TokenAction{
		Ty:    types.TokenActionFinishCreate,
		Value: &types.TokenAction_Tokenfinishcreate{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(finish),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("token"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTokenRevokeTx(parm *TokenRevokeTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TokenRevokeCreate{Symbol: parm.Symbol, Owner: parm.OwnerAddr}
	revoke := &types.TokenAction{
		Ty:    types.TokenActionRevokeCreate,
		Value: &types.TokenAction_Tokenrevokecreate{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(revoke),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("token"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeSellTx(parm *TradeSellTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForSell{
		TokenSymbol:       parm.TokenSymbol,
		AmountPerBoardlot: parm.AmountPerBoardlot,
		MinBoardlot:       parm.MinBoardlot,
		PricePerBoardlot:  parm.PricePerBoardlot,
		TotalBoardlot:     parm.TotalBoardlot,
		Starttime:         0,
		Stoptime:          0,
		Crowdfund:         false,
	}
	sell := &types.Trade{
		Ty:    types.TradeSellLimit,
		Value: &types.Trade_Tokensell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("trade"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeBuyTx(parm *TradeBuyTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForBuy{SellID: parm.SellID, BoardlotCnt: parm.BoardlotCnt}
	buy := &types.Trade{
		Ty:    types.TradeBuyMarket,
		Value: &types.Trade_Tokenbuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("trade"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeRevokeTx(parm *TradeRevokeTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeSell{SellID: parm.SellID}
	buy := &types.Trade{
		Ty:    types.TradeRevokeSell,
		Value: &types.Trade_Tokenrevokesell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("trade"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeBuyLimitTx(parm *TradeBuyLimitTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForBuyLimit{
		TokenSymbol:       parm.TokenSymbol,
		AmountPerBoardlot: parm.AmountPerBoardlot,
		MinBoardlot:       parm.MinBoardlot,
		PricePerBoardlot:  parm.PricePerBoardlot,
		TotalBoardlot:     parm.TotalBoardlot,
	}
	buyLimit := &types.Trade{
		Ty:    types.TradeBuyLimit,
		Value: &types.Trade_Tokenbuylimit{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buyLimit),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("trade"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeSellMarketTx(parm *TradeSellMarketTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForSellMarket{BuyID: parm.BuyID, BoardlotCnt: parm.BoardlotCnt}
	sellMarket := &types.Trade{
		Ty:    types.TradeSellMarket,
		Value: &types.Trade_Tokensellmarket{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(sellMarket),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("trade"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeRevokeBuyTx(parm *TradeRevokeBuyTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeBuy{BuyID: parm.BuyID}
	buy := &types.Trade{
		Ty:    types.TradeRevokeBuy,
		Value: &types.Trade_Tokenrevokebuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("trade"),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) BindMiner(param *types.ReqBindMiner) (*types.ReplyBindMiner, error) {
	ta := &types.TicketAction{}
	tBind := &types.TicketBind{
		MinerAddress:  param.BindAddr,
		ReturnAddress: param.OriginAddr,
	}
	ta.Value = &types.TicketAction_Tbind{Tbind: tBind}
	ta.Ty = types.TicketActionBind
	execer := []byte("ticket")
	to := address.ExecAddress(string(execer))
	txBind := &types.Transaction{Execer: execer, Payload: types.Encode(ta), To: to}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	txBind.Nonce = random.Int63()
	var err error
	txBind.Fee, err = txBind.GetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}
	txBind.Fee += types.MinFee
	txBindHex := types.Encode(txBind)
	txHexStr := hex.EncodeToString(txBindHex)

	return &types.ReplyBindMiner{TxHex: txHexStr}, nil
}

func (c *channelClient) DecodeRawTransaction(param *types.ReqDecodeRawTransaction) (*types.Transaction, error) {
	var tx types.Transaction
	bytes, err := common.FromHex(param.TxHex)
	if err != nil {
		return nil, err
	}
	err = types.Decode(bytes, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (c *channelClient) GetTimeStatus() (*types.TimeStatus, error) {
	var diffTmp []int64
	var ntpTime time.Time
	var local time.Time
	var diff int64
	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond * 100)
		errTimes := 0
		for {
			time.Sleep(time.Millisecond * 100)
			var err error
			ntpTime, err = common.GetNtpTime("time.windows.com:123")
			if err != nil {
				errTimes++
			} else {
				break
			}
			if errTimes == 10 {
				return &types.TimeStatus{NtpTime: "", LocalTime: time.Now().Format("2006-01-02 15:04:05"), Diff: 0}, nil
			}
		}

		local = time.Now()
		diff = local.Unix() - ntpTime.Unix()
		diffTmp = append(diffTmp, diff)
	}
	for j := 0; j < 2; j++ {
		for k := j + 1; k < 3; k++ {
			if diffTmp[j] != diffTmp[k] {
				return &types.TimeStatus{NtpTime: "", LocalTime: time.Now().Format("2006-01-02 15:04:05"), Diff: 0}, nil
			}
		}
	}
	return &types.TimeStatus{NtpTime: ntpTime.Format("2006-01-02 15:04:05"), LocalTime: local.Format("2006-01-02 15:04:05"), Diff: diff}, nil
}
