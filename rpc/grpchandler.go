// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"time"

	"strings"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/queue"
	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
)

// SendTransactionSync send transaction by network and query
func (g *Grpc) SendTransactionSync(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {
	reply, err := g.cli.SendTx(in)
	if err != nil {
		return reply, err
	}
	hash := in.Hash()
	for i := 0; i < 100; i++ {
		detail, err := g.cli.QueryTx(&pb.ReqHash{Hash: hash})
		if err == pb.ErrInvalidParam || err == pb.ErrTypeAsset {
			return nil, err
		}
		if detail != nil {
			return &pb.Reply{IsOk: true, Msg: hash}, nil
		}
		time.Sleep(time.Second / 3)
	}
	return nil, pb.ErrTimeout
}

// SendTransaction send transaction by network
func (g *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {
	return g.cli.SendTx(in)
}

// SendTransactions send transaction by network
func (g *Grpc) SendTransactions(ctx context.Context, in *pb.Transactions) (*pb.Replies, error) {
	if len(in.GetTxs()) == 0 {
		return nil, nil
	}
	reps := &pb.Replies{ReplyList: make([]*pb.Reply, 0, len(in.GetTxs()))}
	for _, tx := range in.GetTxs() {
		reply, err := g.cli.SendTx(tx)
		if err != nil {
			// grpc内部不允许error非空情况下返回参数,需要把错误信息记录在结构中
			reply = &pb.Reply{Msg: []byte(err.Error())}
		}
		reps.ReplyList = append(reps.ReplyList, reply)
	}
	return reps, nil
}

// CreateNoBalanceTxs create multiple transaction with no balance
func (g *Grpc) CreateNoBalanceTxs(ctx context.Context, in *pb.NoBalanceTxs) (*pb.ReplySignRawTx, error) {
	reply, err := g.cli.CreateNoBalanceTxs(in)
	if err != nil {
		return nil, err
	}
	tx := pb.Encode(reply)
	return &pb.ReplySignRawTx{TxHex: common.ToHex(tx)}, nil
}

// CreateNoBalanceTransaction create transaction with no balance
func (g *Grpc) CreateNoBalanceTransaction(ctx context.Context, in *pb.NoBalanceTx) (*pb.ReplySignRawTx, error) {
	params := &pb.NoBalanceTxs{
		TxHexs:  []string{in.GetTxHex()},
		PayAddr: in.GetPayAddr(),
		Privkey: in.GetPrivkey(),
		Expire:  in.GetExpire(),
	}
	reply, err := g.cli.CreateNoBalanceTxs(params)
	if err != nil {
		return nil, err
	}
	tx := pb.Encode(reply)
	return &pb.ReplySignRawTx{TxHex: common.ToHex(tx)}, nil
}

// CreateRawTransaction create rawtransaction of grpc
func (g *Grpc) CreateRawTransaction(ctx context.Context, in *pb.CreateTx) (*pb.UnsignTx, error) {
	reply, err := g.cli.CreateRawTransaction(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

// ReWriteTx re-write raw tx parameters of grpc
func (g *Grpc) ReWriteTx(ctx context.Context, in *pb.ReWriteRawTx) (*pb.UnsignTx, error) {
	reply, err := g.cli.ReWriteRawTx(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

// CreateTransaction create transaction of grpc
func (g *Grpc) CreateTransaction(ctx context.Context, in *pb.CreateTxIn) (*pb.UnsignTx, error) {
	pb.AssertConfig(g.cli)
	cfg := g.cli.GetConfig()
	execer := cfg.ExecName(string(in.Execer))
	exec := pb.LoadExecutorType(execer)
	if exec == nil {
		log.Error("callExecNewTx", "Error", "exec not found")
		return nil, pb.ErrNotSupport
	}
	msg, err := exec.GetAction(in.ActionName)
	if err != nil {
		return nil, err
	}
	//decode protocol buffer
	err = pb.Decode(in.Payload, msg)
	if err != nil {
		return nil, err
	}
	reply, err := pb.CallCreateTx(cfg, execer, in.ActionName, msg)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

// CreateRawTxGroup create rawtransaction for group
func (g *Grpc) CreateRawTxGroup(ctx context.Context, in *pb.CreateTransactionGroup) (*pb.UnsignTx, error) {
	reply, err := g.cli.CreateRawTxGroup(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

// QueryTransaction query transaction by grpc
func (g *Grpc) QueryTransaction(ctx context.Context, in *pb.ReqHash) (*pb.TransactionDetail, error) {
	return g.cli.QueryTx(in)
}

// GetBlocks get blocks by grpc
func (g *Grpc) GetBlocks(ctx context.Context, in *pb.ReqBlocks) (*pb.Reply, error) {
	reply, err := g.cli.GetBlocks(&pb.ReqBlocks{
		Start:    in.Start,
		End:      in.End,
		IsDetail: in.IsDetail,
	})
	if err != nil {
		return nil, err
	}
	return &pb.Reply{
			IsOk: true,
			Msg:  pb.Encode(reply)},
		nil
}

// GetLastHeader get lastheader information
func (g *Grpc) GetLastHeader(ctx context.Context, in *pb.ReqNil) (*pb.Header, error) {
	return g.cli.GetLastHeader()
}

// GetTransactionByAddr get transaction by address
func (g *Grpc) GetTransactionByAddr(ctx context.Context, in *pb.ReqAddr) (*pb.ReplyTxInfos, error) {
	return g.cli.GetTransactionByAddr(in)
}

// GetHexTxByHash get hex transaction by hash
func (g *Grpc) GetHexTxByHash(ctx context.Context, in *pb.ReqHash) (*pb.HexTx, error) {
	reply, err := g.cli.QueryTx(in)
	if err != nil {
		return nil, err
	}
	tx := reply.GetTx()
	if tx == nil {
		return &pb.HexTx{}, nil
	}
	return &pb.HexTx{Tx: common.ToHex(pb.Encode(reply.GetTx()))}, nil
}

// GetTransactionByHashes get transaction by hashes
func (g *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {
	return g.cli.GetTransactionByHash(in)
}

// GetMemPool get mempool contents
func (g *Grpc) GetMemPool(ctx context.Context, in *pb.ReqGetMempool) (*pb.ReplyTxList, error) {
	return g.cli.GetMempool(in)
}

// GetAccounts get  accounts
func (g *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {
	req := &pb.ReqAccountList{WithoutBalance: false}
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletGetAccountList", req)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.WalletAccounts), nil
}

// NewAccount produce new account
func (g *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "NewAccount", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.WalletAccount), nil
}

// WalletTransactionList transaction list of wallet
func (g *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.WalletTxDetails, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletTransactionList", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.WalletTxDetails), nil
}

// ImportPrivkey import privkey
func (g *Grpc) ImportPrivkey(ctx context.Context, in *pb.ReqWalletImportPrivkey) (*pb.WalletAccount, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletImportPrivkey", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.WalletAccount), nil
}

// SendToAddress send to address of coins
func (g *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletSendToAddress", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.ReplyHash), nil
}

// SetTxFee set tx fee
func (g *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletSetFee", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Reply), nil
}

// SetLabl set labl
func (g *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletSetLabel", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.WalletAccount), nil
}

// MergeBalance merge balance of wallet
func (g *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletMergeBalance", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.ReplyHashes), nil
}

// SetPasswd set password
func (g *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletSetPasswd", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Reply), nil
}

// Lock wallet lock
func (g *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletLock", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Reply), nil
}

// UnLock wallet unlock
func (g *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "WalletUnLock", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Reply), nil
}

// GetPeerInfo get peer information
func (g *Grpc) GetPeerInfo(ctx context.Context, req *pb.P2PGetPeerReq) (*pb.PeerList, error) {
	return g.cli.PeerInfo(req)
}

// GetHeaders return headers
func (g *Grpc) GetHeaders(ctx context.Context, in *pb.ReqBlocks) (*pb.Headers, error) {
	return g.cli.GetHeaders(in)
}

// GetLastMemPool return last mempool contents
func (g *Grpc) GetLastMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	return g.cli.GetLastMempool()
}

// GetProperFee return last mempool proper fee
func (g *Grpc) GetProperFee(ctx context.Context, in *pb.ReqProperFee) (*pb.ReplyProperFee, error) {
	return g.cli.GetProperFee(in)
}

// GetBlockOverview get block overview
// GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)   //add by hyb
func (g *Grpc) GetBlockOverview(ctx context.Context, in *pb.ReqHash) (*pb.BlockOverview, error) {
	return g.cli.GetBlockOverview(in)
}

// GetAddrOverview get address overview
func (g *Grpc) GetAddrOverview(ctx context.Context, in *pb.ReqAddr) (*pb.AddrOverview, error) {
	return g.cli.GetAddrOverview(in)
}

// GetBlockHash get block  hash
func (g *Grpc) GetBlockHash(ctx context.Context, in *pb.ReqInt) (*pb.ReplyHash, error) {
	return g.cli.GetBlockHash(in)
}

// GenSeed seed
func (g *Grpc) GenSeed(ctx context.Context, in *pb.GenSeedLang) (*pb.ReplySeed, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "GenSeed", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.ReplySeed), nil
}

// GetSeed get seed
func (g *Grpc) GetSeed(ctx context.Context, in *pb.GetSeedByPw) (*pb.ReplySeed, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "GetSeed", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.ReplySeed), nil
}

// SaveSeed save seed
func (g *Grpc) SaveSeed(ctx context.Context, in *pb.SaveSeedByPw) (*pb.Reply, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "SaveSeed", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Reply), nil
}

// GetWalletStatus get wallet status
func (g *Grpc) GetWalletStatus(ctx context.Context, in *pb.ReqNil) (*pb.WalletStatus, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "GetWalletStatus", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.WalletStatus), nil
}

// GetBalance get balance
func (g *Grpc) GetBalance(ctx context.Context, in *pb.ReqBalance) (*pb.Accounts, error) {
	reply, err := g.cli.GetBalance(in)
	if err != nil {
		return nil, err
	}
	return &pb.Accounts{Acc: reply}, nil
}

// GetAllExecBalance get balance of exec
func (g *Grpc) GetAllExecBalance(ctx context.Context, in *pb.ReqAllExecBalance) (*pb.AllExecBalance, error) {
	return g.cli.GetAllExecBalance(in)
}

// QueryConsensus query consensus
func (g *Grpc) QueryConsensus(ctx context.Context, in *pb.ChainExecutor) (*pb.Reply, error) {
	msg, err := g.cli.QueryConsensus(in)
	if err != nil {
		return nil, err
	}
	var reply pb.Reply
	reply.IsOk = true
	reply.Msg = pb.Encode(msg)
	return &reply, nil
}

// QueryChain query chain
func (g *Grpc) QueryChain(ctx context.Context, in *pb.ChainExecutor) (*pb.Reply, error) {
	msg, err := g.cli.QueryChain(in)
	if err != nil {
		return nil, err
	}
	var reply pb.Reply
	reply.IsOk = true
	reply.Msg = pb.Encode(msg)
	return &reply, nil
}

// ExecWallet  exec wallet
func (g *Grpc) ExecWallet(ctx context.Context, in *pb.ChainExecutor) (*pb.Reply, error) {
	msg, err := g.cli.ExecWallet(in)
	if err != nil {
		return nil, err
	}
	var reply pb.Reply
	reply.IsOk = true
	reply.Msg = pb.Encode(msg)
	return &reply, nil
}

// DumpPrivkey dump Privkey
func (g *Grpc) DumpPrivkey(ctx context.Context, in *pb.ReqString) (*pb.ReplyString, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "DumpPrivkey", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.ReplyString), nil
}

// DumpPrivkeysFile dumps private key to file.
func (g *Grpc) DumpPrivkeysFile(ctx context.Context, in *pb.ReqPrivkeysFile) (*pb.Reply, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "DumpPrivkeysFile", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Reply), nil
}

// ImportPrivkeysFile imports private key from file.
func (g *Grpc) ImportPrivkeysFile(ctx context.Context, in *pb.ReqPrivkeysFile) (*pb.Reply, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "ImportPrivkeysFile", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Reply), nil
}

// Version version
func (g *Grpc) Version(ctx context.Context, in *pb.ReqNil) (*pb.VersionInfo, error) {

	return g.cli.Version()
}

// IsSync is the sync
func (g *Grpc) IsSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {

	return g.cli.IsSync()
}

// IsNtpClockSync is ntp clock sync
func (g *Grpc) IsNtpClockSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {

	return g.cli.IsNtpClockSync()
}

// NetInfo net information
func (g *Grpc) NetInfo(ctx context.Context, in *pb.P2PGetNetInfoReq) (*pb.NodeNetInfo, error) {

	return g.cli.GetNetInfo(in)
}

// GetFatalFailure return  fatal of failure
func (g *Grpc) GetFatalFailure(ctx context.Context, in *pb.ReqNil) (*pb.Int32, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "GetFatalFailure", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.Int32), nil
}

