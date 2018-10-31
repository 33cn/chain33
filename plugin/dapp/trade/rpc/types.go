package rpc

import (
	ptypes "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type channelClient struct {
	rpctypes.ChannelClient
}

type Jrpc struct {
	cli *channelClient
}

type Grpc struct {
	*channelClient
}

func Init(name string, s rpctypes.RPCServer) {
	cli := &channelClient{}
	grpc := &Grpc{channelClient: cli}
	cli.Init(name, s, &Jrpc{cli: cli}, grpc)
	ptypes.RegisterTradeServer(s.GRPC(), grpc)
}

func (this *Jrpc) GetLastMemPool(in types.ReqNil, result *interface{}) error {
	reply, err := this.cli.GetLastMempool()
	if err != nil {
		return err
	}

	{
		var txlist rpctypes.ReplyTxList
		txs := reply.GetTxs()
		for _, tx := range txs {
			tran, err := rpctypes.DecodeTx(tx)
			if err != nil {
				continue
			}
			txlist.Txs = append(txlist.Txs, tran)
		}
		*result = &txlist
	}
	return nil
}
