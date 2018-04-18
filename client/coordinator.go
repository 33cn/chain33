/*
封装系统内部模块间调用的功能接口，支持同步调用和异步调用
一旦新增了模块间调用接口（types.Event*）就应该在QueueProtocolAPI中定义一个接口，并实现
外部使用者通过QueueProtocolAPI直接调用目标模块的功能
*/
package client

import (
	"bytes"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
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
	//storeKey		= "store"
)

var log = log15.New("module", "client")

type QueueCoordinatorOption struct {
	// 发送请求超时时间
	SendTimeout time.Duration
	// 接收应答超时时间
	WaitTimeout time.Duration
}

// 消息通道协议实现
type QueueCoordinator struct {
	// 消息队列
	client queue.Client
	option QueueCoordinatorOption
}

func New(client queue.Client, option *QueueCoordinatorOption) (QueueProtocolAPI, error) {
	if client == nil {
		return nil, types.ErrInvalidParam
	}
	q := &QueueCoordinator{}
	q.client = client
	if option != nil {
		q.option = *option
	} else {
		q.option.SendTimeout = 120 * time.Second
		q.option.WaitTimeout = 60 * time.Second
	}
	return q, nil
}

func (q *QueueCoordinator) query(topic string, ty int64, data interface{}) (queue.Message, error) {
	client := q.client
	msg := client.NewMessage(topic, ty, data)
	err := client.SendTimeout(msg, true, q.option.SendTimeout)
	if err != nil {
		return queue.Message{}, err
	}
	return client.WaitTimeout(msg, q.option.WaitTimeout)
}

func (q *QueueCoordinator) SetOption(option *QueueCoordinatorOption) {
	if option != nil {
		q.option = *option
	}
}

func (q *QueueCoordinator) SendTx(param *types.Transaction) (*types.Reply, error) {
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
		}
	} else {
		err = types.ErrTypeAsset
	}
	return reply, err
}

