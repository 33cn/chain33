package rpc

import (
	"math/rand"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

//提供系统rpc接口

var accountdb = account.NewCoinsAccount()

type channelClient struct {
	queue.Client
	api client.QueueProtocolAPI
}

func (c *channelClient) Init(q queue.Client) {
	c.Client = q
	c.api, _ = client.New(q, nil)
}

func (c *channelClient) CreateRawTransaction(parm *types.CreateTx) ([]byte, error) {
	return c.api.CreateRawTransaction(parm)
}

func (c *channelClient) SendRawTransaction(parm *types.SignedTx) queue.Message {
	msg, err := c.api.SendRawTransaction(parm)
	if err != nil {
		msg.Data = err
	}
	return msg
}

//channel
func (c *channelClient) SendTx(tx *types.Transaction) (*types.Reply, error) {
	return c.api.SendTx(tx)
}

func (c *channelClient) GetBlocks(start int64, end int64, isdetail bool) (*types.BlockDetails, error) {
	return c.api.GetBlocks(&types.ReqBlocks{start, end, isdetail, []string{""}})
}

func (c *channelClient) QueryTx(hash []byte) (*types.TransactionDetail, error) {
	return c.api.QueryTx(&types.ReqHash{hash})
}

func (c *channelClient) GetLastHeader() (*types.Header, error) {
	return c.api.GetLastHeader()
}

func (c *channelClient) GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfos, error) {
	return c.api.GetTransactionByAddr(parm)
}

func (c *channelClient) GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error) {
	return c.api.GetTransactionByHash(parm)
}

func (c *channelClient) GetMempool() (*types.ReplyTxList, error) {
	return c.api.GetMempool()
}

func (c *channelClient) GetAccounts() (*types.WalletAccounts, error) {
	return c.api.WalletGetAccountList()
}

func (c *channelClient) NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error) {
	return c.api.NewAccount(parm)
}

func (c *channelClient) WalletTxList(parm *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	return c.api.WalletTransactionList(parm)
}

func (c *channelClient) ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	return c.api.WalletImportprivkey(parm)
}

func (c *channelClient) SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	return c.api.WalletSendToAddress(parm)
}

func (c *channelClient) SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error) {
	return c.api.WalletSetFee(parm)
}

func (c *channelClient) SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	return c.api.WalletSetLabel(parm)
}

func (c *channelClient) MergeBalance(parm *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	return c.api.WalletMergeBalance(parm)
}

func (c *channelClient) SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error) {
	return c.api.WalletSetPasswd(parm)
}

func (c *channelClient) Lock() (*types.Reply, error) {
	return c.api.WalletLock()
}

func (c *channelClient) UnLock(parm *types.WalletUnLock) (*types.Reply, error) {
	return c.api.WalletUnLock(parm)
}

func (c *channelClient) GetPeerInfo() (*types.PeerList, error) {
	return c.api.PeerInfo()
}

func (c *channelClient) GetHeaders(in *types.ReqBlocks) (*types.Headers, error) {
	return c.api.GetHeaders(&types.ReqBlocks{
		Start:    in.GetStart(),
		End:      in.GetEnd(),
		Isdetail: in.GetIsdetail()})
}

func (c *channelClient) GetLastMemPool(*types.ReqNil) (*types.ReplyTxList, error) {
	return c.api.GetLastMempool(nil)
}

func (c *channelClient) GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error) {
	return c.api.GetBlockOverview(parm)
}

func (c *channelClient) GetAddrOverview(parm *types.ReqAddr) (*types.AddrOverview, error) {
	return c.api.GetAddrOverview(parm)
}

func (c *channelClient) GetBlockHash(parm *types.ReqInt) (*types.ReplyHash, error) {
	return c.api.GetBlockHash(parm)
}

//seed
func (c *channelClient) GenSeed(parm *types.GenSeedLang) (*types.ReplySeed, error) {
	return c.api.GenSeed(parm)
}

func (c *channelClient) SaveSeed(parm *types.SaveSeedByPw) (*types.Reply, error) {
	return c.api.SaveSeed(parm)
}

func (c *channelClient) GetSeed(parm *types.GetSeedByPw) (*types.ReplySeed, error) {
	return c.api.GetSeed(parm)
}

func (c *channelClient) GetWalletStatus() (*WalletStatus, error) {
	reply, err := c.api.GetWalletStatus()
	if nil != err {
		return nil, err
	}
	return (*WalletStatus)(reply), nil
}

