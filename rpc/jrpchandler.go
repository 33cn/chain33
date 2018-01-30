package rpc

import (
	"encoding/hex"
	"fmt"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/types"
)

type Chain33 struct {
	jserver *jsonrpcServer
	cli     IRClient
}

func (req Chain33) CreateRawTransaction(in *types.CreateTx, result *interface{}) error {
	reply, err := req.cli.CreateRawTransaction(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil

}
func (req Chain33) SendRawTransaction(in SignedTx, result *interface{}) error {
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
	reply := req.cli.SendRawTransaction(&stx)
	if reply.GetData().(*types.Reply).IsOk {
		*result = "0x" + hex.EncodeToString(reply.GetData().(*types.Reply).Msg)
		return nil
	} else {

		return fmt.Errorf(string(reply.GetData().(*types.Reply).Msg))
	}
}
func (req Chain33) SendTransaction(in RawParm, result *interface{}) error {
	var parm types.Transaction
	data, err := common.FromHex(in.Data)
	if err != nil {
		return err
	}
	types.Decode(data, &parm)
	log.Debug("SendTransaction", "parm", parm)
	reply := req.cli.SendTx(&parm)
	if reply.GetData().(*types.Reply).IsOk {
		*result = string(reply.GetData().(*types.Reply).Msg)
		return nil
	} else {
		return fmt.Errorf(string(reply.GetData().(*types.Reply).Msg))
	}

}

func (req Chain33) QueryTransaction(in QueryParm, result *interface{}) error {
	var data types.ReqHash
	hash, err := common.FromHex(in.Hash)
	if err != nil {
		return err
	}

	data.Hash = hash
	reply, err := req.cli.QueryTx(data.Hash)
	if err != nil {
		return err
	}

	{ //重新格式化数据

		var transDetail TransactionDetail
		transDetail.Tx = &Transaction{
			Execer:  string(reply.GetTx().GetExecer()),
			Payload: common.ToHex(reply.GetTx().GetPayload()),
			Fee:     reply.GetTx().GetFee(),
			Expire:  reply.GetTx().GetExpire(),
			Nonce:   reply.GetTx().GetNonce(),
			To:      reply.GetTx().GetTo(),
			Signature: &Signature{Ty: reply.GetTx().GetSignature().GetTy(),
				Pubkey:    common.ToHex(reply.GetTx().GetSignature().GetPubkey()),
				Signature: common.ToHex(reply.GetTx().GetSignature().GetSignature())}}
		transDetail.Receipt = &ReceiptData{Ty: reply.GetReceipt().GetTy()}
		logs := reply.GetReceipt().GetLogs()
		for _, log := range logs {
			transDetail.Receipt.Logs = append(transDetail.Receipt.Logs,
				&ReceiptLog{Ty: log.GetTy(), Log: common.ToHex(log.GetLog())})
		}

		for _, proof := range reply.Proofs {
			transDetail.Proofs = append(transDetail.Proofs, common.ToHex(proof))
		}
		transDetail.Height = reply.GetHeight()
		transDetail.Index = reply.GetIndex()
		transDetail.Blocktime = reply.GetBlocktime()
		transDetail.Amount = reply.GetAmount()
		transDetail.Fromaddr = reply.GetFromaddr()
		*result = &transDetail
	}

	return nil

}

func (req Chain33) GetBlocks(in BlockParam, result *interface{}) error {
	var data types.ReqBlocks
	data.End = in.End
	data.Start = in.Start
	data.Isdetail = in.Isdetail
	reply, err := req.cli.GetBlocks(data.Start, data.End, data.Isdetail)
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
				block.Txs = append(block.Txs,
					&Transaction{
						Execer:  string(tx.GetExecer()),
						Payload: common.ToHex(tx.GetPayload()),
						Fee:     tx.GetFee(),
						Expire:  tx.GetExpire(),
						Nonce:   tx.GetNonce(),
						To:      tx.GetTo(),
						Signature: &Signature{Ty: tx.GetSignature().GetTy(),
							Pubkey:    common.ToHex(tx.GetSignature().GetPubkey()),
							Signature: common.ToHex(tx.GetSignature().GetSignature())}})

			}
			bdtl.Block = &block

			for _, rp := range item.Receipts {
				var recp ReceiptData
				recp.Ty = rp.GetTy()
				for _, log := range rp.Logs {
					recp.Logs = append(recp.Logs,
						&ReceiptLog{Ty: log.Ty, Log: common.ToHex(log.GetLog())})
				}
				bdtl.Receipts = append(bdtl.Receipts, &recp)
			}

			blockDetails.Items = append(blockDetails.Items, &bdtl)
		}
		*result = &blockDetails
	}

	return nil

}

