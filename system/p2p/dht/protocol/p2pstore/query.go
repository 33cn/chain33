package p2pstore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

func (p *Protocol) requestPeerInfoForChunk(msg *types.ChunkInfoMsg, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*5)
	defer cancel()
	p.Host.ConnManager().Protect(pid, requestPeerInfoForChunk)
	defer p.Host.ConnManager().Unprotect(pid, requestPeerInfoForChunk)
	stream, err := p.Host.NewStream(ctx, pid, requestPeerInfoForChunk)
	if err != nil {
		return err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	req := types.P2PRequest{
		Request: &types.P2PRequest_ChunkInfoMsg{
			ChunkInfoMsg: msg,
		},
	}

	return protocol.WriteStream(&req, stream)
}

func (p *Protocol) responsePeerInfoForChunk(provider *types.ChunkProvider, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*5)
	defer cancel()
	p.Host.ConnManager().Protect(pid, responsePeerInfoForChunk)
	defer p.Host.ConnManager().Unprotect(pid, responsePeerInfoForChunk)
	stream, err := p.Host.NewStream(ctx, pid, responsePeerInfoForChunk)
	if err != nil {
		return err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	req := types.P2PRequest{
		Request: &types.P2PRequest_Provider{
			Provider: provider,
		},
	}

	return protocol.WriteStream(&req, stream)
}

func (p *Protocol) requestPeerAddr(keyPid, remotePid peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*5)
	defer cancel()
	p.Host.ConnManager().Protect(remotePid, requestPeerAddr)
	defer p.Host.ConnManager().Unprotect(remotePid, requestPeerAddr)
	stream, err := p.Host.NewStream(ctx, remotePid, requestPeerAddr)
	if err != nil {
		return err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	req := types.P2PRequest{
		Request: &types.P2PRequest_Pid{
			Pid: string(keyPid),
		},
	}

	return protocol.WriteStream(&req, stream)
}

func (p *Protocol) responsePeerAddr(info *types.PeerInfo, remotePid peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*5)
	defer cancel()
	p.Host.ConnManager().Protect(remotePid, responsePeerAddr)
	defer p.Host.ConnManager().Unprotect(remotePid, responsePeerAddr)
	stream, err := p.Host.NewStream(ctx, remotePid, responsePeerAddr)
	if err != nil {
		return err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	req := types.P2PRequest{
		Request: &types.P2PRequest_PeerInfo{
			PeerInfo: info,
		},
	}

	return protocol.WriteStream(&req, stream)
}

func (p *Protocol) fetchActivePeers(pid peer.ID, saveAddr bool) ([]peer.ID, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*5)
	defer cancel()
	p.Host.ConnManager().Protect(pid, fetchActivePeer)
	defer p.Host.ConnManager().Unprotect(pid, fetchActivePeer)
	stream, err := p.Host.NewStream(ctx, pid, fetchActivePeer)
	if err != nil {
		return nil, err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer stream.Close()

	var resp types.P2PResponse
	if err = protocol.ReadStream(&resp, stream); err != nil {
		return nil, err
	}
	data, ok := resp.Response.(*types.P2PResponse_PeerInfos)
	if !ok {
		return nil, types2.ErrInvalidMessageType
	}

	if saveAddr {
		return savePeers(data.PeerInfos.PeerInfos, p.Host.Peerstore()), nil
	}

	var peers []peer.ID
	for _, info := range data.PeerInfos.PeerInfos {
		peers = append(peers, peer.ID(info.ID))
	}
	return peers, nil
}

