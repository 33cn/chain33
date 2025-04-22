// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package client 系统接口客户端: 封装 Queue Event
package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/33cn/chain33/rpc/grpcclient"

	"github.com/33cn/chain33/common"

	"github.com/33cn/chain33/common/log/log15"

	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

const (
	mempoolKey = "mempool" // 未打包交易池
	p2pKey     = "p2p"     //
	//rpcKey			= "rpc"
	consensusKey = "consensus" // 共识系统
	//accountKey		= "accout"		// 账号系统
	executorKey   = "execs"      // 交易执行器
	walletKey     = "wallet"     // 钱包
	blockchainKey = "blockchain" // 区块
	storeKey      = "store"
)

var log = log15.New("module", "client")

// QueueProtocolOption queue protocol option
type QueueProtocolOption struct {
	// 发送请求超时时间
	SendTimeout time.Duration
	// 接收应答超时时间
	WaitTimeout time.Duration
}

// QueueProtocol 消息通道协议实现
type QueueProtocol struct {
	// 消息队列
	client queue.Client
	option QueueProtocolOption
}

// New New QueueProtocolAPI interface
func New(client queue.Client, option *QueueProtocolOption) (QueueProtocolAPI, error) {
	if client == nil {
		return nil, types.ErrInvalidParam
	}
	q := &QueueProtocol{}
	q.client = client
	if option != nil {
		q.option = *option
	} else {
		q.option.SendTimeout = time.Duration(-1)
		q.option.WaitTimeout = time.Duration(-1)
	}

	return q, nil
}

func (q *QueueProtocol) send(topic string, ty int64, data interface{}) (*queue.Message, error) {
	client := q.client
	msg := client.NewMessage(topic, ty, data)
	err := client.SendTimeout(msg, true, q.option.SendTimeout)
	if err != nil {
		return &queue.Message{}, err
	}
	reply, err := client.WaitTimeout(msg, q.option.WaitTimeout)
	if err != nil {
		return nil, err
	}
	//NOTE：内部错误情况较多，只对正确流程msg回收
	client.FreeMessage(msg)
	return reply, nil
}

func (q *QueueProtocol) notify(topic string, ty int64, data interface{}) (*queue.Message, error) {
	client := q.client
	msg := client.NewMessage(topic, ty, data)
	err := client.SendTimeout(msg, false, q.option.SendTimeout)
	if err != nil {
		return &queue.Message{}, err
	}
	return msg, err
}

// Notify new and send client message
func (q *QueueProtocol) Notify(topic string, ty int64, data interface{}) (*queue.Message, error) {
	return q.notify(topic, ty, data)
}

// Close close client
func (q *QueueProtocol) Close() {
	q.client.Close()
}

// NewMessage new message
func (q *QueueProtocol) NewMessage(topic string, msgid int64, data interface{}) *queue.Message {
	return q.client.NewMessage(topic, msgid, data)
}

func (q *QueueProtocol) setOption(option *QueueProtocolOption) {
	if option != nil {
		q.option = *option
	}
}

// Send2Mempool send transaction to mempool
func (q *QueueProtocol) Send2Mempool(param *types.Transaction) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("SendTx", "Error", err)
		return nil, err
	}
	msg, err := q.send(mempoolKey, types.EventTx, param)
	if err != nil {
		return nil, err
	}
	reply, ok := msg.GetData().(*types.Reply)
	if ok {
		if reply.GetIsOk() {
			reply.Msg = param.Hash()
		} else {
			msg := string(reply.Msg)
			err = fmt.Errorf(msg)
			reply = nil
		}
	} else {
		err = types.ErrTypeAsset
	}
	q.client.FreeMessage(msg)
	return reply, err
}

func (q *QueueProtocol) send2MainChain(cfg *types.Chain33Config, tx *types.Transaction) (*types.Reply, error) {

	mainGrpc := grpcclient.GetDefaultMainClient()
	if mainGrpc == nil {
		var err error
		mainGrpc, err = grpcclient.NewMainChainClient(cfg, "")
		if err != nil {
			return nil, err
		}
	}
	return mainGrpc.SendTransaction(context.TODO(), tx)
}

