// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"fmt"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

// ProcGetTransactionByAddr 获取地址对应的所有交易信息
// 存储格式key:addr:flag:height ,value:txhash
// key=addr :获取本地参与的所有交易
// key=addr:1 :获取本地作为from方的所有交易
// key=addr:2 :获取本地作为to方的所有交易
func (chain *BlockChain) ProcGetTransactionByAddr(addr *types.ReqAddr) (*types.ReplyTxInfos, error) {
	if addr == nil || len(addr.Addr) == 0 {
		return nil, types.ErrInvalidParam
	}
	//默认取10笔交易数据
	if addr.Count == 0 {
		addr.Count = 10
	}

	if int64(addr.Count) > types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	//入参数校验
	curheigt := chain.GetBlockHeight()
	if addr.GetHeight() > curheigt || addr.GetHeight() < -1 {
		chainlog.Error("ProcGetTransactionByAddr Height err")
		return nil, types.ErrInvalidParam
	}
	if addr.GetDirection() != 0 && addr.GetDirection() != 1 {
		chainlog.Error("ProcGetTransactionByAddr Direction err")
		return nil, types.ErrInvalidParam
	}
	if addr.GetIndex() < 0 || addr.GetIndex() > types.MaxTxsPerBlock {
		chainlog.Error("ProcGetTransactionByAddr Index err")
		return nil, types.ErrInvalidParam
	}
	//查询的drivers--> main 驱动的名称
	//查询的方法：  --> GetTxsByAddr
	//查询的参数：  --> interface{} 类型
	cfg := chain.client.GetConfig()
	txinfos, err := chain.query.Query(cfg.ExecName(cfg.GetCoinExec()), "GetTxsByAddr", addr)
	if err != nil {
		chainlog.Info("ProcGetTransactionByAddr does not exist tx!", "addr", addr, "err", err)
		return nil, err
	}
	return txinfos.(*types.ReplyTxInfos), nil
}

// ProcGetTransactionByHashes 返回类型
//
//	type TransactionDetails struct {
//		Txs []*Transaction
//	}
//
// 通过hashs获取交易详情
func (chain *BlockChain) ProcGetTransactionByHashes(hashs [][]byte) (TxDetails *types.TransactionDetails, err error) {
	if int64(len(hashs)) > types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	var txDetails types.TransactionDetails
	for _, txhash := range hashs {
		txresult, err := chain.GetTxResultFromDb(txhash)
		var txDetail types.TransactionDetail
		if err == nil && txresult != nil {
			setTxDetailFromTxResult(&txDetail, txresult)

			//chainlog.Debug("ProcGetTransactionByHashes", "txDetail", txDetail.String())
			txDetails.Txs = append(txDetails.Txs, &txDetail)
		} else {
			txDetails.Txs = append(txDetails.Txs, &txDetail) //
			chainlog.Debug("ProcGetTransactionByHashes hash no exit", "txhash", common.ToHex(txhash))
		}
	}
	return &txDetails, nil
}

// getTxHashProofs 获取指定txindex在txs中的proof ，注释：index从0开始
func getTxHashProofs(Txs []*types.Transaction, index int32) [][]byte {
	txlen := len(Txs)
	leaves := make([][]byte, txlen)

	for index, tx := range Txs {
		leaves[index] = tx.Hash()
	}

	proofs := merkle.GetMerkleBranch(leaves, uint32(index))
	chainlog.Debug("getTransactionDetail", "index", index, "proofs", proofs)

	return proofs
}

// GetTxResultFromDb 通过txhash 从txindex db中获取tx信息
//
//	type TxResult struct {
//		Height int64
//		Index  int32
//		Tx     *types.Transaction
//	 Receiptdate *ReceiptData
//	}
func (chain *BlockChain) GetTxResultFromDb(txhash []byte) (tx *types.TxResult, err error) {
	txinfo, err := chain.blockStore.GetTx(txhash)
	if err != nil {
		return nil, err
	}
	return txinfo, nil
}

// HasTx 是否包含该交易
func (chain *BlockChain) HasTx(txhash []byte, txHeight int64) (has bool, err error) {

	if txHeight > 0 {
		return chain.txHeightCache.Contains(txhash, txHeight), nil
	}
	return chain.blockStore.HasTx(txhash)
}

