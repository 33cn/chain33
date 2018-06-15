/*
封装系统内部模块间调用的功能接口，支持同步调用和异步调用
一旦新增了模块间调用接口（types.Event*）就应该在QueueProtocolAPI中定义一个接口，并实现
外部使用者通过QueueProtocolAPI直接调用目标模块的功能
*/
package client

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	mempoolKey = "mempool" // 未打包交易池
	p2pKey     = "p2p"     //
	//rpcKey			= "rpc"
	consensusKey = "consensus" // 共识系统
	//accountKey		= "accout"		// 账号系统
	//executorKey		= "execs"		// 交易执行器
	walletKey     = "wallet"     // 钱包
	blockchainKey = "blockchain" // 区块
	storeKey      = "store"
)

var log = log15.New("module", "client")

type QueueProtocolOption struct {
	// 发送请求超时时间
	SendTimeout time.Duration
	// 接收应答超时时间
	WaitTimeout time.Duration
}

// 消息通道协议实现
type QueueProtocol struct {
	// 消息队列
	client queue.Client
	option QueueProtocolOption
}

func New(client queue.Client, option *QueueProtocolOption) (QueueProtocolAPI, error) {
	if client == nil {
		return nil, types.ErrInvalidParam
	}
	q := &QueueProtocol{}
	q.client = client
	if option != nil {
		q.option = *option
	} else {
		q.option.SendTimeout = 120 * time.Second
		q.option.WaitTimeout = 60 * time.Second
	}
	return q, nil
}

func (q *QueueProtocol) query(topic string, ty int64, data interface{}) (queue.Message, error) {
	client := q.client
	msg := client.NewMessage(topic, ty, data)
	err := client.SendTimeout(msg, true, q.option.SendTimeout)
	if err != nil {
		return queue.Message{}, err
	}
	return client.WaitTimeout(msg, q.option.WaitTimeout)
}
func (q *QueueProtocol) Close() {
	q.client.Close()
}

func (q *QueueProtocol) setOption(option *QueueProtocolOption) {
	if option != nil {
		q.option = *option
	}
}
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

func (q *QueueProtocol) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("QueryTx", "Error", err)
		return nil, err
	}
	msg, err := q.query(blockchainKey, types.EventQueryTx, param)
	if err != nil {
		log.Error("QueryTx", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.TransactionDetail); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

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

func (q *QueueProtocol) WalletGetAccountList() (*types.WalletAccounts, error) {
	msg, err := q.query(walletKey, types.EventWalletGetAccountList, &types.ReqNil{})
	if err != nil {
		log.Error("WalletGetAccountList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletAccounts); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

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

func (q *QueueProtocol) WalletImportprivkey(param *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletImportprivkey", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletImportprivkey, param)
	if err != nil {
		log.Error("WalletImportprivkey", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletAccount); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

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

func (q *QueueProtocol) WalletAutoMiner(param *types.MinerFlag) (*types.Reply, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("WalletAutoMiner", "Error", err)
		return nil, err
	}
	msg, err := q.query(walletKey, types.EventWalletAutoMiner, param)
	if err != nil {
		log.Error("WalletAutoMiner", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

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

func (q *QueueProtocol) DumpPrivkey(param *types.ReqStr) (*types.ReplyStr, error) {
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
	if reply, ok := msg.GetData().(*types.ReplyStr); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueProtocol) CloseTickets() (*types.ReplyHashes, error) {
	msg, err := q.query(walletKey, types.EventCloseTickets, &types.ReqNil{})
	if err != nil {
		log.Error("CloseTickets", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHashes); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueProtocol) IsSync() (*types.Reply, error) {
	msg, err := q.query(blockchainKey, types.EventIsSync, &types.ReqNil{})
	if err != nil {
		log.Error("IsSync", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.IsCaughtUp); ok {
		return &types.Reply{IsOk: reply.Iscaughtup}, nil
	} else {
		err = types.ErrTypeAsset
	}
	log.Error("IsSync", "Error", err.Error())
	return nil, err
}

func (q *QueueProtocol) IsNtpClockSync() (*types.Reply, error) {
	msg, err := q.query(blockchainKey, types.EventIsNtpClockSync, &types.ReqNil{})
	if err != nil {
		log.Error("IsNtpClockSync", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.IsNtpClockSync); ok {
		return &types.Reply{IsOk: reply.GetIsntpclocksync()}, nil
	} else {
		err = types.ErrTypeAsset
	}
	log.Error("IsNtpClockSync", "Error", err.Error())
	return nil, err
}

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

func (q *QueueProtocol) Query(param *types.Query) (*types.Message, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("Query", "Error", err)
		return nil, err
	}

	msg, err := q.query(blockchainKey, types.EventQuery, param)
	if err != nil {
		log.Error("Query", "Error", err.Error())
		return nil, err
	}
	ret := msg.GetData().(types.Message)
	return &ret, err
}

func (q *QueueProtocol) Version() (*types.Reply, error) {
	return &types.Reply{IsOk: true, Msg: []byte(version.GetVersion())}, nil
}

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

func (q *QueueProtocol) SignRawTx(param *types.ReqSignRawTx) (*types.ReplySignRawTx, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("Query", "Error", err)
		return nil, err
	}

	data := &types.ReqSignRawTx{
		Addr:    param.GetAddr(),
		Privkey: param.GetPrivkey(),
		TxHex:   param.GetTxHex(),
		Expire:  param.GetExpire(),
		Index:   param.GetIndex(),
	}
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

func (q *QueueProtocol) BindMiner(param *types.ReqBindMiner) (*types.ReplyBindMiner, error) {
	ta := &types.TicketAction{}
	tBind := &types.TicketBind{
		MinerAddress:  param.BindAddr,
		ReturnAddress: param.OriginAddr,
	}
	ta.Value = &types.TicketAction_Tbind{Tbind: tBind}
	ta.Ty = types.TicketActionBind
	execer := []byte("ticket")
	to := address.ExecAddress(string(execer))
	txBind := &types.Transaction{Execer: execer, Payload: types.Encode(ta), To: to}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	txBind.Nonce = random.Int63()
	var err error
	txBind.Fee, err = txBind.GetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}
	txBind.Fee += types.MinFee
	txBindHex := types.Encode(txBind)
	txHexStr := hex.EncodeToString(txBindHex)

	return &types.ReplyBindMiner{TxHex: txHexStr}, nil
}

func (q *QueueProtocol) DecodeRawTransaction(param *types.ReqDecodeRawTransaction) (*types.Transaction, error) {
	var tx types.Transaction
	bytes, err := common.FromHex(param.TxHex)
	if err != nil {
		return nil, err
	}
	err = types.Decode(bytes, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
