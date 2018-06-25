package rpc

import (
	"encoding/hex"
	"math/rand"
	"time"

	"encoding/json"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	tokentype "gitlab.33.cn/chain33/chain33/types/executor/token"
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

// support old rpc create transaction interface. call new imlp
func callExecNewTx(execName, action string, param interface{}) ([]byte, error) {
	exec := types.LoadExecutor(execName)
	if exec == nil {
		log.Error("callExecNewTx", "Error", "exec not found")
		return nil, types.ErrNotSupport
	}
	jsonStr, err := json.Marshal(param)
	if err != nil {
		log.Error("callExecNewTx", "Error", err)
		return nil, err
	}
	tx, err := exec.CreateTx(action, json.RawMessage(jsonStr))
	if err != nil {
		log.Error("callExecNewTx", "Error", err)
		return nil, err
	}
	txHex := types.Encode(tx)
	return txHex, nil
}

func (c *channelClient) CreateRawTransaction(param *types.CreateTx) ([]byte, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("CreateRawTransaction", "Error", err)
		return nil, err
	}

	if param.IsToken {
		return callExecNewTx("token", "", param)
	} else {
		return callExecNewTx("coins", "", param)
	}
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

func (c *channelClient) CreateRawTokenPreCreateTx(parm *tokentype.TokenPreCreateTx) ([]byte, error) {
	return callExecNewTx("token", "TokenPreCreate", parm)
}

func (c *channelClient) CreateRawTokenFinishTx(parm *tokentype.TokenFinishTx) ([]byte, error) {
	return callExecNewTx("token", "TokenFinish", parm)
}

func (c *channelClient) CreateRawTokenRevokeTx(parm *tokentype.TokenRevokeTx) ([]byte, error) {
	return callExecNewTx("token", "TokenRevoke", parm)
}

func (c *channelClient) CreateRawTradeSellTx(parm *TradeSellTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeSellLimit", parm)
}

func (c *channelClient) CreateRawTradeBuyTx(parm *TradeBuyTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeBuyMarket", parm)
}

func (c *channelClient) CreateRawTradeRevokeTx(parm *TradeRevokeTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeSellRevoke", parm)
}

func (c *channelClient) CreateRawTradeBuyLimitTx(parm *TradeBuyLimitTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeBuyLimit", parm)
}

func (c *channelClient) CreateRawTradeSellMarketTx(parm *TradeSellMarketTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeSellMarket", parm)
}

func (c *channelClient) CreateRawTradeRevokeBuyTx(parm *TradeRevokeBuyTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeRevokeBuy", parm)
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
