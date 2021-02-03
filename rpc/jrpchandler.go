// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	wcom "github.com/33cn/chain33/wallet/common"
)

// CreateRawTransaction create rawtransaction by jrpc
func (c *Chain33) CreateRawTransaction(in *rpctypes.CreateTx, result *interface{}) error {
	if in == nil {
		log.Error("CreateRawTransaction", "Error", types.ErrInvalidParam)
		return types.ErrInvalidParam
	}
	inpb := &types.CreateTx{
		To:          in.To,
		Amount:      in.Amount,
		Fee:         in.Fee,
		Note:        []byte(in.Note),
		IsWithdraw:  in.IsWithdraw,
		IsToken:     in.IsToken,
		TokenSymbol: in.TokenSymbol,
		ExecName:    in.ExecName,
		Execer:      in.Execer,
	}
	reply, err := c.cli.CreateRawTransaction(inpb)
	if err != nil {
		return err
	}
	*result = common.ToHex(reply)
	return nil
}

// ReWriteRawTx re-write raw tx by jrpc
func (c *Chain33) ReWriteRawTx(in *rpctypes.ReWriteRawTx, result *interface{}) error {
	inpb := &types.ReWriteRawTx{
		Tx:     in.Tx,
		To:     in.To,
		Fee:    in.Fee,
		Expire: in.Expire,
		Index:  in.Index,
	}

	reply, err := c.cli.ReWriteRawTx(inpb)
	if err != nil {
		return err
	}
	*result = common.ToHex(reply)
	return nil
}

// CreateRawTxGroup create rawtransaction with group
func (c *Chain33) CreateRawTxGroup(in *types.CreateTransactionGroup, result *interface{}) error {
	reply, err := c.cli.CreateRawTxGroup(in)
	if err != nil {
		return err
	}

	*result = common.ToHex(reply)
	return nil
}

// CreateNoBlanaceTxs create multiple transaction with no balance
func (c *Chain33) CreateNoBlanaceTxs(in *types.NoBalanceTxs, result *string) error {
	tx, err := c.cli.CreateNoBalanceTxs(in)
	if err != nil {
		return err
	}
	grouptx := common.ToHex(types.Encode(tx))
	*result = grouptx
	return nil
}

// CreateNoBalanceTransaction create transaction with no balance
func (c *Chain33) CreateNoBalanceTransaction(in *types.NoBalanceTx, result *string) error {
	params := &types.NoBalanceTxs{
		TxHexs:  []string{in.GetTxHex()},
		PayAddr: in.GetPayAddr(),
		Privkey: in.GetPrivkey(),
		Expire:  in.GetExpire(),
	}
	tx, err := c.cli.CreateNoBalanceTxs(params)
	if err != nil {
		return err
	}
	grouptx := common.ToHex(types.Encode(tx))
	*result = grouptx
	return nil
}

// SendTransaction send transaction
func (c *Chain33) SendTransaction(in rpctypes.RawParm, result *interface{}) error {
	var parm types.Transaction
	data, err := common.FromHex(in.Data)
	if err != nil {
		return err
	}
	err = types.Decode(data, &parm)
	if err != nil {
		return err
	}
	log.Debug("SendTransaction", "parm", parm)

	var reply *types.Reply
	//para chain, forward to main chain
	cfg := c.cli.GetConfig()
	if cfg.IsPara() {
		reply, err = c.mainGrpcCli.SendTransaction(context.Background(), &parm)
	} else {
		reply, err = c.cli.SendTx(&parm)
	}

	if err == nil {
		*result = common.ToHex(reply.GetMsg())
	}
	return err
}

// SendTransactionSync send transaction and wait reply
func (c *Chain33) SendTransactionSync(in rpctypes.RawParm, result *interface{}) error {
	err := c.SendTransaction(in, result)
	if err != nil {
		return err
	}
	hash := (*result).(string)
	param := rpctypes.QueryParm{Hash: hash}
	var res interface{}
	for i := 0; i < 100; i++ {
		err = c.QueryTransaction(param, &res)
		if err == types.ErrInvalidParam || err == types.ErrTypeAsset {
			return err
		}
		if _, ok := (res).(*rpctypes.TransactionDetail); ok {
			return nil
		}
		time.Sleep(time.Second / 3)
	}

	return types.ErrTimeout
}

// GetHexTxByHash get hex transaction by hash
func (c *Chain33) GetHexTxByHash(in rpctypes.QueryParm, result *interface{}) error {
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
	*result = common.ToHex(types.Encode(reply.GetTx()))
	return err
}

// QueryTransaction query transaction
func (c *Chain33) QueryTransaction(in rpctypes.QueryParm, result *interface{}) error {
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

	transDetail, err := fmtTxDetail(reply, false)
	if err != nil {
		return err
	}

	*result = transDetail

	return nil
}

// GetBlocks get block information
func (c *Chain33) GetBlocks(in rpctypes.BlockParam, result *interface{}) error {
	reply, err := c.cli.GetBlocks(&types.ReqBlocks{Start: in.Start, End: in.End, IsDetail: in.Isdetail, Pid: []string{""}})
	if err != nil {
		return err
	}
	{
		var blockDetails rpctypes.BlockDetails
		items := reply.GetItems()
		if err := convertBlockDetails(items, &blockDetails, in.Isdetail); err != nil {
			return err
		}
		*result = &blockDetails
	}

	return nil

}