func (q *QueueCoordinator) GetTxList(param *types.TxHashList) (*types.ReplyTxList, error) {
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

func (q *QueueCoordinator) GetBlocks(param *types.ReqBlocks) (*types.BlockDetails, error) {
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
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
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

func (q *QueueCoordinator) GetTransactionByAddr(param *types.ReqAddr) (*types.ReplyTxInfos, error) {
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
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) GetTransactionByHash(param *types.ReqHashes) (*types.TransactionDetails, error) {
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

func (q *QueueCoordinator) GetMempool() (*types.ReplyTxList, error) {
	msg, err := q.query(mempoolKey, types.EventGetMempool, nil)
	if err != nil {
		log.Error("GetMempool", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) WalletGetAccountList() (*types.WalletAccounts, error) {
	msg, err := q.query(walletKey, types.EventWalletGetAccountList, nil)
	if err != nil {
		log.Error("WalletGetAccountList", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletAccounts); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) NewAccount(param *types.ReqNewAccount) (*types.WalletAccount, error) {
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

func (q *QueueCoordinator) WalletTransactionList(param *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
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

func (q *QueueCoordinator) WalletImportprivkey(param *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
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

func (q *QueueCoordinator) WalletSendToAddress(param *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
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

func (q *QueueCoordinator) WalletSetFee(param *types.ReqWalletSetFee) (*types.Reply, error) {
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

func (q *QueueCoordinator) WalletSetLabel(param *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
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

func (q *QueueCoordinator) WalletMergeBalance(param *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
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

func (q *QueueCoordinator) WalletSetPasswd(param *types.ReqWalletSetPasswd) (*types.Reply, error) {
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

func (q *QueueCoordinator) WalletLock() (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventWalletLock, nil)
	if err != nil {
		log.Error("WalletLock", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Reply); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) WalletUnLock(param *types.WalletUnLock) (*types.Reply, error) {
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

func (q *QueueCoordinator) PeerInfo() (*types.PeerList, error) {
	msg, err := q.query(p2pKey, types.EventPeerInfo, nil)
	if err != nil {
		log.Error("PeerInfo", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.PeerList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) GetHeaders(param *types.ReqBlocks) (*types.Headers, error) {
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

func (q *QueueCoordinator) GetLastMempool(param *types.ReqNil) (*types.ReplyTxList, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("GetLastMempool", "Error", err)
		return nil, err
	}
	msg, err := q.query(mempoolKey, types.EventGetLastMempool, param)
	if err != nil {
		log.Error("GetLastMempool", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyTxList); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) GetBlockOverview(param *types.ReqHash) (*types.BlockOverview, error) {
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

func (q *QueueCoordinator) GetAddrOverview(param *types.ReqAddr) (*types.AddrOverview, error) {
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

func (q *QueueCoordinator) GetBlockHash(param *types.ReqInt) (*types.ReplyHash, error) {
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

func (q *QueueCoordinator) GenSeed(param *types.GenSeedLang) (*types.ReplySeed, error) {
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

func (q *QueueCoordinator) SaveSeed(param *types.SaveSeedByPw) (*types.Reply, error) {
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

func (q *QueueCoordinator) GetSeed(param *types.GetSeedByPw) (*types.ReplySeed, error) {
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

func (q *QueueCoordinator) GetWalletStatus() (*types.WalletStatus, error) {
	msg, err := q.query(walletKey, types.EventGetWalletStatus, nil)
	if err != nil {
		log.Error("GetWalletStatus", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.WalletStatus); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) WalletAutoMiner(param *types.MinerFlag) (*types.Reply, error) {
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

func (q *QueueCoordinator) GetTicketCount() (*types.Int64, error) {
	msg, err := q.query(consensusKey, types.EventGetTicketCount, nil)
	if err != nil {
		log.Error("GetTicketCount", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Int64); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) DumpPrivkey(param *types.ReqStr) (*types.ReplyStr, error) {
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

func (q *QueueCoordinator) CloseTickets() (*types.ReplyHashes, error) {
	msg, err := q.query(walletKey, types.EventCloseTickets, nil)
	if err != nil {
		log.Error("CloseTickets", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.ReplyHashes); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) IsSync() (ret bool, err error) {
	ret = false
	msg, err := q.query(blockchainKey, types.EventIsSync, nil)
	if err != nil {
		log.Error("IsSync", "Error", err.Error())
		return
	}
	if reply, ok := msg.GetData().(*types.IsCaughtUp); ok {
		ret = reply.GetIscaughtup()
	} else {
		err = types.ErrTypeAsset
	}
	return
}

func (q *QueueCoordinator) IsNtpClockSync() (ret bool, err error) {
	ret = false
	msg, err := q.query(blockchainKey, types.EventIsNtpClockSync, nil)
	if err != nil {
		log.Error("IsNtpClockSync", "Error", err.Error())
		return
	}
	if reply, ok := msg.GetData().(*types.IsNtpClockSync); ok {
		ret = reply.GetIsntpclocksync()
	} else {
		err = types.ErrTypeAsset
	}
	return
}

func (q *QueueCoordinator) LocalGet(param *types.ReqHash) (*types.LocalReplyValue, error) {
	if param == nil {
		err := types.ErrInvalidParam
		log.Error("LocalGet", "Error", err)
		return nil, err
	}

	var keys [][]byte
	keys = append(keys, func(hash []byte) []byte {
		s := [][]byte{[]byte("TotalFeeKey:"), hash}
		sep := []byte("")
		return bytes.Join(s, sep)
	}(param.Hash))

	msg, err := q.query(walletKey, types.EventLocalGet, &types.LocalDBGet{keys})
	if err != nil {
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.LocalReplyValue); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}

func (q *QueueCoordinator) GetLastHeader() (*types.Header, error) {
	msg, err := q.query(blockchainKey, types.EventGetLastHeader, nil)
	if err != nil {
		log.Error("GetLastHeader", "Error", err.Error())
		return nil, err
	}
	if reply, ok := msg.GetData().(*types.Header); ok {
		return reply, nil
	}
	return nil, types.ErrTypeAsset
}
