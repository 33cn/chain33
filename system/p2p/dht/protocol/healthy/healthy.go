package healthy

import (
	"context"
	"errors"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (p *Protocol) updateFallBehind() {
	//全节点不参与分布式存储，因此不需要更新
	if p.SubConfig.IsFullNode {
		return
	}
	maxHeight := p.queryMaxHeight()
	if maxHeight == -1 {
		return
	}
	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		log.Error("updateFallBehind", "getLastHeaderFromBlockchain error", err)
		return
	}

	atomic.StoreInt64(&p.fallBehind, maxHeight-header.Height)
	log.Info("updateFallBehind", "fall behind", maxHeight-header.Height)
}

func (p *Protocol) queryMaxHeight() int64 {
	peers := p.Host.Network().Peers()
	shuffle(peers)

	maxHeight := int64(-1)
	var count int
	for _, pid := range peers {
		header, err := p.getLastHeaderFromPeer(pid)
		if err != nil {
			log.Error("updateFallBehind", "getLastHeaderFromPeer error", err, "pid", pid)
			continue
		}
		if header.Height > maxHeight {
			maxHeight = header.Height
		}
		//最多访问50个节点，不包含请求失败的
		count++
		if count > MaxQuery {
			break
		}
	}
	return maxHeight
}

func (p *Protocol) getLastHeaderFromPeer(pid peer.ID) (*types.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, protocol.GetLastHeader)
	if err != nil {
		return nil, err
	}
	defer protocol.CloseStream(stream)
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