// GetDuplicateTxHashList 获取重复的交易
func (chain *BlockChain) GetDuplicateTxHashList(txhashlist *types.TxHashList) (duptxhashlist *types.TxHashList, err error) {
	var dupTxHashList types.TxHashList
	//onlyquerycache := false
	//FIXME:这里Count实际传的是区块高度的值，代码中没找到-1的传参，不清楚具体含义
	//对于非txHeight类交易，直接查询数据库，不会有影响，暂时先注释
	//if txhashlist.Count == -1 {
	//	onlyquerycache = true
	//}
	if txhashlist.Expire != nil && len(txhashlist.Expire) != len(txhashlist.Hashes) {
		return nil, types.ErrInvalidParam
	}
	cfg := chain.client.GetConfig()
	for i, txhash := range txhashlist.Hashes {
		expire := int64(0)
		if txhashlist.Expire != nil {
			expire = txhashlist.Expire[i]
		}
		txHeight := types.GetTxHeight(cfg, expire, txhashlist.Count)
		has, err := chain.HasTx(txhash, txHeight)
		if err == nil && has {
			dupTxHashList.Hashes = append(dupTxHashList.Hashes, txhash)
		}
	}
	return &dupTxHashList, nil
}

/*
ProcQueryTxMsg 函数功能：
EventQueryTx(types.ReqHash) : rpc模块会向 blockchain 模块 发送 EventQueryTx(types.ReqHash) 消息 ，
查询交易的默克尔树，回复消息 EventTransactionDetail(types.TransactionDetail)
结构体：
type ReqHash struct {Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`}
type TransactionDetail struct {Hashs [][]byte `protobuf:"bytes,1,rep,name=hashs,proto3" json:"hashs,omitempty"}
*/
func (chain *BlockChain) ProcQueryTxMsg(txhash []byte) (proof *types.TransactionDetail, err error) {
	txresult, err := chain.GetTxResultFromDb(txhash)
	if err != nil {
		return nil, err
	}
	block, err := chain.GetBlock(txresult.Height)
	if err != nil {
		return nil, err
	}

	var txDetail types.TransactionDetail
	cfg := chain.client.GetConfig()
	height := block.Block.GetHeight()
	if chain.isParaChain {
		height = block.Block.GetMainHeight()
	}
	//获取指定tx在txlist中的proof,需要区分ForkRootHash前后proof证明数据
	if !cfg.IsFork(height, "ForkRootHash") {
		proofs := getTxHashProofs(block.Block.Txs, txresult.Index)
		txDetail.Proofs = proofs
	} else {
		txproofs := chain.getMultiLayerProofs(txresult.Height, block.Block.Hash(cfg), block.Block.Txs, txresult.Index)
		txDetail.TxProofs = txproofs
		txDetail.FullHash = block.Block.Txs[txresult.Index].FullHash()
	}

	setTxDetailFromTxResult(&txDetail, txresult)
	return &txDetail, nil
}

func setTxDetailFromTxResult(TransactionDetail *types.TransactionDetail, txresult *types.TxResult) {
	TransactionDetail.Receipt = txresult.Receiptdate
	TransactionDetail.Tx = txresult.GetTx()
	TransactionDetail.Height = txresult.GetHeight()
	TransactionDetail.Index = int64(txresult.GetIndex())
	TransactionDetail.Blocktime = txresult.GetBlocktime()

	//获取Amount
	amount, err := txresult.GetTx().Amount()
	if err != nil {
		// return nil, err
		amount = 0
	}
	TransactionDetail.Amount = amount
	assets, err := txresult.GetTx().Assets()
	if err != nil {
		assets = nil
	}
	TransactionDetail.Assets = assets
	TransactionDetail.ActionName = txresult.GetTx().ActionName()

	//获取from地址
	TransactionDetail.Fromaddr = txresult.GetTx().From()
}

// ProcGetAddrOverview 获取addrOverview
//
//	type  AddrOverview {
//		int64 reciver = 1;
//		int64 balance = 2;
//		int64 txCount = 3;}
func (chain *BlockChain) ProcGetAddrOverview(addr *types.ReqAddr) (*types.AddrOverview, error) {
	cfg := chain.client.GetConfig()
	if addr == nil || len(addr.Addr) == 0 {
		chainlog.Error("ProcGetAddrOverview input err!")
		return nil, types.ErrInvalidParam
	}
	chainlog.Debug("ProcGetAddrOverview", "Addr", addr.GetAddr())

	var addrOverview types.AddrOverview

	//获取地址的reciver
	amount, err := chain.query.Query(cfg.ExecName(cfg.GetCoinExec()), "GetAddrReciver", addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrReciver err", err)
		addrOverview.Reciver = 0
	} else {
		addrOverview.Reciver = amount.(*types.Int64).GetData()
	}
	if err != nil && err != types.ErrEmpty {
		return nil, err
	}

	var reqkey types.ReqKey

	//新的代码不支持PrefixCount查询地址交易计数，executor/localdb.go PrefixCount
	//现有的节点都已经升级了localdb，也就是都支持通过GetAddrTxsCount来获取地址交易计数
	reqkey.Key = []byte(fmt.Sprintf("AddrTxsCount:%s", addr.Addr))
	count, err := chain.query.Query(cfg.ExecName(cfg.GetCoinExec()), "GetAddrTxsCount", &reqkey)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrTxsCount err", err)
		addrOverview.TxCount = 0
	} else {
		addrOverview.TxCount = count.(*types.Int64).GetData()
	}
	chainlog.Debug("GetAddrTxsCount", "TxCount ", addrOverview.TxCount)

	return &addrOverview, nil
}

