package ethrpc

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/33cn/chain33/common"
	"math/big"
	"strconv"
)


type NetApi struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config

}

func NewNetApi( cfg *ctypes.Chain33Config,c queue.Client,api client.QueueProtocolAPI) *NetApi {
	p :=&NetApi{}
	p.cli.Init(c,api)
	p.cfg=cfg
	return p
}

/**
#net_peerCount

req
{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":74}

response
{
    "jsonrpc": "2.0",
    "id": 74,
    "result": "0x01"
}
 */
func (n *NetApi) PeerCount() (string, error) {

	var in  = types.P2PGetPeerReq{}
	reply, err := n.cli.PeerInfo(&in)
	if err != nil {
		return "0x0", err
	}

	numPeers := len(reply.Peers)

	return 	common.ToHex(big.NewInt(int64(numPeers)).Bytes()), nil
}

/**
#net_listening

req
{"jsonrpc":"2.0","method":"net_listening","params":[],"id":74}

response
{
    "jsonrpc": "2.0",
    "id": 74,
    "result": true
}
 */
func (n *NetApi) Listening() (bool, error) {
	return true, nil
}


/**
#net_version

req
{"jsonrpc":"2.0","method":"net_version","params":[],"id":67}

response
{
    "jsonrpc": "2.0",
    "id": 67,
    "result": "0"
}
 */
func (n *NetApi) Version() (string, error) {
	return strconv.FormatInt(int64(n.cfg.GetChainID()), 10),nil
}