func (c *channelClient) GetBalance(in *types.ReqBalance) ([]*types.Account, error) {

	switch in.GetExecer() {
	case "coins":
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := account.CheckAddress(addr); err != nil {
				addr = account.ExecAddress(addr).String()

			}
			exaddrs = append(exaddrs, addr)
		}

		accounts, err := accountdb.LoadAccounts(c.Client, exaddrs)
		if err != nil {
			log.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := account.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {

			acc, err := accountdb.LoadExecAccountQueue(c.Client, addr, execaddress.String())
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
	accountTokendb := account.NewTokenAccountWithoutDB(in.GetTokenSymbol())

	switch in.GetExecer() {
	case "token":
		addrs := in.GetAddresses()
		var queryAddrs []string
		for _, addr := range addrs {
			if err := account.CheckAddress(addr); err != nil {
				addr = string(accountTokendb.AccountKey(addr))
			}
			queryAddrs = append(queryAddrs, addr)
		}

		accounts, err := accountTokendb.LoadAccounts(c.Client, queryAddrs)
		if err != nil {
			log.Error("GetTokenBalance", "err", err.Error(), "token symbol", in.GetTokenSymbol(), "address", queryAddrs)
			return nil, err
		}
		return accounts, nil

	default: //trade
		execaddress := account.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			acc, err := accountTokendb.LoadExecAccountQueue(c.Client, addr, execaddress.String())
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

func (c *channelClient) QueryHash(in *types.Query) (*types.Message, error) {

	msg := c.NewMessage("blockchain", types.EventQuery, in)
	err := c.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	querydata := resp.GetData().(types.Message)
	return &querydata, nil

}

func (c *channelClient) SetAutoMiner(in *types.MinerFlag) (*types.Reply, error) {
	return c.api.WalletAutoMiner(in)
}

func (c *channelClient) GetTicketCount() (*types.Int64, error) {
	return c.api.GetTicketCount()
}

func (c *channelClient) DumpPrivkey(in *types.ReqStr) (*types.ReplyStr, error) {
	return c.api.DumpPrivkey(in)
}

func (c *channelClient) CloseTickets() (*types.ReplyHashes, error) {
	return c.api.CloseTickets()
}

func (c *channelClient) GetTotalCoins(in *types.ReqGetTotalCoins) (*types.ReplyGetTotalCoins, error) {
	//获取地址账户的余额通过account模块
	resp, err := accountdb.GetTotalCoins(c.Client, in)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *channelClient) IsSync() bool {
	reply, err := c.api.IsSync()
	if nil != err {
		reply = false
	}
	return reply
}

func (c *channelClient) IsNtpClockSync() bool {
	reply, err := c.api.IsNtpClockSync()
	if err != nil {
		reply = false
	}
	return reply
}

func (c *channelClient) QueryTotalFee(in *types.ReqHash) (*types.LocalReplyValue, error) {
	return c.api.LocalGet(in)
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
		To:      account.ExecAddress("token").String(),
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
		To:      account.ExecAddress("token").String(),
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
		To:      account.ExecAddress("token").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeSellTx(parm *TradeSellTx) ([]byte, error) {
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
		Ty:    types.TradeSellLimit,
		Value: &types.Trade_Tokensell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeBuyTx(parm *TradeBuyTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForBuy{Sellid: parm.SellId, Boardlotcnt: parm.BoardlotCnt}
	buy := &types.Trade{
		Ty:    types.TradeBuyMarket,
		Value: &types.Trade_Tokenbuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeRevokeTx(parm *TradeRevokeTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeSell{Sellid: parm.SellId}
	buy := &types.Trade{
		Ty:    types.TradeRevokeSell,
		Value: &types.Trade_Tokenrevokesell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) SignRawTx(in *types.ReqSignRawTx) (*types.ReplySignRawTx, error) {
	data := &types.ReqSignRawTx{
		Addr:    in.GetAddr(),
		PrivKey: in.GetPrivKey(),
		TxHex:   in.GetTxHex(),
		Expire:  in.GetExpire(),
	}
	msg := c.NewMessage("wallet", types.EventSignRawTx, data)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("SignRawTx", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.ReplySignRawTx), nil
}

func (c *channelClient) GetNetInfo() (*types.NodeNetInfo, error) {
	msg := c.NewMessage("p2p", types.EventGetNetInfo, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("NetInfo", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.GetData().(*types.NodeNetInfo), nil
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
		To:      account.ExecAddress("trade").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeSellMarketTx(parm *TradeSellMarketTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForSellMarket{Buyid: parm.BuyId, BoardlotCnt: parm.BoardlotCnt}
	sellMarket := &types.Trade{
		Ty:    types.TradeSellMarket,
		Value: &types.Trade_Tokensellmarket{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(sellMarket),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawTradeRevokeBuyTx(parm *TradeRevokeBuyTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeBuy{Buyid: parm.BuyId}
	buy := &types.Trade{
		Ty:    types.TradeRevokeBuy,
		Value: &types.Trade_Tokenrevokebuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("trade"),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("trade").String(),
	}

	data := types.Encode(tx)
	return data, nil
}
