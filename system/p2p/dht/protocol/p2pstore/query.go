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
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

func (p *Protocol) requestPeerInfoForChunk(msg *types.ChunkInfoMsg, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
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
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
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
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
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
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
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
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
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
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
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
//  1. 检查本地
//  2. 检查缓存中记录的提供数据的节点
//  3. 异步获取数据
func (p *Protocol) findChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, peer.ID, error) {
	//优先获取本地p2pStore数据
	bodys, _ := p.loadChunk(req)
	if bodys != nil {
		return bodys, p.Host.ID(), nil
	}
	//检查缓存
	for _, pid := range p.getChunkProviderCache(req.ChunkHash) {
		if bodys, _, _ := p.fetchChunkFromPeer(req, pid); bodys != nil {
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
			bodys, _, err := p.fetchChunkFromPeer(req, pid)
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

func (p *Protocol) getBlocks(req *types.ChunkInfoMsg) (*types.Blocks, error) {
	headerCh := make(chan *types.Headers, 1)
	go func() {
		headers, _ := p.getHeaders(&types.ReqBlocks{Start: req.Start, End: req.End})
		headerCh <- headers
	}()
	bodys, _, err := p.getChunk(req)
	if bodys == nil {
		log.Error("GetChunkBlock", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		return nil, types2.ErrNotFound
	}
	headers := <-headerCh
	if headers == nil {
		log.Error("GetBlockHeader", "error", types2.ErrNotFound)
		return nil, types2.ErrNotFound
	}
	if len(headers.Items) != len(bodys.Items) {
		log.Error("GetBlockHeader", "error", types2.ErrLength, "header length", len(headers.Items), "body length", len(bodys.Items), "start", req.Start, "end", req.End)
		return nil, types2.ErrLength
	}

	var blockList []*types.Block
	for index := range bodys.Items {
		body := bodys.Items[index]
		header := headers.Items[index]
		block := &types.Block{
			Version:    header.Version,
			ParentHash: header.ParentHash,
			TxHash:     header.TxHash,
			StateHash:  header.StateHash,
			Height:     header.Height,
			BlockTime:  header.BlockTime,
			Difficulty: header.Difficulty,
			MainHash:   body.MainHash,
			MainHeight: body.MainHeight,
			Signature:  header.Signature,
			Txs:        body.Txs,
		}
		blockList = append(blockList, block)
	}
	return &types.Blocks{Items: blockList}, nil
}

// getChunk gets chunk data from p2pStore or other peers.
func (p *Protocol) getChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, peer.ID, error) {
	if req == nil {
		return nil, "", types2.ErrInvalidParam
	}
	//优先获取本地p2pStore数据
	bodys, _ := p.loadChunk(req)
	if bodys != nil {
		return bodys, p.Host.ID(), nil
	}
	//本地数据不存在或已过期，则向临近节点查询
	return p.mustFetchChunk(req)
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
	childCtx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	p.Host.ConnManager().Protect(pid, getHeader)
	defer p.Host.ConnManager().Unprotect(pid, getHeader)
	stream, err := p.Host.NewStream(childCtx, pid, getHeader)
	if err != nil {
		return nil, err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Second * 10))
	defer stream.Close()
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ReqBlocks{
			ReqBlocks: param,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream, p.Host.Peerstore().PrivKey(p.Host.ID()))
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
	childCtx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	p.Host.ConnManager().Protect(pid, getChunkRecord)
	defer p.Host.ConnManager().Unprotect(pid, getChunkRecord)
	stream, err := p.Host.NewStream(childCtx, pid, getChunkRecord)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ReqChunkRecords{
			ReqChunkRecords: param,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream, p.Host.Peerstore().PrivKey(p.Host.ID()))
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

// 若网络中有节点保存了该chunk，该方法可以保证查询到
func (p *Protocol) mustFetchChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, peer.ID, error) {
	//TODO: temporary
	for _, conn := range p.Host.Network().Conns() {
		if len(p.RoutingTable.Find(conn.RemotePeer())) == 0 {
			_, _ = p.RoutingTable.TryAddPeer(conn.RemotePeer(), false, true)
		}
	}
	// 递归查询时间上限10分钟
	ctx, cancel := context.WithTimeout(p.Ctx, time.Minute*10)
	defer cancel()

	chunkHash := hex.EncodeToString(req.ChunkHash)
	log.Info("into mustFetchChunk", "start", req.Start, "end", req.End)

	// 先请求缓存provider节点
	for _, pid := range p.getChunkProviderCache(req.ChunkHash) {
		select {
		case <-ctx.Done():
			return nil, "", types2.ErrNotFound
		default:
		}
		start := time.Now()
		bodys, _, err := p.fetchChunkFromPeer(req, pid)
		if err == nil && bodys != nil {
			log.Info("mustFetchChunk found from cache provider", "chunk hash", chunkHash, "start", req.Start, "pid", pid, "maddrs", p.Host.Peerstore().Addrs(pid), "time cost", time.Since(start))
			return bodys, pid, nil
		}
	}

	//保存查询过的节点，防止重复查询
	searchedPeers := make(map[peer.ID]struct{})
	searchedPeers[p.Host.ID()] = struct{}{}

	peers := p.RoutingTable.NearestPeers(genDHTID(req.ChunkHash), 10)
	localPeers := make(chan peer.ID, 100)
	alternativePeers := make(chan peer.ID, 100)
	for _, pid := range peers {
		localPeers <- pid
		searchedPeers[pid] = struct{}{}
	}

	// 优先从已经建立连接的节点上查找数据，因为建立新的连接会耗时，且会导致网络拓扑结构发生变化
	loop1Start := time.Now()
Loop1:
	for {
		select {
		case <-ctx.Done():
			return nil, "", types2.ErrNotFound
		case pid := <-localPeers:
			start := time.Now()
			bodys, nearerPeers, err := p.fetchChunkFromPeer(req, pid)
			if err != nil {
				continue
			}
			if bodys != nil {
				log.Info("mustFetchChunk found better", "chunk hash", chunkHash, "start", req.Start, "pid", pid, "maddrs", p.Host.Peerstore().Addrs(pid), "time cost", time.Since(start))
				return bodys, pid, nil
			}
			for _, pid := range nearerPeers {
				if _, ok := searchedPeers[pid]; ok {
					continue
				}
				if len(p.Host.Network().ConnsToPeer(pid)) != 0 {
					select {
					case localPeers <- pid:
						searchedPeers[pid] = struct{}{}
					default:
						log.Info("mustFetchChunk localPeers channel full", "pid", pid)
					}

				} else if len(p.Host.Peerstore().Addrs(pid)) != 0 {
					select {
					case alternativePeers <- pid:
						searchedPeers[pid] = struct{}{}
					default:
						log.Info("mustFetchChunk alternativePeers channel full", "pid", pid)
					}
				}
			}
		default:
			break Loop1
		}
	}

	log.Error("mustFetchChunk from rt peer not found", "chunk hash", chunkHash, "start", req.Start, "error", types2.ErrNotFound, "time cost", time.Since(loop1Start))

	// 其次从未建立连接但已保存ip等信息的的节点上获取数据
	loop2Start := time.Now()
Loop2:
	for {
		select {
		case <-ctx.Done():
			return nil, "", types2.ErrNotFound
		case pid := <-alternativePeers:
			start := time.Now()
			bodys, nearerPeers, err := p.fetchChunkFromPeer(req, pid)
			if err != nil {
				continue
			}
			if bodys != nil {
				log.Info("mustFetchChunk found from alternative peer", "chunk hash", chunkHash, "start", req.Start, "pid", pid, "maddrs", p.Host.Peerstore().Addrs(pid), "time cost", time.Since(start))
				return bodys, pid, nil
			}
			for _, pid := range nearerPeers {
				if _, ok := searchedPeers[pid]; ok {
					continue
				}
				if len(p.Host.Peerstore().Addrs(pid)) != 0 {
					select {
					case alternativePeers <- pid:
						searchedPeers[pid] = struct{}{}
					default:
						log.Info("mustFetchChunk alternativePeers channel full", "pid", pid)
					}
				}
			}
		default:
			break Loop2
		}
	}

	log.Error("mustFetchChunk not found", "chunk hash", chunkHash, "start", req.Start, "error", types2.ErrNotFound, "time cost", time.Since(loop2Start))

	//如果是分片节点没有在分片网络中找到数据，最后到全节点去请求数据
	ctx2, cancel2 := context.WithTimeout(p.Ctx, time.Minute)
	defer cancel2()
	peerInfos, err := p.Discovery.FindPeers(ctx2, fullNode)
	if err != nil {
		log.Error("mustFetchChunk", "Find full peers error", err)
		return nil, "", types2.ErrNotFound
	}

	ctx3, cancel3 := context.WithTimeout(p.Ctx, time.Minute*3)
	defer cancel3()
	for addrInfo := range peerInfos {
		select {
		case <-ctx3.Done():
			return nil, "", types2.ErrNotFound
		default:
		}
		if addrInfo.ID == p.Host.ID() {
			continue
		}
		log.Info("mustFetchChunk from full node", "pid", addrInfo.ID, "addrs", addrInfo.Addrs)
		start := time.Now()
		bodys, err := p.fetchChunkFromFullPeer(req, addrInfo.ID)
		if err != nil {
			log.Error("mustFetchChunk from full node failed", "pid", addrInfo.ID, "chunk hash", chunkHash, "start", req.Start, "time cost", time.Since(start))
			continue
		}
		log.Info("mustFetchChunk from full node succeed", "pid", addrInfo.ID, "chunk hash", chunkHash, "start", req.Start, "time cost", time.Since(start))
		return bodys, addrInfo.ID, nil
	}

	return nil, "", types2.ErrNotFound
}

func (p *Protocol) fetchChunkFromPeer(params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, []peer.ID, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	p.Host.ConnManager().Protect(pid, fetchChunk)
	defer p.Host.ConnManager().Unprotect(pid, fetchChunk)
	stream, err := p.Host.NewStream(ctx, pid, fetchChunk)
	if err != nil {
		log.Error("fetchChunkFromPeer", "error", err, "start", params.Start)
		return nil, nil, err
	}
	defer stream.Close()
	_ = stream.SetDeadline(time.Now().Add(time.Minute * 5))
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ChunkInfoMsg{
			ChunkInfoMsg: params,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream, p.Host.Peerstore().PrivKey(p.Host.ID()))
	if err != nil {
		log.Error("fetchChunkFromPeer", "SignAndWriteStream error", err, "start", params.Start)
		return nil, nil, err
	}
	var bodys []*types.BlockBody
	var res types.P2PResponse
	for {
		if err := protocol.ReadStream(&res, stream); err != nil {
			log.Error("fetchChunkFromPeer", "ReadStream error", err, "start", params.Start)
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
		// 增加provider缓存
		p.addChunkProviderCache(params.ChunkHash, pid)
		return &types.BlockBodys{
			Items: bodys,
		}, closerPeers, nil
	}

	if len(closerPeers) == 0 {
		return nil, nil, fmt.Errorf(res.Error)
	}
	return nil, closerPeers, nil
}

func (p *Protocol) fetchChunkFromFullPeer(params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, error) {
	bodys, _, err := p.fetchChunkFromPeer(params, pid)
	if bodys == nil {
		return nil, types2.ErrNotFound
	}
	return bodys, err
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

// TODO: 备用
func (p *Protocol) queryAddrInfo(pid peer.ID, queryPeer peer.ID) (*types.PeerInfo, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, queryPeer, fetchPeerAddr)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
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