// GetLastHeader get last header
func (c *Chain33) GetLastHeader(in *types.ReqNil, result *interface{}) error {
	reply, err := c.cli.GetLastHeader()
	if err != nil {
		return err
	}

	{
		var header rpctypes.Header
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

// GetTxByAddr get transaction by address
// GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfo, error)
func (c *Chain33) GetTxByAddr(in types.ReqAddr, result *interface{}) error {
	reply, err := c.cli.GetTransactionByAddr(&in)
	if err != nil {
		return err
	}
	{
		var txinfos rpctypes.ReplyTxInfos
		infos := reply.GetTxInfos()
		for _, info := range infos {
			txinfos.TxInfos = append(txinfos.TxInfos, &rpctypes.ReplyTxInfo{Hash: common.ToHex(info.GetHash()),
				Height: info.GetHeight(), Index: info.GetIndex(), Assets: fmtAsssets(info.Assets)})
		}
		*result = &txinfos
	}

	return nil
}

// GetTxByHashes get transaction by hashes
/*
GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error)
GetMempool() (*types.ReplyTxList, error)
GetAccounts() (*types.WalletAccounts, error)
*/
func (c *Chain33) GetTxByHashes(in rpctypes.ReqHashes, result *interface{}) error {
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
	var txdetails rpctypes.TransactionDetails
	if 0 != len(txs) {
		for _, tx := range txs {
			txDetail, err := fmtTxDetail(tx, in.DisableDetail)
			if err != nil {
				return err
			}
			txdetails.Txs = append(txdetails.Txs, txDetail)
		}
	}
	*result = &txdetails
	return nil
}

func fmtTxDetail(tx *types.TransactionDetail, disableDetail bool) (*rpctypes.TransactionDetail, error) {
	//增加判断，上游接口可能返回空指针
	if tx == nil || tx.GetTx() == nil {
		//参数中hash和返回的detail一一对应，顺序一致
		return nil, nil
	}

	var recp rpctypes.ReceiptData
	recp.Ty = tx.GetReceipt().GetTy()
	logs := tx.GetReceipt().GetLogs()
	if disableDetail {
		logs = nil
	}
	for _, lg := range logs {
		recp.Logs = append(recp.Logs,
			&rpctypes.ReceiptLog{Ty: lg.Ty, Log: common.ToHex(lg.GetLog())})
	}

	var recpResult *rpctypes.ReceiptDataResult
	recpResult, err := rpctypes.DecodeLog(tx.Tx.Execer, &recp)
	if err != nil {
		log.Error("GetTxByHashes", "Failed to DecodeLog for type", err)
		return nil, err
	}

	var proofs []string
	txProofs := tx.GetProofs()
	for _, proof := range txProofs {
		proofs = append(proofs, common.ToHex(proof))
	}

	tran, err := rpctypes.DecodeTx(tx.GetTx())
	if err != nil {
		log.Info("GetTxByHashes", "Failed to DecodeTx due to", err)
		return nil, err
	}
	//对Amount做格式化
	if tx.Amount != 0 {
		tran.Amount = tx.Amount
		tran.AmountFmt = strconv.FormatFloat(float64(tran.Amount)/float64(types.Coin), 'f', 4, 64)
	}
	// swap from with to
	if tx.GetTx().IsWithdraw() {
		tx.Fromaddr, tx.Tx.To = tx.Tx.To, tx.Fromaddr
		tran.To = tx.Tx.GetRealToAddr()
	}
	//交易fullhash
	var fullhash string
	if len(tx.GetFullHash()) != 0 {
		fullhash = common.ToHex(tx.GetFullHash())
	}
	return &rpctypes.TransactionDetail{
		Tx:         tran,
		Height:     tx.GetHeight(),
		Index:      tx.GetIndex(),
		Blocktime:  tx.GetBlocktime(),
		Receipt:    recpResult,
		Proofs:     proofs,
		Amount:     tx.GetAmount(),
		Fromaddr:   tx.GetFromaddr(),
		ActionName: tx.GetActionName(),
		Assets:     fmtAsssets(tx.GetAssets()),
		TxProofs:   fmtTxProofs(tx.GetTxProofs()),
		FullHash:   fullhash,
	}, nil
}

func fmtAsssets(assets []*types.Asset) []*rpctypes.Asset {
	var result []*rpctypes.Asset
	for _, a := range assets {
		asset := &rpctypes.Asset{
			Exec:   a.Exec,
			Symbol: a.Symbol,
			Amount: a.Amount,
		}
		result = append(result, asset)
	}
	return result
}

// GetMempool get mempool information
func (c *Chain33) GetMempool(in *types.ReqGetMempool, result *interface{}) error {
	reply, err := c.cli.GetMempool(in)
	if err != nil {
		return err
	}
	{
		var txlist rpctypes.ReplyTxList
		txs := reply.GetTxs()
		for _, tx := range txs {
			amount, err := tx.Amount()
			if err != nil {
				amount = 0
			}
			tran, err := rpctypes.DecodeTx(tx)
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

// GetAccountsV2 get accounts for version 2
func (c *Chain33) GetAccountsV2(in *types.ReqNil, result *interface{}) error {
	req := &types.ReqAccountList{WithoutBalance: false}
	return c.GetAccounts(req, result)
}

// GetAccounts get accounts
func (c *Chain33) GetAccounts(in *types.ReqAccountList, result *interface{}) error {
	req := types.ReqAccountList{WithoutBalance: in.WithoutBalance}
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletGetAccountList", &req)
	if err != nil {
		return err
	}

	accountsList := reply.(*types.WalletAccounts)
	var accounts rpctypes.WalletAccounts
	for _, wallet := range accountsList.Wallets {
		accounts.Wallets = append(accounts.Wallets, &rpctypes.WalletAccount{Label: wallet.GetLabel(),
			Acc: &rpctypes.Account{Currency: wallet.GetAcc().GetCurrency(), Balance: wallet.GetAcc().GetBalance(),
				Frozen: wallet.GetAcc().GetFrozen(), Addr: wallet.GetAcc().GetAddr()}})
	}

	*result = &accounts
	return nil
}

// NewAccount new a account
func (c *Chain33) NewAccount(in types.ReqNewAccount, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "NewAccount", &in)
	if err != nil {
		return err
	}

	*result = reply
	return nil
}

// WalletTxList transaction list of wallet
func (c *Chain33) WalletTxList(in rpctypes.ReqWalletTransactionList, result *interface{}) error {
	var parm types.ReqWalletTransactionList
	parm.FromTx = []byte(in.FromTx)
	parm.Count = in.Count
	parm.Direction = in.Direction
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletTransactionList", &parm)
	if err != nil {
		return err
	}
	{
		var txdetails rpctypes.WalletTxDetails
		err := rpctypes.ConvertWalletTxDetailToJSON(reply.(*types.WalletTxDetails), &txdetails)
		if err != nil {
			return err
		}
		*result = &txdetails
	}
	return nil
}

// ImportPrivkey import privkey of wallet
func (c *Chain33) ImportPrivkey(in types.ReqWalletImportPrivkey, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletImportPrivkey", &in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// SendToAddress send to address of coins
func (c *Chain33) SendToAddress(in types.ReqWalletSendToAddress, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletSendToAddress", &in)
	if err != nil {
		log.Debug("SendToAddress", "Error", err.Error())
		return err
	}

	log.Debug("sendtoaddr", "msg", reply.String())
	*result = &rpctypes.ReplyHash{Hash: common.ToHex(reply.(*types.ReplyHash).GetHash())}
	log.Debug("SendToAddress", "resulrt", *result)
	return nil
}

// SetTxFee set tx fee
func (c *Chain33) SetTxFee(in types.ReqWalletSetFee, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletSetFee", &in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.(*types.Reply).GetIsOk()
	resp.Msg = string(reply.(*types.Reply).GetMsg())
	*result = &resp
	return nil
}

// SetLabl set lable
func (c *Chain33) SetLabl(in types.ReqWalletSetLabel, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletSetLabel", &in)
	if err != nil {
		return err
	}

	*result = &rpctypes.WalletAccount{Acc: &rpctypes.Account{Addr: reply.(*types.WalletAccount).GetAcc().Addr, Currency: reply.(*types.WalletAccount).GetAcc().GetCurrency(),
		Frozen: reply.(*types.WalletAccount).GetAcc().GetFrozen(), Balance: reply.(*types.WalletAccount).GetAcc().GetBalance()}, Label: reply.(*types.WalletAccount).GetLabel()}
	return nil
}

//GetAccount getAddress by lable
func (c *Chain33) GetAccount(in types.ReqGetAccount, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletGetAccount", &in)
	if err != nil {
		return err
	}

	*result = &rpctypes.WalletAccount{Acc: &rpctypes.Account{Addr: reply.(*types.WalletAccount).GetAcc().Addr, Currency: reply.(*types.WalletAccount).GetAcc().GetCurrency(),
		Frozen: reply.(*types.WalletAccount).GetAcc().GetFrozen(), Balance: reply.(*types.WalletAccount).GetAcc().GetBalance()}, Label: reply.(*types.WalletAccount).GetLabel()}
	return nil

}

// MergeBalance merge balance
func (c *Chain33) MergeBalance(in types.ReqWalletMergeBalance, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletMergeBalance", &in)
	if err != nil {
		return err
	}

	var hashes rpctypes.ReplyHashes
	for _, has := range reply.(*types.ReplyHashes).Hashes {
		hashes.Hashes = append(hashes.Hashes, common.ToHex(has))
	}
	*result = &hashes
	return nil
}

// SetPasswd set password
func (c *Chain33) SetPasswd(in types.ReqWalletSetPasswd, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletSetPasswd", &in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.(*types.Reply).GetIsOk()
	resp.Msg = string(reply.(*types.Reply).GetMsg())
	*result = &resp
	return nil
}

// Lock wallet lock
func (c *Chain33) Lock(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletLock", &in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.(*types.Reply).GetIsOk()
	resp.Msg = string(reply.(*types.Reply).GetMsg())
	*result = &resp
	return nil
}

// UnLock wallet unlock
func (c *Chain33) UnLock(in types.WalletUnLock, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "WalletUnLock", &in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.(*types.Reply).GetIsOk()
	resp.Msg = string(reply.(*types.Reply).GetMsg())
	*result = &resp
	return nil
}

// GetPeerInfo get peer information
func (c *Chain33) GetPeerInfo(in types.P2PGetPeerReq, result *interface{}) error {
	reply, err := c.cli.PeerInfo(&in)
	if err != nil {
		return err
	}
	{

		var peerlist rpctypes.PeerList
		for _, peer := range reply.Peers {
			var pr rpctypes.Peer
			pr.Addr = peer.GetAddr()
			pr.MempoolSize = peer.GetMempoolSize()
			pr.Name = peer.GetName()
			pr.Port = peer.GetPort()
			pr.Self = peer.GetSelf()
			pr.Header = &rpctypes.Header{
				BlockTime:  peer.Header.GetBlockTime(),
				Height:     peer.Header.GetHeight(),
				ParentHash: common.ToHex(peer.GetHeader().GetParentHash()),
				StateHash:  common.ToHex(peer.GetHeader().GetStateHash()),
				TxHash:     common.ToHex(peer.GetHeader().GetTxHash()),
				Version:    peer.GetHeader().GetVersion(),
				Hash:       common.ToHex(peer.GetHeader().GetHash()),
				TxCount:    peer.GetHeader().GetTxCount(),
			}

			pr.Version = peer.GetVersion()
			pr.LocalDBVersion = peer.GetLocalDBVersion()
			pr.StoreDBVersion = peer.GetStoreDBVersion()
			peerlist.Peers = append(peerlist.Peers, &pr)

		}
		*result = &peerlist
	}

	return nil
}

// GetHeaders get headers
func (c *Chain33) GetHeaders(in types.ReqBlocks, result *interface{}) error {
	reply, err := c.cli.GetHeaders(&in)
	if err != nil {
		return err
	}
	var headers rpctypes.Headers
	{
		for _, item := range reply.Items {
			headers.Items = append(headers.Items, &rpctypes.Header{
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

// GetLastMemPool get  contents in last mempool
func (c *Chain33) GetLastMemPool(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.GetLastMempool()
	if err != nil {
		return err
	}

	{
		var txlist rpctypes.ReplyTxList
		txs := reply.GetTxs()
		for _, tx := range txs {
			tran, err := rpctypes.DecodeTx(tx)
			if err != nil {
				continue
			}
			txlist.Txs = append(txlist.Txs, tran)
		}
		*result = &txlist
	}
	return nil
}

// GetProperFee get  contents in proper fee
func (c *Chain33) GetProperFee(in types.ReqProperFee, result *interface{}) error {
	reply, err := c.cli.GetProperFee(&in)
	if err != nil {
		return err
	}
	var properFee rpctypes.ReplyProperFee
	properFee.ProperFee = reply.GetProperFee()
	*result = &properFee
	return nil
}

// GetBlockOverview get overview of block
// GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (c *Chain33) GetBlockOverview(in rpctypes.QueryParm, result *interface{}) error {
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
	var blockOverview rpctypes.BlockOverview

	//获取blockheader信息
	var header rpctypes.Header
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

// GetAddrOverview get overview of address
func (c *Chain33) GetAddrOverview(in types.ReqAddr, result *interface{}) error {
	reply, err := c.cli.GetAddrOverview(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// GetBlockHash get block hash
func (c *Chain33) GetBlockHash(in types.ReqInt, result *interface{}) error {
	reply, err := c.cli.GetBlockHash(&in)
	if err != nil {
		return err
	}
	var replyHash rpctypes.ReplyHash
	replyHash.Hash = common.ToHex(reply.GetHash())
	*result = &replyHash
	return nil
}

// GenSeed seed
func (c *Chain33) GenSeed(in types.GenSeedLang, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "GenSeed", &in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// SaveSeed save seed
func (c *Chain33) SaveSeed(in types.SaveSeedByPw, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "SaveSeed", &in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.(*types.Reply).GetIsOk()
	resp.Msg = string(reply.(*types.Reply).GetMsg())
	*result = &resp
	return nil
}

// GetSeed get seed
func (c *Chain33) GetSeed(in types.GetSeedByPw, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "GetSeed", &in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// GetWalletStatus get status of wallet
func (c *Chain33) GetWalletStatus(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "GetWalletStatus", &in)
	if err != nil {
		return err
	}

	status := rpctypes.WalletStatus{
		IsWalletLock: reply.(*types.WalletStatus).IsWalletLock,
		IsAutoMining: reply.(*types.WalletStatus).IsAutoMining,
		IsHasSeed:    reply.(*types.WalletStatus).IsHasSeed,
		IsTicketLock: reply.(*types.WalletStatus).IsTicketLock,
	}
	*result = &status
	return nil
}

// GetBalance get balance
func (c *Chain33) GetBalance(in types.ReqBalance, result *interface{}) error {
	//增加addr地址的校验
	for _, addr := range in.GetAddresses() {
		err := address.CheckAddress(addr)
		if err != nil {
			if err = address.CheckMultiSignAddress(addr); err != nil {
				return types.ErrInvalidAddress
			}
		}
	}
	balances, err := c.cli.GetBalance(&in)
	if err != nil {
		return err
	}

	*result = fmtAccount(balances)
	return nil
}

// GetAllExecBalance get all balance of exec
func (c *Chain33) GetAllExecBalance(in types.ReqAllExecBalance, result *interface{}) error {
	balance, err := c.cli.GetAllExecBalance(&in)
	if err != nil {
		return err
	}

	allBalance := &rpctypes.AllExecBalance{Addr: in.Addr}
	for _, execAcc := range balance.ExecAccount {
		res := &rpctypes.ExecAccount{Execer: execAcc.Execer}
		acc := &rpctypes.Account{
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

// ExecWallet exec wallet
func (c *Chain33) ExecWallet(in *rpctypes.ChainExecutor, result *interface{}) error {
	hash, err := common.FromHex(in.StateHash)
	if err != nil {
		return err
	}
	param, err := wcom.QueryData.DecodeJSON(in.Driver, in.FuncName, in.Payload)
	if err != nil {
		return err
	}
	execdata := &types.ChainExecutor{
		Driver:    in.Driver,
		FuncName:  in.FuncName,
		StateHash: hash,
		Param:     types.Encode(param),
	}
	msg, err := c.cli.ExecWallet(execdata)
	if err != nil {
		return err
	}
	var jsonmsg json.RawMessage
	jsonmsg, err = types.PBToJSON(msg)
	if err != nil {
		return err
	}
	*result = jsonmsg
	return nil
}

// Query query
func (c *Chain33) Query(in rpctypes.Query4Jrpc, result *interface{}) error {
	cfg := c.cli.GetConfig()
	execty := types.LoadExecutorType(in.Execer)
	if execty == nil {
		log.Error("Query", "funcname", in.FuncName, "err", types.ErrNotSupport)
		return types.ErrNotSupport
	}
	decodePayload, err := execty.CreateQuery(in.FuncName, in.Payload)
	if err != nil {
		log.Error("EventQuery1", "err", err.Error(), "funcName", in.FuncName)
		return err
	}
	resp, err := c.cli.Query(cfg.ExecName(in.Execer), in.FuncName, decodePayload)
	if err != nil {
		log.Error("EventQuery2", "err", err.Error())
		return err
	}
	var jsonmsg json.RawMessage
	jsonmsg, err = execty.QueryToJSON(in.FuncName, resp)
	*result = jsonmsg
	if err != nil {
		log.Error("EventQuery3", "err", err.Error())
		return err
	}
	return nil
}

// DumpPrivkey dump privkey
func (c *Chain33) DumpPrivkey(in types.ReqString, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "DumpPrivkey", &in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// DumpPrivkeysFile dumps private key to file.
func (c *Chain33) DumpPrivkeysFile(in types.ReqPrivkeysFile, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "DumpPrivkeysFile", &in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.(*types.Reply).GetIsOk()
	resp.Msg = string(reply.(*types.Reply).GetMsg())
	*result = &resp
	return nil
}

// ImportPrivkeysFile imports private key from file.
func (c *Chain33) ImportPrivkeysFile(in types.ReqPrivkeysFile, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "ImportPrivkeysFile", &in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.(*types.Reply).GetIsOk()
	resp.Msg = string(reply.(*types.Reply).GetMsg())
	*result = &resp
	return nil
}

// Version get software version
func (c *Chain33) Version(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.Version()
	if err != nil {
		return err
	}
	*result = resp
	return nil
}

// GetTotalCoins get total coins
func (c *Chain33) GetTotalCoins(in *types.ReqGetTotalCoins, result *interface{}) error {
	resp, err := c.cli.GetTotalCoins(in)
	if err != nil {
		return err
	}
	*result = resp
	return nil
}

// IsSync is sync or not
func (c *Chain33) IsSync(in *types.ReqNil, result *interface{}) error {
	reply, err := c.cli.IsSync()
	if err != nil {
		return err
	}
	ret := false
	if reply != nil {
		ret = reply.IsOk
	}
	*result = ret
	return nil
}

// IsNtpClockSync  is ntp clock sync
func (c *Chain33) IsNtpClockSync(in *types.ReqNil, result *interface{}) error {
	reply, err := c.cli.IsNtpClockSync()
	if err != nil {
		return err
	}
	ret := false
	if reply != nil {
		ret = reply.IsOk
	}
	*result = ret
	return nil
}

// QueryTotalFee query total fee
func (c *Chain33) QueryTotalFee(in *types.LocalDBGet, result *interface{}) error {
	if in == nil || len(in.Keys) != 1 {
		return types.ErrInvalidParam
	}
	totalFeePrefix := []byte("TotalFeeKey:")
	//add prefix if not exist
	if !bytes.HasPrefix(in.Keys[0], totalFeePrefix) {
		in.Keys[0] = append(totalFeePrefix, in.Keys[0]...)
	}
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

// SignRawTx signature the rawtransaction
func (c *Chain33) SignRawTx(in *types.ReqSignRawTx, result *interface{}) error {
	req := types.ReqSignRawTx{Addr: in.Addr, Privkey: in.Privkey, TxHex: in.TxHex, Expire: in.Expire,
		Index: in.Index, Token: in.Token, Fee: in.Fee, NewToAddr: in.NewToAddr}
	reply, err := c.cli.ExecWalletFunc("wallet", "SignRawTx", &req)
	if err != nil {
		return err
	}
	*result = reply.(*types.ReplySignRawTx).TxHex
	return nil
}

// GetNetInfo get net information
func (c *Chain33) GetNetInfo(in types.P2PGetNetInfoReq, result *interface{}) error {
	resp, err := c.cli.GetNetInfo(&in)
	if err != nil {
		return err
	}
	*result = &rpctypes.NodeNetinfo{
		Externaladdr: resp.GetExternaladdr(),
		Localaddr:    resp.GetLocaladdr(),
		Service:      resp.GetService(),
		Outbounds:    resp.GetOutbounds(),
		Inbounds:     resp.GetInbounds(),
		Peerstore:    resp.GetPeerstore(),
		Routingtable: resp.GetRoutingtable(),
		Ratein:       resp.GetRatein(),
		Rateout:      resp.GetRateout(),
		Ratetotal:    resp.GetRatetotal(),
	}
	return nil
}

// GetFatalFailure return fatal failure
func (c *Chain33) GetFatalFailure(in *types.ReqNil, result *interface{}) error {
	reply, err := c.cli.ExecWalletFunc("wallet", "FatalFailure", &types.ReqNil{})
	if err != nil {
		return err
	}
	*result = reply.(*types.Int32).Data
	return nil
}

// DecodeRawTransaction 考虑交易组的解析统一返回ReplyTxList列表
func (c *Chain33) DecodeRawTransaction(in *types.ReqDecodeRawTransaction, result *interface{}) error {
	tx, err := c.cli.DecodeRawTransaction(in)
	if err != nil {
		return err
	}
	txs, err := tx.GetTxGroup()
	if err != nil {
		return err
	}
	var rpctxs rpctypes.ReplyTxList
	if txs == nil {
		res, err := rpctypes.DecodeTx(tx)
		if err != nil {
			return err
		}
		rpctxs.Txs = append(rpctxs.Txs, res)
	} else {
		for _, rpctx := range txs.GetTxs() {
			res, err := rpctypes.DecodeTx(rpctx)
			if err != nil {
				return err
			}
			rpctxs.Txs = append(rpctxs.Txs, res)
		}
	}
	*result = &rpctxs
	return nil
}

// GetTimeStatus get status of time
func (c *Chain33) GetTimeStatus(in *types.ReqNil, result *interface{}) error {
	reply, err := c.cli.GetTimeStatus()
	if err != nil {
		return err
	}
	timeStatus := &rpctypes.TimeStatus{
		NtpTime:   reply.NtpTime,
		LocalTime: reply.LocalTime,
		Diff:      reply.Diff,
	}
	*result = timeStatus
	return nil
}

// CloseQueue close queue
func (c *Chain33) CloseQueue(in *types.ReqNil, result *interface{}) error {
	go func() {
		time.Sleep(time.Millisecond * 100)
		_, err := c.cli.CloseQueue()
		if err != nil {
			return
		}
	}()

	*result = &types.Reply{IsOk: true}
	return nil
}

// GetLastBlockSequence get sequence last block
func (c *Chain33) GetLastBlockSequence(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.GetLastBlockSequence()
	if err != nil {
		return err
	}
	*result = resp.GetData()
	return nil
}

// GetBlockSequences get the block loading sequence number information for the specified interval
func (c *Chain33) GetBlockSequences(in rpctypes.BlockParam, result *interface{}) error {
	resp, err := c.cli.GetBlockSequences(&types.ReqBlocks{Start: in.Start, End: in.End, IsDetail: in.Isdetail, Pid: []string{""}})
	if err != nil {
		return err
	}
	var BlkSeqs rpctypes.ReplyBlkSeqs
	items := resp.GetItems()
	for _, item := range items {
		BlkSeqs.BlkSeqInfos = append(BlkSeqs.BlkSeqInfos, &rpctypes.ReplyBlkSeq{Hash: common.ToHex(item.GetHash()),
			Type: item.GetType()})
	}
	*result = &BlkSeqs
	return nil
}

// GetBlockByHashes get block information by hashes
func (c *Chain33) GetBlockByHashes(in rpctypes.ReqHashes, result *interface{}) error {
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
	{
		var blockDetails rpctypes.BlockDetails
		items := reply.Items
		if err := convertBlockDetails(items, &blockDetails, !in.DisableDetail); err != nil {
			return err
		}
		*result = &blockDetails
	}
	return nil
}

// CreateTransaction create transaction
func (c *Chain33) CreateTransaction(in *rpctypes.CreateTxIn, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	cfg := c.cli.GetConfig()
	btx, err := types.CallCreateTxJSON(cfg, cfg.ExecName(in.Execer), in.ActionName, in.Payload)
	if err != nil {
		return err
	}
	*result = common.ToHex(btx)
	return nil
}

// ConvertExectoAddr convert exec to address
func (c *Chain33) ConvertExectoAddr(in rpctypes.ExecNameParm, result *string) error {
	*result = address.ExecAddress(in.ExecName)
	return nil
}

// GetExecBalance get balance exec
func (c *Chain33) GetExecBalance(in *types.ReqGetExecBalance, result *interface{}) error {
	resp, err := c.cli.GetExecBalance(in)
	if err != nil {
		return err
	}
	//*result = resp
	*result = hex.EncodeToString(types.Encode(resp))
	return nil
}

// AddPushSubscribe  add Seq CallBack
func (c *Chain33) AddPushSubscribe(in *types.PushSubscribeReq, result *interface{}) error {
	resp, err := c.cli.AddPushSubscribe(in)
	log.Error("AddPushSubscribe", "err", err, "reply", resp)

	if err != nil {
		return err
	}
	*result = resp
	return nil
}

// ListPushes  List Seq CallBack
func (c *Chain33) ListPushes(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.ListPushes()
	if err != nil {
		return err
	}
	*result = resp
	return nil
}

// GetPushSeqLastNum  Get Seq Call Back Last Num
func (c *Chain33) GetPushSeqLastNum(in *types.ReqString, result *interface{}) error {
	resp, err := c.cli.GetPushSeqLastNum(in)
	if err != nil {
		return err
	}
	*result = resp
	return nil
}

func convertBlockDetails(details []*types.BlockDetail, retDetails *rpctypes.BlockDetails, isDetail bool) error {
	for _, item := range details {
		var bdtl rpctypes.BlockDetail
		var block rpctypes.Block
		if item == nil || item.GetBlock() == nil {
			retDetails.Items = append(retDetails.Items, nil)
			continue
		}
		block.BlockTime = item.Block.GetBlockTime()
		block.Height = item.Block.GetHeight()
		block.Version = item.Block.GetVersion()
		block.ParentHash = common.ToHex(item.Block.GetParentHash())
		block.StateHash = common.ToHex(item.Block.GetStateHash())
		block.TxHash = common.ToHex(item.Block.GetTxHash())
		block.Difficulty = item.Block.Difficulty
		block.MainHash = common.ToHex(item.Block.MainHash)
		block.MainHeight = item.Block.MainHeight
		if item.Block.Signature != nil {
			block.Signature = &rpctypes.Signature{Ty: item.Block.Signature.Ty, Pubkey: common.ToHex(item.Block.Signature.Pubkey),
				Signature: common.ToHex(item.Block.Signature.Signature)}
		}

		txs := item.Block.GetTxs()
		if isDetail && len(txs) != len(item.Receipts) { //只有获取详情时才需要校验txs和Receipts的数量是否相等CHAIN33-540
			return types.ErrDecode
		}
		for _, tx := range txs {
			tran, err := rpctypes.DecodeTx(tx)
			if err != nil {
				continue
			}
			block.Txs = append(block.Txs, tran)
		}
		bdtl.Block = &block

		for i, rp := range item.Receipts {
			var recp rpctypes.ReceiptData
			recp.Ty = rp.GetTy()
			for _, log := range rp.Logs {
				recp.Logs = append(recp.Logs,
					&rpctypes.ReceiptLog{Ty: log.Ty, Log: common.ToHex(log.GetLog())})
			}
			rd, err := rpctypes.DecodeLog(txs[i].Execer, &recp)
			if err != nil {
				continue
			}
			bdtl.Receipts = append(bdtl.Receipts, rd)
		}

		retDetails.Items = append(retDetails.Items, &bdtl)
	}
	return nil
}

func fmtAccount(balances []*types.Account) []*rpctypes.Account {
	var accounts []*rpctypes.Account
	for _, balance := range balances {
		accounts = append(accounts, &rpctypes.Account{Addr: balance.GetAddr(),
			Balance:  balance.GetBalance(),
			Currency: balance.GetCurrency(),
			Frozen:   balance.GetFrozen()})
	}
	return accounts
}

// GetCoinSymbol get coin symbol
func (c *Chain33) GetCoinSymbol(in types.ReqNil, result *interface{}) error {
	cfg := c.cli.GetConfig()
	symbol := cfg.GetCoinSymbol()
	resp := types.ReplyString{Data: symbol}
	log.Warn("GetCoinSymbol", "Symbol", symbol)
	*result = &resp
	return nil
}

func fmtTxProofs(txProofs []*types.TxProof) []*rpctypes.TxProof {
	var result []*rpctypes.TxProof
	for _, txproof := range txProofs {
		var proofs []string
		for _, proof := range txproof.GetProofs() {
			proofs = append(proofs, common.ToHex(proof))
		}
		rpctxproof := &rpctypes.TxProof{
			Proofs:   proofs,
			Index:    txproof.GetIndex(),
			RootHash: common.ToHex(txproof.GetRootHash()),
		}
		result = append(result, rpctxproof)
	}
	return result
}

// NetProtocols get net information
func (c *Chain33) NetProtocols(in types.ReqNil, result *interface{}) error {
	resp, err := c.cli.NetProtocols(&in)
	if err != nil {
		return err
	}

	*result = resp
	return nil
}

//GetSequenceByHash get sequcen by hashes
func (c *Chain33) GetSequenceByHash(in rpctypes.ReqHashes, result *interface{}) error {
	if len(in.Hashes) != 0 && common.IsHex(in.Hashes[0]) {
		var req types.ReqHash
		req.Upgrade = in.DisableDetail
		req.Hash = common.HexToHash(in.Hashes[0]).Bytes()
		seq, err := c.cli.GetSequenceByHash(&req)
		if err != nil {
			return err
		}
		*result = seq
		return nil
	}

	return types.ErrInvalidParam

}

//GetBlockBySeq get block by seq
func (c *Chain33) GetBlockBySeq(in types.Int64, result *interface{}) error {

	blockSeq, err := c.cli.GetBlockBySeq(&in)
	if err != nil {
		return err
	}
	var bseq rpctypes.BlockSeq
	var retDetail rpctypes.BlockDetails

	bseq.Num = blockSeq.Num
	err = convertBlockDetails([]*types.BlockDetail{blockSeq.Detail}, &retDetail, false)
	bseq.Detail = retDetail.Items[0]
	bseq.Seq = &rpctypes.BlockSequence{Hash: common.ToHex(blockSeq.Seq.Hash), Type: blockSeq.Seq.Type}
	*result = bseq
	return err

}

func convertHeader(header *types.Header, message *rpctypes.Header) {

	message.BlockTime = header.GetBlockTime()
	message.Height = header.GetHeight()
	message.ParentHash = common.ToHex(header.GetParentHash())
	message.StateHash = common.ToHex(header.GetStateHash())
	message.TxHash = common.ToHex(header.GetTxHash())
	message.Version = header.GetVersion()
	message.Hash = common.ToHex(header.GetHash())
	message.TxCount = header.GetTxCount()
	message.Difficulty = header.GetDifficulty()
	if header.Signature != nil {
		message.Signature = &rpctypes.Signature{Ty: header.Signature.Ty, Pubkey: common.ToHex(header.Signature.Pubkey),
			Signature: common.ToHex(header.Signature.Signature)}
	}
}

//GetParaTxByTitle get paraTx by title
func (c *Chain33) GetParaTxByTitle(req types.ReqParaTxByTitle, result *interface{}) error {
	paraTxDetails, err := c.cli.GetParaTxByTitle(&req)
	if err != nil {
		return err
	}
	var paraDetails rpctypes.ParaTxDetails
	convertParaTxDetails(paraTxDetails, &paraDetails)
	*result = paraDetails
	return nil
}

//LoadParaTxByTitle load paratx by title
func (c *Chain33) LoadParaTxByTitle(req types.ReqHeightByTitle, result *interface{}) error {

	reply, err := c.cli.LoadParaTxByTitle(&req)
	if err != nil {
		return err
	}
	var replyHeight rpctypes.ReplyHeightByTitle
	replyHeight.Title = reply.Title
	for _, item := range reply.Items {
		replyHeight.Items = append(replyHeight.Items, &rpctypes.BlockInfo{Height: item.Height, Hash: common.ToHex(item.Hash)})
	}

	*result = replyHeight
	return nil
}

func convertParaTxDetails(details *types.ParaTxDetails, message *rpctypes.ParaTxDetails) {
	for _, item := range details.Items {
		var ptxDetail rpctypes.ParaTxDetail
		var header rpctypes.Header
		convertHeader(item.Header, &header)
		ptxDetail.Header = &header
		ptxDetail.Type = item.Type
		ptxDetail.Index = item.Index
		ptxDetail.ChildHash = common.ToHex(item.ChildHash)
		for _, proof := range item.Proofs {
			ptxDetail.Proofs = append(ptxDetail.Proofs, common.ToHex(proof))
		}
		for _, detail := range item.TxDetails {
			var txDetail rpctypes.TxDetail
			txDetail.Index = detail.Index
			for _, proof := range detail.Proofs {
				txDetail.Proofs = append(txDetail.Proofs, common.ToHex(proof))
			}

			var receipt rpctypes.ReceiptData
			receipt.Ty = detail.Receipt.Ty
			for _, log := range detail.Receipt.Logs {
				receipt.Logs = append(receipt.Logs, &rpctypes.ReceiptLog{Ty: log.Ty, Log: common.ToHex(log.Log)})
			}
			txDetail.Receipt = &receipt
			tranTx, err := rpctypes.DecodeTx(detail.Tx)
			if err != nil {
				continue
			}
			txDetail.Tx = tranTx
			ptxDetail.TxDetails = append(ptxDetail.TxDetails, &txDetail)
		}
		message.Items = append(message.Items, &ptxDetail)
	}

}

//GetParaTxByHeight get paraTx by block height
func (c *Chain33) GetParaTxByHeight(req types.ReqParaTxByHeight, result *interface{}) error {
	paraTxDetails, err := c.cli.GetParaTxByHeight(&req)
	if err != nil {
		return err
	}
	var ptxDetails rpctypes.ParaTxDetails
	convertParaTxDetails(paraTxDetails, &ptxDetails)
	*result = ptxDetails
	return nil

}

//QueryChain querychain by chain executor
func (c *Chain33) QueryChain(in rpctypes.ChainExecutor, result *interface{}) error {
	var qin = new(types.ChainExecutor)
	msg, err := types.QueryFunc.DecodeJSON(in.Driver, in.FuncName, in.Payload)
	if err != nil {
		log.Error("QueryChain", "DecodeJSON err", err, "driver", in.Driver,
			"func name", in.FuncName, "payload size", len(in.Payload))
		return err
	}

	qin.Driver = in.Driver
	qin.FuncName = in.FuncName
	qin.Param = types.Encode(msg)
	if in.StateHash != "" {
		qin.StateHash = common.HexToHash(in.StateHash).Bytes()
	}

	msg, err = c.cli.QueryChain(qin)
	if err != nil {
		log.Error("QueryChain", "err", err)
		return err
	}
	var jsonMsg json.RawMessage
	jsonMsg, err = types.PBToJSON(msg)
	if err != nil {
		return err
	}
	*result = jsonMsg
	return err
}