// CloseQueue close queue
func (g *Grpc) CloseQueue(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	go func() {
		time.Sleep(time.Millisecond * 100)
		_, err := g.cli.CloseQueue()
		if err != nil {
			log.Error("CloseQueue", "Error", err)
		}
	}()

	return &pb.Reply{IsOk: true}, nil
}

// GetLastBlockSequence get last block sequence
func (g *Grpc) GetLastBlockSequence(ctx context.Context, in *pb.ReqNil) (*pb.Int64, error) {
	return g.cli.GetLastBlockSequence()
}

// GetBlockByHashes get block by hashes
func (g *Grpc) GetBlockByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.BlockDetails, error) {
	return g.cli.GetBlockByHashes(in)
}

// GetSequenceByHash get block sequece by hash
func (g *Grpc) GetSequenceByHash(ctx context.Context, in *pb.ReqHash) (*pb.Int64, error) {
	return g.cli.GetSequenceByHash(in)
}

// GetBlockBySeq get block with hash by seq
func (g *Grpc) GetBlockBySeq(ctx context.Context, in *pb.Int64) (*pb.BlockSeq, error) {
	return g.cli.GetBlockBySeq(in)
}

// SignRawTx signature rawtransaction
func (g *Grpc) SignRawTx(ctx context.Context, in *pb.ReqSignRawTx) (*pb.ReplySignRawTx, error) {
	reply, err := g.cli.ExecWalletFunc("wallet", "SignRawTx", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.ReplySignRawTx), nil
}

