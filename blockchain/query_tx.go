// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

//ProcGetTransactionByAddr 获取地址对应的所有交易信息
//存储格式key:addr:flag:height ,value:txhash
//key=addr :获取本地参与的所有交易
//key=addr:1 :获取本地作为from方的所有交易
//key=addr:2 :获取本地作为to方的所有交易
func (chain *BlockChain) ProcGetTransactionByAddr(addr *types.ReqAddr) (*types.ReplyTxInfos, error) {
	if addr == nil || len(addr.Addr) == 0 {
		return nil, types.ErrInvalidParam
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
	txinfos, err := chain.query.Query(types.ExecName("coins"), "GetTxsByAddr", addr)
	if err != nil {
		chainlog.Info("ProcGetTransactionByAddr does not exist tx!", "addr", addr, "err", err)
		return nil, err
	}
	return txinfos.(*types.ReplyTxInfos), nil
}

//ProcGetTransactionByHashes 返回类型
//type TransactionDetails struct {
//	Txs []*Transaction
//}
//通过hashs获取交易详情
func (chain *BlockChain) ProcGetTransactionByHashes(hashs [][]byte) (TxDetails *types.TransactionDetails, err error) {
	//chainlog.Info("ProcGetTransactionByHashes", "txhash len:", len(hashs))
	var txDetails types.TransactionDetails
	for _, txhash := range hashs {
		txresult, err := chain.GetTxResultFromDb(txhash)
		if err == nil && txresult != nil {
			var txDetail types.TransactionDetail
			setTxDetailFromTxResult(&txDetail, txresult)

			//chainlog.Debug("ProcGetTransactionByHashes", "txDetail", txDetail.String())
			txDetails.Txs = append(txDetails.Txs, &txDetail)
		} else {
			txDetails.Txs = append(txDetails.Txs, nil)
			chainlog.Debug("ProcGetTransactionByHashes hash no exit", "txhash", common.ToHex(txhash))
		}
	}
	return &txDetails, nil
}

//GetTransactionProofs 获取指定txindex  在txs中的TransactionDetail ，注释：index从0开始
func GetTransactionProofs(Txs []*types.Transaction, index int32) ([][]byte, error) {
	txlen := len(Txs)

	//计算tx的hash值
	leaves := make([][]byte, txlen)
	for index, tx := range Txs {
		leaves[index] = tx.Hash()
		//chainlog.Info("GetTransactionDetail txhash", "index", index, "txhash", tx.Hash())
	}

	proofs := merkle.GetMerkleBranch(leaves, uint32(index))
	chainlog.Debug("GetTransactionDetail proofs", "proofs", proofs)

	return proofs, nil
}

//GetTxResultFromDb 通过txhash 从txindex db中获取tx信息
//type TxResult struct {
//	Height int64
//	Index  int32
//	Tx     *types.Transaction
//  Receiptdate *ReceiptData
//}
func (chain *BlockChain) GetTxResultFromDb(txhash []byte) (tx *types.TxResult, err error) {
	txinfo, err := chain.blockStore.GetTx(txhash)
	if err != nil {
		return nil, err
	}
	return txinfo, nil
}

//HasTx 是否包含该交易
func (chain *BlockChain) HasTx(txhash []byte, onlyquerycache bool) (has bool, err error) {
	has = chain.cache.HasCacheTx(txhash)
	if has {
		return true, nil
	}
	if onlyquerycache {
		return has, nil
	}
	return chain.blockStore.HasTx(txhash)
}

//GetDuplicateTxHashList 获取重复的交易
func (chain *BlockChain) GetDuplicateTxHashList(txhashlist *types.TxHashList) (duptxhashlist *types.TxHashList, err error) {
	var dupTxHashList types.TxHashList
	onlyquerycache := false
	if txhashlist.Count == -1 {
		onlyquerycache = true
	}
	if txhashlist.Expire != nil && len(txhashlist.Expire) != len(txhashlist.Hashes) {
		return nil, types.ErrInvalidParam
	}
	for i, txhash := range txhashlist.Hashes {
		expire := int64(0)
		if txhashlist.Expire != nil {
			expire = txhashlist.Expire[i]
		}
		txHeight := types.GetTxHeight(expire, txhashlist.Count)
		//在txHeight > 0 的情况下，可以安全的查询cache
		if txHeight > 0 {
			onlyquerycache = true
		}
		has, err := chain.HasTx(txhash, onlyquerycache)
		if err == nil && has {
			dupTxHashList.Hashes = append(dupTxHashList.Hashes, txhash)
		}
	}
	return &dupTxHashList, nil
}

/*ProcQueryTxMsg 函数功能：
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
	var TransactionDetail types.TransactionDetail
	//获取指定tx在txlist中的proof
	proofs, err := GetTransactionProofs(block.Block.Txs, txresult.Index)
	if err != nil {
		return nil, err
	}
	TransactionDetail.Proofs = proofs
	setTxDetailFromTxResult(&TransactionDetail, txresult)
	return &TransactionDetail, nil
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
	addr := txresult.GetTx().From()
	TransactionDetail.Fromaddr = addr
	if TransactionDetail.GetTx().IsWithdraw() {
		//swap from and to
		TransactionDetail.Fromaddr, TransactionDetail.Tx.To = TransactionDetail.Tx.To, TransactionDetail.Fromaddr
	}
}

//ProcGetAddrOverview 获取addrOverview
//type  AddrOverview {
//	int64 reciver = 1;
//	int64 balance = 2;
//	int64 txCount = 3;}
func (chain *BlockChain) ProcGetAddrOverview(addr *types.ReqAddr) (*types.AddrOverview, error) {

	if addr == nil || len(addr.Addr) == 0 {
		chainlog.Error("ProcGetAddrOverview input err!")
		return nil, types.ErrInvalidParam
	}
	chainlog.Debug("ProcGetAddrOverview", "Addr", addr.GetAddr())

	var addrOverview types.AddrOverview

	//获取地址的reciver
	amount, err := chain.query.Query(types.ExecName("coins"), "GetAddrReciver", addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrReciver err", err)
		addrOverview.Reciver = 0
	} else {
		addrOverview.Reciver = amount.(*types.Int64).GetData()
	}
	beg := types.Now()
	curdbver, err := types.G("dbversion")
	if err != nil {
		return nil, err
	}
	var reqkey types.ReqKey
	if curdbver.(int64) == 0 {
		//旧的数据库获取地址对应的交易count，使用前缀查找的方式获取
		//前缀和util.go 文件中的CalcTxAddrHashKey保持一致
		reqkey.Key = []byte(fmt.Sprintf("TxAddrHash:%s:%s", addr.Addr, ""))
		count, err := chain.query.Query(types.ExecName("coins"), "GetPrefixCount", &reqkey)
		if err != nil {
			chainlog.Error("ProcGetAddrOverview", "GetPrefixCount err", err)
			addrOverview.TxCount = 0
		} else {
			addrOverview.TxCount = count.(*types.Int64).GetData()
		}
		chainlog.Debug("GetPrefixCount", "cost ", types.Since(beg))
	} else {
		//新的数据库直接使用key值查找就可以
		//前缀和util.go 文件中的calcAddrTxsCountKey保持一致
		reqkey.Key = []byte(fmt.Sprintf("AddrTxsCount:%s", addr.Addr))
		count, err := chain.query.Query(types.ExecName("coins"), "GetAddrTxsCount", &reqkey)
		if err != nil {
			chainlog.Error("ProcGetAddrOverview", "GetAddrTxsCount err", err)
			addrOverview.TxCount = 0
		} else {
			addrOverview.TxCount = count.(*types.Int64).GetData()
		}
		chainlog.Debug("GetAddrTxsCount", "cost ", types.Since(beg))
	}
	return &addrOverview, nil
}
