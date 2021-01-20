package p2pstore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

func (p *Protocol) fetchCloserPeers(key []byte, count int, pid peer.ID) ([]peer.ID, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	p.Host.ConnManager().Protect(pid, fetchShardPeer)
	defer p.Host.ConnManager().Unprotect(pid, fetchShardPeer)
	stream, err := p.Host.NewStream(ctx, pid, fetchShardPeer)
	if err != nil {
		return nil, err
	}
	defer protocol.CloseStream(stream)
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
	closerPeers := saveCloserPeers(resp.CloserPeers, p.Host.Peerstore())
	return closerPeers, nil
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
	return p.mustFetchChunk(p.Ctx, req, true)
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
	defer protocol.CloseStream(stream)
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
	defer protocol.CloseStream(stream)
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
	// 从多个节点请求ChunkRecords, 并从中选择多数节点返回的数据
	recordsCache := make(map[string]*types.ChunkRecords)
	recordsCount := make(map[string]int)
	for i, pid := range p.ShardHealthyRoutingTable.ListPeers() {
		records, err := p.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err, "start", param.Start, "end", param.End)
			continue
		}
		sum := common.Sha256(types.Encode(records))
		recordsCache[string(sum)] = records
		recordsCount[string(sum)]++
		log.Info("getChunkRecords", "peer", pid, "start", param.Start, "end", param.End)
		if i > 10 && len(recordsCount) != 0 {
			break
		}
	}
	var records *types.ChunkRecords
	var maxCount int
	for sum, count := range recordsCount {
		if count > maxCount {
			records = recordsCache[sum]
		}
	}

	return records
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
	defer protocol.CloseStream(stream)
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
func (p *Protocol) mustFetchChunk(pctx context.Context, req *types.ChunkInfoMsg, queryFull bool) (*types.BlockBodys, peer.ID, error) {
	//递归查询时间上限一小时
	ctx, cancel := context.WithTimeout(pctx, time.Minute*5)
	defer cancel()

	//保存查询过的节点，防止重复查询
	searchedPeers := make(map[peer.ID]struct{})
	searchedPeers[p.Host.ID()] = struct{}{}
	peers := p.ShardHealthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), AlphaValue)
	log.Info("into mustFetchChunk", "shard healthy peers len", p.ShardHealthyRoutingTable.Size(), "start", req.Start, "end", req.End)
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
				log.Info("mustFetchChunk found", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "pid", pid, "maddrs", p.Host.Peerstore().Addrs(pid), "time cost", time.Since(start))
				return bodys, pid, nil
			}
			break
		}
		peers = nearerPeers
	}

	log.Error("mustFetchChunk not found", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "error", types2.ErrNotFound)
	if !queryFull {
		return nil, "", types2.ErrNotFound
	}
	//如果是分片节点没有在分片网络中找到数据，最后到全节点去请求数据
	ctx2, cancel2 := context.WithTimeout(ctx, time.Second*10)
	defer cancel2()
	peerInfos, err := p.FindPeers(ctx2, fullNode)
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
			log.Error("mustFetchChunk from full node failed", "pid", addrInfo.ID, "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start)
			continue
		}
		log.Info("mustFetchChunk from full node succeed", "pid", addrInfo.ID, "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start)
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
	defer protocol.CloseStream(stream)
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
	closerPeers := saveCloserPeers(res.CloserPeers, p.Host.Peerstore())
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

func (p *Protocol) checkChunkInNetwork(req *types.ChunkInfoMsg) (peer.ID, bool) {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*10)
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	random := rand.Int63n(req.End-req.Start+1) + req.Start
	req2 := &types.ChunkInfoMsg{
		ChunkHash: req.ChunkHash,
		Start:     random,
		End:       random,
	}
	//query a random block of the chunk, to make sure that the chunk exists in the extension.
	_, pid, err := p.mustFetchChunk(ctx, req2, false)
	return pid, err == nil
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

func (p *Protocol) checkNetworkAndStoreChunk(req *types.ChunkInfoMsg) error {
	//先检查之前的chunk是否以在网络中查到
	infos := p.getHistoryChunkInfos(req, 3)
	for i := len(infos) - 1; i >= 0; i-- {
		info := infos[i]
		if _, ok := p.checkChunkInNetwork(info); ok {
			infos = infos[i+1:]
			break
		}
	}
	infos = append(infos, req)
	var err error
	for _, info := range infos {
		log.Info("checkNetworkAndStoreChunk storing", "chunk hash", hex.EncodeToString(info.ChunkHash), "start", info.Start)
		if err = p.storeChunk(info); err != nil {
			log.Error("checkNetworkAndStoreChunk", "store chunk error", err, "chunkhash", hex.EncodeToString(info.ChunkHash), "start", info.Start)
			continue
		}
		//本地存储之后立即到其他节点做一次备份
		p.notifyStoreChunk(req)
	}
	return err
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
	resp, err := p.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if records, ok := resp.GetData().(*types.ChunkRecords); ok {
		return records, nil
	}

	return nil, types2.ErrNotFound
}

func (p *Protocol) getHistoryChunkInfos(in *types.ChunkInfoMsg, count int64) []*types.ChunkInfoMsg {
	chunkLen := in.End - in.Start + 1
	req := &types.ReqChunkRecords{
		Start: in.Start/chunkLen - count,
		End:   in.Start/chunkLen - 1,
	}
	if req.End < 0 {
		return nil
	}
	if req.Start < 0 {
		req.Start = 0
	}
	records, err := p.getChunkRecordFromBlockchain(req)
	if err != nil {
		log.Error("getHistoryChunkRecords", "getChunkRecordFromBlockchain error", err, "start", req.Start, "end", req.End, "records", records)
		return nil
	}
	var chunkInfos []*types.ChunkInfoMsg
	for _, info := range records.Infos {
		chunkInfos = append(chunkInfos, &types.ChunkInfoMsg{
			ChunkHash: info.ChunkHash,
			Start:     info.Start,
			End:       info.End,
		})
	}

	return chunkInfos
}

func saveCloserPeers(peerInfos []*types.PeerInfo, store peerstore.Peerstore) []peer.ID {
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
		store.AddAddrs(pid, maddrs, time.Minute*30)
		peers = append(peers, pid)
	}
	return peers
}
