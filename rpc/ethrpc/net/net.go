package net

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"strconv"
)

type NetApi struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config
}

func NewNetApi(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	p := &NetApi{}
	p.cli.Init(c, api)
	p.cfg = cfg
	return p
}

func (n *NetApi) PeerCount() (string, error) {

	var in = types.P2PGetPeerReq{}
	reply, err := n.cli.PeerInfo(&in)
	if err != nil {
		return "0x0", err
	}

	numPeers := len(reply.Peers)
	return hexutil.EncodeUint64(uint64(numPeers)), nil
}

func (n *NetApi) Listening() (bool, error) {
	return true, nil
}

func (n *NetApi) Version() (string, error) {
	return strconv.FormatInt(int64(n.cfg.GetChainID()), 10), nil
}
