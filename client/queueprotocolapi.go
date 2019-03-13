// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// QueueProtocolAPI 消息通道交互API接口定义
type QueueProtocolAPI interface {
	Version() (*types.VersionInfo, error)
	Close()
	NewMessage(topic string, msgid int64, data interface{}) *queue.Message
	Notify(topic string, ty int64, data interface{}) (*queue.Message, error)
	// +++++++++++++++ mempool interfaces begin
	// 同步发送交易信息到指定模块，获取应答消息 types.EventTx
	SendTx(param *types.Transaction) (*types.Reply, error)
	// types.EventTxList
	GetTxList(param *types.TxHashList) (*types.ReplyTxList, error)
	// types.EventGetMempool
	GetMempool() (*types.ReplyTxList, error)
	// types.EventGetLastMempool
	GetLastMempool() (*types.ReplyTxList, error)
	// types.EventGetProperFee
	GetProperFee() (*types.ReplyProperFee, error)
	// +++++++++++++++ execs interfaces begin
	// types.EventBlockChainQuery
	Query(driver, funcname string, param types.Message) (types.Message, error)
	QueryConsensus(param *types.ChainExecutor) (types.Message, error)
	QueryConsensusFunc(driver string, funcname string, param types.Message) (types.Message, error)
	QueryChain(param *types.ChainExecutor) (types.Message, error)
	ExecWalletFunc(driver string, funcname string, param types.Message) (types.Message, error)
	ExecWallet(param *types.ChainExecutor) (types.Message, error)
	// --------------- execs interfaces end

	// +++++++++++++++ p2p interfaces begin
	// types.EventPeerInfo
	PeerInfo() (*types.PeerList, error)
	// types.EventGetNetInfo
	GetNetInfo() (*types.NodeNetInfo, error)
	// --------------- p2p interfaces end
	// +++++++++++++++ wallet interfaces begin
	// types.EventLocalGet
	LocalGet(param *types.LocalDBGet) (*types.LocalReplyValue, error)
	// types.EventLocalNew
	LocalNew(param *types.ReqNil) (*types.Int64, error)
	// types.EventLocalClose
	LocalClose(param *types.Int64) error
	// types.EventLocalBeign
	LocalBegin(param *types.Int64) error
	// types.EventLocalCommit
	LocalCommit(param *types.Int64) error
	// types.EventLocalRollback
	LocalRollback(param *types.Int64) error
	// types.EventLocalSet
	LocalSet(param *types.LocalDBSet) error
	// types.EventLocalList
	LocalList(param *types.LocalDBList) (*types.LocalReplyValue, error)
	// types.EventWalletGetAccountList
	WalletGetAccountList(req *types.ReqAccountList) (*types.WalletAccounts, error)
	// types.EventNewAccount
	NewAccount(param *types.ReqNewAccount) (*types.WalletAccount, error)
	// types.EventWalletTransactionList
	WalletTransactionList(param *types.ReqWalletTransactionList) (*types.WalletTxDetails, error)
	// types.EventWalletImportprivkey
	WalletImportprivkey(param *types.ReqWalletImportPrivkey) (*types.WalletAccount, error)
	// types.EventWalletSendToAddress
	WalletSendToAddress(param *types.ReqWalletSendToAddress) (*types.ReplyHash, error)
	// types.EventWalletSetFee
	WalletSetFee(param *types.ReqWalletSetFee) (*types.Reply, error)
	// types.EventWalletSetLabel
	WalletSetLabel(param *types.ReqWalletSetLabel) (*types.WalletAccount, error)
	// types.EventWalletMergeBalance
	WalletMergeBalance(param *types.ReqWalletMergeBalance) (*types.ReplyHashes, error)
	// types.EventWalletSetPasswd
	WalletSetPasswd(param *types.ReqWalletSetPasswd) (*types.Reply, error)
	// types.EventWalletLock
	WalletLock() (*types.Reply, error)
	// types.EventWalletUnLock
	WalletUnLock(param *types.WalletUnLock) (*types.Reply, error)
	// types.EventGenSeed
	GenSeed(param *types.GenSeedLang) (*types.ReplySeed, error)
	// types.EventSaveSeed
	SaveSeed(param *types.SaveSeedByPw) (*types.Reply, error)
	// types.EventGetSeed
	GetSeed(param *types.GetSeedByPw) (*types.ReplySeed, error)
	// types.EventGetWalletStatus
	GetWalletStatus() (*types.WalletStatus, error)
	// types.EventDumpPrivkey
	DumpPrivkey(param *types.ReqString) (*types.ReplyString, error)
	// types.EventSignRawTx
	SignRawTx(param *types.ReqSignRawTx) (*types.ReplySignRawTx, error)
	GetFatalFailure() (*types.Int32, error)
	// types.EventCreateTransaction 由服务器协助创建一个交易
	WalletCreateTx(param *types.ReqCreateTransaction) (*types.Transaction, error)
	// types.EventGetBlocks
	GetBlocks(param *types.ReqBlocks) (*types.BlockDetails, error)
	// types.EventQueryTx
	QueryTx(param *types.ReqHash) (*types.TransactionDetail, error)
	// types.EventGetTransactionByAddr
	GetTransactionByAddr(param *types.ReqAddr) (*types.ReplyTxInfos, error)
	// types.EventGetTransactionByHash
	GetTransactionByHash(param *types.ReqHashes) (*types.TransactionDetails, error)
	// types.EventGetHeaders
	GetHeaders(param *types.ReqBlocks) (*types.Headers, error)
	// types.EventGetBlockOverview
	GetBlockOverview(param *types.ReqHash) (*types.BlockOverview, error)
	// types.EventGetAddrOverview
	GetAddrOverview(param *types.ReqAddr) (*types.AddrOverview, error)
	// types.EventGetBlockHash
	GetBlockHash(param *types.ReqInt) (*types.ReplyHash, error)
	// types.EventIsSync
	IsSync() (*types.Reply, error)
	// types.EventIsNtpClockSync
	IsNtpClockSync() (*types.Reply, error)
	// types.EventGetLastHeader
	GetLastHeader() (*types.Header, error)

	//types.EventGetLastBlockSequence:
	GetLastBlockSequence() (*types.Int64, error)
	//types.EventGetBlockSequences:
	GetBlockSequences(param *types.ReqBlocks) (*types.BlockSequences, error)
	//types.EventGetBlockByHashes:
	GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error)
	//types.EventGetBlockBySeq:
	GetBlockBySeq(param *types.Int64) (*types.BlockSeq, error)
	//types.EventGetSequenceByHash:
	GetSequenceByHash(param *types.ReqHash) (*types.Int64, error)

	// --------------- blockchain interfaces end

	// +++++++++++++++ store interfaces begin
	StoreGet(*types.StoreGet) (*types.StoreReplyValue, error)
	StoreGetTotalCoins(*types.IterateRangeByStateHash) (*types.ReplyGetTotalCoins, error)
	StoreList(param *types.StoreList) (*types.StoreListReply, error)
	// --------------- store interfaces end

	// +++++++++++++++ other interfaces begin
	// close chain33
	CloseQueue() (*types.Reply, error)
	// --------------- other interfaces end
	// types.EventAddBlockSeqCB
	AddSeqCallBack(param *types.BlockSeqCB) (*types.Reply, error)

	// types.EventListBlockSeqCB
	ListSeqCallBack() (*types.BlockSeqCBs, error)
	// types.EventGetSeqCBLastNum
	GetSeqCallBackLastNum(param *types.ReqString) (*types.Int64, error)
}
