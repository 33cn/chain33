package rpc

import (
	"encoding/hex"
	"fmt"
	"strings"

	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/types"
	pb "gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	pr "google.golang.org/grpc/peer"
)

func (g *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.SendTx(in)
}

func (g *Grpc) CreateRawTransaction(ctx context.Context, in *pb.CreateTx) (*pb.UnsignTx, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.CreateRawTransaction(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

func (g *Grpc) SendRawTransaction(ctx context.Context, in *pb.SignedTx) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply := g.cli.SendRawTransaction(in)
	if reply.GetData().(*pb.Reply).IsOk {
		return reply.GetData().(*pb.Reply), nil
	} else {
		return nil, fmt.Errorf(string(reply.GetData().(*pb.Reply).Msg))
	}

}

func (g *Grpc) QueryTransaction(ctx context.Context, in *pb.ReqHash) (*pb.TransactionDetail, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.QueryTx(in.Hash)
}

func (g *Grpc) GetBlocks(ctx context.Context, in *pb.ReqBlocks) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetBlocks(in.Start, in.End, in.Isdetail)
	if err != nil {
		return nil, err
	}

			IsOk: true,

}

func (g *Grpc) GetLastHeader(ctx context.Context, in *pb.ReqNil) (*pb.Header, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetLastHeader()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetTransactionByAddr(ctx context.Context, in *pb.ReqAddr) (*pb.ReplyTxInfos, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetTxByAddr(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetHexTxByHash(ctx context.Context, in *pb.ReqHash) (*pb.HexTx, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.QueryTx(in.GetHash())
	if err != nil {
		return nil, err
	}
	return &pb.HexTx{Tx: hex.EncodeToString(types.Encode(reply.GetTx()))}, nil
}

func (g *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetTxByHashes(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetMempool()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetAccounts()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.NewAccount(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.WalletTxDetails, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.WalletTxList(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) ImportPrivKey(ctx context.Context, in *pb.ReqWalletImportPrivKey) (*pb.WalletAccount, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.ImportPrivkey(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.SendToAddress(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.SetTxFee(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.SetLabl(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.MergeBalance(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.SetPasswd(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.Lock()
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (g *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.UnLock(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetPeerInfo(ctx context.Context, in *pb.ReqNil) (*pb.PeerList, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetPeerInfo()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetHeaders(ctx context.Context, in *pb.ReqBlocks) (*pb.Headers, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetHeaders(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetLastMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetLastMemPool(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

//add by hyb
//GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (g *Grpc) GetBlockOverview(ctx context.Context, in *pb.ReqHash) (*pb.BlockOverview, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetBlockOverview(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
func (g *Grpc) GetAddrOverview(ctx context.Context, in *pb.ReqAddr) (*pb.AddrOverview, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetAddrOverview(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetBlockHash(ctx context.Context, in *pb.ReqInt) (*pb.ReplyHash, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetBlockHash(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

//seed
func (g *Grpc) GenSeed(ctx context.Context, in *pb.GenSeedLang) (*pb.ReplySeed, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GenSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetSeed(ctx context.Context, in *pb.GetSeedByPw) (*pb.ReplySeed, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) SaveSeed(ctx context.Context, in *pb.SaveSeedByPw) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.SaveSeed(in)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (g *Grpc) GetWalletStatus(ctx context.Context, in *pb.ReqNil) (*pb.WalletStatus, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetWalletStatus()
	if err != nil {
		return nil, err
	}
	return (*pb.WalletStatus)(reply), nil
}

func (g *Grpc) GetBalance(ctx context.Context, in *pb.ReqBalance) (*pb.Accounts, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	reply, err := g.cli.GetBalance(in)
	if err != nil {
		return nil, err
	}
	return &pb.Accounts{Acc: reply}, nil
}

func (g *Grpc) GetTokenBalance(ctx context.Context, in *pb.ReqTokenBalance) (*pb.Accounts, error) {
	reply, err := g.cli.GetTokenBalance(in)
	if err != nil {
		return nil, err
	}
	return &pb.Accounts{Acc: reply}, nil
}

func (g *Grpc) QueryChain(ctx context.Context, in *pb.Query) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	result, err := g.cli.QueryHash(in)
	if err != nil {
		return nil, err
	}
	var reply types.Reply
	reply.IsOk = true
	reply.Msg = types.Encode(*result)

	return &reply, nil
}

func (g *Grpc) SetAutoMining(ctx context.Context, in *pb.MinerFlag) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	result, err := g.cli.SetAutoMiner(in)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (g *Grpc) GetTicketCount(ctx context.Context, in *types.ReqNil) (*pb.Int64, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	result, err := g.cli.GetTicketCount()
	if err != nil {
		return nil, err
	}
	return result, nil

}
func (g *Grpc) DumpPrivkey(ctx context.Context, in *pb.ReqStr) (*pb.ReplyStr, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	result, err := g.cli.DumpPrivkey(in)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (g *Grpc) CloseTickets(ctx context.Context, in *pb.ReqNil) (*pb.ReplyHashes, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	result, err := g.cli.CloseTickets()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (g *Grpc) Version(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}

	return &pb.Reply{IsOk: true, Msg: []byte(version.GetVersion())}, nil
}

func (g *Grpc) IsSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}

	return &pb.Reply{IsOk: g.cli.IsSync()}, nil
}

func (g *Grpc) IsNtpClockSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}

	return &pb.Reply{IsOk: g.cli.IsNtpClockSync()}, nil
}

func (g *Grpc) NetInfo(ctx context.Context, in *pb.ReqNil) (*pb.NodeNetInfo, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	resp, err := g.cli.GetNetInfo()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (g *Grpc) checkWhitlist(ctx context.Context) bool {

	getctx, ok := pr.FromContext(ctx)
	if ok {
		remoteaddr := strings.Split(getctx.Addr.String(), ":")[0]
		return checkWhitlist(remoteaddr)
	}
	return false
}
