// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"time"

	"strings"

	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
)

// SendTransaction send transaction by network
func (g *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {
	return g.cli.SendTx(in)
}

// CreateNoBalanceTransaction create transaction with no balance
func (g *Grpc) CreateNoBalanceTransaction(ctx context.Context, in *pb.NoBalanceTx) (*pb.ReplySignRawTx, error) {
	reply, err := g.cli.CreateNoBalanceTransaction(in)
	if err != nil {
		return nil, err
	}
	tx := pb.Encode(reply)
	return &pb.ReplySignRawTx{TxHex: hex.EncodeToString(tx)}, nil
}

// CreateRawTransaction create rawtransaction of grpc
func (g *Grpc) CreateRawTransaction(ctx context.Context, in *pb.CreateTx) (*pb.UnsignTx, error) {
	reply, err := g.cli.CreateRawTransaction(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

// ReWriteRawTx re-write raw tx parameters of grpc
func (g *Grpc) ReWriteRawTx(ctx context.Context, in *pb.ReWriteRawTx) (*pb.UnsignTx, error) {
	reply, err := g.cli.ReWriteRawTx(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

// CreateTransaction create transaction of grpc
func (g *Grpc) CreateTransaction(ctx context.Context, in *pb.CreateTxIn) (*pb.UnsignTx, error) {
	execer := pb.ExecName(string(in.Execer))
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
	reply, err := pb.CallCreateTx(execer, in.ActionName, msg)
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
	return &pb.HexTx{Tx: hex.EncodeToString(pb.Encode(reply.GetTx()))}, nil
}

// GetTransactionByHashes get transaction by hashes
func (g *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {
	return g.cli.GetTransactionByHash(in)
}

// GetMemPool get mempool contents
func (g *Grpc) GetMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	return g.cli.GetMempool()
}

// GetAccounts get  accounts
func (g *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {
	req := &pb.ReqAccountList{WithoutBalance: false}
	return g.cli.WalletGetAccountList(req)
}

// NewAccount produce new account
func (g *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {
	return g.cli.NewAccount(in)
}

// WalletTransactionList transaction list of wallet
func (g *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.WalletTxDetails, error) {
	return g.cli.WalletTransactionList(in)
}

// ImportPrivkey import privkey
func (g *Grpc) ImportPrivkey(ctx context.Context, in *pb.ReqWalletImportPrivkey) (*pb.WalletAccount, error) {
	return g.cli.WalletImportprivkey(in)
}

// SendToAddress send to address of coins
func (g *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {
	return g.cli.WalletSendToAddress(in)
}

// SetTxFee set tx fee
func (g *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {
	return g.cli.WalletSetFee(in)
}

// SetLabl set labl
func (g *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {
	return g.cli.WalletSetLabel(in)
}

// MergeBalance merge balance of wallet
func (g *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {
	return g.cli.WalletMergeBalance(in)
}

// SetPasswd set password
func (g *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {
	return g.cli.WalletSetPasswd(in)
}

// Lock wallet lock
func (g *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	return g.cli.WalletLock()
}

// UnLock wallet unlock
func (g *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {
	return g.cli.WalletUnLock(in)
}

// GetPeerInfo get peer information
func (g *Grpc) GetPeerInfo(ctx context.Context, in *pb.ReqNil) (*pb.PeerList, error) {
	return g.cli.PeerInfo()
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
func (g *Grpc) GetProperFee(ctx context.Context, in *pb.ReqNil) (*pb.ReplyProperFee, error) {
	return g.cli.GetProperFee()
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
	return g.cli.GenSeed(in)
}

// GetSeed get seed
func (g *Grpc) GetSeed(ctx context.Context, in *pb.GetSeedByPw) (*pb.ReplySeed, error) {
	return g.cli.GetSeed(in)
}

// SaveSeed save seed
func (g *Grpc) SaveSeed(ctx context.Context, in *pb.SaveSeedByPw) (*pb.Reply, error) {
	return g.cli.SaveSeed(in)
}

// GetWalletStatus get wallet status
func (g *Grpc) GetWalletStatus(ctx context.Context, in *pb.ReqNil) (*pb.WalletStatus, error) {
	return g.cli.GetWalletStatus()
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

	return g.cli.DumpPrivkey(in)
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
func (g *Grpc) NetInfo(ctx context.Context, in *pb.ReqNil) (*pb.NodeNetInfo, error) {

	return g.cli.GetNetInfo()
}

// GetFatalFailure return  fatal of failure
func (g *Grpc) GetFatalFailure(ctx context.Context, in *pb.ReqNil) (*pb.Int32, error) {
	return g.cli.GetFatalFailure()
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
	return g.cli.SignRawTx(in)
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
	keys := strings.Split(string(in.Key), "-")
	if len(keys) == 2 {
		return &pb.Int64{Data: pb.GetDappFork(keys[0], keys[1])}, nil
	}
	return &pb.Int64{Data: pb.GetFork(string(in.Key))}, nil
}