// getTxFullHashProofs 获取指定txinde在txs中的proof, ForkRootHash之后使用fullhash来计算
func getTxFullHashProofs(Txs []*types.Transaction, index int32) [][]byte {
	txlen := len(Txs)
	leaves := make([][]byte, txlen)

	for index, tx := range Txs {
		leaves[index] = tx.FullHash()
	}

	proofs := merkle.GetMerkleBranch(leaves, uint32(index))

	return proofs
}

// 获取指定交易在子链中的路径证明,使用fullhash
func (chain *BlockChain) getMultiLayerProofs(height int64, blockHash []byte, Txs []*types.Transaction, index int32) []*types.TxProof {
	var proofs []*types.TxProof

	//平行链节点上的交易都是单层merkle树，使用FullHash直接计算即可
	if chain.isParaChain {
		txProofs := getTxFullHashProofs(Txs, index)
		proof := types.TxProof{Proofs: txProofs, Index: uint32(index)}
		proofs = append(proofs, &proof)
		return proofs
	}

	//主链节点的处理
	title, haveParaTx := types.GetParaExecTitleName(string(Txs[index].Execer))
	if !haveParaTx {
		title = types.MainChainName
	}

	replyparaTxs, err := chain.LoadParaTxByHeight(height, "", 0, 1)
	if err != nil || 0 == len(replyparaTxs.Items) {
		filterlog.Error("getMultiLayerProofs", "height", height, "blockHash", common.ToHex(blockHash), "err", err)
		return nil
	}

	//获取本子链的第一笔交易的index值，以及最后一笔交易的index值
	var startIndex int32
	var childIndex uint32
	var hashes [][]byte
	var exist bool
	var childHash []byte
	var txCount int32

	for _, paratx := range replyparaTxs.Items {
		if title == paratx.Title && bytes.Equal(blockHash, paratx.GetHash()) {
			startIndex = paratx.GetStartIndex()
			childIndex = paratx.ChildHashIndex
			childHash = paratx.ChildHash
			txCount = paratx.GetTxCount()
			exist = true
		}
		hashes = append(hashes, paratx.ChildHash)
	}
	//只有主链的交易,没有其他平行链的交易
	if len(replyparaTxs.Items) == 1 && startIndex == 0 && childIndex == 0 {
		txProofs := getTxFullHashProofs(Txs, index)
		proof := types.TxProof{Proofs: txProofs, Index: uint32(index)}
		proofs = append(proofs, &proof)
		return proofs
	}

	endindex := startIndex + txCount
	//不存在或者获取到的索引错误直接返回
	if !exist || startIndex > index || endindex > int32(len(Txs)) || childIndex >= uint32(len(replyparaTxs.Items)) {
		filterlog.Error("getMultiLayerProofs", "height", height, "blockHash", common.ToHex(blockHash), "exist", exist, "replyparaTxs", replyparaTxs)
		filterlog.Error("getMultiLayerProofs", "startIndex", startIndex, "index", index, "childIndex", childIndex)
		return nil
	}

	//主链和平行链交易都存在,获取本子链所有交易的hash值
	var childHashes [][]byte
	for i := startIndex; i < endindex; i++ {
		childHashes = append(childHashes, Txs[i].FullHash())
	}
	txOnChildIndex := index - startIndex

	//计算单笔交易在子链中的路径证明
	txOnChildBranch := merkle.GetMerkleBranch(childHashes, uint32(txOnChildIndex))
	txproof := types.TxProof{Proofs: txOnChildBranch, Index: uint32(txOnChildIndex), RootHash: childHash}
	proofs = append(proofs, &txproof)

	//子链在整个区块中的路径证明
	childBranch := merkle.GetMerkleBranch(hashes, childIndex)
	childproof := types.TxProof{Proofs: childBranch, Index: childIndex}
	proofs = append(proofs, &childproof)
	return proofs
}