// QueryRandNum query randHash from ticket
func (g *Grpc) QueryRandNum(ctx context.Context, in *pb.ReqRandHash) (*pb.ReplyHash, error) {
	reply, err := g.cli.Query(in.ExecName, "RandNumHash", in)
	if err != nil {
		return nil, err
	}
	return reply.(*pb.ReplyHash), nil
}

// GetFork get fork height by fork key
func (g *Grpc) GetFork(ctx context.Context, in *pb.ReqKey) (*pb.Int64, error) {
	pb.AssertConfig(g.cli)
	cfg := g.cli.GetConfig()
	keys := strings.Split(string(in.Key), "-")
	if len(keys) == 2 {
		return &pb.Int64{Data: cfg.GetDappFork(keys[0], keys[1])}, nil
	}
	return &pb.Int64{Data: cfg.GetFork(string(in.Key))}, nil
}

// GetParaTxByTitle 通过seq以及title获取对应平行连的交易
func (g *Grpc) GetParaTxByTitle(ctx context.Context, in *pb.ReqParaTxByTitle) (*pb.ParaTxDetails, error) {
	return g.cli.GetParaTxByTitle(in)
}

// LoadParaTxByTitle //获取拥有此title交易的区块高度
func (g *Grpc) LoadParaTxByTitle(ctx context.Context, in *pb.ReqHeightByTitle) (*pb.ReplyHeightByTitle, error) {
	return g.cli.LoadParaTxByTitle(in)
}