// SendTx send transaction to mempool with forward logic in parachain
func (q *QueueProtocol) SendTx(tx *types.Transaction) (*types.Reply, error) {

	cfg := q.GetConfig()
	if types.IsForward2MainChainTx(cfg, tx) {
		return q.send2MainChain(cfg, tx)
	}
	return q.Send2Mempool(tx)
}

// GetTxList get transactions from mempool
func (q *QueueProtocol) GetTxList(param *types.TxHashList) (*types.ReplyTxList, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetTxList", "Error", err)
		return nil, err
	}
	msg, err := q.send(mempoolKey, types.EventTxList, param)
	if err != nil {
		log.Error("GetTxList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetTxListByAddr get transactions by addr from mempool
func (q *QueueProtocol) GetTxListByAddr(param *types.ReqAddrs) (*types.TransactionDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetTxList", "Error", err)
		return nil, err
	}
	msg, err := q.send(mempoolKey, types.EventGetAddrTxs, param)
	if err != nil {
		log.Error("GetTxList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.TransactionDetails); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// RemoveTxsByHashList remove txs by tx  hash list
func (q *QueueProtocol) RemoveTxsByHashList(hashList *types.TxHashList) error {
	if hashList == nil {
		err := types.ErrInvalidParam
		log.Error("RemoveTxsByHashList", "Error", err)
		return err
	}
	msg, err := q.send(mempoolKey, types.EventDelTxList, hashList)
	if err != nil {
		log.Error("RemoveTxsByHashList", "Error", err.Error())
		return err
	}
	var ok bool
	err, ok = msg.GetData().(error)
	if !ok {
		return err
	}
	return nil
}

// GetBlocks get block detail from blockchain
func (q *QueueProtocol) GetBlocks(param *types.ReqBlocks) (*types.BlockDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBlocks", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetBlocks, param)
	if err != nil {
		log.Error("GetBlocks", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.BlockDetails); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("GetBlocks", "Error", err.Error())
	return nil, err
}

// QueryTx query transaction detail by transaction hash from blockchain
func (q *QueueProtocol) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Debug("QueryTx", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventQueryTx, param)
	if err != nil {
		log.Debug("QueryTx", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.TransactionDetail); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetTransactionByAddr get transaction by address
func (q *QueueProtocol) GetTransactionByAddr(param *types.ReqAddr) (*types.ReplyTxInfos, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetTransactionByAddr", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetTransactionByAddr, param)
	if err != nil {
		log.Error("GetTransactionByAddr", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxInfos); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("GetTransactionByAddr", "Error", err)
	return nil, types.ErrTypeAsset
}

// GetTransactionByHash get transactions by hash from blockchain
func (q *QueueProtocol) GetTransactionByHash(param *types.ReqHashes) (*types.TransactionDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetTransactionByHash", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetTransactionByHash, param)
	if err != nil {
		log.Error("GetTransactionByHash", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.TransactionDetails); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetMempool get transactions from mempool
func (q *QueueProtocol) GetMempool(req *types.ReqGetMempool) (*types.ReplyTxList, error) {
	msg, err := q.send(mempoolKey, types.EventGetMempool, req)
	if err != nil {
		log.Error("GetMempool", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// PeerInfo query peer list
func (q *QueueProtocol) PeerInfo(req *types.P2PGetPeerReq) (*types.PeerList, error) {
	msg, err := q.send(p2pKey, types.EventPeerInfo, req)
	if err != nil {
		log.Error("PeerInfo", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.PeerList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// NetProtocols protocols list
func (q *QueueProtocol) NetProtocols(req *types.ReqNil) (*types.NetProtocolInfos, error) {
	msg, err := q.send(p2pKey, types.EventNetProtocols, req)
	if err != nil {
		log.Error("PeerInfo", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.NetProtocolInfos); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetHeaders get block headers by height
func (q *QueueProtocol) GetHeaders(param *types.ReqBlocks) (*types.Headers, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetHeaders", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetHeaders, param)
	if err != nil {
		log.Error("GetHeaders", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Headers); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetLastMempool get transactions from last mempool
func (q *QueueProtocol) GetLastMempool() (*types.ReplyTxList, error) {
	msg, err := q.send(mempoolKey, types.EventGetLastMempool, &types.ReqNil{})
	if err != nil {
		log.Error("GetLastMempool", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetProperFee get proper fee from mempool
func (q *QueueProtocol) GetProperFee(req *types.ReqProperFee) (*types.ReplyProperFee, error) {
	msg, err := q.send(mempoolKey, types.EventGetProperFee, req)
	if err != nil {
		log.Error("GetProperFee", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyProperFee); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetBlockOverview get block head detil by hash
func (q *QueueProtocol) GetBlockOverview(param *types.ReqHash) (*types.BlockOverview, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBlockOverview", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetBlockOverview, param)
	if err != nil {
		log.Error("GetBlockOverview", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.BlockOverview); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetAddrOverview get block head detil by address
func (q *QueueProtocol) GetAddrOverview(param *types.ReqAddr) (*types.AddrOverview, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetAddrOverview", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetAddrOverview, param)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.AddrOverview); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetBlockHash get blockHash by height
func (q *QueueProtocol) GetBlockHash(param *types.ReqInt) (*types.ReplyHash, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBlockHash", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetBlockHash, param)
	if err != nil {
		log.Error("GetBlockHash", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// Query the query interface
func (q *QueueProtocol) Query(driver, funcname string, param types.Message) (types.Message, error) {
	if types.IsNilP(param) {
		err := types.ErrInvalidParam
		log.Error("Query", "Error", err)
		return nil, err
	}
	query := &types.ChainExecutor{Driver: driver, FuncName: funcname, Param: types.Encode(param)}
	return q.QueryChain(query)
}

// QueryConsensus query consensus data
func (q *QueueProtocol) QueryConsensus(param *types.ChainExecutor) (types.Message, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("ExecWallet", "Error", err)
		return nil, err
	}
	msg, err := q.send(consensusKey, types.EventConsensusQuery, param)
	if err != nil {
		log.Error("query QueryConsensus", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(types.Message); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// ExecWalletFunc exec wallet function interface
func (q *QueueProtocol) ExecWalletFunc(driver string, funcname string, param types.Message) (types.Message, error) {
	if types.IsNilP(param) {
		err := types.ErrInvalidParam
		log.Error("ExecWalletFunc", "Error", err)
		return nil, err
	}
	query := &types.ChainExecutor{Driver: driver, FuncName: funcname, Param: types.Encode(param)}
	return q.ExecWallet(query)
}

// QueryConsensusFunc query consensus function
func (q *QueueProtocol) QueryConsensusFunc(driver string, funcname string, param types.Message) (types.Message, error) {
	if types.IsNilP(param) {
		err := types.ErrInvalidParam
		log.Error("QueryConsensusFunc", "Error", err)
		return nil, err
	}
	query := &types.ChainExecutor{Driver: driver, FuncName: funcname, Param: types.Encode(param)}
	return q.QueryConsensus(query)
}

// ExecWallet exec wallet function
func (q *QueueProtocol) ExecWallet(param *types.ChainExecutor) (types.Message, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("ExecWallet", "Error", err)
		return nil, err
	}
	msg, err := q.send(walletKey, types.EventWalletExecutor, param)
	if err != nil {
		log.Error("ExecWallet", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(types.Message); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// IsSync query the blockchain sync state
func (q *QueueProtocol) IsSync() (*types.Reply, error) {
	msg, err := q.send(blockchainKey, types.EventIsSync, &types.ReqNil{})
	if err != nil {
		log.Error("IsSync", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.IsCaughtUp); ok {
		return &types.Reply{IsOk: reply.Iscaughtup}, nil
	}
	err = types.ErrTypeAsset
	log.Error("IsSync", "Error", err.Error())
	return nil, err
}

// IsNtpClockSync query the ntp clock sync state
func (q *QueueProtocol) IsNtpClockSync() (*types.Reply, error) {
	msg, err := q.send(blockchainKey, types.EventIsNtpClockSync, &types.ReqNil{})
	if err != nil {
		log.Error("IsNtpClockSync", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.IsNtpClockSync); ok {
		return &types.Reply{IsOk: reply.GetIsntpclocksync()}, nil
	}
	err = types.ErrTypeAsset

	log.Error("IsNtpClockSync", "Error", err.Error())
	return nil, err
}

// LocalGet get value from local db by key
func (q *QueueProtocol) LocalGet(param *types.LocalDBGet) (*types.LocalReplyValue, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalGet", "Error", err)
		return nil, err
	}

	msg, err := q.send(blockchainKey, types.EventLocalGet, param)
	if err != nil {
		log.Error("LocalGet", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.LocalReplyValue); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// LocalSet set key value in local db
func (q *QueueProtocol) LocalSet(param *types.LocalDBSet) error {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalSet", "Error", err)
		return err
	}
	_, err := q.send(blockchainKey, types.EventLocalSet, param)
	if err != nil {
		log.Error("LocalSet", "Error", err.Error())
		return err
	}
	return nil
}

// LocalNew new a localdb object
func (q *QueueProtocol) LocalNew(readOnly bool) (*types.Int64, error) {
	msg, err := q.send(blockchainKey, types.EventLocalNew, readOnly)
	if err != nil {
		log.Error("LocalNew", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// LocalBegin begin a transaction
func (q *QueueProtocol) LocalBegin(param *types.Int64) error {
	_, err := q.send(blockchainKey, types.EventLocalBegin, param)
	if err != nil {
		log.Error("LocalBegin", "Error", err.Error())
		return err
	}
	return nil
}

// LocalClose begin a transaction
func (q *QueueProtocol) LocalClose(param *types.Int64) error {
	_, err := q.send(blockchainKey, types.EventLocalClose, param)
	if err != nil {
		log.Error("LocalClose", "Error", err.Error())
		return err
	}
	return nil
}

// LocalCommit commit a transaction
func (q *QueueProtocol) LocalCommit(param *types.Int64) error {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalCommit", "Error", err)
		return err
	}
	_, err := q.send(blockchainKey, types.EventLocalCommit, param)
	if err != nil {
		log.Error("LocalCommit", "Error", err.Error())
		return err
	}
	return nil
}

// LocalRollback rollback a transaction
func (q *QueueProtocol) LocalRollback(param *types.Int64) error {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalRollback", "Error", err)
		return err
	}
	_, err := q.send(blockchainKey, types.EventLocalRollback, param)
	if err != nil {
		log.Error("LocalRollback", "Error", err.Error())
		return err
	}
	return nil
}

// LocalList get value list from local db by key list
func (q *QueueProtocol) LocalList(param *types.LocalDBList) (*types.LocalReplyValue, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalList", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventLocalList, param)
	if err != nil {
		log.Error("LocalList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.LocalReplyValue); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetLastHeader get the current head detail
func (q *QueueProtocol) GetLastHeader() (*types.Header, error) {
	msg, err := q.send(blockchainKey, types.EventGetLastHeader, &types.ReqNil{})
	if err != nil {
		log.Error("GetLastHeader", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Header); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("GetLastHeader", "Error", err.Error())
	return nil, err
}

// Version get the software version
func (q *QueueProtocol) Version() (*types.VersionInfo, error) {
	types.AssertConfig(q.client)
	return &types.VersionInfo{
		Title:   q.client.GetConfig().GetTitle(),
		App:     version.GetAppVersion(),
		Chain33: version.GetVersion(),
		LocalDb: version.GetLocalDBVersion(),
		ChainID: q.client.GetConfig().GetChainID(),
	}, nil
}

// GetNetInfo get the net information
func (q *QueueProtocol) GetNetInfo(req *types.P2PGetNetInfoReq) (*types.NodeNetInfo, error) {
	msg, err := q.send(p2pKey, types.EventGetNetInfo, req)
	if err != nil {
		log.Error("GetNetInfo", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.NodeNetInfo); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("GetNetInfo", "Error", err.Error())
	return nil, err
}

// StoreSet set value by statehash and key to statedb
func (q *QueueProtocol) StoreSet(param *types.StoreSetWithSync) (*types.ReplyHash, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreSet", "Error", err)
		return nil, err
	}

	msg, err := q.send(storeKey, types.EventStoreSet, param)
	if err != nil {
		log.Error("StoreSet", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("StoreSet", "Error", err.Error())
	return nil, err
}

// StoreGet get value by statehash and key from statedb
func (q *QueueProtocol) StoreGet(param *types.StoreGet) (*types.StoreReplyValue, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreGet", "Error", err)
		return nil, err
	}

	msg, err := q.send(storeKey, types.EventStoreGet, param)
	if err != nil {
		log.Error("StoreGet", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.StoreReplyValue); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("StoreGet", "Error", err.Error())
	return nil, err
}

// StoreMemSet Memset kvs by statehash to statedb
func (q *QueueProtocol) StoreMemSet(param *types.StoreSetWithSync) (*types.ReplyHash, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreMemSet", "Error", err)
		return nil, err
	}

	msg, err := q.send(storeKey, types.EventStoreMemSet, param)
	if err != nil {
		log.Error("StoreMemSet", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("StoreMemSet", "Error", err.Error())
	return nil, err
}

// StoreCommit commit kvs by statehash to statedb
func (q *QueueProtocol) StoreCommit(param *types.ReqHash) (*types.ReplyHash, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreCommit", "Error", err)
		return nil, err
	}

	msg, err := q.send(storeKey, types.EventStoreCommit, param)
	if err != nil {
		log.Error("StoreCommit", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("StoreCommit", "Error", err.Error())
	return nil, err
}

// StoreRollback rollback kvs by statehash to statedb
func (q *QueueProtocol) StoreRollback(param *types.ReqHash) (*types.ReplyHash, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreRollback", "Error", err)
		return nil, err
	}

	msg, err := q.send(storeKey, types.EventStoreRollback, param)
	if err != nil {
		log.Error("StoreRollback", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("StoreRollback", "Error", err.Error())
	return nil, err
}

// StoreDel del kvs by statehash to statedb
func (q *QueueProtocol) StoreDel(param *types.StoreDel) (*types.ReplyHash, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreDel", "Error", err)
		return nil, err
	}

	msg, err := q.send(storeKey, types.EventStoreDel, param)
	if err != nil {
		log.Error("StoreDel", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("StoreDel", "Error", err.Error())
	return nil, err
}

// StoreList query list from statedb
func (q *QueueProtocol) StoreList(param *types.StoreList) (*types.StoreListReply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreList", "Error", err)
		return nil, err
	}

	msg, err := q.send(storeKey, types.EventStoreList, param)
	if err != nil {
		log.Error("StoreList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.StoreListReply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// StoreGetTotalCoins get total coins from statedb
func (q *QueueProtocol) StoreGetTotalCoins(param *types.IterateRangeByStateHash) (*types.ReplyGetTotalCoins, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreGetTotalCoins", "Error", err)
		return nil, err
	}
	msg, err := q.send(storeKey, types.EventStoreGetTotalCoins, param)
	if err != nil {
		log.Error("StoreGetTotalCoins", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyGetTotalCoins); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("StoreGetTotalCoins", "Error", err.Error())
	return nil, err
}

// CloseQueue close client queue
func (q *QueueProtocol) CloseQueue() (*types.Reply, error) {
	return q.client.CloseQueue()
}

// GetLastBlockSequence 获取最新的block执行序列号
func (q *QueueProtocol) GetLastBlockSequence() (*types.Int64, error) {
	msg, err := q.send(blockchainKey, types.EventGetLastBlockSequence, &types.ReqNil{})
	if err != nil {
		log.Error("GetLastBlockSequence", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetSequenceByHash 通过hash获取对应的执行序列号
func (q *QueueProtocol) GetSequenceByHash(param *types.ReqHash) (*types.Int64, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetSequenceByHash", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetSeqByHash, param)
	if err != nil {
		log.Error("GetSequenceByHash", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetBlockByHashes get block detail list by hash list
func (q *QueueProtocol) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBlockByHashes", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetBlockByHashes, param)
	if err != nil {
		log.Error("GetBlockByHashes", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.BlockDetails); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("GetBlockByHashes", "Error", err.Error())
	return nil, err
}

// GetBlockBySeq get block detail and hash by seq
func (q *QueueProtocol) GetBlockBySeq(param *types.Int64) (*types.BlockSeq, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBlockBySeq", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetBlockBySeq, param)
	if err != nil {
		log.Error("GetBlockBySeq", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.BlockSeq); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetBlockSequences block执行序列号
func (q *QueueProtocol) GetBlockSequences(param *types.ReqBlocks) (*types.BlockSequences, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBlockSequences", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetBlockSequences, param)
	if err != nil {
		log.Error("GetBlockSequences", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.BlockSequences); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("GetBlockSequences", "Error", err)
	return nil, err
}

// QueryChain query chain
func (q *QueueProtocol) QueryChain(param *types.ChainExecutor) (types.Message, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("QueryChain", "Error", err)
		return nil, err
	}
	msg, err := q.send(executorKey, types.EventBlockChainQuery, param)
	if err != nil {
		log.Error("QueryChain", "Error", err, "driver", param.Driver, "func", param.FuncName)
		return nil, err
	}
	if reply, ok := msg.GetData().(types.Message); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("QueryChain", "Error", err)
	return nil, err
}

// AddPushSubscribe Add Seq CallBack
func (q *QueueProtocol) AddPushSubscribe(param *types.PushSubscribeReq) (*types.ReplySubscribePush, error) {
	msg, err := q.send(blockchainKey, types.EventSubscribePush, param)
	if err != nil {
		log.Error("AddPushSubscribe", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplySubscribePush); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// ListPushes List Seq CallBacks
func (q *QueueProtocol) ListPushes() (*types.PushSubscribes, error) {
	msg, err := q.send(blockchainKey, types.EventListPushes, &types.ReqNil{})
	if err != nil {
		log.Error("ListPushes", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.PushSubscribes); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetPushSeqLastNum Get Seq Call Back Last Num
func (q *QueueProtocol) GetPushSeqLastNum(param *types.ReqString) (*types.Int64, error) {
	msg, err := q.send(blockchainKey, types.EventGetPushLastNum, param)
	if err != nil {
		log.Error("GetPushSeqLastNum", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetLastBlockMainSequence 获取最新的block执行序列号
func (q *QueueProtocol) GetLastBlockMainSequence() (*types.Int64, error) {
	msg, err := q.send(blockchainKey, types.EventGetLastBlockMainSequence, &types.ReqNil{})
	if err != nil {
		log.Error("GetLastBlockMainSequence", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetMainSequenceByHash 通过hash获取对应的执行序列号
func (q *QueueProtocol) GetMainSequenceByHash(param *types.ReqHash) (*types.Int64, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetMainSequenceByHash", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetMainSeqByHash, param)
	if err != nil {
		log.Error("GetMainSequenceByHash", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetParaTxByTitle 通过seq以及title获取对应平行连的交易
func (q *QueueProtocol) GetParaTxByTitle(param *types.ReqParaTxByTitle) (*types.ParaTxDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetParaTxByTitle", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetParaTxByTitle, param)
	if err != nil {
		log.Error("GetParaTxByTitle", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ParaTxDetails); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// LoadParaTxByTitle //获取拥有此title交易的区块高度
func (q *QueueProtocol) LoadParaTxByTitle(param *types.ReqHeightByTitle) (*types.ReplyHeightByTitle, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LoadParaTxByTitle", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetHeightByTitle, param)
	if err != nil {
		log.Error("LoadParaTxByTitle", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHeightByTitle); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetParaTxByHeight //通过区块高度列表+title获取平行链交易
func (q *QueueProtocol) GetParaTxByHeight(param *types.ReqParaTxByHeight) (*types.ParaTxDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetParaTxByHeight", "Error", err)
		return nil, err
	}
	msg, err := q.send(blockchainKey, types.EventGetParaTxByTitleAndHeight, param)
	if err != nil {
		log.Error("GetParaTxByHeight", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ParaTxDetails); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetConfig 通过seq以及title获取对应平行连的交易
func (q *QueueProtocol) GetConfig() *types.Chain33Config {
	if q.client == nil {
		panic("client is nil, can not get Chain33Config")
	}
	cfg := q.client.GetConfig()
	if cfg == nil {
		panic("Chain33Config is nil")
	}
	return cfg
}

// SendDelayTx send delay transaction to mempool
func (q *QueueProtocol) SendDelayTx(param *types.DelayTx, waitReply bool) (*types.Reply, error) {
	if param.GetTx() == nil {
		err := types.ErrNilTransaction
		log.Error("SendDelayTx", "Error", err)
		return nil, err
	}
	// 不需要阻塞等待
	if !waitReply {
		err := q.client.SendTimeout(
			q.client.NewMessage(mempoolKey, types.EventAddDelayTx, param),
			true, q.option.SendTimeout)
		if err != nil {
			log.Error("SendDelayTx", "txHash", common.ToHex(param.GetTx().Hash()), "send msg err", err.Error())
			return nil, err
		}
		return nil, nil
	}

	msg, err := q.send(mempoolKey, types.EventAddDelayTx, param)
	if err != nil {
		log.Error("SendDelayTx", "txHash", common.ToHex(param.GetTx().Hash()), "send msg err", err.Error())
		return nil, err
	}
	reply, ok := msg.GetData().(*types.Reply)
	if !ok {
		return nil, types.ErrTypeAsset
	}

	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	reply.Msg = param.GetTx().Hash()
	return reply, err
}

// AddBlacklist add peer to blacklist
func (q *QueueProtocol) AddBlacklist(req *types.BlackPeer) (*types.Reply, error) {
	msg, err := q.send(p2pKey, types.EventAddBlacklist, req)
	if err != nil {
		log.Error("AddBlacklist", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}

	return nil, types.ErrInvalidParam
}

// DelBlacklist delete peer from blacklist
func (q *QueueProtocol) DelBlacklist(req *types.BlackPeer) (*types.Reply, error) {
	msg, err := q.send(p2pKey, types.EventDelBlacklist, req)
	if err != nil {
		log.Error("DelBlacklist", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrInvalidParam
}

// ShowBlacklist show all blacklist peers
func (q *QueueProtocol) ShowBlacklist(req *types.ReqNil) (*types.Blacklist, error) {
	msg, err := q.send(p2pKey, types.EventShowBlacklist, req)
	if err != nil {
		log.Error("DelBlacklist", "Error", err.Error())
		return nil, err
	}

	if reply, ok := msg.GetData().(*types.Blacklist); ok {
		return reply, nil
	}

	return nil, types.ErrInvalidParam

}

// DialPeer  dial the the specified peer
func (q *QueueProtocol) DialPeer(req *types.SetPeer) (*types.Reply, error) {
	msg, err := q.send(p2pKey, types.EventDialPeer, req)
	if err != nil {
		log.Error("DialPeer", "Error", err.Error())
		return nil, err
	}

	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrInvalidParam

}

// ClosePeer close the specified peer
func (q *QueueProtocol) ClosePeer(req *types.SetPeer) (*types.Reply, error) {
	msg, err := q.send(p2pKey, types.EventClosePeer, req)
	if err != nil {
		log.Error("ClosePeer", "Error", err.Error())
		return nil, err
	}

	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrInvalidParam

}

// GetHighestBlockNum return heightest block num.
func (q *QueueProtocol) GetHighestBlockNum(param *types.ReqNil) (*types.ReplyBlockHeight, error) {
	msg, err := q.send(blockchainKey, types.EventHighestBlock, param)
	if err != nil {
		log.Error("ClosePeer", "Error", err.Error())
		return nil, err
	}

	if reply, ok := msg.GetData().(*types.ReplyBlockHeight); ok {
		return reply, nil
	}
	return nil, types.ErrInvalidParam
}

// GetFinalizedBlock get finalized block choice
func (q *QueueProtocol) GetFinalizedBlock() (*types.SnowChoice, error) {

	cli := q.client
	msg := cli.NewMessage("blockchain", types.EventSnowmanLastChoice, &types.ReqNil{})
	err := cli.Send(msg, true)
	if err != nil {
		return nil, err
	}

	reply, err := cli.Wait(msg)
	if err != nil {
		return nil, err
	}
	return reply.GetData().(*types.SnowChoice), nil
}
