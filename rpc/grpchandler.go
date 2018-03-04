package rpc

import (
	"encoding/hex"
	"fmt"

	"code.aliyun.com/chain33/chain33/types"
	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
)

type Grpc rpcServer

func (g *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {
	reply := g.server.SendTx(in)
	if reply.GetData().(*pb.Reply).IsOk {
		return reply.GetData().(*pb.Reply), nil
	} else {
		return nil, fmt.Errorf(string(reply.GetData().(*pb.Reply).Msg))
	}
}

func (g *Grpc) CreateRawTransaction(ctx context.Context, in *pb.CreateTx) (*pb.UnsignTx, error) {
	reply, err := g.server.CreateRawTransaction(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

func (g *Grpc) SendRawTransaction(ctx context.Context, in *pb.SignedTx) (*pb.Reply, error) {
	reply := g.server.SendRawTransaction(in)
	if reply.GetData().(*pb.Reply).IsOk {
		return reply.GetData().(*pb.Reply), nil
	} else {
		return nil, fmt.Errorf(string(reply.GetData().(*pb.Reply).Msg))
	}

}

func (g *Grpc) QueryTransaction(ctx context.Context, in *pb.ReqHash) (*pb.TransactionDetail, error) {

	return g.server.QueryTx(in.Hash)
}

func (g *Grpc) GetBlocks(ctx context.Context, in *pb.ReqBlocks) (*pb.Reply, error) {

	reply, err := g.server.GetBlocks(in.Start, in.End, in.Isdetail)
	if err != nil {
		return nil, err
	}

	return &pb.Reply{IsOk: true, Msg: (interface{}(reply)).(*pb.Reply).Msg}, nil

}

func (g *Grpc) GetLastHeader(ctx context.Context, in *pb.ReqNil) (*pb.Header, error) {

	reply, err := g.server.GetLastHeader()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetTransactionByAddr(ctx context.Context, in *pb.ReqAddr) (*pb.ReplyTxInfos, error) {

	reply, err := g.server.GetTxByAddr(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetHexTxByHash(ctx context.Context, in *pb.ReqHash) (*pb.HexTx, error) {
	reply, err := g.server.QueryTx(in.GetHash())
	if err != nil {
		return nil, err
	}
	return &pb.HexTx{Tx: hex.EncodeToString(types.Encode(reply.GetTx()))}, nil
}

func (g *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {

	reply, err := g.server.GetTxByHashes(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {

	reply, err := g.server.GetMempool()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {

	reply, err := g.server.GetAccounts()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {

	reply, err := g.server.NewAccount(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.WalletTxDetails, error) {

	reply, err := g.server.WalletTxList(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) ImportPrivKey(ctx context.Context, in *pb.ReqWalletImportPrivKey) (*pb.WalletAccount, error) {

	reply, err := g.server.ImportPrivkey(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {

	reply, err := g.server.SendToAddress(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {

	reply, err := g.server.SetTxFee(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {

	reply, err := g.server.SetLabl(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {

	reply, err := g.server.MergeBalance(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {

	reply, err := g.server.SetPasswd(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {

	reply, err := g.server.Lock()
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (g *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {

	reply, err := g.server.UnLock(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetPeerInfo(ctx context.Context, in *pb.ReqNil) (*pb.PeerList, error) {

	reply, err := g.server.GetPeerInfo()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetHeaders(ctx context.Context, in *pb.ReqBlocks) (*pb.Headers, error) {
	reply, err := g.server.GetHeaders(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetLastMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	reply, err := g.server.GetLastMemPool(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

//add by hyb
//GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (g *Grpc) GetBlockOverview(ctx context.Context, in *pb.ReqHash) (*pb.BlockOverview, error) {
	reply, err := g.server.GetBlockOverview(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (g *Grpc) GetAddrOverview(ctx context.Context, in *pb.ReqAddr) (*pb.AddrOverview, error) {
	reply, err := g.server.GetAddrOverview(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetBlockHash(ctx context.Context, in *pb.ReqInt) (*pb.ReplyHash, error) {
	reply, err := g.server.GetBlockHash(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

//seed
func (g *Grpc) GenSeed(ctx context.Context, in *pb.GenSeedLang) (*pb.ReplySeed, error) {
	reply, err := g.server.GenSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetSeed(ctx context.Context, in *pb.GetSeedByPw) (*pb.ReplySeed, error) {
	reply, err := g.server.GetSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SaveSeed(ctx context.Context, in *pb.SaveSeedByPw) (*pb.Reply, error) {
	reply, err := g.server.SaveSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetWalletStatus(ctx context.Context, in *pb.ReqNil) (*pb.WalletStatus, error) {

	reply, err := g.server.GetWalletStatus()
	if err != nil {
		return nil, err
	}
	return (*pb.WalletStatus)(reply), nil
}

func (g *Grpc) GetBalance(ctx context.Context, in *pb.ReqBalance) (*pb.Accounts, error) {
	reply, err := g.server.GetBalance(in)
	if err != nil {
		return nil, err
	}
	return &pb.Accounts{Acc: reply}, nil
}

func (g *Grpc) QueryChain(ctx context.Context, in *pb.Query) (*pb.Reply, error) {
	result, err := g.server.QueryHash(in)
	if err != nil {
		return nil, err
	}
	var reply types.Reply
	reply.IsOk = true
	reply.Msg = types.Encode(*result)

	return &reply, nil
}

func (g *Grpc) SetAutoMining(ctx context.Context, in *pb.MinerFlag) (*pb.Reply, error) {
	result, err := g.server.SetAutoMiner(in)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (g *Grpc) GetTicketCount(ctx context.Context, in *types.ReqNil) (*pb.Int64, error) {

	result, err := g.server.GetTicketCount()
	if err != nil {
		return nil, err
	}
	return result, nil

}
func (g *Grpc) DumpPrivkey(ctx context.Context, in *pb.ReqStr) (*pb.ReplyStr, error) {
	result, err := g.server.DumpPrivkey(in)
	if err != nil {
		return nil, err
	}
	return result, nil
}