// GetParaTxByHeight //通过区块高度列表+title获取平行链交易
func (g *Grpc) GetParaTxByHeight(ctx context.Context, in *pb.ReqParaTxByHeight) (*pb.ParaTxDetails, error) {
	return g.cli.GetParaTxByHeight(in)
}

// GetAccount 通过地址标签获取账户地址以及账户余额信息
func (g *Grpc) GetAccount(ctx context.Context, in *pb.ReqGetAccount) (*pb.WalletAccount, error) {
	acc, err := g.cli.ExecWalletFunc("wallet", "WalletGetAccount", in)
	if err != nil {
		return nil, err
	}

	return acc.(*pb.WalletAccount), nil
}

// GetServerTime get server time
func (g *Grpc) GetServerTime(ctx context.Context, in *pb.ReqNil) (*pb.ServerTime, error) {
	serverTime := &pb.ServerTime{
		CurrentTimestamp: pb.Now().Unix(),
	}
	return serverTime, nil
}

// GetCryptoList 获取加密算法列表
func (g *Grpc) GetCryptoList(ctx context.Context, in *pb.ReqNil) (*pb.CryptoList, error) {
	return g.cli.GetCryptoList(), nil
}

// GetAddressDrivers 获取已注册地址插件
func (g *Grpc) GetAddressDrivers(ctx context.Context, in *pb.ReqNil) (*pb.AddressDrivers, error) {
	return g.cli.GetAddressDrivers(), nil
}

