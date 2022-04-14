package admin

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	ctypes "github.com/33cn/chain33/types"
)

type AdminApi struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config
}

func NewAdminApi(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	p := &AdminApi{}
	p.cli.Init(c, api)
	p.cfg = cfg
	return p
}

func (p *AdminApi) Peers() ([]*rpctypes.Peer, error) {
	var in = types.P2PGetPeerReq{}
	reply, err := p.cli.PeerInfo(&in)
	if err != nil {
		return nil, err
	}

	var peerlist rpctypes.PeerList
	for _, peer := range reply.Peers {
		var pr rpctypes.Peer
		pr.Addr = peer.GetAddr()
		pr.MempoolSize = peer.GetMempoolSize()
		pr.Name = peer.GetName()
		pr.Port = peer.GetPort()
		pr.Self = peer.GetSelf()
		pr.Header = &rpctypes.Header{
			BlockTime:  peer.Header.GetBlockTime(),
			Height:     peer.Header.GetHeight(),
			ParentHash: common.ToHex(peer.GetHeader().GetParentHash()),
			StateHash:  common.ToHex(peer.GetHeader().GetStateHash()),
			TxHash:     common.ToHex(peer.GetHeader().GetTxHash()),
			Version:    peer.GetHeader().GetVersion(),
			Hash:       common.ToHex(peer.GetHeader().GetHash()),
			TxCount:    peer.GetHeader().GetTxCount(),
		}

		pr.Version = peer.GetVersion()
		pr.LocalDBVersion = peer.GetLocalDBVersion()
		pr.StoreDBVersion = peer.GetStoreDBVersion()
		pr.RunningTime = peer.GetRunningTime()
		pr.FullNode = peer.GetFullNode()
		pr.Blocked = peer.GetBlocked()
		peerlist.Peers = append(peerlist.Peers, &pr)

	}
	return peerlist.Peers, nil
}

func (p *AdminApi) Datadir() (string, error) {
	mcfg := p.cfg.GetModuleConfig()
	dbpath := mcfg.BlockChain.DbPath
	return dbpath, nil
}

func (p *AdminApi) NodeInfo() (*rpctypes.Peer, error) {

	peers, err := p.Peers()
	if err != nil {
		return nil, err
	}

	for _, peer := range peers {
		if peer.Self {
			return peer, nil
		}
	}

	return nil, nil
}
