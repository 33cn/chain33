package rpc

import (
	"gitlab.33.cn/chain33/chain33/common"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
)

// 显示指定地址的公钥对信息，可以作为后续交易参数
func (g *channelClient) ShowPrivacyKey(ctx context.Context, in *types.ReqString) (*pty.ReplyPrivacyPkPair, error) {
	data, err := g.ExecWalletFunc(pty.PrivacyX, "ShowPrivacyKey", in)
	if err != nil {
		return nil, err
	}
	return data.(*pty.ReplyPrivacyPkPair), nil
}

// 创建一系列UTXO
func (g *channelClient) CreateUTXOs(ctx context.Context, in *pty.ReqCreateUTXOs) (*types.Reply, error) {
	data, err := g.ExecWalletFunc(pty.PrivacyX, "CreateUTXOs", in)
	if err != nil {
		return nil, err
	}
	return data.(*types.Reply), nil
}

// 将资金从公开到隐私转移
func (g *channelClient) MakeTxPublic2Privacy(ctx context.Context, in *pty.ReqPub2Pri) (*types.Reply, error) {
	data, err := g.ExecWalletFunc(pty.PrivacyX, "Public2Privacy", in)
	if err != nil {
		return nil, err
	}
	return data.(*types.Reply), nil
}

// 将资产从隐私到隐私进行转移
func (g *channelClient) MakeTxPrivacy2Privacy(ctx context.Context, in *pty.ReqPri2Pri) (*types.Reply, error) {
	data, err := g.ExecWalletFunc(pty.PrivacyX, "Privacy2Privacy", in)
	if err != nil {
		return nil, err
	}
	return data.(*types.Reply), nil
}

// 将资产从隐私到公开进行转移
func (g *channelClient) MakeTxPrivacy2Public(ctx context.Context, in *pty.ReqPri2Pub) (*types.Reply, error) {
	data, err := g.ExecWalletFunc(pty.PrivacyX, "Privacy2Public", in)
	if err != nil {
		return nil, err
	}
	return data.(*types.Reply), nil
}

// 扫描UTXO以及获取扫描UTXO后的状态
func (g *channelClient) RescanUtxos(ctx context.Context, in *pty.ReqRescanUtxos) (*pty.RepRescanUtxos, error) {
	data, err := g.ExecWalletFunc(pty.PrivacyX, "RescanUtxos", in)
	if err != nil {
		return nil, err
	}
	return data.(*pty.RepRescanUtxos), nil
}

// 使能隐私账户
func (g *channelClient) EnablePrivacy(ctx context.Context, in *pty.ReqEnablePrivacy) (*pty.RepEnablePrivacy, error) {
	data, err := g.ExecWalletFunc(pty.PrivacyX, "EnablePrivacy", in)
	if err != nil {
		return nil, err
	}
	return data.(*pty.RepEnablePrivacy), nil
}

func (c *Jrpc) ShowPrivacyAccountInfo(in *pty.ReqPPrivacyAccount, result *interface{}) error {
	//reply, err := c.cli.ExecWalletFunc(pty.PrivacyX, "ShowPrivacyAccountInfo", in)
	reply, err := c.cli.ExecWalletFunc(pty.PrivacyX, "PrivacyAccountInfo", in)
	if err != nil {
		return err
	}
	*result, err = types.PBToJson(reply)
	return err
}

/////////////////privacy///////////////
func (c *Jrpc) ShowPrivacyAccountSpend(in *pty.ReqPrivBal4AddrToken, result *interface{}) error {
	if 0 == len(in.Addr) {
		return types.ErrInvalidParam
	}
	account, err := c.cli.ExecWalletFunc(pty.PrivacyX, "ShowPrivacyAccountSpend", in)
	if err != nil {
		log.Info("ShowPrivacyAccountSpend", "return err info", err)
		return err
	}
	*result = account
	return nil
}

func (c *Jrpc) ShowPrivacykey(in *types.ReqString, result *interface{}) error {
	reply, err := c.cli.ShowPrivacyKey(context.Background(), in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Jrpc) MakeTxPublic2privacy(in *pty.ReqPub2Pri, result *interface{}) error {
	reply, err := c.cli.MakeTxPublic2Privacy(context.Background(), in)
	if err != nil {
		return err
	}

	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Jrpc) MakeTxPrivacy2privacy(in *pty.ReqPri2Pri, result *interface{}) error {
	reply, err := c.cli.MakeTxPrivacy2Privacy(context.Background(), in)
	if err != nil {
		return err
	}

	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Jrpc) MakeTxPrivacy2public(in *pty.ReqPri2Pub, result *interface{}) error {
	reply, err := c.cli.MakeTxPrivacy2Public(context.Background(), in)
	if err != nil {
		return err
	}
	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}

	return nil
}

func (c *Jrpc) CreateUTXOs(in *pty.ReqCreateUTXOs, result *interface{}) error {

	reply, err := c.cli.CreateUTXOs(context.Background(), in)
	if err != nil {
		return err
	}
	*result = rpctypes.ReplyHash{Hash: common.ToHex(reply.GetMsg())}
	return nil
}

// PrivacyTxList get all privacy transaction list by param
func (c *Jrpc) PrivacyTxList(in *pty.ReqPrivacyTransactionList, result *interface{}) error {
	if in.Direction != 0 && in.Direction != 1 {
		return types.ErrInvalidParam
	}
	reply, err := c.cli.ExecWalletFunc(pty.PrivacyX, "PrivacyTransactionList", in)
	if err != nil {
		return err
	}
	var txdetails rpctypes.WalletTxDetails
	err = rpctypes.ConvertWalletTxDetailToJson(reply.(*types.WalletTxDetails), &txdetails)
	if err != nil {
		return err
	}
	*result = &txdetails
	return nil
}

func (c *Jrpc) RescanUtxos(in *pty.ReqRescanUtxos, result *interface{}) error {
	reply, err := c.cli.RescanUtxos(context.Background(), in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (c *Jrpc) EnablePrivacy(in *pty.ReqEnablePrivacy, result *interface{}) error {
	reply, err := c.cli.EnablePrivacy(context.Background(), in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}
