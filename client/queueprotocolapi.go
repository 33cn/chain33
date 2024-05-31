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
	// types.EventGetAddrTxs
	GetTxListByAddr(param *types.ReqAddrs) (*types.TransactionDetails, error)
	// types.EventTxList
	GetTxList(param *types.TxHashList) (*types.ReplyTxList, error)
	// types.EventGetMempool
	GetMempool(req *types.ReqGetMempool) (*types.ReplyTxList, error)
	// types.EventGetLastMempool
	GetLastMempool() (*types.ReplyTxList, error)
	// types.EventGetProperFee
	GetProperFee(req *types.ReqProperFee) (*types.ReplyProperFee, error)
	//types.EventDelTxList
	RemoveTxsByHashList(hashList *types.TxHashList) error
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
	PeerInfo(param *types.P2PGetPeerReq) (*types.PeerList, error)
	// types.EventGetNetInfo
	GetNetInfo(param *types.P2PGetNetInfoReq) (*types.NodeNetInfo, error)
	//type.EventNetProtocols
	NetProtocols(*types.ReqNil) (*types.NetProtocolInfos, error)
	// --------------- p2p interfaces end
	// +++++++++++++++ wallet interfaces begin
	// types.EventLocalGet
	LocalGet(param *types.LocalDBGet) (*types.LocalReplyValue, error)
	// types.EventLocalNew
	LocalNew(readOnly bool) (*types.Int64, error)
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

	// 在平行链上获得主链Sequence相关的接口
	//types.EventGetLastBlockSequence:
	GetLastBlockMainSequence() (*types.Int64, error)
	//types.EventGetSequenceByHash:
	GetMainSequenceByHash(param *types.ReqHash) (*types.Int64, error)
	GetHighestBlockNum(param *types.ReqNil) (*types.ReplyBlockHeight, error)
	// --------------- blockchain interfaces end

	// +++++++++++++++ store interfaces begin
	StoreSet(param *types.StoreSetWithSync) (*types.ReplyHash, error)
	StoreGet(*types.StoreGet) (*types.StoreReplyValue, error)
	StoreMemSet(param *types.StoreSetWithSync) (*types.ReplyHash, error)
	StoreCommit(param *types.ReqHash) (*types.ReplyHash, error)
	StoreRollback(param *types.ReqHash) (*types.ReplyHash, error)
	StoreDel(param *types.StoreDel) (*types.ReplyHash, error)
	StoreGetTotalCoins(*types.IterateRangeByStateHash) (*types.ReplyGetTotalCoins, error)
	StoreList(param *types.StoreList) (*types.StoreListReply, error)
	// --------------- store interfaces end

	// +++++++++++++++ other interfaces begin
	// close chain33
	CloseQueue() (*types.Reply, error)
	// --------------- other interfaces end
	// types.EventAddBlockSeqCB
	AddPushSubscribe(param *types.PushSubscribeReq) (*types.ReplySubscribePush, error)
	// types.EventListBlockSeqCB
	ListPushes() (*types.PushSubscribes, error)
	// types.EventGetSeqCBLastNum
	GetPushSeqLastNum(param *types.ReqString) (*types.Int64, error)
	// types.EventGetParaTxByTitle
	GetParaTxByTitle(param *types.ReqParaTxByTitle) (*types.ParaTxDetails, error)
	// types.EventGetHeightByTitle
	LoadParaTxByTitle(param *types.ReqHeightByTitle) (*types.ReplyHeightByTitle, error)
	// types.EventGetParaTxByTitleAndHeight
	GetParaTxByHeight(param *types.ReqParaTxByHeight) (*types.ParaTxDetails, error)
	// get chain config
	GetConfig() *types.Chain33Config
	// send delay tx
	SendDelayTx(param *types.DelayTx, waitReply bool) (*types.Reply, error)
	//AddBlacklist add blacklist
	AddBlacklist(req *types.BlackPeer) (*types.Reply, error)
	//DelBlacklist  del blacklist
	DelBlacklist(req *types.BlackPeer) (*types.Reply, error)
	//ShowBlacklist  show blacklist
	ShowBlacklist(req *types.ReqNil) (*types.Blacklist, error)
	//DialPeer dial the specified  peer
	DialPeer(in *types.SetPeer) (*types.Reply, error)
	//ClosePeer close specified peer
	ClosePeer(in *types.SetPeer) (*types.Reply, error)
	//GetFinalizedBlock get finalized block choice
	GetFinalizedBlock() (*types.SnowChoice, error)
}