func (p *Protocol) fetchShardPeers(key []byte, count int, pid peer.ID) ([]peer.ID, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*5)
	defer cancel()
	p.Host.ConnManager().Protect(pid, fetchShardPeer)
	defer p.Host.ConnManager().Unprotect(pid, fetchShardPeer)
	stream, err := p.Host.NewStream(ctx, pid, fetchShardPeer)
	if err != nil {
		return nil, err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer stream.Close()
	req := types.P2PRequest{
		Request: &types.P2PRequest_ReqPeers{
			ReqPeers: &types.ReqPeers{
				ReferKey: key,
				Count:    int32(count),
			},
		},
	}
	if err = protocol.WriteStream(&req, stream); err != nil {
		return nil, err
	}
	var resp types.P2PResponse
	if err = protocol.ReadStream(&resp, stream); err != nil {
		return nil, err
	}
	closerPeers := savePeers(resp.CloserPeers, p.Host.Peerstore())
	return closerPeers, nil
}

// findChunk
//	1. 检查本地
//  2. 检查缓存中记录的提供数据的节点
//  3. 异步获取数据
func (p *Protocol) findChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, peer.ID, error) {
	//优先获取本地p2pStore数据
	bodys, _ := p.getChunkBlock(req)
	if bodys != nil {
		return bodys, p.Host.ID(), nil
	}
	//检查缓存
	for _, pid := range p.getChunkProviderCache(req.ChunkHash) {
		if bodys, _, _ := p.fetchChunkFromPeer(p.Ctx, req, pid); bodys != nil {
			return bodys, pid, nil
		}
	}
	// 异步获取数据
	return p.findChunkAsync(req)
}

func (p *Protocol) findChunkAsync(req *types.ChunkInfoMsg) (*types.BlockBodys, peer.ID, error) {
	chunkHash := hex.EncodeToString(req.ChunkHash)
	wakeupCh := make(chan struct{}, 1)
	p.wakeupMutex.Lock()
	p.wakeup[chunkHash] = wakeupCh
	p.wakeupMutex.Unlock()

	// 发起异步请求
	for _, pid := range p.RoutingTable.NearestPeers(genDHTID(req.ChunkHash), AlphaValue) {
		if err := p.requestPeerInfoForChunk(req, pid); err != nil {
			log.Error("findChunkAsync", "requestPeerInfoForChunk error", err, "pid", pid)
		}
	}

	// 等待异步消息
	select {
	case <-wakeupCh:
		for _, pid := range p.getChunkProviderCache(req.ChunkHash) {
			bodys, _, err := p.fetchChunkFromPeer(p.Ctx, req, pid)
			if err != nil {
				log.Error("findChunkAsync", "fetchChunkFromPeer error", err, "pid", pid)
				continue
			}
			if bodys != nil {
				return bodys, pid, nil
			}
		}

	case <-time.After(time.Minute * 3):
		p.wakeupMutex.Lock()
		delete(p.wakeup, chunkHash)
		p.wakeupMutex.Unlock()
		log.Error("findChunkAsync", "error", "timeout", "chunkHash", chunkHash)
		break
	}

	return nil, "", types2.ErrNotFound
}

//getChunk gets chunk data from p2pStore or other peers.
func (p *Protocol) getChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, peer.ID, error) {
	if req == nil {
		return nil, "", types2.ErrInvalidParam
	}
	//优先获取本地p2pStore数据
	bodys, _ := p.getChunkBlock(req)
	if bodys != nil {
		return bodys, p.Host.ID(), nil
	}
	//本地数据不存在或已过期，则向临近节点查询
	return p.mustFetchChunk(req)
}

func (p *Protocol) getHeadersOld(param *types.ReqBlocks) (*types.Headers, peer.ID) {
	req := types.P2PGetHeaders{
		StartHeight: param.Start,
		EndHeight:   param.End,
	}
	for _, peerID := range param.Pid {
		pid, err := peer.Decode(peerID)
		if err != nil {
			log.Error("getHeaders", "decode pid error", err)
			continue
		}
		headers, err := p.getHeadersFromPeerOld(&req, pid)
		if err != nil {
			continue
		}
		return headers, pid
	}
	return nil, ""
}

