package rpc

import (
	"fmt"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/types"
)

type JRpcRequest struct {
	jserver *jsonrpcServer
}

//发送交易信息到topic=rpc 的queue 中

//{"execer":"xxx","palyload":"xx","signature":{"ty":1,"pubkey":"xx","signature":"xxx"},"Fee":12}
type JTransparm struct {
	Execer    string
	Payload   string
	Signature *Signature
	Fee       int64
}
type Signature struct {
	Ty        int32
	Pubkey    string
	Signature string
}

type RawParm struct {
	Data string
}

func (req JRpcRequest) SendTransaction(in RawParm, result *interface{}) error {
	fmt.Println("jrpc transaction:", in)
	var parm types.Transaction
	types.Decode([]byte(in.Data), &parm)
	fmt.Println("parm", parm)
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply := cli.SendTx(&parm)
	*result = string(reply.GetData().(*types.Reply).Msg)
	return reply.Err()

}

type QueryParm struct {
	hash string
}

func (req JRpcRequest) QueryTransaction(in QueryParm, result *interface{}) error {
	var data types.ReqHash

	data.Hash = common.FromHex(in.hash)
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.QueryTx(data.Hash)
	if err != nil {
		return err
	}
	*result = reply
	return nil

}

type BlockParam struct {
	Start    int64
	End      int64
	Isdetail bool
}

func (req JRpcRequest) GetBlocks(in BlockParam, result *interface{}) error {
	var data types.ReqBlocks
	data.End = in.End
	data.Start = in.Start
	data.Isdetail = in.Isdetail
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.GetBlocks(data.Start, data.End, data.Isdetail)
	if err != nil {
		return err
	}
	*result = reply
	return nil

}

func (req JRpcRequest) GetLastHeader(result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.GetLastHeader()
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

type ReqAddr struct {
	Addr string
}

//GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfo, error)
func (req JRpcRequest) GetTxByAddr(in ReqAddr, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	var parm types.ReqAddr
	parm.Addr = in.Addr
	reply, err := cli.GetTxByAddr(&parm)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

/*
GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error)
	GetMempool() (*types.ReplyTxList, error)
	GetAccounts() (*types.WalletAccounts, error)
*/

type ReqHashes struct {
	Hashes []string
}

func (req JRpcRequest) GetTxByHashes(in ReqHashes, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	var parm types.ReqHashes
	parm.Hashes = make([][]byte, 0)
	for _, v := range in.Hashes {
		hb := common.FromHex(v)
		parm.Hashes = append(parm.Hashes, hb)

	}
	reply, err := cli.GetTxByHashes(&parm)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) GetMempool(result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.GetMempool()
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) GetAccounts(in *types.ReqNil, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.GetAccounts()
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

/*
	NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error)
	WalletTxList(parm *types.ReqWalletTransactionList) (*types.TransactionDetails, error)
	ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error)
	SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error)

*/

func (req JRpcRequest) NewAccount(in types.ReqNewAccount, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)

	reply, err := cli.NewAccount(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) WalletTxList(in types.ReqWalletTransactionList, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)

	reply, err := cli.WalletTxList(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) ImportPrivkey(in types.ReqWalletImportPrivKey, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)

	reply, err := cli.ImportPrivkey(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

type ReqWalletSendToAddress struct {
	From   string `protobuf:"bytes,1,opt,name=from" json:"from,omitempty"`
	To     string `protobuf:"bytes,2,opt,name=to" json:"to,omitempty"`
	Amount int64  `protobuf:"varint,3,opt,name=amount" json:"amount,omitempty"`
	Note   string `protobuf:"bytes,4,opt,name=note" json:"note,omitempty"`
}

func (req JRpcRequest) SendToAddress(in types.ReqWalletSendToAddress, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.SendToAddress(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

/*
	SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error)
	SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error)
	MergeBalance(parm *types.ReqWalletMergeBalance) (*types.ReplyHashes, error)
	SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error)
*/

func (req JRpcRequest) SetTxFee(in types.ReqWalletSetFee, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.SetTxFee(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) SetLabl(in types.ReqWalletSetLabel, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.SetLabl(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) MergeBalance(in types.ReqWalletMergeBalance, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.MergeBalance(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) SetPasswd(in types.ReqWalletSetPasswd, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.SetPasswd(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

/*
	Lock() (*types.Reply, error)
	UnLock(parm *types.WalletUnLock) (*types.Reply, error)
	GetPeerInfo() (*types.PeerList, error)
*/

func (req JRpcRequest) Lock(result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.Lock()
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) UnLock(in types.WalletUnLock, result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.UnLock(&in)
	if err != nil {
		return err
	}
	*result = reply
	return nil
}

func (req JRpcRequest) GetPeerInfo(result *interface{}) error {
	cli := NewClient("channel", "")
	cli.SetQueue(req.jserver.q)
	reply, err := cli.GetPeerInfo()
	if err != nil {
		return err
	}
	*result = reply
	return nil
}
