package rpc

import (
	"fmt"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/types"
)

type Chain33 struct {
	jserver *jsonrpcServer
	cli     IRClient
}

func (req Chain33) SendTransaction(in RawParm, result *interface{}) error {
	fmt.Println("jrpc transaction:", in)
	var parm types.Transaction
	types.Decode(common.FromHex(in.Data), &parm)
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

	data.Hash = common.FromHex(in.Hash)
	reply, err := req.cli.QueryTx(data.Hash)
	if err != nil {
		return err
	}

	{ //重新格式化数据

		var transDetail TransactionDetail
		transDetail.Tx = &Transaction{
			Execer:  common.ToHex(reply.Tx.Execer),
			Payload: common.ToHex(reply.Tx.Payload),
			Fee:     reply.Tx.Fee,
			Expire:  reply.Tx.Expire,
			Nonce:   reply.Tx.Nonce,
			To:      reply.Tx.To,
			Signature: &Signature{Ty: reply.Tx.Signature.Ty,
				Pubkey:    common.ToHex(reply.Tx.Signature.Pubkey),
				Signature: common.ToHex(reply.Tx.Signature.Signature)}}

		transDetail.Receipt = &ReceiptData{Ty: reply.Receipt.Ty}
		for _, log := range reply.Receipt.Logs {
			transDetail.Receipt.Logs = append(transDetail.Receipt.Logs,
				&ReceiptLog{Ty: log.Ty, Log: common.ToHex(log.GetLog())})
		}

		for _, proof := range reply.Proofs {
			transDetail.Proofs = append(transDetail.Proofs, common.ToHex(proof))
		}

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
		for _, item := range reply.Items {
			var bdtl BlockDetail
			var block Block
			block.BlockTime = item.Block.GetBlockTime()
			block.Height = item.Block.GetHeight()
			block.Version = item.Block.GetVersion()
			block.ParentHash = common.ToHex(item.Block.GetParentHash())
			block.StateHash = common.ToHex(item.Block.GetStateHash())
			block.TxHash = common.ToHex(item.Block.GetTxHash())
			for _, tx := range item.Block.Txs {
				block.Txs = append(block.Txs,
					&Transaction{
						Execer:  common.ToHex(tx.GetExecer()),
						Payload: common.ToHex(tx.GetPayload()),
						Fee:     tx.Fee,
						Expire:  tx.Expire,
						Nonce:   tx.Nonce,
						To:      tx.To,
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
		*result = &header
	}

	return nil
}

//GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfo, error)
func (req Chain33) GetTxByAddr(in ReqAddr, result *interface{}) error {

	var parm types.ReqAddr
	parm.Addr = in.Addr
	reply, err := req.cli.GetTxByAddr(&parm)
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

	var parm types.ReqHashes
	parm.Hashes = make([][]byte, 0)
	for _, v := range in.Hashes {
		hb := common.FromHex(v)
		parm.Hashes = append(parm.Hashes, hb)

	}
	reply, err := req.cli.GetTxByHashes(&parm)
	if err != nil {
		return err
	}
	{
		var txdetails TransactionDetails
		for _, tx := range reply.Txs {

			txdetails.Txs = append(txdetails.Txs,
				&Transaction{
					Execer:  common.ToHex(tx.Tx.GetExecer()),
					Payload: common.ToHex(tx.Tx.GetPayload()),
					Fee:     tx.Tx.Fee,
					Expire:  tx.Tx.Expire,
					Nonce:   tx.Tx.Nonce,
					To:      tx.Tx.To,
					Signature: &Signature{Ty: tx.Tx.GetSignature().GetTy(),
						Pubkey:    common.ToHex(tx.Tx.GetSignature().GetPubkey()),
						Signature: common.ToHex(tx.Tx.GetSignature().GetSignature())}})

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
		for _, tx := range reply.Txs {
			txlist.Txs = append(txlist.Txs,
				&Transaction{
					Execer:  common.ToHex(tx.GetExecer()),
					Payload: common.ToHex(tx.GetPayload()),
					Fee:     tx.Fee,
					Expire:  tx.Expire,
					Nonce:   tx.Nonce,
					To:      tx.To,
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
	parm.FromTx = common.FromHex(in.FromTx)
	parm.Count = in.Count
	parm.Direction = in.Direction
	reply, err := req.cli.WalletTxList(&parm)
	if err != nil {
		return err
	}
	{
		var txdetails TransactionDetails
		for _, tx := range reply.Txs {
			txdetails.Txs = append(txdetails.Txs,
				&Transaction{
					Execer:  common.ToHex(tx.Tx.GetExecer()),
					Payload: common.ToHex(tx.Tx.GetPayload()),
					Fee:     tx.Tx.Fee,
					Expire:  tx.Tx.Expire,
					Nonce:   tx.Tx.Nonce,
					To:      tx.Tx.To,
					Signature: &Signature{Ty: tx.Tx.GetSignature().GetTy(),
						Pubkey:    common.ToHex(tx.Tx.GetSignature().GetPubkey()),
						Signature: common.ToHex(tx.Tx.GetSignature().GetSignature())}})

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
