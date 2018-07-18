package rpc

import (
	"encoding/hex"
	"math/rand"

	"encoding/json"

	"reflect"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	tokentype "gitlab.33.cn/chain33/chain33/types/executor/token"
	tradetype "gitlab.33.cn/chain33/chain33/types/executor/trade"
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
	execName = types.ExecName(execName)
	exec := types.LoadExecutor(execName)
	if exec == nil {
		log.Error("callExecNewTx", "Error", "exec not found")
		return nil, types.ErrNotSupport
	}

	// param is interface{type, var-nil}, check with nil always fail
	if reflect.ValueOf(param).IsNil() {
		log.Error("callExecNewTx", "Error", "param in nil")
		return nil, types.ErrInvalidParam
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

func (c *channelClient) GetAllExecBalance(in *types.ReqAddr) (*types.AllExecBalance, error) {
	addr := in.Addr
	err := address.CheckAddress(addr)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}
	var addrs []string
	addrs = append(addrs, addr)
	allBalance := &types.AllExecBalance{Addr: addr}
	for _, exec := range types.AllowUserExec {
		execer := string(exec)
		params := &types.ReqBalance{
			Addresses: addrs,
			Execer:    execer,
		}
		res, err := c.GetBalance(params)
		if err != nil {
			continue
		}
		if len(res) < 1 {
			continue
		}
		acc := res[0]
		if acc.Balance == 0 && acc.Frozen == 0 {
			continue
		}
		execAcc := &types.ExecAccount{Execer: execer, Account: acc}
		allBalance.ExecAccount = append(allBalance.ExecAccount, execAcc)
	}
	return allBalance, nil
}

//TODO:和GetBalance进行泛化处理，同时LoadAccounts和LoadExecAccountQueue也需要进行泛化处理, added by hzj
func (c *channelClient) GetTokenBalance(in *types.ReqTokenBalance) ([]*types.Account, error) {
	accountTokendb, err := account.NewAccountDB(types.ExecName("token"), in.GetTokenSymbol(), nil)
	if err != nil {
		return nil, err
	}
	switch in.GetExecer() {
	case types.ExecName("token"):
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

func (c *channelClient) CreateRawTradeSellTx(parm *tradetype.TradeSellTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeSellLimit", parm)
}

func (c *channelClient) CreateRawTradeBuyTx(parm *tradetype.TradeBuyTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeBuyMarket", parm)
}

func (c *channelClient) CreateRawTradeRevokeTx(parm *tradetype.TradeRevokeTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeSellRevoke", parm)
}

func (c *channelClient) CreateRawTradeBuyLimitTx(parm *tradetype.TradeBuyLimitTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeBuyLimit", parm)
}

func (c *channelClient) CreateRawTradeSellMarketTx(parm *tradetype.TradeSellMarketTx) ([]byte, error) {
	return callExecNewTx("trade", "TradeSellMarket", parm)
}

func (c *channelClient) CreateRawTradeRevokeBuyTx(parm *tradetype.TradeRevokeBuyTx) ([]byte, error) {
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
	random := rand.New(rand.NewSource(types.Now().UnixNano()))
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
	ntpTime := common.GetRealTimeRetry(types.NtpHosts, 10)
	local := types.Now()
	if ntpTime.IsZero() {
		return &types.TimeStatus{NtpTime: "", LocalTime: local.Format("2006-01-02 15:04:05"), Diff: 0}, nil
	}
	diff := local.Unix() - ntpTime.Unix()
	return &types.TimeStatus{NtpTime: ntpTime.Format("2006-01-02 15:04:05"), LocalTime: local.Format("2006-01-02 15:04:05"), Diff: diff}, nil
}

func (c *channelClient) CreateRawRelayOrderTx(parm *RelayOrderTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayCreate{
		Operation: parm.Operation,
		Coin:      parm.Coin,
		Amount:    parm.Amount,
		Addr:      parm.Addr,
		CoinWaits: parm.CoinWait,
		BtyAmount: parm.BtyAmount,
	}
	sell := &types.RelayAction{
		Ty:    types.RelayActionCreate,
		Value: &types.RelayAction_Create{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayAcceptTx(parm *RelayAcceptTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayAccept{OrderId: parm.OrderId, CoinAddr: parm.CoinAddr, CoinWaits: parm.CoinWait}
	val := &types.RelayAction{
		Ty:    types.RelayActionAccept,
		Value: &types.RelayAction_Accept{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayRevokeTx(parm *RelayRevokeTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayRevoke{OrderId: parm.OrderId, Target: parm.Target, Action: parm.Action}
	val := &types.RelayAction{
		Ty:    types.RelayActionRevoke,
		Value: &types.RelayAction_Revoke{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayConfirmTx(parm *RelayConfirmTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayConfirmTx{OrderId: parm.OrderId, TxHash: parm.TxHash}
	val := &types.RelayAction{
		Ty:    types.RelayActionConfirmTx,
		Value: &types.RelayAction_ConfirmTx{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayVerifyBTCTx(parm *RelayVerifyBTCTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayVerifyCli{
		OrderId:    parm.OrderId,
		RawTx:      parm.RawTx,
		TxIndex:    parm.TxIndex,
		MerkBranch: parm.MerklBranch,
		BlockHash:  parm.BlockHash}
	val := &types.RelayAction{
		Ty:    types.RelayActionVerifyCmdTx,
		Value: &types.RelayAction_VerifyCli{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelaySaveBTCHeadTx(parm *RelaySaveBTCHeadTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	head := &types.BtcHeader{
		Hash:         parm.Hash,
		PreviousHash: parm.PreviousHash,
		MerkleRoot:   parm.MerkleRoot,
		Height:       parm.Height,
		IsReset:      parm.IsReset,
	}

	v := &types.BtcHeaders{}
	v.BtcHeader = append(v.BtcHeader, head)

	val := &types.RelayAction{
		Ty:    types.RelayActionRcvBTCHeaders,
		Value: &types.RelayAction_BtcHeaders{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(types.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	data := types.Encode(tx)
	return data, nil
}