func (p *Protocol) getHeadersFromPeerOld(req *types.P2PGetHeaders, pid peer.ID) (*types.Headers, error) {
	p.Host.ConnManager().Protect(pid, getHeaderOld)
	defer p.Host.ConnManager().Unprotect(pid, getHeaderOld)
	stream, err := p.Host.NewStream(p.Ctx, pid, getHeaderOld)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	err = protocol.WriteStream(&types.MessageHeaderReq{
		Message: req,
	}, stream)
	if err != nil {
		return nil, err
	}
	var resp types.MessageHeaderResp
	err = protocol.ReadStream(&resp, stream)
	if err != nil {
		return nil, err
	}
	return &types.Headers{
		Items: resp.Message.Headers,
	}, nil
}

func (p *Protocol) getHeaders(param *types.ReqBlocks) (*types.Headers, peer.ID) {
	for _, peerID := range param.Pid {
		pid, err := peer.Decode(peerID)
		if err != nil {
			log.Error("getHeaders", "decode pid error", err)
			continue
		}
		headers, err := p.getHeadersFromPeer(param, pid)
		if err != nil {
			log.Error("getHeaders", "peer", pid, "error", err)
			continue
		}
		return headers, pid
	}

	if len(param.Pid) != 0 {
		return nil, ""
	}

	for _, pid := range p.RoutingTable.ListPeers() {
		headers, err := p.getHeadersFromPeer(param, pid)
		if err != nil {
			log.Error("getHeaders", "peer", pid, "error", err)
			continue
		}
		return headers, pid
	}
	log.Error("getHeaders", "error", types2.ErrNotFound)
	return nil, ""
}

func (p *Protocol) getHeadersFromPeer(param *types.ReqBlocks, pid peer.ID) (*types.Headers, error) {
	childCtx, cancel := context.WithTimeout(p.Ctx, 30*time.Second)
	defer cancel()
	p.Host.ConnManager().Protect(pid, getHeader)
	defer p.Host.ConnManager().Unprotect(pid, getHeader)
	stream, err := p.Host.NewStream(childCtx, pid, getHeader)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ReqBlocks{
			ReqBlocks: param,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream)
	if err != nil {
		log.Error("getHeadersFromPeer", "SignAndWriteStream error", err)
		return nil, err
	}
	var res types.P2PResponse
	err = protocol.ReadStreamAndAuthenticate(&res, stream)
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Response.(*types.P2PResponse_BlockHeaders).BlockHeaders, nil
}

func (p *Protocol) getChunkRecords(param *types.ReqChunkRecords) *types.ChunkRecords {
	for _, sPid := range param.Pid {
		pid, err := peer.Decode(sPid)
		if err != nil {
			continue
		}
		records, err := p.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err, "start", param.Start, "end", param.End)
			continue
		}
		log.Info("getChunkRecords", "peer", pid, "error", err, "start", param.Start, "end", param.End)
		return records
	}

	for _, pid := range p.RoutingTable.ListPeers() {
		records, err := p.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err, "start", param.Start, "end", param.End)
			continue
		}
		log.Info("getChunkRecords", "peer", pid, "error", err, "start", param.Start, "end", param.End)
		return records
	}
	return nil
}

func (p *Protocol) getChunkRecordsFromPeer(param *types.ReqChunkRecords, pid peer.ID) (*types.ChunkRecords, error) {
	childCtx, cancel := context.WithTimeout(p.Ctx, 30*time.Second)
	defer cancel()
	p.Host.ConnManager().Protect(pid, getChunkRecord)
	defer p.Host.ConnManager().Unprotect(pid, getChunkRecord)
	stream, err := p.Host.NewStream(childCtx, pid, getChunkRecord)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ReqChunkRecords{
			ReqChunkRecords: param,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream)
	if err != nil {
		log.Error("getChunkRecordsFromPeer", "SignAndWriteStream error", err)
		return nil, err
	}

	var res types.P2PResponse
	err = protocol.ReadStreamAndAuthenticate(&res, stream)
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Response.(*types.P2PResponse_ChunkRecords).ChunkRecords, nil
}