// SendDelayTransaction send delay tx
func (g *Grpc) SendDelayTransaction(ctx context.Context, in *pb.DelayTx) (*pb.Reply, error) {
	return g.cli.SendDelayTx(in, true)
}

// GetWalletRecoverAddress get recover addr
func (g *Grpc) GetWalletRecoverAddress(ctx context.Context, in *pb.ReqGetWalletRecoverAddr) (*pb.ReplyString, error) {
	return g.cli.GetWalletRecoverAddr(in)
}

// SignWalletRecoverTx sign wallet recover tx
func (g *Grpc) SignWalletRecoverTx(ctx context.Context, in *pb.ReqSignWalletRecoverTx) (*pb.ReplySignRawTx, error) {
	return g.cli.SignWalletRecoverTx(in)
}

// GetFinalizedBlock get finalized block choice
func (g *Grpc) GetFinalizedBlock(ctx context.Context, in *pb.ReqNil) (*pb.SnowChoice, error) {

	choice, err := g.cli.GetFinalizedBlock()
	if err != nil {
		return nil, err
	}

	return choice, nil
}

// GetChainConfig 获取chain config 参数
func (g *Grpc) GetChainConfig(ctx context.Context, in *pb.ReqNil) (*pb.ChainConfigInfo, error) {
	cfg := g.cli.GetConfig()
	currentH := int64(0)
	lastH, err := g.GetLastHeader(ctx, in)
	if err == nil {
		currentH = lastH.GetHeight()
	}
	return &pb.ChainConfigInfo{
		Title:            cfg.GetTitle(),
		CoinExec:         cfg.GetCoinExec(),
		CoinSymbol:       cfg.GetCoinSymbol(),
		CoinPrecision:    cfg.GetCoinPrecision(),
		TokenPrecision:   cfg.GetTokenPrecision(),
		ChainID:          cfg.GetChainID(),
		MaxTxFee:         cfg.GetMaxTxFee(currentH),
		MinTxFeeRate:     cfg.GetMinTxFeeRate(),
		MaxTxFeeRate:     cfg.GetMaxTxFeeRate(),
		IsPara:           cfg.IsPara(),
		DefaultAddressID: address.GetDefaultAddressID(),
	}, nil
}

