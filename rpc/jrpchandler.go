package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/types"
	retrievetype "gitlab.33.cn/chain33/chain33/types/executor/retrieve"
	tokentype "gitlab.33.cn/chain33/chain33/types/executor/token"
	tradetype "gitlab.33.cn/chain33/chain33/types/executor/trade"
)

func (c *Chain33) CreateRawTransaction(in *types.CreateTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTransaction(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil

}

func (c *Chain33) CreateNoBalanceTransaction(in *types.NoBalanceTx, result *string) error {
	tx, err := c.cli.CreateNoBalanceTransaction(in)
	if err != nil {
		return err
	}
	grouptx := hex.EncodeToString(types.Encode(tx))
	*result = grouptx
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

//used only in parachain
func forwardTranToMainNet(in RawParm, result *interface{}) error {
	if rpcCfg.GetMainnetJrpcAddr() == "" {
		return types.ErrInvalidMainnetRpcAddr
	}
	rpc, err := NewJSONClient(rpcCfg.GetMainnetJrpcAddr())

	if err != nil {
		return err
	}

	err = rpc.Call("Chain33.SendTransaction", in, result)
	return err
}

func (c *Chain33) SendTransaction(in RawParm, result *interface{}) error {
	if types.IsPara() {
		return forwardTranToMainNet(in, result)
	}
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
		header.Difficulty = reply.GetDifficulty()
		/* 空值，斩不显示
		Signature: &Signature{
			Ty:        reply.GetSignature().GetTy(),
			Pubkey:    common.ToHex(reply.GetSignature().GetPubkey()),
			Signature: common.ToHex(reply.GetSignature().GetSignature()),
		}
		*/
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
			parm.Hashes = append(parm.Hashes, nil)
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
				txdetails.Txs = append(txdetails.Txs, nil)
				continue
			}
			txProofs := tx.GetProofs()
			for _, proof := range txProofs {
				proofs = append(proofs, common.ToHex(proof))
			}
			tran, err := DecodeTx(tx.GetTx())
			if err != nil {
				log.Info("GetTxByHashes", "Failed to DecodeTx due to", err)
				txdetails.Txs = append(txdetails.Txs, nil)
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
			tran, err := DecodeTx(tx)
			if err != nil {
				continue
			}
			tran.Amount = amount
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
	reply, err := c.cli.WalletTransactionList(&parm)
	if err != nil {
		return err
	}
	{
		var txdetails WalletTxDetails
		c.convertWalletTxDetailToJson(reply, &txdetails)
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
	if types.IsPara() {
		createTx := &types.CreateTx{
			To:          in.GetTo(),
			Amount:      in.GetAmount(),
			Fee:         1e5,
			Note:        in.GetNote(),
			IsWithdraw:  false,
			IsToken:     true,
			TokenSymbol: in.GetTokenSymbol(),
			ExecName:    types.ExecName(types.TokenX),
		}
		tx, err := c.cli.CreateRawTransaction(createTx)
		if err != nil {
			log.Debug("ParaChain CreateRawTransaction", "Error", err.Error())
			return err
		}
		//不需要自己去导出私钥，signRawTx 里面只需带入公钥地址，也回优先去查出相应的私钥，前提是私钥已经导入
		reqSignRawTx := &types.ReqSignRawTx{
			Addr:    in.From,
			Privkey: "",
			TxHex:   hex.EncodeToString(tx),
			Expire:  "300s",
			Index:   0,
			Token:   "",
		}
		replySignRawTx, err := c.cli.SignRawTx(reqSignRawTx)
		if err != nil {
			log.Debug("ParaChain SignRawTx", "Error", err.Error())
			return err
		}
		rawParm := RawParm{
			Token: "",
			Data:  replySignRawTx.GetTxHex(),
		}
		var txHash interface{}
		err = forwardTranToMainNet(rawParm, &txHash)
		if err != nil {
			log.Debug("ParaChain forwardTranToMainNet", "Error", err.Error())
			return err
		}
		*result = &ReplyHash{Hash: txHash.(string)}
		return nil
	}
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
				Difficulty: item.GetDifficulty(),
				/* 空值，斩不显示
				Signature: &Signature{
					Ty:        item.GetSignature().GetTy(),
					Pubkey:    common.ToHex(item.GetSignature().GetPubkey()),
					Signature: common.ToHex(item.GetSignature().GetSignature()),
				},
				*/
				Version: item.GetVersion()})
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
	header.Difficulty = reply.GetHead().GetDifficulty()
	/* 空值，斩不显示
	header.Signature = &Signature{
		Ty:        reply.GetHead().GetSignature().GetTy(),
		Pubkey:    common.ToHex(reply.GetHead().GetSignature().GetPubkey()),
		Signature: common.ToHex(reply.GetHead().GetSignature().GetSignature()),
	}
	*/
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

func (c *Chain33) GetAllExecBalance(in types.ReqAddr, result *interface{}) error {
	balance, err := c.cli.GetAllExecBalance(&in)
	if err != nil {
		return err
	}

	allBalance := &AllExecBalance{Addr: in.Addr}
	for _, execAcc := range balance.ExecAccount {
		res := &ExecAccount{Execer: execAcc.Execer}
		acc := &Account{
			Balance:  execAcc.Account.GetBalance(),
			Currency: execAcc.Account.GetCurrency(),
			Frozen:   execAcc.Account.GetFrozen(),
		}
		res.Account = acc
		allBalance.ExecAccount = append(allBalance.ExecAccount, res)
	}
	*result = allBalance
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
	trans := types.LoadQueryType(in.FuncName)
	if trans == nil {
		// 不是所有的合约都需要做类型转化， 没有修改的合约走老的接口
		// 另外给部分合约的代码修改的时间
		//log.Info("EventQuery", "Old Query called", in.FuncName)
		// return c.QueryOld(in, result)

		// now old code all move to type/executor, test and then remove old code
		log.Error("Query", "funcname", in.FuncName, "err", types.ErrNotSupport)
		return types.ErrNotSupport
	}
	decodePayload, err := trans.Input(in.Payload)
	if err != nil {
		log.Error("EventQuery", "err", err.Error())
		return err
	}

	resp, err := c.cli.Query(&types.Query{Execer: []byte(types.ExecName(in.Execer)), FuncName: in.FuncName, Payload: decodePayload})
	if err != nil {
		log.Error("EventQuery", "err", err.Error())
		return err
	}

	*result, err = trans.(types.RpcQueryType).Output(resp)
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
	unkownPl := make(map[string]interface{})
	if types.ExecName(types.CoinsX) == string(tx.Execer) {
		var action types.CoinsAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			unkownPl["unkownpayload"] = string(tx.GetPayload())
			pl = unkownPl
			fmt.Println(pl)
		} else {
			pl = &action
		}
	} else if types.ExecName(types.TicketX) == string(tx.Execer) {
		var action types.TicketAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			unkownPl["unkownpayload"] = string(tx.GetPayload())
			pl = unkownPl
		} else {
			pl = &action
		}
	} else if types.ExecName(types.HashlockX) == string(tx.Execer) {
		var action types.HashlockAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			unkownPl["unkownpayload"] = string(tx.GetPayload())
			pl = unkownPl
		} else {
			pl = &action
		}
	} else if types.ExecName(types.TokenX) == string(tx.Execer) {
		var action types.TokenAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			unkownPl["unkownpayload"] = string(tx.GetPayload())
			pl = unkownPl
		} else {
			pl = &action
		}
	} else if types.ExecName(types.TradeX) == string(tx.Execer) {
		var action types.Trade
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			unkownPl["unkownpayload"] = string(tx.GetPayload())
			pl = unkownPl
		} else {
			pl = &action
		}
		pl = &action
	} else if types.ExecName(types.PrivacyX) == string(tx.Execer) {
		var action types.PrivacyAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			return nil, err
		}
		pl = &action
	} else if types.ExecName(types.EvmX) == string(tx.Execer) {
		var action types.EVMContractAction
		err := types.Decode(tx.GetPayload(), &action)
		if err != nil {
			unkownPl["unkownpayload"] = string(tx.GetPayload())
			pl = unkownPl
		} else {
			pl = &action
		}
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
		Fee:        tx.Fee,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.To,
		From:       tx.From(),
		GroupCount: tx.GroupCount,
		Header:     common.ToHex(tx.Header),
		Next:       common.ToHex(tx.Next),
	}
	return result, nil
}

