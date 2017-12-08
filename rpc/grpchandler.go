package rpc

import (
	"golang.org/x/net/context"

	pb "code.aliyun.com/chain33/chain33/types"
)

type Grpc struct {
	gserver *grpcServer
}

func (req *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {

	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)

	reply := cli.SendTx(in)
	return &pb.Reply{IsOk: true, Msg: reply.GetData().(*pb.Reply).Msg}, reply.Err()

}
func (req *Grpc) QueryTransaction(ctx context.Context, in *pb.ReqHash) (*pb.TransactionDetail, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	return cli.QueryTx(in.Hash)
}

func (req *Grpc) GetBlocks(ctx context.Context, in *pb.ReqBlocks) (*pb.Reply, error) {

	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)

	reply, err := cli.GetBlocks(in.Start, in.End, in.Isdetail)
	if err != nil {
		return nil, err
	}

	return &pb.Reply{IsOk: true, Msg: (interface{}(reply)).(*pb.Reply).Msg}, nil

}

func (req *Grpc) GetLastHeader(ctx context.Context, in *pb.ReqNil) (*pb.Header, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.GetLastHeader()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetTransactionByAddr(ctx context.Context, in *pb.ReqAddr) (*pb.ReplyTxInfo, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.GetTxByAddr(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (req *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.GetTxByHashes(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.GetMempool()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.GetAccounts()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.NewAccount(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.TransactionDetails, error) {

	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.WalletTxList(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) ImportPrivKey(ctx context.Context, in *pb.ReqWalletImportPrivKey) (*pb.WalletAccount, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.ImportPrivkey(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.SendToAddress(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.SetTxFee(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.SetLabl(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.MergeBalance(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.SetPasswd(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.Lock()
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (req *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.UnLock(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (req *Grpc) GetPeerInfo(ctx context.Context, in *pb.ReqNil) (*pb.PeerList, error) {
	cli := NewClient("channel", "")
	cli.SetQueue(req.gserver.q)
	reply, err := cli.GetPeerInfo()
	if err != nil {
		return nil, err
	}

	return reply, nil
}