// ConvertExectoAddr 根据执行器的名字创建地址
func (g *Grpc) ConvertExectoAddr(ctx context.Context, in *pb.ReqString) (*pb.ReplyString, error) {
	addr := address.ExecAddress(in.GetData())
	return &pb.ReplyString{Data: addr}, nil
}

// GetCoinSymbol get coin symbol
func (g *Grpc) GetCoinSymbol(ctx context.Context, in *pb.ReqNil) (*pb.ReplyString, error) {
	return &pb.ReplyString{Data: g.cli.GetConfig().GetCoinSymbol()}, nil
}

// GetBlockSequences ...
func (g *Grpc) GetBlockSequences(ctx context.Context, in *pb.ReqBlocks) (*pb.BlockSequences, error) {
	return g.cli.GetBlockSequences(in)
}

// AddPushSubscribe ...
func (g *Grpc) AddPushSubscribe(ctx context.Context, in *pb.PushSubscribeReq) (*pb.ReplySubscribePush, error) {
	return g.cli.AddPushSubscribe(in)
}

// ListPushes  列举推送服务
func (g *Grpc) ListPushes(ctx context.Context, in *pb.ReqNil) (*pb.PushSubscribes, error) {
	resp, err := g.cli.ListPushes()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPushSeqLastNum  Get Seq Call Back Last Num
func (g *Grpc) GetPushSeqLastNum(ctx context.Context, in *pb.ReqString) (*pb.Int64, error) {
	return g.cli.GetPushSeqLastNum(in)
}

// SubEvent 订阅消息推送服务
func (g *Grpc) SubEvent(in *pb.ReqSubscribe, resp pb.Chain33_SubEventServer) error {
	sub := g.hashTopic(in.Name)
	dataChan := make(chan *queue.Message, 128)
	if sub == nil {
		sub = &subInfo{topic: in.GetName(), subType: (PushType)(in.Type).string(), subChan: make(map[chan *queue.Message]string), since: time.Now()}
		//相关订阅信息加入到缓存中
		g.addSubInfo(sub)
		g.addSubChan(in.GetName(), dataChan)
		var subReq pb.PushSubscribeReq
		subReq.Encode = "grpc"
		subReq.Name = in.GetName()
		subReq.Contract = in.GetContract()
		subReq.Type = in.GetType()
		if in.GetFromBlock() != 0 { //从指定高度订阅
			hash, err := g.GetBlockHash(context.Background(), &pb.ReqInt{Height: in.GetFromBlock()})
			if err != nil {
				return err
			}
			seqs, err := g.GetSequenceByHash(context.Background(), &pb.ReqHash{Hash: hash.Hash})
			if err != nil {
				return err
			}
			subReq.LastHeight = in.GetFromBlock()
			subReq.LastBlockHash = common.HashHex(hash.Hash)
			subReq.LastSequence = seqs.Data
		}
		reply, err := g.cli.AddPushSubscribe(&subReq)
		if err != nil {
			log.Error("grpc SubEvent", "AddPushSubscribe", err)
			g.delSubInfo(in.GetName(), nil)
			return err
		}
		if !reply.GetIsOk() {
			g.delSubInfo(in.GetName(), nil)
			return errors.New(reply.GetMsg())
		}

	} else {
		g.addSubChan(in.GetName(), dataChan)
	}
	defer func() {
		g.delSubInfo(in.GetName(), dataChan)
	}()
	var err error
	for msg := range dataChan {

		pushData, ok := msg.GetData().(*pb.PushData)
		if ok {
			err = resp.Send(pushData)
			if err != nil {
				log.Error("grpc send ", "err:", err.Error())
				break
			}
		} else {
			log.Error("grpc SubEvent", msg)
		}
	}
	return err
}

// UnSubEvent 取消订阅
func (g *Grpc) UnSubEvent(ctx context.Context, in *pb.ReqString) (*pb.Reply, error) {
	//删除缓存的TopicID
	err := g.delSubInfo(in.GetData(), nil)
	if err != nil {
		return nil, err
	}
	//TODO 发送指令给blockchain 停止推送
	return &pb.Reply{IsOk: true}, nil
}
