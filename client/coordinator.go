/*
封装系统内部模块间调用的功能接口，支持同步调用和异步调用
一旦新增了模块间调用接口（types.Event*）就应该在QueueProtocolAPI中定义一个接口，并实现
外部使用者通过QueueProtocolAPI直接调用目标模块的功能
*/
package client

import (
	"bytes"
	"github.com/pkg/errors"
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

// 消息通道协议实现
type QueueCoordinator struct {
	client queue.Client // 消息队列
}

func NewQueueAPI(client queue.Client) (QueueProtocolAPI, error) {
	if nil == client {
		return nil, errors.New("Invalid param")
	}
	q := &QueueCoordinator{}
	q.client = client
	return q, nil
}

func (q *QueueCoordinator) query(topic string, ty int64, data interface{}) (queue.Message, error) {
	client := q.client
	msg := client.NewMessage(topic, ty, data)
	err := client.Send(msg, true)
	if nil != err {
		return queue.Message{}, err
	}
	return client.Wait(msg)
}

func (q *QueueCoordinator) GetTx(param *types.Transaction) (*types.Reply, error) {
	msg, err := q.query(mempoolKey, types.EventTx, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) GetTxList(param *types.TxHashList) (*types.ReplyTxList, error) {
	msg, err := q.query(mempoolKey, types.EventTxList, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyTxList), nil
}

func (q *QueueCoordinator) GetBlocks(param *types.ReqBlocks) (*types.BlockDetails, error) {
	msg, err := q.query(blockchainKey, types.EventGetBlocks, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.BlockDetails), nil
}

func (q *QueueCoordinator) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	msg, err := q.query(blockchainKey, types.EventQueryTx, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.TransactionDetail), nil
}

func (q *QueueCoordinator) GetTransactionByAddr(param *types.ReqAddr) (*types.ReplyTxInfos, error) {
	msg, err := q.query(blockchainKey, types.EventGetTransactionByAddr, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyTxInfos), nil
}

func (q *QueueCoordinator) GetTransactionByHash(param *types.ReqHashes) (*types.TransactionDetails, error) {
	msg, err := q.query(blockchainKey, types.EventGetTransactionByHash, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.TransactionDetails), nil
}

func (q *QueueCoordinator) GetMempool() (*types.ReplyTxList, error) {
	msg, err := q.query(blockchainKey, types.EventGetMempool, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyTxList), nil
}

func (q *QueueCoordinator) WalletGetAccountList() (*types.WalletAccounts, error) {
	msg, err := q.query(walletKey, types.EventWalletGetAccountList, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.WalletAccounts), nil
}

func (q *QueueCoordinator) NewAccount(param *types.ReqNewAccount) (*types.WalletAccount, error) {
	msg, err := q.query(walletKey, types.EventNewAccount, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.WalletAccount), nil
}

func (q *QueueCoordinator) WalletTransactionList(param *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	msg, err := q.query(walletKey, types.EventWalletTransactionList, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.WalletTxDetails), nil
}

func (q *QueueCoordinator) WalletImportprivkey(param *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	msg, err := q.query(walletKey, types.EventWalletImportprivkey, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.WalletAccount), nil
}

func (q *QueueCoordinator) WalletSendToAddress(param *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	msg, err := q.query(walletKey, types.EventWalletSendToAddress, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHash), nil
}

func (q *QueueCoordinator) WalletSetFee(param *types.ReqWalletSetFee) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventWalletSetFee, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) WalletSetLabel(param *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	msg, err := q.query(walletKey, types.EventWalletSetLabel, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.WalletAccount), nil
}

func (q *QueueCoordinator) WalletMergeBalance(param *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	msg, err := q.query(walletKey, types.EventWalletMergeBalance, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHashes), nil
}

func (q *QueueCoordinator) WalletSetPasswd(param *types.ReqWalletSetPasswd) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventWalletSetPasswd, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) WalletLock() (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventWalletLock, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) WalletUnLock() (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventWalletUnLock, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) PeerInfo() (*types.PeerList, error) {
	msg, err := q.query(p2pKey, types.EventPeerInfo, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.PeerList), nil
}

func (q *QueueCoordinator) GetHeaders(param *types.ReqBlocks) (*types.Headers, error) {
	msg, err := q.query(blockchainKey, types.EventGetHeaders, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Headers), nil
}

