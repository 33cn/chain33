package rpc

import (
	"golang.org/x/net/context"

	pb "code.aliyun.com/chain33/chain33/types"
)

type Grpc struct {
	gserver *grpcServer
	cli     IRClient
}

func (req *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {

	reply := req.cli.SendTx(in)
	return &pb.Reply{IsOk: true, Msg: reply.GetData().(*pb.Reply).Msg}, reply.Err()

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