//若网络中有节点保存了该chunk，该方法可以保证查询到
func (p *Protocol) mustFetchChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, peer.ID, error) {
	//递归查询时间上限5分钟
	ctx, cancel := context.WithTimeout(p.Ctx, time.Minute*5)
	defer cancel()

	chunkHash := hex.EncodeToString(req.ChunkHash)
	//保存查询过的节点，防止重复查询
	searchedPeers := make(map[peer.ID]struct{})
	searchedPeers[p.Host.ID()] = struct{}{}
	peers := p.RoutingTable.NearestPeers(genDHTID(req.ChunkHash), AlphaValue)
	log.Info("into mustFetchChunk", "start", req.Start, "end", req.End)
	for len(peers) != 0 {
		var nearerPeers []peer.ID
		var bodys *types.BlockBodys
		var err error
		for _, pid := range peers {
			if _, ok := searchedPeers[pid]; ok {
				continue
			}
			searchedPeers[pid] = struct{}{}
			start := time.Now()
			bodys, nearerPeers, err = p.fetchChunkFromPeer(ctx, req, pid)
			if err != nil {
				continue
			}
			if bodys != nil {
				log.Info("mustFetchChunk found", "chunk hash", chunkHash, "start", req.Start, "pid", pid, "maddrs", p.Host.Peerstore().Addrs(pid), "time cost", time.Since(start))
				return bodys, pid, nil
			}
			break
		}
		peers = nearerPeers
	}

	log.Error("mustFetchChunk not found", "chunk hash", chunkHash, "start", req.Start, "error", types2.ErrNotFound)

	//如果是分片节点没有在分片网络中找到数据，最后到全节点去请求数据
	ctx2, cancel2 := context.WithTimeout(ctx, time.Minute*3)
	defer cancel2()
	peerInfos, err := p.Discovery.FindPeers(ctx2, fullNode)
	if err != nil {
		log.Error("mustFetchChunk", "Find full peers error", err)
		return nil, "", types2.ErrNotFound
	}

	for addrInfo := range peerInfos {
		if addrInfo.ID == p.Host.ID() {
			continue
		}
		log.Info("mustFetchChunk", "pid", addrInfo.ID, "addrs", addrInfo.Addrs)
		bodys, pid := p.fetchChunkFromFullPeer(ctx, req, addrInfo.ID)
		if bodys == nil {
			log.Error("mustFetchChunk from full node failed", "pid", addrInfo.ID, "chunk hash", chunkHash, "start", req.Start)
			continue
		}
		log.Info("mustFetchChunk from full node succeed", "pid", addrInfo.ID, "chunk hash", chunkHash, "start", req.Start)
		return bodys, pid, nil
	}

	return nil, "", types2.ErrNotFound
}

func (p *Protocol) fetchChunkFromPeer(ctx context.Context, params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, []peer.ID, error) {
	childCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	p.Host.ConnManager().Protect(pid, fetchChunk)
	defer p.Host.ConnManager().Unprotect(pid, fetchChunk)
	stream, err := p.Host.NewStream(childCtx, pid, fetchChunk)
	if err != nil {
		log.Error("fetchChunkFromPeer", "error", err)
		return nil, nil, err
	}
	defer stream.Close()
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ChunkInfoMsg{
			ChunkInfoMsg: params,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream)
	if err != nil {
		log.Error("fetchChunkFromPeer", "SignAndWriteStream error", err)
		return nil, nil, err
	}
	var bodys []*types.BlockBody
	var res types.P2PResponse
	for {
		if err := protocol.ReadStream(&res, stream); err != nil {
			log.Error("fetchChunkFromPeer", "ReadStream error", err)
			return nil, nil, err
		}
		body, ok := res.Response.(*types.P2PResponse_BlockBody)
		if !ok {
			break
		}
		bodys = append(bodys, body.BlockBody)
	}
	closerPeers := savePeers(res.CloserPeers, p.Host.Peerstore())
	if int64(len(bodys)) == params.End-params.Start+1 {
		return &types.BlockBodys{
			Items: bodys,
		}, closerPeers, nil
	}

	if len(closerPeers) == 0 {
		return nil, nil, fmt.Errorf(res.Error)
	}
	return nil, closerPeers, nil
}