func (req Chain33) GetLastHeader(in *types.ReqNil, result *interface{}) error {

	reply, err := req.cli.GetLastHeader()
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
func (req Chain33) GetTxByAddr(in types.ReqAddr, result *interface{}) error {

	reply, err := req.cli.GetTxByAddr(&in)
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

func (req Chain33) GetTxByHashes(in ReqHashes, result *interface{}) error {
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
	reply, err := req.cli.GetTxByHashes(&parm)
	if err != nil {
		return err
	}
	{
		var txdetails TransactionDetails
		txs := reply.GetTxs()
		for _, tx := range txs {
			var recp ReceiptData
			logs := tx.GetReceipt().GetLogs()
			recp.Ty = tx.GetReceipt().GetTy()
			for _, lg := range logs {
				recp.Logs = append(recp.Logs,
					&ReceiptLog{Ty: lg.Ty, Log: common.ToHex(lg.GetLog())})
			}

			var proofs []string
			txProofs := tx.GetProofs()
			for _, proof := range txProofs {
				proofs = append(proofs, common.ToHex(proof))
			}

			txdetails.Txs = append(txdetails.Txs,
				&TransactionDetail{
					Tx: &Transaction{
						Execer:  string(tx.GetTx().GetExecer()),
						Payload: common.ToHex(tx.GetTx().GetPayload()),
						Fee:     tx.GetTx().GetFee(),
						Expire:  tx.GetTx().GetExpire(),
						Nonce:   tx.GetTx().GetNonce(),
						To:      tx.GetTx().GetTo(),
						Signature: &Signature{Ty: tx.GetTx().GetSignature().GetTy(),
							Pubkey:    common.ToHex(tx.GetTx().GetSignature().GetPubkey()),
							Signature: common.ToHex(tx.GetTx().GetSignature().GetSignature())},
					},
					Height:    tx.GetHeight(),
					Index:     tx.GetIndex(),
					Blocktime: tx.GetBlocktime(),
					Receipt:   &recp,
					Proofs:    proofs,
					Amount:    tx.GetAmount(),
					Fromaddr:  tx.GetFromaddr(),
				})
		}

		*result = &txdetails
	}

	return nil
}

func (req Chain33) GetMempool(in *types.ReqNil, result *interface{}) error {

	reply, err := req.cli.GetMempool()
	if err != nil {
		return err
	}
	{
		var txlist ReplyTxList
		txs := reply.GetTxs()
		for _, tx := range txs {
			txlist.Txs = append(txlist.Txs,
				&Transaction{
					Execer:  string(tx.GetExecer()),
					Payload: common.ToHex(tx.GetPayload()),
					Fee:     tx.GetFee(),
					Expire:  tx.GetExpire(),
					Nonce:   tx.GetNonce(),
					To:      tx.GetTo(),
					Signature: &Signature{Ty: tx.GetSignature().GetTy(),
						Pubkey:    common.ToHex(tx.GetSignature().GetPubkey()),
						Signature: common.ToHex(tx.GetSignature().GetSignature())}})
		}
		*result = &txlist
	}

	return nil
}

func (req Chain33) GetAccounts(in *types.ReqNil, result *interface{}) error {

	reply, err := req.cli.GetAccounts()
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

func (req Chain33) NewAccount(in types.ReqNewAccount, result *interface{}) error {
	reply, err := req.cli.NewAccount(&in)
	if err != nil {
		return err
	}

	*result = reply
	return nil
}

func (req Chain33) WalletTxList(in ReqWalletTransactionList, result *interface{}) error {
	var parm types.ReqWalletTransactionList
	parm.FromTx = []byte(in.FromTx)
	parm.Count = in.Count
	parm.Direction = in.Direction
	reply, err := req.cli.WalletTxList(&parm)
	if err != nil {
		return err
	}
	{

		var txdetails WalletTxDetails

		for _, tx := range reply.TxDetails {
			var recp ReceiptData
			logs := tx.GetReceipt().GetLogs()

			for _, lg := range logs {
				recp.Ty = lg.GetTy()

				recp.Logs = append(recp.Logs,
					&ReceiptLog{Ty: lg.Ty, Log: common.ToHex(lg.GetLog())})
			}

			txdetails.TxDetails = append(txdetails.TxDetails, &WalletTxDetail{
				Tx: &Transaction{
					Execer:  string(tx.GetTx().GetExecer()),
					Payload: common.ToHex(tx.GetTx().GetPayload()),
					Fee:     tx.GetTx().GetFee(),
					Expire:  tx.GetTx().GetExpire(),
					Nonce:   tx.GetTx().GetNonce(),
					To:      tx.GetTx().GetTo(),
					Signature: &Signature{
						Pubkey:    common.ToHex(tx.GetTx().GetSignature().GetPubkey()),
						Signature: common.ToHex(tx.GetTx().GetSignature().GetSignature()),
					},
				},
				Receipt:   &recp,
				Height:    tx.GetHeight(),
				Index:     tx.GetIndex(),
				Blocktime: tx.GetBlocktime(),
				Amount:    tx.GetAmount(),
				Fromaddr:  tx.GetFromaddr(),
				Txhash:    common.ToHex(tx.GetTxhash()),
			})

		}
		*result = &txdetails
	}

	return nil
}

func (req Chain33) ImportPrivkey(in types.ReqWalletImportPrivKey, result *interface{}) error {
	reply, err := req.cli.ImportPrivkey(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req Chain33) SendToAddress(in types.ReqWalletSendToAddress, result *interface{}) error {
	log.Debug("Rpc SendToAddress", "Tx", in)
	reply, err := req.cli.SendToAddress(&in)
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

func (req Chain33) SetTxFee(in types.ReqWalletSetFee, result *interface{}) error {
	reply, err := req.cli.SetTxFee(&in)
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

func (req Chain33) SetLabl(in types.ReqWalletSetLabel, result *interface{}) error {
	reply, err := req.cli.SetLabl(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req Chain33) MergeBalance(in types.ReqWalletMergeBalance, result *interface{}) error {
	reply, err := req.cli.MergeBalance(&in)
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

func (req Chain33) SetPasswd(in types.ReqWalletSetPasswd, result *interface{}) error {
	reply, err := req.cli.SetPasswd(&in)
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

func (req Chain33) Lock(in types.ReqNil, result *interface{}) error {
	reply, err := req.cli.Lock()
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

func (req Chain33) UnLock(in types.WalletUnLock, result *interface{}) error {
	reply, err := req.cli.UnLock(&in)
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}

func (req Chain33) GetPeerInfo(in types.ReqNil, result *interface{}) error {
	reply, err := req.cli.GetPeerInfo()
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
			}
			peerlist.Peers = append(peerlist.Peers, &pr)
		}
		*result = &peerlist
	}

	return nil
}

func (req Chain33) GetHeaders(in types.ReqBlocks, result *interface{}) error {
	reply, err := req.cli.GetHeaders(&in)
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

func (req Chain33) GetLastMemPool(in types.ReqNil, result *interface{}) error {
	reply, err := req.cli.GetLastMemPool(&in)
	if err != nil {
		return err
	}

	{
		var txlist ReplyTxList
		txs := reply.GetTxs()
		for _, tx := range txs {
			txlist.Txs = append(txlist.Txs,
				&Transaction{
					Execer:  string(tx.GetExecer()),
					Payload: common.ToHex(tx.GetPayload()),
					Fee:     tx.GetFee(),
					Expire:  tx.GetExpire(),
					Nonce:   tx.GetNonce(),
					To:      tx.GetTo(),
					Signature: &Signature{Ty: tx.GetSignature().GetTy(),
						Pubkey:    common.ToHex(tx.GetSignature().GetPubkey()),
						Signature: common.ToHex(tx.GetSignature().GetSignature())}})
		}
		*result = &txlist
	}
	return nil
}

//GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (req Chain33) GetBlockOverview(in QueryParm, result *interface{}) error {
	var data types.ReqHash
	hash, err := common.FromHex(in.Hash)
	if err != nil {
		return err
	}

	data.Hash = hash
	reply, err := req.cli.GetBlockOverview(&data)
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
	blockOverview.Head = &header

	//获取blocktxhashs信息
	for _, has := range reply.GetTxHashes() {
		blockOverview.TxHashes = append(blockOverview.TxHashes, common.ToHex(has))
	}

	blockOverview.TxCount = reply.GetTxCount()
	*result = &blockOverview
	return nil
}
func (req Chain33) GetAddrOverview(in types.ReqAddr, result *interface{}) error {
	reply, err := req.cli.GetAddrOverview(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req Chain33) GetBlockHash(in types.ReqInt, result *interface{}) error {
	reply, err := req.cli.GetBlockHash(&in)
	if err != nil {
		return err
	}
	var replyHash ReplyHash
	replyHash.Hash = common.ToHex(reply.GetHash())
	*result = &replyHash
	return nil
}

//seed
func (req Chain33) GenSeed(in types.GenSeedLang, result *interface{}) error {
	reply, err := req.cli.GenSeed(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}
func (req Chain33) SaveSeed(in types.SaveSeedByPw, result *interface{}) error {
	reply, err := req.cli.SaveSeed(&in)
	if err != nil {
		return err
	}

	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}
func (req Chain33) GetSeed(in types.GetSeedByPw, result *interface{}) error {
	reply, err := req.cli.GetSeed(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req Chain33) GetWalletStatus(in types.ReqNil, result *interface{}) error {
	reply, err := req.cli.GetWalletStatus()
	if err != nil {
		return err
	}
	var resp Reply
	resp.IsOk = reply.GetIsOk()
	resp.Msg = string(reply.GetMsg())
	*result = &resp
	return nil
}
