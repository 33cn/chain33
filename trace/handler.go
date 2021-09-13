package trace

import (
	"errors"
	"fmt"
	"github.com/33cn/chain33/types"
	"github.com/gorilla/mux"
	"github.com/hpcloud/tail"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/multiformats/go-multiaddr"
	"io"
	"net/http"
	"os"
)

type peersResponse struct {
	BlackPeers []*types.BlackInfo `json:"blackpeers,omitempty"`
	Peers      []kbucket.PeerInfo `json:"peers,omitempty"`
}

// Peer holds information about a Peer.
type Peer struct {
	Address  string `json:"address"`
	FullNode bool   `json:"fullNode"`
}

type chainStateResponse struct {
	//TODO
}

func (s *Service) blacklistPeersHandler(w http.ResponseWriter, r *http.Request) {

	if s.P2PEnv==nil{
		BadRequest(w, "service init faild")
		return
	}
	OK(w, peersResponse{
		BlackPeers: s.ConnBlackList.List().Blackinfo,
	})
}
func (s *Service) peersHandler(w http.ResponseWriter, r *http.Request) {
	//s.RoutingTable.
	OK(w, peersResponse{
		Peers: s.RoutingTable.GetPeerInfos(),
	})
}

func (s *Service) logsHandler(w http.ResponseWriter, r *http.Request) {
	_,err:=os.Stat(s.ChainCfg.GetModuleConfig().Log.LogFile)
	if err!=nil{
		BadRequest(w, err.Error()+fmt.Sprintf("filepath:%s",s.ChainCfg.GetModuleConfig().Log.LogFile))
		return
	}

	seek := &tail.SeekInfo{}
	seek.Offset = 0
	seek.Whence = io.SeekEnd
	//RateLimiter: ratelimiter.NewLeakyBucket(1800, time.Second)
	t, err := tail.TailFile(s.ChainCfg.GetModuleConfig().Log.LogFile, tail.Config{Follow: true,Location: seek})
	if err!=nil{
		BadRequest(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	for line := range t.Lines {

		_, err= w.Write([]byte(line.Text+"\n"))
		if err!=nil{
			BadRequest(w, err.Error())
			return
		}
		w.(http.Flusher).Flush()

	}



	//OK(w, string(line))

}

func (s *Service) peerConnectHandler(w http.ResponseWriter, r *http.Request) {
	//指定一个peer去发起连接
	addr, err := multiaddr.NewMultiaddr("/" + mux.Vars(r)["multi-address"])
	if err != nil {
		BadRequest(w, err)
		return
	}
	log.Info("peerConnectHandler", "addr", addr)
	//TODO 对指定的节点发起连接

	var peerAddr string
	OK(w, peerAddr)
}

// chainStateHandler returns the current chain state.
func (s *Service) chainStateHandler(w http.ResponseWriter, _ *http.Request) {

	OK(w, chainStateResponse{
		//TODO 获取区块链的状态数据
	})
}

//获取当前的拓扑结构数据
func (s *Service) topologyHandler(w http.ResponseWriter, _ *http.Request) {
	//TODO DHT 路由表数据
	BadRequest(w, errors.New("no support"))
}