func decodeUserWrite(payload []byte) *userWrite {
	var article userWrite
	if len(payload) != 0 {
		if payload[0] == '#' {
			data := bytes.SplitN(payload[1:], []byte("#"), 2)
			if len(data) == 2 {
				article.Topic = string(data[0])
				article.Content = string(data[1])
				return &article
			}
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

		logType := types.LoadLog(int64(l.Ty))
		if logType == nil {
			log.Error("Fail to DecodeLog", "type", l.Ty)
			lTy = "unkownType"
			logIns = nil
		} else {
			logIns, err = logType.Decode(lLog)
			lTy = logType.Name()
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

func (c *Chain33) CreateRawTokenPreCreateTx(in *tokentype.TokenPreCreateTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenPreCreateTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTokenFinishTx(in *tokentype.TokenFinishTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenFinishTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTokenRevokeTx(in *tokentype.TokenRevokeTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTokenRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeSellTx(in *tradetype.TradeSellTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeSellTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeBuyTx(in *tradetype.TradeBuyTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeBuyTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeRevokeTx(in *tradetype.TradeRevokeTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeBuyLimitTx(in *tradetype.TradeBuyLimitTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeBuyLimitTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeSellMarketTx(in *tradetype.TradeSellMarketTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeSellMarketTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawTradeRevokeBuyTx(in *tradetype.TradeRevokeBuyTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTradeRevokeBuyTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRetrieveBackupTx(in *retrievetype.RetrieveBackupTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRetrieveBackupTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRetrievePrepareTx(in *retrievetype.RetrievePrepareTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRetrievePrepareTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRetrievePerformTx(in *retrievetype.RetrievePerformTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRetrievePerformTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRetrieveCancelTx(in *retrievetype.RetrieveCancelTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRetrieveCancelTx(in)
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

func (c *Chain33) CreateBindMiner(in *types.ReqBindMiner, result *interface{}) error {
	if in.Amount%(10000*types.Coin) != 0 || in.Amount < 0 {
		return types.ErrAmount
	}
	err := address.CheckAddress(in.BindAddr)
	if err != nil {
		return err
	}
	err = address.CheckAddress(in.OriginAddr)
	if err != nil {
		return err
	}

	if in.CheckBalance {
		getBalance := &types.ReqBalance{Addresses: []string{in.OriginAddr}, Execer: "coins"}
		balances, err := c.cli.GetBalance(getBalance)
		if err != nil {
			return err
		}
		if len(balances) == 0 {
			return types.ErrInputPara
		}
		if balances[0].Balance < in.Amount+2*types.Coin {
			return types.ErrNoBalance
		}
	}

	reply, err := c.cli.BindMiner(in)
	if err != nil {
		return err
	}

	*result = reply
	return nil
}

func (c *Chain33) DecodeRawTransaction(in *types.ReqDecodeRawTransaction, result *interface{}) error {
	reply, err := c.cli.DecodeRawTransaction(in)
	if err != nil {
		return err
	}
	res, err := DecodeTx(reply)
	if err != nil {
		return err
	}
	*result = res
	return nil
}

func (c *Chain33) GetTimeStatus(in *types.ReqNil, result *interface{}) error {
	reply, err := c.cli.GetTimeStatus()
	if err != nil {
		return err
	}

	timeStatus := &TimeStatus{
		NtpTime:   reply.NtpTime,
		LocalTime: reply.LocalTime,
		Diff:      reply.Diff,
	}

	*result = timeStatus

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

func (c *Chain33) CreateTrasaction(in types.ReqCreateTransaction, result *interface{}) error {
	reply, err := c.cli.CreateTrasaction(&in)
	if err != nil {
		return err
	}
	txHex := types.Encode(reply)
	*result = hex.EncodeToString(txHex)
	return nil
}

func (c *Chain33) ShowPrivacyAccountInfo(in types.ReqPPrivacyAccount, result *interface{}) error {
	reply, err := c.cli.ShowPrivacyAccountInfo(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) CloseQueue(in *types.ReqNil, result *interface{}) error {
	go func() {
		time.Sleep(time.Millisecond * 100)
		c.cli.CloseQueue()
	}()

	*result = &types.Reply{IsOk: true}
	return nil
}

func (c *Chain33) GetLastBlockSequence(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.GetLastBlockSequence()
	if err != nil {
		return err
	}
	*result = resp.GetData()
	return nil
}

// 获取指定区间的block加载序列号信息。输入信息只使用：start，end
func (c *Chain33) GetBlockSequences(in BlockParam, result *interface{}) error {
	resp, err := c.cli.GetBlockSequences(&types.ReqBlocks{Start: in.Start, End: in.End, IsDetail: in.Isdetail, Pid: []string{""}})
	if err != nil {
		return err
	}
	var BlkSeqs ReplyBlkSeqs
	items := resp.GetItems()
	for _, item := range items {
		BlkSeqs.BlkSeqInfos = append(BlkSeqs.BlkSeqInfos, &ReplyBlkSeq{Hash: common.ToHex(item.GetHash()),
			Type: item.GetType()})
	}
	*result = &BlkSeqs
	return nil
}

// 通过block hash 获取对应的block信息
func (c *Chain33) GetBlockByHashes(in ReqHashes, result *interface{}) error {
	log.Warn("GetBlockByHashes", "hashes", in)
	var parm types.ReqHashes
	parm.Hashes = make([][]byte, 0)
	for _, v := range in.Hashes {
		hb, err := common.FromHex(v)
		if err != nil {
			parm.Hashes = append(parm.Hashes, nil)
			continue
		}
		parm.Hashes = append(parm.Hashes, hb)

	}
	reply, err := c.cli.GetBlockByHashes(&parm)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) CreateTransaction(in *TransactionCreate, result *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	exec := types.LoadExecutor(in.Execer)
	if exec == nil {
		return types.ErrExecNameNotAllow
	}
	tx, err := exec.CreateTx(in.ActionName, in.Payload)
	if err != nil {
		log.Error("CreateTransaction", "err", err.Error())
		return err
	}
	*result = tx
	return nil
}

func (c *Chain33) CreateRawRelayOrderTx(in *RelayOrderTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayOrderTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRelayAcceptTx(in *RelayAcceptTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayAcceptTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Chain33) CreateRawRelayRevokeTx(in *RelayRevokeTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Chain33) CreateRawRelayConfirmTx(in *RelayConfirmTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayConfirmTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Chain33) CreateRawRelayVerifyBTCTx(in *RelayVerifyBTCTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayVerifyBTCTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRelaySaveBTCHeadTx(in *RelaySaveBTCHeadTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelaySaveBTCHeadTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)

	return nil
}

func (c *Chain33) convertWalletTxDetailToJson(in *types.WalletTxDetails, out *WalletTxDetails) error {
	if in == nil || out == nil {
		return types.ErrInvalidParams
	}
	for _, tx := range in.TxDetails {
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
		out.TxDetails = append(out.TxDetails, &WalletTxDetail{
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
	return nil
}

// PrivacyTxList get all privacy transaction list by param
func (c *Chain33) PrivacyTxList(in *types.ReqPrivacyTransactionList, result *interface{}) error {
	reply, err := c.cli.PrivacyTransactionList(in)
	if err != nil {
		return err
	}
	{
		var txdetails WalletTxDetails
		c.convertWalletTxDetailToJson(reply, &txdetails)
		*result = &txdetails
	}
	return nil
}

func (c *Chain33) RescanUtxos(in types.ReqRescanUtxos, result *interface{}) error {
	reply, err := c.cli.RescanUtxos(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}
