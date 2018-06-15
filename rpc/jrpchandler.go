package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Chain33) CreateRawTransaction(in *types.CreateTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTransaction(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil

}

func (c *Chain33) SendRawTransaction(in SignedTx, result *interface{}) error {
	var stx types.SignedTx
	var err error

	stx.Pubkey, err = hex.DecodeString(in.Pubkey)
	if err != nil {
		return err
	}

	stx.Sign, err = hex.DecodeString(in.Sign)
	if err != nil {
		return err
	}
	stx.Unsign, err = hex.DecodeString(in.Unsign)
	if err != nil {
		return err
	}
	stx.Ty = in.Ty
	reply, err := c.cli.SendRawTransaction(&stx)
	if err != nil {
		return err
	}
	if reply.IsOk {
		*result = "0x" + hex.EncodeToString(reply.Msg)
		return nil
	} else {
		return fmt.Errorf(string(reply.Msg))
	}
}

func (c *Chain33) SendTransaction(in RawParm, result *interface{}) error {
	var parm types.Transaction
	data, err := common.FromHex(in.Data)
	if err != nil {
		return err
	}
	types.Decode(data, &parm)
	log.Debug("SendTransaction", "parm", parm)
	reply, err := c.cli.SendTx(&parm)
	if err == nil {
		*result = common.ToHex(reply.GetMsg())
	}
	return err
}

func (c *Chain33) GetHexTxByHash(in QueryParm, result *interface{}) error {
	var data types.ReqHash
	hash, err := common.FromHex(in.Hash)
	if err != nil {
		return err
	}
	data.Hash = hash
	reply, err := c.cli.QueryTx(&data)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(types.Encode(reply.GetTx()))
	return err
}

func (c *Chain33) QueryTransaction(in QueryParm, result *interface{}) error {
	var data types.ReqHash
	hash, err := common.FromHex(in.Hash)
	if err != nil {
		return err
	}

	data.Hash = hash
	reply, err := c.cli.QueryTx(&data)
	if err != nil {
		return err
	}

	{ //重新格式化数据

		var transDetail TransactionDetail
		transDetail.Tx, err = DecodeTx(reply.GetTx())
		if err != nil {
			return err
		}

		receiptTmp := &ReceiptData{Ty: reply.GetReceipt().GetTy()}
		logs := reply.GetReceipt().GetLogs()
		for _, log := range logs {
			receiptTmp.Logs = append(receiptTmp.Logs,
				&ReceiptLog{Ty: log.GetTy(), Log: common.ToHex(log.GetLog())})
		}

		transDetail.Receipt, err = DecodeLog(receiptTmp)
		if err != nil {
			return err
		}

		for _, proof := range reply.Proofs {
			transDetail.Proofs = append(transDetail.Proofs, common.ToHex(proof))
		}
		transDetail.Height = reply.GetHeight()
		transDetail.Index = reply.GetIndex()
		transDetail.Blocktime = reply.GetBlocktime()
		transDetail.Amount = reply.GetAmount()
		transDetail.Fromaddr = reply.GetFromaddr()
		transDetail.ActionName = reply.GetActionName()

		*result = &transDetail
	}
	return nil
}

func (c *Chain33) GetBlocks(in BlockParam, result *interface{}) error {
	reply, err := c.cli.GetBlocks(&types.ReqBlocks{Start: in.Start, End: in.End, IsDetail: in.Isdetail, Pid: []string{""}})
	if err != nil {
		return err
	}
	{
		var blockDetails BlockDetails
		items := reply.GetItems()
		for _, item := range items {
			var bdtl BlockDetail
			var block Block
			block.BlockTime = item.Block.GetBlockTime()
			block.Height = item.Block.GetHeight()
			block.Version = item.Block.GetVersion()
			block.ParentHash = common.ToHex(item.Block.GetParentHash())
			block.StateHash = common.ToHex(item.Block.GetStateHash())
			block.TxHash = common.ToHex(item.Block.GetTxHash())
			txs := item.Block.GetTxs()
			for _, tx := range txs {
				tran, err := DecodeTx(tx)
				if err != nil {
					continue
				}
				block.Txs = append(block.Txs, tran)
			}
			bdtl.Block = &block

			for _, rp := range item.Receipts {
				var recp ReceiptData
				recp.Ty = rp.GetTy()
				for _, log := range rp.Logs {
					recp.Logs = append(recp.Logs,
						&ReceiptLog{Ty: log.Ty, Log: common.ToHex(log.GetLog())})
				}
				rd, err := DecodeLog(&recp)
				if err != nil {
					continue
				}
				bdtl.Receipts = append(bdtl.Receipts, rd)
			}

			blockDetails.Items = append(blockDetails.Items, &bdtl)
		}
		*result = &blockDetails
	}

	return nil

}

func (c *Chain33) GetLastHeader(in *types.ReqNil, result *interface{}) error {

	reply, err := c.cli.GetLastHeader()
	if err != nil {
		return err
	}

	{
		var header Header
		header.BlockTime = reply.GetBlockTime()
		header.Height = reply.GetHeight()
		header.ParentHash = common.ToHex(reply.GetParentHash())
		header.StateHash = common.ToHex(reply.GetStateHash())
		header.TxHash = common.ToHex(reply.GetTxHash())
		header.Version = reply.GetVersion()
		header.Hash = common.ToHex(reply.GetHash())
		header.TxCount = reply.TxCount
		*result = &header
	}

	return nil
}

//GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfo, error)
func (c *Chain33) GetTxByAddr(in types.ReqAddr, result *interface{}) error {
	reply, err := c.cli.GetTransactionByAddr(&in)
	if err != nil {
		return err
	}
	{
		var txinfos ReplyTxInfos
		infos := reply.GetTxInfos()
		for _, info := range infos {
			txinfos.TxInfos = append(txinfos.TxInfos, &ReplyTxInfo{Hash: common.ToHex(info.GetHash()),
				Height: info.GetHeight(), Index: info.GetIndex()})
		}
		*result = &txinfos
	}

	return nil
}

/*
GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error)
	GetMempool() (*types.ReplyTxList, error)
	GetAccounts() (*types.WalletAccounts, error)
*/

func (c *Chain33) GetTxByHashes(in ReqHashes, result *interface{}) error {
	log.Warn("GetTxByHashes", "hashes", in)
	var parm types.ReqHashes
	parm.Hashes = make([][]byte, 0)
	for _, v := range in.Hashes {
		//hb := common.FromHex(v)
		hb, err := common.FromHex(v)
		if err != nil {
			continue
		}
		parm.Hashes = append(parm.Hashes, hb)

	}
	reply, err := c.cli.GetTransactionByHash(&parm)
	if err != nil {
		return err
	}
	txs := reply.GetTxs()
	log.Info("GetTxByHashes", "get tx with count", len(txs))
	var txdetails TransactionDetails
	if 0 != len(txs) {
		for _, tx := range txs {
			var recp ReceiptData
			var proofs []string
			var recpResult *ReceiptDataResult
			var err error
			recp.Ty = tx.GetReceipt().GetTy()
			logs := tx.GetReceipt().GetLogs()
			if in.DisableDetail {
				logs = nil
			}
			for _, lg := range logs {
				recp.Logs = append(recp.Logs,
					&ReceiptLog{Ty: lg.Ty, Log: common.ToHex(lg.GetLog())})
			}
			recpResult, err = DecodeLog(&recp)
			if err != nil {
				log.Error("GetTxByHashes", "Failed to DecodeLog for type", err)
				continue
			}
			txProofs := tx.GetProofs()
			for _, proof := range txProofs {
				proofs = append(proofs, common.ToHex(proof))
			}
			tran, err := DecodeTx(tx.GetTx())
			if err != nil {
				log.Info("GetTxByHashes", "Failed to DecodeTx due to", err)
				continue
			}
			txdetails.Txs = append(txdetails.Txs,
				&TransactionDetail{
					Tx:         tran,
					Height:     tx.GetHeight(),
					Index:      tx.GetIndex(),
					Blocktime:  tx.GetBlocktime(),
					Receipt:    recpResult,
					Proofs:     proofs,
					Amount:     tx.GetAmount(),
					Fromaddr:   tx.GetFromaddr(),
					ActionName: tx.GetActionName(),
				})
		}
	}
	*result = &txdetails
	return nil
}

func (c *Chain33) GetMempool(in *types.ReqNil, result *interface{}) error {

	reply, err := c.cli.GetMempool()
	if err != nil {
		return err
	}
	{
		var txlist ReplyTxList
		txs := reply.GetTxs()
		for _, tx := range txs {
			amount, err := tx.Amount()
			if err != nil {
				amount = 0
			}
			from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
			tran, err := DecodeTx(tx)
			if err != nil {
				continue
			}
			tran.Amount = amount
			tran.From = from
			txlist.Txs = append(txlist.Txs, tran)
		}
		*result = &txlist
	}

	return nil
}

func (c *Chain33) GetAccounts(in *types.ReqNil, result *interface{}) error {

	reply, err := c.cli.WalletGetAccountList()
	if err != nil {
		return err
	}
	var accounts WalletAccounts
	for _, wallet := range reply.Wallets {
		accounts.Wallets = append(accounts.Wallets, &WalletAccount{Label: wallet.GetLabel(),
			Acc: &Account{Currency: wallet.GetAcc().GetCurrency(), Balance: wallet.GetAcc().GetBalance(),
				Frozen: wallet.GetAcc().GetFrozen(), Addr: wallet.GetAcc().GetAddr()}})
	}
	*result = &accounts
	return nil
}

/*
	NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error)
	WalletTxList(parm *types.ReqWalletTransactionList) (*types.TransactionDetails, error)
	ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error)
	SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error)

*/

func (c *Chain33) NewAccount(in types.ReqNewAccount, result *interface{}) error {
	reply, err := c.cli.NewAccount(&in)
	if err != nil {
		return err
	}

	*result = reply
	return nil
}

func (c *Chain33) WalletTxList(in ReqWalletTransactionList, result *interface{}) error {
	var parm types.ReqWalletTransactionList
	parm.FromTx = []byte(in.FromTx)
	parm.Count = in.Count
	parm.Direction = in.Direction
	parm.Isprivacy = in.Isprivacy
	reply, err := c.cli.WalletTransactionList(&parm)
	if err != nil {
		return err
	}
	{
		var txdetails WalletTxDetails

		for _, tx := range reply.TxDetails {
			var recp ReceiptData
			logs := tx.GetReceipt().GetLogs()
			recp.Ty = tx.GetReceipt().GetTy()
			for _, lg := range logs {
				recp.Logs = append(recp.Logs,
					&ReceiptLog{Ty: lg.Ty, Log: common.ToHex(lg.GetLog())})
			}
			rd, err := DecodeLog(&recp)
			if err != nil {
				continue
			}
			tran, err := DecodeTx(tx.GetTx())
			if err != nil {
				continue
			}
			txdetails.TxDetails = append(txdetails.TxDetails, &WalletTxDetail{
				Tx:         tran,
				Receipt:    rd,
				Height:     tx.GetHeight(),
				Index:      tx.GetIndex(),
				BlockTime:  tx.GetBlocktime(),
				Amount:     tx.GetAmount(),
				FromAddr:   tx.GetFromaddr(),
				TxHash:     common.ToHex(tx.GetTxhash()),
				ActionName: tx.GetActionName(),
			})
		}
		*result = &txdetails
	}

	return nil
}

func (c *Chain33) ImportPrivkey(in types.ReqWalletImportPrivKey, result *interface{}) error {
	reply, err := c.cli.WalletImportprivkey(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) SendToAddress(in types.ReqWalletSendToAddress, result *interface{}) error {
	log.Debug("Rpc SendToAddress", "Tx", in)
	reply, err := c.cli.WalletSendToAddress(&in)
	if err != nil {
		log.Debug("SendToAddress", "Error", err.Error())
		return err
	}
	log.Debug("sendtoaddr", "msg", reply.String())
	*result = &ReplyHash{Hash: common.ToHex(reply.GetHash())}
	log.Debug("SendToAddress", "resulrt", *result)
	return nil
}

/*
	SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error)
	SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error)
	MergeBalance(parm *types.ReqWalletMergeBalance) (*types.ReplyHashes, error)
	SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error)
*/

func (c *Chain33) SetTxFee(in types.ReqWalletSetFee, result *interface{}) error {
	reply, err := c.cli.WalletSetFee(&in)
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

func (c *Chain33) SetLabl(in types.ReqWalletSetLabel, result *interface{}) error {
	reply, err := c.cli.WalletSetLabel(&in)
	if err != nil {
		return err
	}

	*result = &WalletAccount{Acc: &Account{Addr: reply.GetAcc().Addr, Currency: reply.GetAcc().GetCurrency(),
		Frozen: reply.GetAcc().GetFrozen(), Balance: reply.GetAcc().GetBalance()}, Label: reply.GetLabel()}
	return nil
}

func (c *Chain33) MergeBalance(in types.ReqWalletMergeBalance, result *interface{}) error {
	reply, err := c.cli.WalletMergeBalance(&in)
	if err != nil {
		return err
	}
	{
		var hashes ReplyHashes
		for _, has := range reply.Hashes {
			hashes.Hashes = append(hashes.Hashes, common.ToHex(has))
		}
		*result = &hashes
	}

	return nil
}

func (c *Chain33) SetPasswd(in types.ReqWalletSetPasswd, result *interface{}) error {
	reply, err := c.cli.WalletSetPasswd(&in)
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

/*
	Lock() (*types.Reply, error)
	UnLock(parm *types.WalletUnLock) (*types.Reply, error)
	GetPeerInfo() (*types.PeerList, error)
*/

func (c *Chain33) Lock(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.WalletLock()
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

func (c *Chain33) UnLock(in types.WalletUnLock, result *interface{}) error {
	reply, err := c.cli.WalletUnLock(&in)
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

func (c *Chain33) GetPeerInfo(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.PeerInfo()
	if err != nil {
		return err
	}
	{

		var peerlist PeerList
		for _, peer := range reply.Peers {
			var pr Peer
			pr.Addr = peer.GetAddr()
			pr.MempoolSize = peer.GetMempoolSize()
			pr.Name = peer.GetName()
			pr.Port = peer.GetPort()
			pr.Self = peer.GetSelf()
			pr.Header = &Header{
				BlockTime:  peer.Header.GetBlockTime(),
				Height:     peer.Header.GetHeight(),
				ParentHash: common.ToHex(peer.GetHeader().GetParentHash()),
				StateHash:  common.ToHex(peer.GetHeader().GetStateHash()),
				TxHash:     common.ToHex(peer.GetHeader().GetTxHash()),
				Version:    peer.GetHeader().GetVersion(),
				Hash:       common.ToHex(peer.GetHeader().GetHash()),
				TxCount:    peer.GetHeader().GetTxCount(),
			}
			peerlist.Peers = append(peerlist.Peers, &pr)
		}
		*result = &peerlist
	}

	return nil
}

func (c *Chain33) GetHeaders(in types.ReqBlocks, result *interface{}) error {
	reply, err := c.cli.GetHeaders(&in)
	if err != nil {
		return err
	}
	var headers Headers
	{
		for _, item := range reply.Items {
			headers.Items = append(headers.Items, &Header{
				BlockTime:  item.GetBlockTime(),
				TxCount:    item.GetTxCount(),
				Hash:       common.ToHex(item.GetHash()),
				Height:     item.GetHeight(),
				ParentHash: common.ToHex(item.GetParentHash()),
				StateHash:  common.ToHex(item.GetStateHash()),
				TxHash:     common.ToHex(item.GetTxHash()),
				Version:    item.GetVersion()})
		}
		*result = &headers
	}
	return nil
}

func (c *Chain33) GetLastMemPool(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.GetLastMempool()
	if err != nil {
		return err
	}

	{
		var txlist ReplyTxList
		txs := reply.GetTxs()
		for _, tx := range txs {
			tran, err := DecodeTx(tx)
			if err != nil {
				continue
			}
			txlist.Txs = append(txlist.Txs, tran)
		}
		*result = &txlist
	}
	return nil
}

//GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (c *Chain33) GetBlockOverview(in QueryParm, result *interface{}) error {
	var data types.ReqHash
	hash, err := common.FromHex(in.Hash)
	if err != nil {
		return err
	}

	data.Hash = hash
	reply, err := c.cli.GetBlockOverview(&data)
	if err != nil {
		return err
	}
	var blockOverview BlockOverview

	//获取blockheader信息
	var header Header
	header.BlockTime = reply.GetHead().GetBlockTime()
	header.Height = reply.GetHead().GetHeight()
	header.ParentHash = common.ToHex(reply.GetHead().GetParentHash())
	header.StateHash = common.ToHex(reply.GetHead().GetStateHash())
	header.TxHash = common.ToHex(reply.GetHead().GetTxHash())
	header.Version = reply.GetHead().GetVersion()
	header.Hash = common.ToHex(reply.GetHead().GetHash())
	header.TxCount = reply.GetHead().GetTxCount()
	blockOverview.Head = &header

	//获取blocktxhashs信息
	for _, has := range reply.GetTxHashes() {
		blockOverview.TxHashes = append(blockOverview.TxHashes, common.ToHex(has))
	}

	blockOverview.TxCount = reply.GetTxCount()
	*result = &blockOverview
	return nil
}

func (c *Chain33) GetAddrOverview(in types.ReqAddr, result *interface{}) error {
	reply, err := c.cli.GetAddrOverview(&in)
	if err != nil {
		return err
	}
	type AddrOverview struct {
		Reciver int64 `json:"reciver"`
		Balance int64 `json:"balance"`
		TxCount int64 `json:"txCount"`
	}

	*result = (*AddrOverview)(reply)
	return nil
}

func (c *Chain33) GetBlockHash(in types.ReqInt, result *interface{}) error {
	reply, err := c.cli.GetBlockHash(&in)
	if err != nil {
		return err
	}
	var replyHash ReplyHash
	replyHash.Hash = common.ToHex(reply.GetHash())
	*result = &replyHash
	return nil
}

//seed
func (c *Chain33) GenSeed(in types.GenSeedLang, result *interface{}) error {
	reply, err := c.cli.GenSeed(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) SaveSeed(in types.SaveSeedByPw, result *interface{}) error {
	reply, err := c.cli.SaveSeed(&in)
	if err != nil {
		return err
	}

	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

func (c *Chain33) GetSeed(in types.GetSeedByPw, result *interface{}) error {
	reply, err := c.cli.GetSeed(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) GetWalletStatus(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.GetWalletStatus()
	if err != nil {
		return err
	}

	*result = *(*WalletStatus)(reply)
	return nil
}

func (c *Chain33) GetBalance(in types.ReqBalance, result *interface{}) error {

	balances, err := c.cli.GetBalance(&in)
	if err != nil {
		return err
	}
	var accounts []*Account
	for _, balance := range balances {
		accounts = append(accounts, &Account{Addr: balance.GetAddr(),
			Balance:  balance.GetBalance(),
			Currency: balance.GetCurrency(),
			Frozen:   balance.GetFrozen()})
	}
	*result = accounts
	return nil
}

func (c *Chain33) GetTokenBalance(in types.ReqTokenBalance, result *interface{}) error {

	balances, err := c.cli.GetTokenBalance(&in)
	if err != nil {
		return err
	}
	var accounts []*Account
	for _, balance := range balances {
		accounts = append(accounts, &Account{Addr: balance.GetAddr(),
			Balance:  balance.GetBalance(),
			Currency: balance.GetCurrency(),
			Frozen:   balance.GetFrozen()})
	}
	*result = accounts
	return nil
}

func (c *Chain33) QueryOld(in Query4Jrpc, result *interface{}) error {
	decodePayload, err := protoPayload(in.Execer, in.FuncName, &in.Payload)
	if err != nil {
		return err
	}

	resp, err := c.cli.Query(&types.Query{Execer: []byte(in.Execer), FuncName: in.FuncName, Payload: decodePayload})
	if err != nil {
		log.Error("EventQuery", "err", err.Error())
		return err
	}

	*result = resp
	return nil
}

func (c *Chain33) Query(in Query4Jrpc, result *interface{}) error {
	trans, ok := types.RpcTypeUtilMap[in.FuncName]
	if !ok {
		// 不是所有的合约都需要做类型转化， 没有修改的合约走老的接口
		// 另外给部分合约的代码修改的时间
		//log.Info("EventQuery", "Old Query called", in.FuncName)
		return c.QueryOld(in, result)
	}
	decodePayload, err := trans.(types.RpcTypeQuery).Input(in.Payload)
	if err != nil {
		log.Error("EventQuery", "err", err.Error())
		return err
	}

	resp, err := c.cli.Query(&types.Query{Execer: []byte(in.Execer), FuncName: in.FuncName, Payload: decodePayload})
	if err != nil {
		log.Error("EventQuery", "err", err.Error())
		return err
	}

	*result, err = trans.(types.RpcTypeQuery).Output(resp)
	if err != nil {
		log.Error("EventQuery", "err", err.Error())
		return err
	}

	return nil
}

func (c *Chain33) SetAutoMining(in types.MinerFlag, result *interface{}) error {
	resp, err := c.cli.WalletAutoMiner(&in)
	if err != nil {
		log.Error("SetAutoMiner", "err", err.Error())
		return err
	}
	var reply Reply
	reply.IsOk = resp.GetIsOk()
	reply.Msg = string(resp.GetMsg())
	*result = &reply
	return nil
}

func (c *Chain33) GetTicketCount(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.GetTicketCount()
	if err != nil {
		return err
	}
	*result = resp.GetData()
	return nil

}

func (c *Chain33) DumpPrivkey(in types.ReqStr, result *interface{}) error {
	reply, err := c.cli.DumpPrivkey(&in)
	if err != nil {
		return err
	}

	*result = reply
	return nil
}

func (c *Chain33) CloseTickets(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.CloseTickets()
	if err != nil {
		return err
	}
	var hashes ReplyHashes
	for _, has := range resp.Hashes {
		hashes.Hashes = append(hashes.Hashes, hex.EncodeToString(has))
	}
	*result = &hashes
	return nil
}

func (c *Chain33) Version(in *types.ReqNil, result *interface{}) error {
	*result = version.GetVersion()
	return nil
}

func (c *Chain33) GetTotalCoins(in *types.ReqGetTotalCoins, result *interface{}) error {
	resp, err := c.cli.GetTotalCoins(in)
	if err != nil {
		return err
	}
	*result = resp
	return nil
}

func (c *Chain33) IsSync(in *types.ReqNil, result *interface{}) error {
	reply, _ := c.cli.IsSync()
	ret := false
	if reply != nil {
		ret = reply.IsOk
	}
	*result = ret
	return nil
}

func DecodeTx(tx *types.Transaction) (*Transaction, error) {
	if tx == nil {
		return nil, types.ErrEmpty
	}
	var pl interface{}
	if "coins" == string(tx.Execer) {
		var action types.CoinsAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if "ticket" == string(tx.Execer) {
		var action types.TicketAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if "hashlock" == string(tx.Execer) {
		var action types.HashlockAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if "token" == string(tx.Execer) {
		var action types.TokenAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if "trade" == string(tx.Execer) {
		var action types.Trade
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if types.PrivacyX == string(tx.Execer) {
		var action types.PrivacyAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if "evm" == string(tx.Execer) {
		var action types.EVMContractAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if "user.write" == string(tx.Execer) {
		pl = decodeUserWrite(tx.GetPayload())
	} else {
		pl = map[string]interface{}{"rawlog": common.ToHex(tx.GetPayload())}
	}
	result := &Transaction{
		Execer:     string(tx.Execer),
		Payload:    pl,
		RawPayload: common.ToHex(tx.GetPayload()),
		Signature: &Signature{
			Ty:        tx.GetSignature().GetTy(),
			Pubkey:    common.ToHex(tx.GetSignature().GetPubkey()),
			Signature: common.ToHex(tx.GetSignature().GetSignature()),
		},
		Fee:    tx.Fee,
		Expire: tx.Expire,
		Nonce:  tx.Nonce,
		To:     tx.To,
	}
	return result, nil
}

func decodeUserWrite(payload []byte) *userWrite {
	var article userWrite
	if payload[0] == '#' {
		data := bytes.SplitN(payload[1:], []byte("#"), 2)
		if len(data) == 2 {
			article.Topic = string(data[0])
			article.Content = string(data[1])
			return &article
		}
	}
	article.Topic = ""
	article.Content = string(payload)
	return &article
}

func DecodeLog(rlog *ReceiptData) (*ReceiptDataResult, error) {
	var rTy string
	switch rlog.Ty {
	case 0:
		rTy = "ExecErr"
	case 1:
		rTy = "ExecPack"
	case 2:
		rTy = "ExecOk"
	default:
		rTy = "Unkown"
	}
	rd := &ReceiptDataResult{Ty: rlog.Ty, TyName: rTy}

	for _, l := range rlog.Logs {
		var lTy string
		var logIns interface{}

		lLog, err := hex.DecodeString(l.Log[2:])
		if err != nil {
			return nil, err
		}

		switch l.Ty {
		case types.TyLogErr:
			lTy = "LogErr"
			logIns = string(lLog)
		case types.TyLogFee:
			lTy = "LogFee"
			var logTmp types.ReceiptAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTransfer:
			lTy = "LogTransfer"
			var logTmp types.ReceiptAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogGenesis:
			lTy = "LogGenesis"
			logIns = nil
		case types.TyLogDeposit:
			lTy = "LogDeposit"
			var logTmp types.ReceiptAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogExecTransfer:
			lTy = "LogExecTransfer"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogExecWithdraw:
			lTy = "LogExecWithdraw"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogExecDeposit:
			lTy = "LogExecDeposit"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogExecFrozen:
			lTy = "LogExecFrozen"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogExecActive:
			lTy = "LogExecActive"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogGenesisTransfer:
			lTy = "LogGenesisTransfer"
			var logTmp types.ReceiptAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogGenesisDeposit:
			lTy = "LogGenesisDeposit"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogNewTicket:
			lTy = "LogNewTicket"
			var logTmp types.ReceiptTicket
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogCloseTicket:
			lTy = "LogCloseTicket"
			var logTmp types.ReceiptTicket
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogMinerTicket:
			lTy = "LogMinerTicket"
			var logTmp types.ReceiptTicket
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTicketBind:
			lTy = "LogTicketBind"
			var logTmp types.ReceiptTicketBind
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogPreCreateToken:
			lTy = "LogPreCreateToken"
			var logTmp types.ReceiptToken
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogFinishCreateToken:
			lTy = "LogFinishCreateToken"
			var logTmp types.ReceiptToken
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogRevokeCreateToken:
			lTy = "LogRevokeCreateToken"
			var logTmp types.ReceiptToken
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTradeSellLimit:
			lTy = "LogTradeSell"
			var logTmp types.ReceiptTradeSell
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTradeBuyMarket:
			lTy = "LogTradeBuy"
			var logTmp types.ReceiptTradeBuyMarket
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTradeSellRevoke:
			lTy = "LogTradeRevoke"
			var logTmp types.ReceiptTradeRevoke
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTradeBuyLimit:
			lTy = "LogTradeBuyLimit"
			var logTmp types.ReceiptTradeBuyLimit
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTradeSellMarket:
			lTy = "LogTradeSellMarket"
			var logTmp types.ReceiptSellMarket
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTradeBuyRevoke:
			lTy = "LogTradeBuyRevoke"
			var logTmp types.ReceiptTradeBuyRevoke
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenTransfer:
			lTy = "LogTokenTransfer"
			var logTmp types.ReceiptAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenDeposit:
			lTy = "LogTokenDeposit"
			var logTmp types.ReceiptAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenExecTransfer:
			lTy = "LogTokenExecTransfer"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenExecWithdraw:
			lTy = "LogTokenExecWithdraw"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenExecDeposit:
			lTy = "LogTokenExecDeposit"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenExecFrozen:
			lTy = "LogTokenExecFrozen"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenExecActive:
			lTy = "LogTokenExecActive"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenGenesisTransfer:
			lTy = "LogTokenGenesisTransfer"
			var logTmp types.ReceiptAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogTokenGenesisDeposit:
			lTy = "LogTokenGenesisDeposit"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogPrivacyFee:
			lTy = "LogPrivacyFee"
			var logTmp types.ReceiptExecAccountTransfer
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogPrivacyOutput:
			lTy = "LogPrivacyOutput"
			var logTmp types.ReceiptPrivacyOutput
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogCallContract:
			lTy = "LogCallContract"
			var logTmp types.ReceiptEVMContract
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogContractData:
			lTy = "LogContractData"
			var logTmp types.EVMContractData
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogContractState:
			lTy = "LogContractState"
			var logTmp types.EVMContractState
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		case types.TyLogModifyConfig:
			lTy = "LogModifyConfig"
			var logTmp types.ReceiptConfig
			err = types.Decode(lLog, &logTmp)
			if err != nil {
				return nil, err
			}
			logIns = logTmp
		default:
			log.Error("Fail to DecodeLog", "type", l.Ty)
			lTy = "unkownType"
			logIns = nil
		}
		rd.Logs = append(rd.Logs, &ReceiptLogResult{Ty: l.Ty, TyName: lTy, Log: logIns, RawLog: l.Log})
	}
	return rd, nil
}

func (c *Chain33) IsNtpClockSync(in *types.ReqNil, result *interface{}) error {
	reply, _ := c.cli.IsNtpClockSync()
	ret := false
	if reply != nil {
		ret = reply.IsOk
	}
	*result = ret
	return nil
}

func (c *Chain33) QueryTotalFee(in *types.LocalDBGet, result *interface{}) error {
	reply, err := c.cli.LocalGet(in)
	if err != nil {
		return err
	}

	var fee types.TotalFee
	err = types.Decode(reply.Values[0], &fee)
	if err != nil {
		return err
	}
	*result = fee
	return nil
}

func (c *Chain33) CreateRawTokenPreCreateTx(in *TokenPreCreateTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenPreCreateTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTokenFinishTx(in *TokenFinishTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenFinishTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTokenRevokeTx(in *TokenRevokeTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeSellTx(in *TradeSellTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeSellTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeBuyTx(in *TradeBuyTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeBuyTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeRevokeTx(in *TradeRevokeTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeBuyLimitTx(in *TradeBuyLimitTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeBuyLimitTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeSellMarketTx(in *TradeSellMarketTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeSellMarketTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeRevokeBuyTx(in *TradeRevokeBuyTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeRevokeBuyTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) SignRawTx(in *types.ReqSignRawTx, result *interface{}) error {
	resp, err := c.cli.SignRawTx(in)
	if err != nil {
		return err
	}
	*result = resp.TxHex
	return nil
}

func (c *Chain33) GetNetInfo(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.GetNetInfo()
	if err != nil {
		return err
	}

	*result = &NodeNetinfo{resp.GetExternaladdr(), resp.GetLocaladdr(), resp.GetService(), resp.GetOutbounds(), resp.GetInbounds()}
	return nil
}

func (c *Chain33) GetFatalFailure(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.GetFatalFailure()
	if err != nil {
		return err
	}
	*result = resp.GetData()
	return nil

}

func (c *Chain33) QueryTicketStat(in *types.LocalDBGet, result *interface{}) error {
	reply, err := c.cli.LocalGet(in)
	if err != nil {
		return err
	}

	var ticketStat types.TicketStatistic
	err = types.Decode(reply.Values[0], &ticketStat)
	if err != nil {
		return err
	}
	*result = ticketStat
	return nil
}

func (c *Chain33) QueryTicketInfo(in *types.LocalDBGet, result *interface{}) error {
	reply, err := c.cli.LocalGet(in)
	if err != nil {
		return err
	}

	var ticketInfo types.TicketMinerInfo
	err = types.Decode(reply.Values[0], &ticketInfo)
	if err != nil {
		return err
	}
	*result = ticketInfo
	return nil
}

func (c *Chain33) QueryTicketInfoList(in *types.LocalDBList, result *interface{}) error {
	reply, err := c.cli.LocalList(in)
	if err != nil {
		return err
	}

	var ticketInfo types.TicketMinerInfo
	var ticketList []types.TicketMinerInfo
	for _, v := range reply.Values {
		err = types.Decode(v, &ticketInfo)
		if err != nil {
			return err
		}
		ticketList = append(ticketList, ticketInfo)
	}
	*result = ticketList
	return nil
}

/////////////////privacy///////////////
func (c *Chain33) ShowPrivacyBalance(in types.ReqPrivBal4AddrToken, result *interface{}) error {
	account, err := c.cli.ShowPrivacyBalance(&in)
	if err != nil {
		log.Info("ShowPrivacyBalance", "return err info", err)
		return err
	}
	*result = account
	return nil
}

func (c *Chain33) ShowPrivacyAccount(in types.ReqPrivBal4AddrToken, result *interface{}) error {
	account, err := c.cli.ShowPrivacyAccount(&in)
	if err != nil {
		log.Info("ShowPrivacyAccount", "return err info", err)
		return err
	}
	*result = account
	return nil
}

func (c *Chain33) ShowPrivacyAccountSpend(in types.ReqPrivBal4AddrToken, result *interface{}) error {
	account, err := c.cli.ShowPrivacyAccountSpend(&in)
	if err != nil {
		log.Info("ShowPrivacyAccountSpend", "return err info", err)
		return err
	}
	*result = account
	return nil
}

func (c *Chain33) ShowPrivacykey(in types.ReqStr, result *interface{}) error {
	reply, err := c.cli.ShowPrivacyKey(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) MakeTxPublic2privacy(in types.ReqPub2Pri, result *interface{}) error {
	reply, err := c.cli.Publick2Privacy(&in)
	if err != nil {
		return err
	}

	*result = ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Chain33) MakeTxPrivacy2privacy(in types.ReqPri2Pri, result *interface{}) error {
	reply, err := c.cli.Privacy2Privacy(&in)
	if err != nil {
		return err
	}

	*result = ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Chain33) MakeTxPrivacy2public(in types.ReqPri2Pub, result *interface{}) error {
	reply, err := c.cli.Privacy2Public(&in)
	if err != nil {
		return err
	}
	*result = ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Chain33) CreateUTXOs(in types.ReqCreateUTXOs, result *interface{}) error {

	reply, err := c.cli.CreateUTXOs(&in)
	if err != nil {
		return err
	}
	*result = ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}
