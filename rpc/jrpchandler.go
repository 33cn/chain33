// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/rpc/jsonclient"
	"github.com/33cn/chain33/types"
	wcom "github.com/33cn/chain33/wallet/common"

	rpctypes "github.com/33cn/chain33/rpc/types"
)

// CreateRawTransaction create rawtransaction by jrpc
func (c *Chain33) CreateRawTransaction(in *types.CreateTx, result *interface{}) error {
	reply, err := c.cli.CreateRawTransaction(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

// CreateRawTxGroup create rawtransaction with group
func (c *Chain33) CreateRawTxGroup(in *types.CreateTransactionGroup, result *interface{}) error {
	reply, err := c.cli.CreateRawTxGroup(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

// CreateNoBalanceTransaction create transaction with no balance
func (c *Chain33) CreateNoBalanceTransaction(in *types.NoBalanceTx, result *string) error {
	tx, err := c.cli.CreateNoBalanceTransaction(in)
	if err != nil {
		return err
	}
	grouptx := hex.EncodeToString(types.Encode(tx))
	*result = grouptx
	return nil
}

// SendRawTransaction send rawtransacion
func (c *Chain33) SendRawTransaction(in rpctypes.SignedTx, result *interface{}) error {
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
	}
	return fmt.Errorf(string(reply.Msg))
}

// used only in parachain
func forwardTranToMainNet(in rpctypes.RawParm, result *interface{}) error {
	if rpcCfg.MainnetJrpcAddr == "" {
		return types.ErrInvalidMainnetRPCAddr
	}
	rpc, err := jsonclient.NewJSONClient(rpcCfg.MainnetJrpcAddr)

	if err != nil {
		return err
	}

	err = rpc.Call("Chain33.SendTransaction", in, result)
	return err
}

// SendTransaction send transaction
func (c *Chain33) SendTransaction(in rpctypes.RawParm, result *interface{}) error {
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
	*result = hex.EncodeToString(types.Encode(reply.GetTx()))
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
		for _, item := range items {
			var bdtl rpctypes.BlockDetail
			var block rpctypes.Block
			block.BlockTime = item.Block.GetBlockTime()
			block.Height = item.Block.GetHeight()
			block.Version = item.Block.GetVersion()
			block.ParentHash = common.ToHex(item.Block.GetParentHash())
			block.StateHash = common.ToHex(item.Block.GetStateHash())
			block.TxHash = common.ToHex(item.Block.GetTxHash())
			txs := item.Block.GetTxs()
			if in.Isdetail && len(txs) != len(item.Receipts) { //只有获取详情时才需要校验txs和Receipts的数量是否相等CHAIN33-540
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

			blockDetails.Items = append(blockDetails.Items, &bdtl)
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
				Height: info.GetHeight(), Index: info.GetIndex(), Assets: info.Assets})
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
			txDetail, _ := fmtTxDetail(tx, in.DisableDetail)
			txdetails.Txs = append(txdetails.Txs, txDetail)
		}
	}
	*result = &txdetails
	return nil
}

func fmtTxDetail(tx *types.TransactionDetail, disableDetail bool) (*rpctypes.TransactionDetail, error) {
	//增加判断，上游接口可能返回空指针
	if tx == nil {
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
		Assets:     tx.GetAssets(),
	}, nil
}

// GetMempool get mempool information
func (c *Chain33) GetMempool(in *types.ReqNil, result *interface{}) error {

	reply, err := c.cli.GetMempool()
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
	reply, err := c.cli.WalletGetAccountList(in)
	if err != nil {
		return err
	}
	var accounts rpctypes.WalletAccounts
	for _, wallet := range reply.Wallets {
		accounts.Wallets = append(accounts.Wallets, &rpctypes.WalletAccount{Label: wallet.GetLabel(),
			Acc: &rpctypes.Account{Currency: wallet.GetAcc().GetCurrency(), Balance: wallet.GetAcc().GetBalance(),
				Frozen: wallet.GetAcc().GetFrozen(), Addr: wallet.GetAcc().GetAddr()}})
	}
	*result = &accounts
	return nil
}

// NewAccount new a account
func (c *Chain33) NewAccount(in types.ReqNewAccount, result *interface{}) error {
	reply, err := c.cli.NewAccount(&in)
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
	reply, err := c.cli.WalletTransactionList(&parm)
	if err != nil {
		return err
	}
	{
		var txdetails rpctypes.WalletTxDetails
		rpctypes.ConvertWalletTxDetailToJSON(reply, &txdetails)
		*result = &txdetails
	}
	return nil
}

// ImportPrivkey import privkey of wallet
func (c *Chain33) ImportPrivkey(in types.ReqWalletImportPrivkey, result *interface{}) error {
	reply, err := c.cli.WalletImportprivkey(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// SendToAddress send to address of coins
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
			ExecName:    types.ExecName("token"),
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
		rawParm := rpctypes.RawParm{
			Token: "",
			Data:  replySignRawTx.GetTxHex(),
		}
		var txHash interface{}
		err = forwardTranToMainNet(rawParm, &txHash)
		if err != nil {
			log.Debug("ParaChain forwardTranToMainNet", "Error", err.Error())
			return err
		}
		*result = &rpctypes.ReplyHash{Hash: txHash.(string)}
		return nil
	}
	reply, err := c.cli.WalletSendToAddress(&in)
	if err != nil {
		log.Debug("SendToAddress", "Error", err.Error())
		return err
	}
	log.Debug("sendtoaddr", "msg", reply.String())
	*result = &rpctypes.ReplyHash{Hash: common.ToHex(reply.GetHash())}
	log.Debug("SendToAddress", "resulrt", *result)
	return nil
}

// SetTxFee set tx fee
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
	var resp rpctypes.Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

// SetLabl set lable
func (c *Chain33) SetLabl(in types.ReqWalletSetLabel, result *interface{}) error {
	reply, err := c.cli.WalletSetLabel(&in)
	if err != nil {
		return err
	}

	*result = &rpctypes.WalletAccount{Acc: &rpctypes.Account{Addr: reply.GetAcc().Addr, Currency: reply.GetAcc().GetCurrency(),
		Frozen: reply.GetAcc().GetFrozen(), Balance: reply.GetAcc().GetBalance()}, Label: reply.GetLabel()}
	return nil
}

// MergeBalance merge balance
func (c *Chain33) MergeBalance(in types.ReqWalletMergeBalance, result *interface{}) error {
	reply, err := c.cli.WalletMergeBalance(&in)
	if err != nil {
		return err
	}
	{
		var hashes rpctypes.ReplyHashes
		for _, has := range reply.Hashes {
			hashes.Hashes = append(hashes.Hashes, common.ToHex(has))
		}
		*result = &hashes
	}

	return nil
}

// SetPasswd set password
func (c *Chain33) SetPasswd(in types.ReqWalletSetPasswd, result *interface{}) error {
	reply, err := c.cli.WalletSetPasswd(&in)
	if err != nil {
		return err
	}
	var resp rpctypes.Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

// Lock wallet lock
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
	var resp rpctypes.Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

// UnLock wallet unlock
func (c *Chain33) UnLock(in types.WalletUnLock, result *interface{}) error {
	reply, err := c.cli.WalletUnLock(&in)
	if err != nil {
		return err
	}
	var resp rpctypes.Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

// GetPeerInfo get peer information
func (c *Chain33) GetPeerInfo(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.PeerInfo()
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
	reply, err := c.cli.GenSeed(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// SaveSeed save seed
func (c *Chain33) SaveSeed(in types.SaveSeedByPw, result *interface{}) error {
	reply, err := c.cli.SaveSeed(&in)
	if err != nil {
		return err
	}

	var resp rpctypes.Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

// GetSeed get seed
func (c *Chain33) GetSeed(in types.GetSeedByPw, result *interface{}) error {
	reply, err := c.cli.GetSeed(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// GetWalletStatus get status of wallet
func (c *Chain33) GetWalletStatus(in types.ReqNil, result *interface{}) error {
	reply, err := c.cli.GetWalletStatus()
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

// GetBalance get balance
func (c *Chain33) GetBalance(in types.ReqBalance, result *interface{}) error {

	balances, err := c.cli.GetBalance(&in)
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

// GetAllExecBalance get all balance of exec
func (c *Chain33) GetAllExecBalance(in types.ReqAddr, result *interface{}) error {
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
	execty := types.LoadExecutorType(in.Execer)
	if execty == nil {
		log.Error("Query", "funcname", in.FuncName, "err", types.ErrNotSupport)
		return types.ErrNotSupport
	}

	decodePayload, err := execty.CreateQuery(in.FuncName, in.Payload)
	if err != nil {
		log.Error("EventQuery1", "err", err.Error())
		return err
	}
	resp, err := c.cli.Query(types.ExecName(in.Execer), in.FuncName, decodePayload)
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
	reply, err := c.cli.DumpPrivkey(&in)
	if err != nil {
		return err
	}

	*result = reply
	return nil
}

// Version version
func (c *Chain33) Version(in *types.ReqNil, result *interface{}) error {
	*result = version.GetVersion()
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
	reply, _ := c.cli.IsSync()
	ret := false
	if reply != nil {
		ret = reply.IsOk
	}
	*result = ret
	return nil
}

// IsNtpClockSync  is ntp clock sync
func (c *Chain33) IsNtpClockSync(in *types.ReqNil, result *interface{}) error {
	reply, _ := c.cli.IsNtpClockSync()
	ret := false
	if reply != nil {
		ret = reply.IsOk
	}
	*result = ret
	return nil
}

// QueryTotalFee query total fee
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

// SignRawTx signature the rawtransaction
func (c *Chain33) SignRawTx(in *types.ReqSignRawTx, result *interface{}) error {
	resp, err := c.cli.SignRawTx(in)
	if err != nil {
		return err
	}
	*result = resp.TxHex
	return nil
}

// GetNetInfo get net information
func (c *Chain33) GetNetInfo(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.GetNetInfo()
	if err != nil {
		return err
	}
	*result = &rpctypes.NodeNetinfo{
		Externaladdr: resp.GetExternaladdr(),
		Localaddr:    resp.GetLocaladdr(),
		Service:      resp.GetService(),
		Outbounds:    resp.GetOutbounds(),
		Inbounds:     resp.GetInbounds(),
	}
	return nil
}

// GetFatalFailure return fatal failure
func (c *Chain33) GetFatalFailure(in *types.ReqNil, result *interface{}) error {
	resp, err := c.cli.GetFatalFailure()
	if err != nil {
		return err
	}
	*result = resp.GetData()
	return nil

}

// QueryTicketStat quert stat of ticket
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

// QueryTicketInfo query ticket information
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

// QueryTicketInfoList query ticket list information
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

// DecodeRawTransaction decode rawtransaction
func (c *Chain33) DecodeRawTransaction(in *types.ReqDecodeRawTransaction, result *interface{}) error {
	reply, err := c.cli.DecodeRawTransaction(in)
	if err != nil {
		return err
	}
	res, err := rpctypes.DecodeTx(reply)
	if err != nil {
		return err
	}
	*result = res
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

// WalletCreateTx wallet create tx
func (c *Chain33) WalletCreateTx(in types.ReqCreateTransaction, result *interface{}) error {
	reply, err := c.cli.WalletCreateTx(&in)
	if err != nil {
		return err
	}
	txHex := types.Encode(reply)
	*result = hex.EncodeToString(txHex)
	return nil
}

// CloseQueue close queue
func (c *Chain33) CloseQueue(in *types.ReqNil, result *interface{}) error {
	go func() {
		time.Sleep(time.Millisecond * 100)
		c.cli.CloseQueue()
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
	*result = reply
	return nil
}

// CreateTransaction create transaction
func (c *Chain33) CreateTransaction(in *rpctypes.CreateTxIn, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	exec := types.LoadExecutorType(in.Execer)
	if exec == nil {
		return types.ErrExecNameNotAllow
	}
	tx, err := exec.CreateTx(in.ActionName, in.Payload)
	if err != nil {
		log.Error("CreateTransaction", "err", err.Error())
		return err
	}
	*result = hex.EncodeToString(types.Encode(tx))
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
