package p2pstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

//getChunk gets chunk data from p2pStore or other peers.
func (s *StoreProtocol) getChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}

	//优先获取本地p2pStore数据
	bodys, _ := s.getChunkBlock(req.ChunkHash)
	if bodys != nil {
		l := int64(len(bodys.Items))
		start, end := req.Start%l, req.End%l+1
		bodys.Items = bodys.Items[start:end]
		return bodys, nil
	}

	//本地数据不存在或已过期，则向临近节点查询
	//首先从本地路由表获取 *3* 个最近的节点
	peers := s.Discovery.FindNearestPeers(peer.ID(genChunkPath(req.ChunkHash)), AlphaValue)
	return s.mustFetchChunk(req, peersToMap(peers))
}

func (s *StoreProtocol) getHeaders(param *types.ReqBlocks) *types.Headers {
	for _, pid := range s.Discovery.RoutingTable() {
		headers, err := s.getHeadersFromPeer(param, pid)
		if err != nil {
			log.Error("getHeaders", "peer", pid, "error", err)
			continue
		}
		return headers
	}

	log.Error("getHeaders", "error", types2.ErrNotFound)
	return &types.Headers{}
}

func (s *StoreProtocol) getHeadersFromPeer(param *types.ReqBlocks, pid peer.ID) (*types.Headers, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, GetHeader)
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
	err = protocol.ReadResponseAndAuthenticate(&res, stream)
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Response.(*types.P2PResponse_BlockHeaders).BlockHeaders, nil
}

func (s *StoreProtocol) getChunkRecords(param *types.ReqChunkRecords) *types.ChunkRecords {
	for _, prettyID := range param.Pid {
		pid, err := peer.IDB58Decode(prettyID)
		if err != nil {
			log.Error("getChunkRecords", "decode pid error", err)
		}
		records, err := s.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "addr", s.Host.Peerstore().Addrs(pid), "error", err)
			continue
		}
		return records
	}

	for _, pid := range s.Discovery.RoutingTable() {
		records, err := s.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err)
			continue
		}
		return records
	}

	log.Error("getChunkRecords", "error", types2.ErrNotFound)
	return nil
}

func (s *StoreProtocol) getChunkRecordsFromPeer(param *types.ReqChunkRecords, pid peer.ID) (*types.ChunkRecords, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, GetChunkRecord)
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
	err = protocol.ReadResponseAndAuthenticate(&res, stream)
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Response.(*types.P2PResponse_ChunkRecords).ChunkRecords, nil
}

func (s *StoreProtocol) mustFetchChunk(req *types.ChunkInfoMsg, peers map[peer.ID]struct{}) (*types.BlockBodys, error) {
	//递归查询时间上限一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	for {
		bodys, newPeers := s.fetchChunkOrNearerPeersAsync(ctx, req, peers)
		if bodys != nil {
			return bodys, nil
		}
		if len(newPeers) == 0 {
			break
		}
		peers = newPeers
	}
	return nil, types2.ErrNotFound
}

func (s *StoreProtocol) fetchChunkOrNearerPeersAsync(ctx context.Context, param *types.ChunkInfoMsg, peerMap map[peer.ID]struct{}) (*types.BlockBodys, map[peer.ID]struct{}) {

	responseCh := make(chan interface{}, len(peerMap))
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	// 多个节点并发请求
	for peerID := range peerMap {
		if peerID == s.Host.ID() {
			responseCh <- nil
			continue
		}
		go func(pid peer.ID) {
			bodys, pids, err := s.fetchChunkOrNearerPeers(cancelCtx, param, pid)
			if bodys != nil {
				responseCh <- bodys
			} else if len(pids) != 0 {
				responseCh <- pids
			} else {
				responseCh <- nil
				log.Error("fetchChunkOrNearerPeers", "error", err, "peer id", pid)
			}
		}(peerID)
	}

	//map用于去重
	newPeerMap := make(map[peer.ID]struct{})
	for range peerMap {
		res := <-responseCh
		switch t := res.(type) {
		case *types.BlockBodys:
			//查到了区块数据，直接返回
			return t, nil
		case []peer.ID:
			//没查到区块数据，返回了更近的节点信息
			for _, newPeer := range t {
				//过滤掉已经查询过的节点
				if _, ok := peerMap[newPeer]; !ok {
					newPeerMap[newPeer] = struct{}{}
				}
				if len(newPeerMap) == AlphaValue {
					//直接返回新peer，加快查询速度
					return nil, newPeerMap
				}
			}
		}
	}
	return nil, newPeerMap
}

