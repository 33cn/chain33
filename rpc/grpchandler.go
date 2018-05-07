package rpc

import (
	"encoding/hex"
	"fmt"
	"strings"

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
	return g.cli.SendRawTransaction(in)
}

func (g *Grpc) QueryTransaction(ctx context.Context, in *pb.ReqHash) (*pb.TransactionDetail, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.QueryTx(in)
}

func (g *Grpc) GetBlocks(ctx context.Context, in *pb.ReqBlocks) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}

	reply, err := g.cli.GetBlocks(&pb.ReqBlocks{in.Start, in.End, in.Isdetail, []string{""}})
	if err != nil {
		return nil, err
	}
	return &pb.Reply{
			IsOk: true,
			Msg:  pb.Encode(reply)},
		nil
}

func (g *Grpc) GetLastHeader(ctx context.Context, in *pb.ReqNil) (*pb.Header, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetLastHeader()
}

func (g *Grpc) GetTransactionByAddr(ctx context.Context, in *pb.ReqAddr) (*pb.ReplyTxInfos, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetTransactionByAddr(in)
}

func (g *Grpc) GetHexTxByHash(ctx context.Context, in *pb.ReqHash) (*pb.HexTx, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
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

func (g *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetTransactionByHash(in)
}

func (g *Grpc) GetMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetMempool()
}

func (g *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletGetAccountList()
}

func (g *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.NewAccount(in)
}

func (g *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.WalletTxDetails, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletTransactionList(in)
}

func (g *Grpc) ImportPrivKey(ctx context.Context, in *pb.ReqWalletImportPrivKey) (*pb.WalletAccount, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletImportprivkey(in)
}

func (g *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletSendToAddress(in)
}

func (g *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletSetFee(in)
}

func (g *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletSetLabel(in)
}

func (g *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletMergeBalance(in)
}

func (g *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletSetPasswd(in)
}

func (g *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletLock()
}

func (g *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletUnLock(in)
}

func (g *Grpc) GetPeerInfo(ctx context.Context, in *pb.ReqNil) (*pb.PeerList, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.PeerInfo()
}

func (g *Grpc) GetHeaders(ctx context.Context, in *pb.ReqBlocks) (*pb.Headers, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetHeaders(in)
}

func (g *Grpc) GetLastMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetLastMempool()
}

//add by hyb
//GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (g *Grpc) GetBlockOverview(ctx context.Context, in *pb.ReqHash) (*pb.BlockOverview, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetBlockOverview(in)
}

func (g *Grpc) GetAddrOverview(ctx context.Context, in *pb.ReqAddr) (*pb.AddrOverview, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetAddrOverview(in)
}

func (g *Grpc) GetBlockHash(ctx context.Context, in *pb.ReqInt) (*pb.ReplyHash, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetBlockHash(in)
}

//seed
func (g *Grpc) GenSeed(ctx context.Context, in *pb.GenSeedLang) (*pb.ReplySeed, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GenSeed(in)
}

func (g *Grpc) GetSeed(ctx context.Context, in *pb.GetSeedByPw) (*pb.ReplySeed, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetSeed(in)
}

func (g *Grpc) SaveSeed(ctx context.Context, in *pb.SaveSeedByPw) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.SaveSeed(in)
}

func (g *Grpc) GetWalletStatus(ctx context.Context, in *pb.ReqNil) (*pb.WalletStatus, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetWalletStatus()
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
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
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
	msg, err := g.cli.Query(in)
	if err != nil {
		return nil, err
	}
	var reply pb.Reply
	reply.IsOk = true
	reply.Msg = pb.Encode(*msg)
	return &reply, nil
}

func (g *Grpc) SetAutoMining(ctx context.Context, in *pb.MinerFlag) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.WalletAutoMiner(in)
}

func (g *Grpc) GetTicketCount(ctx context.Context, in *pb.ReqNil) (*pb.Int64, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetTicketCount()
}

func (g *Grpc) DumpPrivkey(ctx context.Context, in *pb.ReqStr) (*pb.ReplyStr, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.DumpPrivkey(in)
}

func (g *Grpc) CloseTickets(ctx context.Context, in *pb.ReqNil) (*pb.ReplyHashes, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.CloseTickets()
}

func (g *Grpc) Version(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.Version()
}

func (g *Grpc) IsSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.IsSync()
}

func (g *Grpc) IsNtpClockSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.IsNtpClockSync()
}

func (g *Grpc) NetInfo(ctx context.Context, in *pb.ReqNil) (*pb.NodeNetInfo, error) {
	if !g.checkWhitlist(ctx) {
		return nil, fmt.Errorf("reject")
	}
	return g.cli.GetNetInfo()
}

func (g *Grpc) checkWhitlist(ctx context.Context) bool {

	getctx, ok := pr.FromContext(ctx)
	if ok {
		remoteaddr := strings.Split(getctx.Addr.String(), ":")[0]
		return checkWhitlist(remoteaddr)
	}
	return false
}
