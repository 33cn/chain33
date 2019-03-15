// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package client 系统接口客户端: 封装 Queue Event
package client

import (
	"fmt"
	"time"

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

func (q *QueueProtocol) query(topic string, ty int64, data interface{}) (*queue.Message, error) {
	client := q.client
	msg := client.NewMessage(topic, ty, data)
	err := client.SendTimeout(msg, true, q.option.SendTimeout)
	if err != nil {
		return &queue.Message{}, err
	}
	return client.WaitTimeout(msg, q.option.WaitTimeout)
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

// SendTx send transaction to mempool
func (q *QueueProtocol) SendTx(param *types.Transaction) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("SendTx", "Error", err)
		return nil, err
	}
	msg, err := q.query(mempoolKey, types.EventTx, param)
	if err != nil {
		log.Error("SendTx", "Error", err.Error())
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
	return reply, err
}

// GetTxList get transactions from mempool
func (q *QueueProtocol) GetTxList(param *types.TxHashList) (*types.ReplyTxList, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetTxList", "Error", err)
		return nil, err
	}
	msg, err := q.query(mempoolKey, types.EventTxList, param)
	if err != nil {
		log.Error("GetTxList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetBlocks get block detail from blockchain
func (q *QueueProtocol) GetBlocks(param *types.ReqBlocks) (*types.BlockDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetBlocks", "Error", err)
		return nil, err
	}
	msg, err := q.query(blockchainKey, types.EventGetBlocks, param)
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
	msg, err := q.query(blockchainKey, types.EventQueryTx, param)
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
	msg, err := q.query(blockchainKey, types.EventGetTransactionByAddr, param)
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
	msg, err := q.query(blockchainKey, types.EventGetTransactionByHash, param)
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
func (q *QueueProtocol) GetMempool() (*types.ReplyTxList, error) {
	msg, err := q.query(mempoolKey, types.EventGetMempool, &types.ReqNil{})
	if err != nil {
		log.Error("GetMempool", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletGetAccountList get account list from wallet
func (q *QueueProtocol) WalletGetAccountList(req *types.ReqAccountList) (*types.WalletAccounts, error) {
	msg, err := q.query(walletKey, types.EventWalletGetAccountList, req)
	if err != nil {
		log.Error("WalletGetAccountList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletAccounts); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// NewAccount new account in wallet
func (q *QueueProtocol) NewAccount(param *types.ReqNewAccount) (*types.WalletAccount, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("NewAccount", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventNewAccount, param)
	if err != nil {
		log.Error("NewAccount", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletAccount); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletTransactionList get transactions from wallet
func (q *QueueProtocol) WalletTransactionList(param *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletTransactionList", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletTransactionList, param)
	if err != nil {
		log.Error("WalletTransactionList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletTxDetails); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletImportprivkey import privkey in wallet
func (q *QueueProtocol) WalletImportprivkey(param *types.ReqWalletImportPrivkey) (*types.WalletAccount, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletImportprivkey", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletImportPrivkey, param)
	if err != nil {
		log.Error("WalletImportprivkey", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletAccount); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletSendToAddress req send to address
func (q *QueueProtocol) WalletSendToAddress(param *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletSendToAddress", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletSendToAddress, param)
	if err != nil {
		log.Error("WalletSendToAddress", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletSetFee set wallet transaction fee
func (q *QueueProtocol) WalletSetFee(param *types.ReqWalletSetFee) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletSetFee", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletSetFee, param)
	if err != nil {
		log.Error("WalletSetFee", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletSetLabel set wallet address and label
func (q *QueueProtocol) WalletSetLabel(param *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletSetLabel", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletSetLabel, param)
	if err != nil {
		log.Error("WalletSetLabel", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletAccount); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletMergeBalance merge balance to one address
func (q *QueueProtocol) WalletMergeBalance(param *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletMergeBalance", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletMergeBalance, param)
	if err != nil {
		log.Error("WalletMergeBalance", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHashes); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletSetPasswd set wallet passwd
func (q *QueueProtocol) WalletSetPasswd(param *types.ReqWalletSetPasswd) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletSetPasswd", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletSetPasswd, param)
	if err != nil {
		log.Error("WalletSetPasswd", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletLock lock wallet
func (q *QueueProtocol) WalletLock() (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventWalletLock, &types.ReqNil{})
	if err != nil {
		log.Error("WalletLock", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletUnLock unlock wallet
func (q *QueueProtocol) WalletUnLock(param *types.WalletUnLock) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletUnLock", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletUnLock, param)
	if err != nil {
		log.Error("WalletUnLock", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// PeerInfo query peer list
func (q *QueueProtocol) PeerInfo() (*types.PeerList, error) {
	msg, err := q.query(p2pKey, types.EventPeerInfo, &types.ReqNil{})
	if err != nil {
		log.Error("PeerInfo", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.PeerList); ok {
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
	msg, err := q.query(blockchainKey, types.EventGetHeaders, param)
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
	msg, err := q.query(mempoolKey, types.EventGetLastMempool, &types.ReqNil{})
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
func (q *QueueProtocol) GetProperFee() (*types.ReplyProperFee, error) {
	msg, err := q.query(mempoolKey, types.EventGetProperFee, &types.ReqNil{})
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
	msg, err := q.query(blockchainKey, types.EventGetBlockOverview, param)
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
	msg, err := q.query(blockchainKey, types.EventGetAddrOverview, param)
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
	msg, err := q.query(blockchainKey, types.EventGetBlockHash, param)
	if err != nil {
		log.Error("GetBlockHash", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHash); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GenSeed generate seed return the seed
func (q *QueueProtocol) GenSeed(param *types.GenSeedLang) (*types.ReplySeed, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GenSeed", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventGenSeed, param)
	if err != nil {
		log.Error("GenSeed", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplySeed); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// SaveSeed save the wallet seed
func (q *QueueProtocol) SaveSeed(param *types.SaveSeedByPw) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("SaveSeed", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventSaveSeed, param)
	if err != nil {
		log.Error("SaveSeed", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetSeed get seed from wallet
func (q *QueueProtocol) GetSeed(param *types.GetSeedByPw) (*types.ReplySeed, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetSeed", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventGetSeed, param)
	if err != nil {
		log.Error("GetSeed", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplySeed); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetWalletStatus get wallet current status
func (q *QueueProtocol) GetWalletStatus() (*types.WalletStatus, error) {
	msg, err := q.query(walletKey, types.EventGetWalletStatus, &types.ReqNil{})
	if err != nil {
		log.Error("GetWalletStatus", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletStatus); ok {
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
	msg, err := q.query(consensusKey, types.EventConsensusQuery, param)
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
	msg, err := q.query(walletKey, types.EventWalletExecutor, param)
	if err != nil {
		log.Error("ExecWallet", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(types.Message); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// DumpPrivkey dump privkey by wallet
func (q *QueueProtocol) DumpPrivkey(param *types.ReqString) (*types.ReplyString, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("DumpPrivkey", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventDumpPrivkey, param)
	if err != nil {
		log.Error("DumpPrivkey", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyString); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// IsSync query the blockchain sync state
func (q *QueueProtocol) IsSync() (*types.Reply, error) {
	msg, err := q.query(blockchainKey, types.EventIsSync, &types.ReqNil{})
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
	msg, err := q.query(blockchainKey, types.EventIsNtpClockSync, &types.ReqNil{})
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

	msg, err := q.query(blockchainKey, types.EventLocalGet, param)
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
	_, err := q.query(blockchainKey, types.EventLocalSet, param)
	if err != nil {
		log.Error("LocalSet", "Error", err.Error())
		return err
	}
	return nil
}

//LocalNew new a localdb object
func (q *QueueProtocol) LocalNew(param *types.ReqNil) (*types.Int64, error) {
	msg, err := q.query(blockchainKey, types.EventLocalNew, nil)
	if err != nil {
		log.Error("LocalNew", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

//LocalBegin begin a transaction
func (q *QueueProtocol) LocalBegin(param *types.Int64) error {
	_, err := q.query(blockchainKey, types.EventLocalBegin, param)
	if err != nil {
		log.Error("LocalBegin", "Error", err.Error())
		return err
	}
	return nil
}

//LocalClose begin a transaction
func (q *QueueProtocol) LocalClose(param *types.Int64) error {
	_, err := q.query(blockchainKey, types.EventLocalClose, param)
	if err != nil {
		log.Error("LocalClose", "Error", err.Error())
		return err
	}
	return nil
}

//LocalCommit commit a transaction
func (q *QueueProtocol) LocalCommit(param *types.Int64) error {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalCommit", "Error", err)
		return err
	}
	_, err := q.query(blockchainKey, types.EventLocalCommit, param)
	if err != nil {
		log.Error("LocalCommit", "Error", err.Error())
		return err
	}
	return nil
}

//LocalRollback rollback a transaction
func (q *QueueProtocol) LocalRollback(param *types.Int64) error {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalRollback", "Error", err)
		return err
	}
	_, err := q.query(blockchainKey, types.EventLocalRollback, param)
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
	msg, err := q.query(blockchainKey, types.EventLocalList, param)
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
	msg, err := q.query(blockchainKey, types.EventGetLastHeader, &types.ReqNil{})
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
	return &types.VersionInfo{
		Title:   types.GetTitle(),
		App:     version.GetAppVersion(),
		Chain33: version.GetVersion(),
		LocalDb: version.GetLocalDBVersion(),
	}, nil
}

// GetNetInfo get the net information
func (q *QueueProtocol) GetNetInfo() (*types.NodeNetInfo, error) {
	msg, err := q.query(p2pKey, types.EventGetNetInfo, &types.ReqNil{})
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

// SignRawTx sign transaction return the sign tx data
func (q *QueueProtocol) SignRawTx(param *types.ReqSignRawTx) (*types.ReplySignRawTx, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("Query", "Error", err)
		return nil, err
	}
	data := param
	msg, err := q.query(walletKey, types.EventSignRawTx, data)
	if err != nil {
		log.Error("SignRawTx", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplySignRawTx); ok {
		return reply, nil
	}
	err = types.ErrTypeAsset
	log.Error("SignRawTx", "Error", err.Error())
	return nil, err
}

// StoreGet get value by statehash and key from statedb
func (q *QueueProtocol) StoreGet(param *types.StoreGet) (*types.StoreReplyValue, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreGet", "Error", err)
		return nil, err
	}

	msg, err := q.query(storeKey, types.EventStoreGet, param)
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

//StoreList query list from statedb
func (q *QueueProtocol) StoreList(param *types.StoreList) (*types.StoreListReply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("StoreList", "Error", err)
		return nil, err
	}

	msg, err := q.query(storeKey, types.EventStoreList, param)
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
	msg, err := q.query(storeKey, types.EventStoreGetTotalCoins, param)
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

// GetFatalFailure get fatal failure from wallet
func (q *QueueProtocol) GetFatalFailure() (*types.Int32, error) {
	msg, err := q.query(walletKey, types.EventFatalFailure, &types.ReqNil{})
	if err != nil {
		log.Error("GetFatalFailure", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int32); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// CloseQueue close client queue
func (q *QueueProtocol) CloseQueue() (*types.Reply, error) {
	return q.client.CloseQueue()
}

// GetLastBlockSequence 获取最新的block执行序列号
func (q *QueueProtocol) GetLastBlockSequence() (*types.Int64, error) {
	msg, err := q.query(blockchainKey, types.EventGetLastBlockSequence, &types.ReqNil{})
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
	msg, err := q.query(blockchainKey, types.EventGetSeqByHash, param)
	if err != nil {
		log.Error("GetSequenceByHash", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {

		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// WalletCreateTx create transaction
func (q *QueueProtocol) WalletCreateTx(param *types.ReqCreateTransaction) (*types.Transaction, error) {
	msg, err := q.query(walletKey, types.EventWalletCreateTx, param)
	if err != nil {
		log.Error("CreateTrasaction", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Transaction); ok {
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
	msg, err := q.query(blockchainKey, types.EventGetBlockByHashes, param)
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
	msg, err := q.query(blockchainKey, types.EventGetBlockBySeq, param)
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
	msg, err := q.query(blockchainKey, types.EventGetBlockSequences, param)
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
	msg, err := q.query(executorKey, types.EventBlockChainQuery, param)
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

// GetTicketCount get ticket count from consensus
func (q *QueueProtocol) GetTicketCount() (*types.Int64, error) {
	msg, err := q.query(consensusKey, types.EventGetTicketCount, &types.ReqNil{})
	if err != nil {
		log.Error("GetTicketCount", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// AddSeqCallBack Add Seq CallBack
func (q *QueueProtocol) AddSeqCallBack(param *types.BlockSeqCB) (*types.Reply, error) {

	msg, err := q.query(blockchainKey, types.EventAddBlockSeqCB, param)
	if err != nil {
		log.Error("AddSeqCallBack", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// ListSeqCallBack List Seq CallBacks
func (q *QueueProtocol) ListSeqCallBack() (*types.BlockSeqCBs, error) {

	msg, err := q.query(blockchainKey, types.EventListBlockSeqCB, &types.ReqNil{})
	if err != nil {
		log.Error("ListSeqCallBack", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.BlockSeqCBs); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

// GetSeqCallBackLastNum Get Seq Call Back Last Num
func (q *QueueProtocol) GetSeqCallBackLastNum(param *types.ReqString) (*types.Int64, error) {

	msg, err := q.query(blockchainKey, types.EventGetSeqCBLastNum, param)
	if err != nil {
		log.Error("ListSeqCallBack", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}
