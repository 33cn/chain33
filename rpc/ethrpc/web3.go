package ethrpc

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Web3 struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config

}

func NewWeb3( cfg *ctypes.Chain33Config,c queue.Client,api client.QueueProtocolAPI) *Web3 {
	w:=&Web3{}
	w.cli.Init(c,api)
	w.cfg=cfg
	return w
}


//Sha3   web3_sha3
//Returns Keccak-256 (not the standardized SHA3-256) of the given data.
func (w*Web3)Sha3(input string)(string,error){
	data,err:= common.FromHex(input)
	if err!=nil{
		return "",err
	}
	return hexutil.Encode(common.Sha3(data)),nil
}