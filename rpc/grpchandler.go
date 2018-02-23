package rpc

import (
	"fmt"

	"code.aliyun.com/chain33/chain33/types"
	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
)

type Grpc struct {
	gserver *grpcServer
	cli     IRClient
}

func (req *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {

	reply := req.cli.SendTx(in)
	if reply.GetData().(*pb.Reply).IsOk {
		return reply.GetData().(*pb.Reply), nil
	} else {
		return nil, fmt.Errorf(string(reply.GetData().(*pb.Reply).Msg))
	}

}

func (req *Grpc) CreateRawTransaction(ctx context.Context, in *pb.CreateTx) (*pb.UnsignTx, error) {
	reply, err := req.cli.CreateRawTransaction(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

func (req *Grpc) SendRawTransaction(ctx context.Context, in *pb.SignedTx) (*pb.Reply, error) {
	reply := req.cli.SendRawTransaction(in)
	if reply.GetData().(*pb.Reply).IsOk {
		return reply.GetData().(*pb.Reply), nil
	} else {
		return nil, fmt.Errorf(string(reply.GetData().(*pb.Reply).Msg))
	}

}
func (req *Grpc) QueryTransaction(ctx context.Context, in *pb.ReqHash) (*pb.TransactionDetail, error) {

	return req.cli.QueryTx(in.Hash)
}

func (req *Grpc) GetBlocks(ctx context.Context, in *pb.ReqBlocks) (*pb.Reply, error) {

	reply, err := req.cli.GetBlocks(in.Start, in.End, in.Isdetail)
	if err != nil {
		return nil, err
	}

	return &pb.Reply{IsOk: true, Msg: (interface{}(reply)).(*pb.Reply).Msg}, nil

}

func (req *Grpc) GetLastHeader(ctx context.Context, in *pb.ReqNil) (*pb.Header, error) {

	reply, err := req.cli.GetLastHeader()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetTransactionByAddr(ctx context.Context, in *pb.ReqAddr) (*pb.ReplyTxInfos, error) {

	reply, err := req.cli.GetTxByAddr(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (req *Grpc) GetHexTxByHash(ctx context.Context, in *pb.ReqHash) (*pb.HexTx, error) {
	reply, err := req.cli.QueryTx(in.GetHash())
	if err != nil {
		return nil, err
	}
	return &pb.HexTx{Tx: reply.GetTx().String()}, nil
}
func (req *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {

	reply, err := req.cli.GetTxByHashes(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {

	reply, err := req.cli.GetMempool()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {

	reply, err := req.cli.GetAccounts()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {

	reply, err := req.cli.NewAccount(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.WalletTxDetails, error) {

	reply, err := req.cli.WalletTxList(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) ImportPrivKey(ctx context.Context, in *pb.ReqWalletImportPrivKey) (*pb.WalletAccount, error) {

	reply, err := req.cli.ImportPrivkey(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {

	reply, err := req.cli.SendToAddress(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {

	reply, err := req.cli.SetTxFee(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {

	reply, err := req.cli.SetLabl(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {

	reply, err := req.cli.MergeBalance(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {

	reply, err := req.cli.SetPasswd(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {

	reply, err := req.cli.Lock()
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (req *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {

	reply, err := req.cli.UnLock(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetPeerInfo(ctx context.Context, in *pb.ReqNil) (*pb.PeerList, error) {

	reply, err := req.cli.GetPeerInfo()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetHeaders(ctx context.Context, in *pb.ReqBlocks) (*pb.Headers, error) {
	reply, err := req.cli.GetHeaders(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetLastMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	reply, err := req.cli.GetLastMemPool(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

//add by hyb
//GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (req *Grpc) GetBlockOverview(ctx context.Context, in *pb.ReqHash) (*pb.BlockOverview, error) {
	reply, err := req.cli.GetBlockOverview(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (req *Grpc) GetAddrOverview(ctx context.Context, in *pb.ReqAddr) (*pb.AddrOverview, error) {
	reply, err := req.cli.GetAddrOverview(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetBlockHash(ctx context.Context, in *pb.ReqInt) (*pb.ReplyHash, error) {
	reply, err := req.cli.GetBlockHash(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

//seed
func (req *Grpc) GenSeed(ctx context.Context, in *pb.GenSeedLang) (*pb.ReplySeed, error) {
	reply, err := req.cli.GenSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetSeed(ctx context.Context, in *pb.GetSeedByPw) (*pb.ReplySeed, error) {
	reply, err := req.cli.GetSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SaveSeed(ctx context.Context, in *pb.SaveSeedByPw) (*pb.Reply, error) {
	reply, err := req.cli.SaveSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetWalletStatus(ctx context.Context, in *pb.ReqNil) (*pb.WalletStatus, error) {

	reply, err := req.cli.GetWalletStatus()
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (req *Grpc) GetBalance(ctx context.Context, in *pb.ReqBalance) (*pb.Accounts, error) {
	reply, err := req.cli.GetBalance(in)
	if err != nil {
		return nil, err
	}
	return &pb.Accounts{Acc: reply}, nil
}

func (req *Grpc) QueryChain(ctx context.Context, in *pb.Query) (*pb.Reply, error) {
	result, err := req.cli.QueryHash(in)
	if err != nil {
		return nil, err
	}
	var reply types.Reply
	reply.IsOk = true
	reply.Msg = types.Encode(*result)

	return &reply, nil
}

func (req *Grpc) SetAutoMining(ctx context.Context, in *pb.MinerFlag) (*pb.Reply, error) {
	result, err := req.cli.SetAutoMiner(in)
	if err != nil {
		return nil, err
	}
	return result, nil
}
