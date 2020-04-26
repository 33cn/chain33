package healthy

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	MaxQuery = 50
)

var log = log15.New("module", "protocol.sync")

type Protocol struct {
	*protocol.P2PEnv //协议共享接口变量
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

func InitProtocol(env *protocol.P2PEnv) {
	s := Protocol{env}
	s.Host.SetStreamHandler(protocol.IsSync, protocol.HandlerWithRW(s.HandleStreamIsSync))
	s.Host.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(s.HandleStreamIsHealthy))
	s.Host.SetStreamHandler(protocol.GetLastHeader, protocol.HandlerWithRW(s.HandleStreamLastHeader))
}

func (p *Protocol) HandleStreamIsSync(req *types.P2PRequest, res *types.P2PResponse) error {
	peers := p.Host.Network().Peers()
	if len(peers) > MaxQuery {
		peers = peers[:MaxQuery]
	}

	maxHeight := int64(-1)
	for _, pid := range peers {
		header, err := p.getLastHeaderFromPeer(pid)
		if err != nil {
			log.Error("HandleStreamIsSync", "getLastHeader error", err, "pid", pid)
			continue
		}
		if header.Height > maxHeight {
			maxHeight = header.Height
		}
	}

	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		return err
	}

	var isSync bool
	if header.Height >= maxHeight {
		isSync = true
	}
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: isSync,
		},
	}
	return nil
}

func (p *Protocol) HandleStreamIsHealthy(req *types.P2PRequest, res *types.P2PResponse) error {
	maxFallBehind := req.Request.(*types.P2PRequest_HealthyHeight).HealthyHeight
	peers := p.Host.Network().Peers()
	if len(peers) > MaxQuery {
		//打乱顺序
		shuffle(peers)
		peers = peers[:MaxQuery]
	}

	maxHeight := int64(-1)
	for _, pid := range peers {
		header, err := p.getLastHeaderFromPeer(pid)
		if err != nil {
			log.Error("HandleStreamIsHealthy", "getLastHeader error", err, "pid", pid)
			continue
		}
		if header.Height > maxHeight {
			maxHeight = header.Height
		}
	}

	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		return err
	}

	var isHealthy bool
	if header.Height >= maxHeight-maxFallBehind {
		isHealthy = true
	}
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: isHealthy,
		},
	}
	return nil
}

func (p *Protocol) HandleStreamLastHeader(req *types.P2PRequest, res *types.P2PResponse) error {
	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		return err
	}
	res.Response = &types.P2PResponse_LastHeader{
		LastHeader: header,
	}
	return nil
}

func (p *Protocol) getLastHeaderFromPeer(pid peer.ID) (*types.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, protocol.GetLastHeader)
	if err != nil {
		return nil, err
	}
	msg := types.P2PRequest{}
	err = protocol.WriteStream(&msg, stream)
	if err != nil {
		return nil, err
	}

	var res types.P2PResponse
	err = protocol.ReadStream(&res, stream)
	if err != nil {
		return nil, err
	}

	if header, ok := res.Response.(*types.P2PResponse_LastHeader); ok {
		return header.LastHeader, nil
	}

	return nil, errors.New(res.Error)
}

func (p *Protocol) getLastHeaderFromBlockChain() (*types.Header, error) {
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	reply, err := p.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if header, ok := reply.Data.(*types.Header); ok {
		return header, nil
	}
	return nil, types2.ErrNotFound
}

func shuffle(slice []peer.ID) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}
