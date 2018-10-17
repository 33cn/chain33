package rpc

import (
	"encoding/hex"
	"time"

	pb "gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
)

func (g *Grpc) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Reply, error) {
	return g.cli.SendTx(in)
}

func (g *Grpc) CreateNoBalanceTransaction(ctx context.Context, in *pb.NoBalanceTx) (*pb.ReplySignRawTx, error) {
	reply, err := g.cli.CreateNoBalanceTransaction(in)
	if err != nil {
		return nil, err
	}
	tx := pb.Encode(reply)
	return &pb.ReplySignRawTx{TxHex: hex.EncodeToString(tx)}, nil
}

func (g *Grpc) CreateRawTransaction(ctx context.Context, in *pb.CreateTx) (*pb.UnsignTx, error) {
	reply, err := g.cli.CreateRawTransaction(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

func (g *Grpc) CreateTransaction(ctx context.Context, in *pb.CreateTxIn) (*pb.UnsignTx, error) {
	exec := pb.LoadExecutorType(string(in.Execer))
	if exec == nil {
		log.Error("callExecNewTx", "Error", "exec not found")
		return nil, pb.ErrNotSupport
	}
	msg, err := exec.GetAction(in.ActionName)
	if err != nil {
		return nil, err
	}
	reply, err := pb.CallCreateTx(string(in.Execer), in.ActionName, msg)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

func (g *Grpc) CreateRawTxGroup(ctx context.Context, in *pb.CreateTransactionGroup) (*pb.UnsignTx, error) {
	reply, err := g.cli.CreateRawTxGroup(in)
	if err != nil {
		return nil, err
	}
	return &pb.UnsignTx{Data: reply}, nil
}

func (g *Grpc) SendRawTransaction(ctx context.Context, in *pb.SignedTx) (*pb.Reply, error) {
	return g.cli.SendRawTransaction(in)
}

func (g *Grpc) QueryTransaction(ctx context.Context, in *pb.ReqHash) (*pb.TransactionDetail, error) {
	return g.cli.QueryTx(in)
}

func (g *Grpc) GetBlocks(ctx context.Context, in *pb.ReqBlocks) (*pb.Reply, error) {
	reply, err := g.cli.GetBlocks(&pb.ReqBlocks{in.Start, in.End, in.IsDetail, []string{""}})
	if err != nil {
		return nil, err
	}
	return &pb.Reply{
			IsOk: true,
			Msg:  pb.Encode(reply)},
		nil
}

func (g *Grpc) GetLastHeader(ctx context.Context, in *pb.ReqNil) (*pb.Header, error) {
	return g.cli.GetLastHeader()
}

func (g *Grpc) GetTransactionByAddr(ctx context.Context, in *pb.ReqAddr) (*pb.ReplyTxInfos, error) {
	return g.cli.GetTransactionByAddr(in)
}

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

func (g *Grpc) GetTransactionByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.TransactionDetails, error) {
	return g.cli.GetTransactionByHash(in)
}

func (g *Grpc) GetMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	return g.cli.GetMempool()
}

func (g *Grpc) GetAccounts(ctx context.Context, in *pb.ReqNil) (*pb.WalletAccounts, error) {
	req := &pb.ReqAccountList{WithoutBalance: false}
	return g.cli.WalletGetAccountList(req)
}

func (g *Grpc) NewAccount(ctx context.Context, in *pb.ReqNewAccount) (*pb.WalletAccount, error) {
	return g.cli.NewAccount(in)
}

func (g *Grpc) WalletTransactionList(ctx context.Context, in *pb.ReqWalletTransactionList) (*pb.WalletTxDetails, error) {
	return g.cli.WalletTransactionList(in)
}

func (g *Grpc) ImportPrivkey(ctx context.Context, in *pb.ReqWalletImportPrivkey) (*pb.WalletAccount, error) {
	return g.cli.WalletImportprivkey(in)
}

func (g *Grpc) SendToAddress(ctx context.Context, in *pb.ReqWalletSendToAddress) (*pb.ReplyHash, error) {
	return g.cli.WalletSendToAddress(in)
}

func (g *Grpc) SetTxFee(ctx context.Context, in *pb.ReqWalletSetFee) (*pb.Reply, error) {
	return g.cli.WalletSetFee(in)
}

func (g *Grpc) SetLabl(ctx context.Context, in *pb.ReqWalletSetLabel) (*pb.WalletAccount, error) {
	return g.cli.WalletSetLabel(in)
}

func (g *Grpc) MergeBalance(ctx context.Context, in *pb.ReqWalletMergeBalance) (*pb.ReplyHashes, error) {
	return g.cli.WalletMergeBalance(in)
}

func (g *Grpc) SetPasswd(ctx context.Context, in *pb.ReqWalletSetPasswd) (*pb.Reply, error) {
	return g.cli.WalletSetPasswd(in)
}

func (g *Grpc) Lock(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	return g.cli.WalletLock()
}

func (g *Grpc) UnLock(ctx context.Context, in *pb.WalletUnLock) (*pb.Reply, error) {
	return g.cli.WalletUnLock(in)
}

func (g *Grpc) GetPeerInfo(ctx context.Context, in *pb.ReqNil) (*pb.PeerList, error) {
	return g.cli.PeerInfo()
}

func (g *Grpc) GetHeaders(ctx context.Context, in *pb.ReqBlocks) (*pb.Headers, error) {
	return g.cli.GetHeaders(in)
}

func (g *Grpc) GetLastMemPool(ctx context.Context, in *pb.ReqNil) (*pb.ReplyTxList, error) {
	return g.cli.GetLastMempool()
}

//add by hyb
//GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
func (g *Grpc) GetBlockOverview(ctx context.Context, in *pb.ReqHash) (*pb.BlockOverview, error) {
	return g.cli.GetBlockOverview(in)
}

func (g *Grpc) GetAddrOverview(ctx context.Context, in *pb.ReqAddr) (*pb.AddrOverview, error) {
	return g.cli.GetAddrOverview(in)
}

func (g *Grpc) GetBlockHash(ctx context.Context, in *pb.ReqInt) (*pb.ReplyHash, error) {
	return g.cli.GetBlockHash(in)
}

//seed
func (g *Grpc) GenSeed(ctx context.Context, in *pb.GenSeedLang) (*pb.ReplySeed, error) {
	return g.cli.GenSeed(in)
}

func (g *Grpc) GetSeed(ctx context.Context, in *pb.GetSeedByPw) (*pb.ReplySeed, error) {
	return g.cli.GetSeed(in)
}

func (g *Grpc) SaveSeed(ctx context.Context, in *pb.SaveSeedByPw) (*pb.Reply, error) {
	return g.cli.SaveSeed(in)
}

func (g *Grpc) GetWalletStatus(ctx context.Context, in *pb.ReqNil) (*pb.WalletStatus, error) {
	return g.cli.GetWalletStatus()
}

func (g *Grpc) GetBalance(ctx context.Context, in *pb.ReqBalance) (*pb.Accounts, error) {
	reply, err := g.cli.GetBalance(in)
	if err != nil {
		return nil, err
	}
	return &pb.Accounts{Acc: reply}, nil
}

func (g *Grpc) GetAllExecBalance(ctx context.Context, in *pb.ReqAddr) (*pb.AllExecBalance, error) {
	return g.cli.GetAllExecBalance(in)
}

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

func (g *Grpc) DumpPrivkey(ctx context.Context, in *pb.ReqString) (*pb.ReplyString, error) {

	return g.cli.DumpPrivkey(in)
}

func (g *Grpc) Version(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {

	return g.cli.Version()
}

func (g *Grpc) IsSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {

	return g.cli.IsSync()
}

func (g *Grpc) IsNtpClockSync(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {

	return g.cli.IsNtpClockSync()
}

func (g *Grpc) NetInfo(ctx context.Context, in *pb.ReqNil) (*pb.NodeNetInfo, error) {

	return g.cli.GetNetInfo()
}

func (g *Grpc) GetFatalFailure(ctx context.Context, in *pb.ReqNil) (*pb.Int32, error) {
	return g.cli.GetFatalFailure()
}

func (g *Grpc) CloseQueue(ctx context.Context, in *pb.ReqNil) (*pb.Reply, error) {
	go func() {
		time.Sleep(time.Millisecond * 100)
		g.cli.CloseQueue()
	}()

	return &pb.Reply{IsOk: true}, nil
}

func (g *Grpc) GetLastBlockSequence(ctx context.Context, in *pb.ReqNil) (*pb.Int64, error) {
	return g.cli.GetLastBlockSequence()
}
func (g *Grpc) GetBlockSequences(ctx context.Context, in *pb.ReqBlocks) (*pb.BlockSequences, error) {
	return g.cli.GetBlockSequences(in)
}
func (g *Grpc) GetBlockByHashes(ctx context.Context, in *pb.ReqHashes) (*pb.BlockDetails, error) {
	return g.cli.GetBlockByHashes(in)
}

func (g *Grpc) SignRawTx(ctx context.Context, in *pb.ReqSignRawTx) (*pb.ReplySignRawTx, error) {
	return g.cli.SignRawTx(in)
}
