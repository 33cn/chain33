package p2pstore

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

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
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	msg := types.P2PStoreRequest{
		ProtocolID: GetHeader,
		Data: &types.P2PStoreRequest_ReqBlocks{
			ReqBlocks: param,
		},
	}
	err = writeMessage(rw.Writer, &msg)
	if err != nil {
		log.Error("getHeadersFromPeer", "stream write error", err)
		return nil, err
	}
	var res types.P2PStoreResponse
	err = readMessage(rw.Reader, &res)
	if err != nil {
		return nil, err
	}
	if res.ErrorInfo != "" {
		return nil, errors.New(res.ErrorInfo)
	}
	return res.Result.(*types.P2PStoreResponse_Headers).Headers, nil
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
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	msg := types.P2PStoreRequest{
		ProtocolID: GetChunkRecord,
		Data: &types.P2PStoreRequest_ReqChunkRecords{
			ReqChunkRecords: param,
		},
	}
	err = writeMessage(rw.Writer, &msg)
	if err != nil {
		log.Error("getChunkRecordsFromPeer", "stream write error", err)
		return nil, err
	}

	var res types.P2PStoreResponse
	err = readMessage(rw.Reader, &res)
	if err != nil {
		return nil, err
	}
	if res.ErrorInfo != "" {
		return nil, errors.New(res.ErrorInfo)
	}
	return res.Result.(*types.P2PStoreResponse_ChunkRecords).ChunkRecords, nil
}

func (s *StoreProtocol) mustFetchChunk(req *types.ChunkInfoMsg, peers []peer.ID) (*types.BlockBodys, error) {
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

func (s *StoreProtocol) fetchChunkOrNearerPeersAsync(ctx context.Context, param *types.ChunkInfoMsg, peers []peer.ID) (*types.BlockBodys, []peer.ID) {

	responseCh := make(chan interface{}, len(peers))
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	// 多个节点并发请求
	for _, peerID := range peers {
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

	var peerList []peer.ID
	for range peers {
		res := <-responseCh
		switch t := res.(type) {
		case *types.BlockBodys:
			//查到了区块数据，直接返回
			return t, nil
		case []peer.ID:
			//没查到区块数据，返回了更近的节点信息
			peerList = append(peerList, t...)
			if len(peerList) >= AlphaValue {
				//直接返回新peer，加快查询速度
				return nil, t
			}
		}
	}

	//TODO 若超过3个，排序选择最优的三个节点
	if len(peerList) > AlphaValue {
		peerList = peerList[:AlphaValue]
	}

	return nil, peerList
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
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	msg := types.P2PStoreRequest{
		ProtocolID: FetchChunk,
		Data: &types.P2PStoreRequest_ChunkInfoMsg{
			ChunkInfoMsg: params,
		},
	}
	err = writeMessage(rw.Writer, &msg)
	if err != nil {
		log.Error("fetchChunkOrNearerPeers", "stream write error", err)
		return nil, nil, err
	}
	var res types.P2PStoreResponse
	err = readMessage(rw.Reader, &res)
	if err != nil {
		log.Error("fetchChunkFromPeer", "read response error", err, "chunk hash", hex.EncodeToString(params.ChunkHash))
		return nil, nil, err
	}
	log.Info("fetchChunkFromPeer", "remote", pid.Pretty(), "chunk hash", hex.EncodeToString(params.ChunkHash))

	switch v := res.Result.(type) {
	case *types.P2PStoreResponse_BlockBodys:
		return v.BlockBodys, nil, nil
	case *types.P2PStoreResponse_AddrInfo:
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
		if res.ErrorInfo != "" {
			return nil, nil, errors.New(res.ErrorInfo)
		}
	}

	return nil, nil, types2.ErrNotFound
}

func (s *StoreProtocol) checkLastChunk(in *types.ChunkInfoMsg) {
	l := in.End - in.Start + 1
	req := &types.ReqChunkRecords{
		Start: in.Start/l-1,
		End:  in.End/l-1,
	}
	if req.Start < 0 {
		return
	}
	records, err := s.getChunkRecordFromBlockchain(req)
	if err != nil {
		log.Error("checkLastChunk", "getChunkRecordFromBlockchain error", err, "start", req.Start, "end", req.End)
		return
	}
	if len(records.Infos) != 1 {
		log.Error("checkLastChunk", "record len", len(records.Infos))
	}
	info := &types.ChunkInfoMsg{
		ChunkHash: records.Infos[0].ChunkHash,
		Start: records.Infos[0].Start,
		End: records.Infos[0].End,
	}
	bodys, err := s.GetChunk(info)
	if err == nil && bodys != nil && len(bodys.Items) == int(l) {
		return
	}
	err = s.storeChunk(info)
	if err != nil {
		log.Error("checkLastChunk", "chunk hash", hex.EncodeToString(info.ChunkHash), "start", info.Start, "end", info.End, "error", err)
	}

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
	if reply, ok := resp.GetData().(*types.Reply); ok {
		return nil, errors.New(string(reply.Msg))
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
	records, ok := resp.GetData().(*types.ChunkRecords)
	if !ok {
		return nil, types2.ErrNotFound
	}
	return records, nil
}