func (s *StoreProtocol) fetchChunkOrNearerPeers(ctx context.Context, params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, []peer.ID, error) {
	childCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, FetchChunk)
	if err != nil {
		log.Error("getBlocksFromRemote", "error", err)
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
		log.Error("fetchChunkOrNearerPeers", "SignAndWriteStream error", err)
		return nil, nil, err
	}
	var res types.P2PResponse
	err = protocol.ReadResponseAndAuthenticate(&res, stream)
	if err != nil {
		log.Error("fetchChunkFromPeer", "read response error", err, "chunk hash", hex.EncodeToString(params.ChunkHash))
		return nil, nil, err
	}
	log.Info("fetchChunkFromPeer", "remote", pid.Pretty(), "chunk hash", hex.EncodeToString(params.ChunkHash))

	switch v := res.Response.(type) {
	case *types.P2PResponse_BlockBodys:
		return v.BlockBodys, nil, nil
	case *types.P2PResponse_AddrInfo:
		var addrInfos []peer.AddrInfo
		err = json.Unmarshal(v.AddrInfo, &addrInfos)
		if err != nil {
			log.Error("fetchChunkOrNearerPeers", "addrInfo error", err)
		}
		var peerList []peer.ID
		//如果对端节点返回了addrInfo，把节点信息加入到PeerStore，并返回节点id
		for _, addrInfo := range addrInfos {
			s.Host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, time.Hour)
			peerList = append(peerList, addrInfo.ID)
		}
		return nil, peerList, nil
	default:
		if res.Error != "" {
			return nil, nil, errors.New(res.Error)
		}
	}

	return nil, nil, types2.ErrNotFound
}

func (s *StoreProtocol) checkLastChunk(in *types.ChunkInfoMsg) {
	l := in.End - in.Start + 1
	req := &types.ReqChunkRecords{
		Start: in.Start/l - 1,
		End:   in.End/l - 1,
	}
	if req.Start < 0 {
		return
	}
	records, err := s.getChunkRecordFromBlockchain(req)
	if err != nil || len(records.Infos) != 1 {
		log.Error("checkLastChunk", "getChunkRecordFromBlockchain error", err, "start", req.Start, "end", req.End, "records", records)
		return
	}
	info := &types.ChunkInfoMsg{
		ChunkHash: records.Infos[0].ChunkHash,
		Start:     records.Infos[0].Start,
		End:       records.Infos[0].End,
	}
	bodys, err := s.getChunk(info)
	if err == nil && bodys != nil && len(bodys.Items) == int(l) {
		return
	}
	//网络中找不到上一个chunk,先把上一个chunk保存到本地p2pstore
	log.Debug("checkLastChunk", "chunk num", info.Start, "chunk hash", hex.EncodeToString(info.ChunkHash))
	err = s.storeChunk(info)
	if err != nil {
		log.Error("checkLastChunk", "chunk hash", hex.EncodeToString(info.ChunkHash), "start", info.Start, "end", info.End, "error", err)
	}
}

func (s *StoreProtocol) storeChunk(req *types.ChunkInfoMsg) error {
	//先检查上个chunk是否可以在网络中查到
	s.checkLastChunk(req)
	//如果p2pStore已保存数据，只更新时间即可
	if err := s.updateChunk(req); err == nil {
		return nil
	}
	//blockchain通知p2pStore保存数据，则blockchain应该有数据
	bodys, err := s.getChunkFromBlockchain(req)
	if err != nil {
		log.Error("StoreChunk", "getChunkFromBlockchain error", err)
		return err
	}
	err = s.addChunkBlock(req, bodys)
	if err != nil {
		log.Error("StoreChunk", "addChunkBlock error", err)
		return err
	}

	//本地存储之后立即到其他节点做一次备份
	s.notifyStoreChunk(req)
	return nil
}

func (s *StoreProtocol) getChunkFromBlockchain(param *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	if param == nil {
		return nil, types2.ErrInvalidParam
	}
	msg := s.QueueClient.NewMessage("blockchain", types.EventGetChunkBlockBody, param)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if bodys, ok := resp.GetData().(*types.BlockBodys); ok {
		return bodys, nil
	}
	return nil, types2.ErrNotFound
}

func (s *StoreProtocol) getChunkRecordFromBlockchain(req *types.ReqChunkRecords) (*types.ChunkRecords, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}
	msg := s.QueueClient.NewMessage("blockchain", types.EventGetChunkRecord, req)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if records, ok := resp.GetData().(*types.ChunkRecords); ok {
		return records, nil
	}

	return nil, types2.ErrNotFound
}

func peersToMap(peers []peer.ID) map[peer.ID]struct{} {
	m := make(map[peer.ID]struct{})
	for _, pid := range peers {
		m[pid] = struct{}{}
	}
	return m
}