func (q *QueueCoordinator) GetLastMempool(param *types.ReqNil) (*types.ReplyTxList, error) {
	msg, err := q.query(mempoolKey, types.EventGetLastMempool, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyTxList), nil
}

func (q *QueueCoordinator) GetBlockOverview(param *types.ReqHash) (*types.BlockOverview, error) {
	msg, err := q.query(blockchainKey, types.EventGetBlockOverview, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.BlockOverview), nil
}

func (q *QueueCoordinator) GetAddrOverview(param *types.ReqAddr) (*types.BlockOverview, error) {
	msg, err := q.query(blockchainKey, types.EventGetAddrOverview, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.BlockOverview), nil
}

func (q *QueueCoordinator) GetBlockHash(param *types.ReqInt) (*types.ReplyHash, error) {
	msg, err := q.query(blockchainKey, types.EventGetBlockHash, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHash), nil
}

func (q *QueueCoordinator) GenSeed(param *types.GenSeedLang) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventGenSeed, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) SaveSeed(param *types.SaveSeedByPw) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventSaveSeed, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) GetSeed(param *types.GetSeedByPw) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventGetSeed, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) GetWalletStatus() (*types.WalletStatus, error) {
	msg, err := q.query(walletKey, types.EventGetWalletStatus, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.WalletStatus), nil
}

func (q *QueueCoordinator) WalletAutoMiner(param *types.MinerFlag) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventWalletAutoMiner, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) GetTicketCount() (*types.Int64, error) {
	msg, err := q.query(consensusKey, types.EventGetTicketCount, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Int64), nil
}

func (q *QueueCoordinator) DumpPrivkey(param *types.ReqStr) (*types.ReplyStr, error) {
	msg, err := q.query(walletKey, types.EventWalletAutoMiner, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyStr), nil
}

func (q *QueueCoordinator) CloseTickets() (*types.ReplyHashes, error) {
	msg, err := q.query(walletKey, types.EventCloseTickets, nil)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHashes), nil
}

func (q *QueueCoordinator) IsSync() (ret bool, err error) {
	ret = false
	msg, err := q.query(blockchainKey, types.EventIsSync, nil)
	if nil != err {
		return
	}
	ret = msg.GetData().(*types.IsCaughtUp).GetIscaughtup()
	return
}

func (q *QueueCoordinator) TokenPreCreate(param *types.ReqTokenPreCreate) (*types.ReplyHash, error) {
	msg, err := q.query(walletKey, types.EventTokenPreCreate, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHash), nil
}

func (q *QueueCoordinator) TokenFinishCreate(param *types.ReqTokenFinishCreate) (*types.ReplyHash, error) {
	msg, err := q.query(walletKey, types.EventTokenFinishCreate, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHash), nil
}

func (q *QueueCoordinator) TokenRevokeCreate(param *types.ReqTokenRevokeCreate) (*types.ReplyHash, error) {
	msg, err := q.query(walletKey, types.EventTokenRevokeCreate, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHash), nil
}

func (q *QueueCoordinator) SellToken(param *types.ReqSellToken) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventSellToken, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) BuyToken(param *types.ReqBuyToken) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventBuyToken, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) RevokeSellToken(param *types.ReqRevokeSell) (*types.Reply, error) {
	msg, err := q.query(walletKey, types.EventRevokeSellToken, param)
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) IsNtpClockSync() (ret bool, err error) {
	ret = false
	msg, err := q.query(blockchainKey, types.EventIsNtpClockSync, nil)
	if nil != err {
		return
	}
	ret = msg.GetData().(*types.IsNtpClockSync).GetIsntpclocksync()
	return
}

func (q *QueueCoordinator) LocalGet(param *types.ReqHash) (*types.LocalReplyValue, error) {
	var keys [][]byte
	keys = append(keys, func(hash []byte) []byte {
		s := [][]byte{[]byte("TotalFeeKey:"), hash}
		sep := []byte("")
		return bytes.Join(s, sep)
	}(param.Hash))

	msg, err := q.query(walletKey, types.EventLocalGet, &types.LocalDBGet{keys})
	if nil != err {
		return nil, err
	}
	return msg.GetData().(*types.LocalReplyValue), nil
}
