package rpc

import (
	"gitlab.33.cn/chain33/chain33/common"
	context "golang.org/x/net/context"
)

// 显示指定地址的公钥对信息，可以作为后续交易参数
func (g *Grpc) ShowPrivacyKey(ctx context.Context, in *pb.ReqString) (*pb.ReplyPrivacyPkPair, error) {
	return g.cli.ShowPrivacyKey(in)

}

// 创建一系列UTXO
func (g *Grpc) CreateUTXOs(ctx context.Context, in *pb.ReqCreateUTXOs) (*pb.Reply, error) {
	return g.cli.CreateUTXOs(in)
}

// 将资金从公开到隐私转移
func (g *Grpc) MakeTxPublic2Privacy(ctx context.Context, in *pb.ReqPub2Pri) (*pb.Reply, error) {
	return g.cli.Publick2Privacy(in)
}

// 将资产从隐私到隐私进行转移
func (g *Grpc) MakeTxPrivacy2Privacy(ctx context.Context, in *pb.ReqPri2Pri) (*pb.Reply, error) {
	return g.cli.Privacy2Privacy(in)
}

// 将资产从隐私到公开进行转移
func (g *Grpc) MakeTxPrivacy2Public(ctx context.Context, in *pb.ReqPri2Pub) (*pb.Reply, error) {
	return g.cli.Privacy2Public(in)
}

// 扫描UTXO以及获取扫描UTXO后的状态
func (g *Grpc) RescanUtxos(ctx context.Context, in *pb.ReqRescanUtxos) (*pb.RepRescanUtxos, error) {
	return g.cli.RescanUtxos(in)
}

// 使能隐私账户
func (g *Grpc) EnablePrivacy(ctx context.Context, in *pb.ReqEnablePrivacy) (*pb.RepEnablePrivacy, error) {
	return g.cli.EnablePrivacy(in)
}

func (c *Chain33) ShowPrivacyAccountInfo(in types.ReqPPrivacyAccount, result *interface{}) error {
	reply, err := c.cli.ShowPrivacyAccountInfo(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

/////////////////privacy///////////////
func (c *Chain33) ShowPrivacyAccountSpend(in types.ReqPrivBal4AddrToken, result *interface{}) error {
	account, err := c.cli.ShowPrivacyAccountSpend(&in)
	if err != nil {
		log.Info("ShowPrivacyAccountSpend", "return err info", err)
		return err
	}
	*result = account
	return nil
}

func (c *Chain33) ShowPrivacykey(in types.ReqStr, result *interface{}) error {
	reply, err := c.cli.ShowPrivacyKey(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) MakeTxPublic2privacy(in types.ReqPub2Pri, result *interface{}) error {
	reply, err := c.cli.Publick2Privacy(&in)
	if err != nil {
		return err
	}

	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Chain33) MakeTxPrivacy2privacy(in types.ReqPri2Pri, result *interface{}) error {
	reply, err := c.cli.Privacy2Privacy(&in)
	if err != nil {
		return err
	}

	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Chain33) MakeTxPrivacy2public(in types.ReqPri2Pub, result *interface{}) error {
	reply, err := c.cli.Privacy2Public(&in)
	if err != nil {
		return err
	}
	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Chain33) CreateUTXOs(in types.ReqCreateUTXOs, result *interface{}) error {

	reply, err := c.cli.CreateUTXOs(&in)
	if err != nil {
		return err
	}
	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

// PrivacyTxList get all privacy transaction list by param
func (c *Chain33) PrivacyTxList(in *types.ReqPrivacyTransactionList, result *interface{}) error {
	reply, err := c.cli.PrivacyTransactionList(in)
	if err != nil {
		return err
	}
	{
		var txdetails rpctypes.WalletTxDetails
		c.convertWalletTxDetailToJson(reply, &txdetails)
		*result = &txdetails
	}
	return nil
}

func (c *Chain33) RescanUtxos(in types.ReqRescanUtxos, result *interface{}) error {
	reply, err := c.cli.RescanUtxos(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Chain33) EnablePrivacy(in types.ReqEnablePrivacy, result *interface{}) error {
	reply, err := c.cli.EnablePrivacy(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}