func (p *Protocol) fetchChunkFromFullPeer(ctx context.Context, params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, peer.ID) {
	bodys, peers, err := p.fetchChunkFromPeer(ctx, params, pid)
	if err != nil {
		log.Error("fetchChunkFromFullPeer", "error", err)
		return nil, ""
	}
	if bodys != nil {
		return bodys, pid
	}
	for _, pid := range peers {
		bodys, _, err = p.fetchChunkFromPeer(ctx, params, pid)
		if err != nil {
			log.Error("fetchChunkFromFullPeer 2", "error", err, "pid", pid)
			continue
		}
		if bodys != nil {
			return bodys, pid
		}
	}
	return nil, ""
}

func (p *Protocol) storeChunk(req *types.ChunkInfoMsg) error {

	//如果p2pStore已保存数据，只更新时间即可
	if err := p.updateChunk(req); err == nil {
		return nil
	}
	//blockchain通知p2pStore保存数据，则blockchain应该有数据
	bodys, err := p.getChunkFromBlockchain(req)
	if err != nil {
		return err
	}
	err = p.addChunkBlock(req, bodys)
	if err != nil {
		return err
	}
	return nil
}

func (p *Protocol) getChunkFromBlockchain(param *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	if param == nil {
		return nil, types2.ErrInvalidParam
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetChunkBlockBody, param)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := p.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if bodys, ok := resp.GetData().(*types.BlockBodys); ok {
		return bodys, nil
	}
	return nil, types2.ErrNotFound
}

func (p *Protocol) getChunkRecordFromBlockchain(req *types.ReqChunkRecords) (*types.ChunkRecords, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetChunkRecord, req)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := p.QueueClient.WaitTimeout(msg, time.Second)
	if err != nil {
		return nil, err
	}
	if records, ok := resp.GetData().(*types.ChunkRecords); ok {
		return records, nil
	}

	return nil, types2.ErrNotFound
}

//TODO: 备用
func (p *Protocol) queryAddrInfo(pid peer.ID, queryPeer peer.ID) (*types.PeerInfo, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*5)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, queryPeer, fetchPeerAddr)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	req := types.P2PRequest{
		Request: &types.P2PRequest_Pid{
			Pid: string(pid),
		},
	}
	err = protocol.WriteStream(&req, stream)
	if err != nil {
		return nil, err
	}
	var resp types.P2PResponse
	err = protocol.ReadStream(&resp, stream)
	if err != nil {
		return nil, err
	}
	data, ok := resp.Response.(*types.P2PResponse_PeerInfo)
	if !ok {
		return nil, types2.ErrInvalidMessageType
	}
	return data.PeerInfo, nil
}

func savePeers(peerInfos []*types.PeerInfo, store peerstore.Peerstore) []peer.ID {
	var peers []peer.ID
	for _, peerInfo := range peerInfos {
		if peerInfo == nil {
			continue
		}
		var maddrs []multiaddr.Multiaddr
		for _, addr := range peerInfo.MultiAddr {
			maddr, err := multiaddr.NewMultiaddrBytes(addr)
			if err != nil {
				continue
			}
			maddrs = append(maddrs, maddr)
		}
		pid := peer.ID(peerInfo.ID)
		store.AddAddrs(pid, maddrs, time.Hour*24)
		peers = append(peers, pid)
	}
	return peers
}
