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

func NewAdminApi( cfg *ctypes.Chain33Config,c queue.Client,api client.QueueProtocolAPI) *AdminApi {
	p :=&AdminApi{}
	p.cli.Init(c,api)
	p.cfg=cfg
	return p
}



/**
#admin_peers

req

{
  "method":"admin_peers",
  "params":[],
  "id":1,
  "jsonrpc":"2.0"
}

response

{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        {
            "addr": "192.168.0.19",
            "port": 13803,
            "name": "16Uiu2HAmBdwm5i6Ao6hBedNXHSM44ZUhM4243s5yJGAKPyHRjESw",
            "mempoolSize": 0,
            "self": true,
            "header": {
                "version": 0,
                "parentHash": "0x56a00b884045e7087a6df93b58da8ba06d8127e3f9bf822b3c57271bf493aa62",
                "txHash": "0x91b60784d75497bc7c2ed681e51a985d933ec63ee6adf032ba6024a6adcb3dc9",
                "stateHash": "0x593cfe7c38f21d87f2377983a911601140b679763e7e0439b830eab45dc26187",
                "height": 2,
                "blockTime": 1647411454,
                "txCount": 1,
                "hash": "0x560f19205f79a41bd89ccfc288dc89fe9902c33afafce9a43d00ed5c6c8e082e",
                "difficulty": 0
            },
            "version": "6.0.0-3877-g1405d606a-1405d606@1.0.0",
            "runningTime": "0.044 minutes"
        }
    ]
}
*/

func (p *AdminApi) Peers() ([]*rpctypes.Peer, error) {
	var in  = types.P2PGetPeerReq{}
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


/**
#admin_datadir

req
{
  "method":"admin_datadir",
  "params":[],
  "id":1,
  "jsonrpc":"2.0"
}

response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "datadir"
}

*/

func (p *AdminApi) Datadir() (string, error) {
	mcfg := p.cfg.GetModuleConfig()
	dbpath := mcfg.BlockChain.DbPath
	return dbpath, nil
}


/**
#admin_nodeInfo

req

{
  "method":"admin_nodeInfo",
  "params":[],
  "id":1,
  "jsonrpc":"2.0"
}

response

{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "addr": "192.168.0.19",
        "port": 13803,
        "name": "16Uiu2HAmBdwm5i6Ao6hBedNXHSM44ZUhM4243s5yJGAKPyHRjESw",
        "mempoolSize": 0,
        "self": true,
        "header": {
            "version": 0,
            "parentHash": "0x56a00b884045e7087a6df93b58da8ba06d8127e3f9bf822b3c57271bf493aa62",
            "txHash": "0x91b60784d75497bc7c2ed681e51a985d933ec63ee6adf032ba6024a6adcb3dc9",
            "stateHash": "0x593cfe7c38f21d87f2377983a911601140b679763e7e0439b830eab45dc26187",
            "height": 2,
            "blockTime": 1647411454,
            "txCount": 1,
            "hash": "0x560f19205f79a41bd89ccfc288dc89fe9902c33afafce9a43d00ed5c6c8e082e",
            "difficulty": 0
        },
        "version": "6.0.0-3877-g1405d606a-1405d606@1.0.0",
        "runningTime": "0.262 minutes"
    }
}
 */
func (p *AdminApi)NodeInfo() (*rpctypes.Peer, error)  {

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


