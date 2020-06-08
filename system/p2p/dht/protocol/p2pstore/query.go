package p2pstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

//getChunk gets chunk data from p2pStore or other peers.
func (p *Protocol) getChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}

	//优先获取本地p2pStore数据
	bodys, _ := p.getChunkBlock(req.ChunkHash)
	if bodys != nil {
		l := int64(len(bodys.Items))
		start, end := req.Start%l, req.End%l+1
		bodys.Items = bodys.Items[start:end]
		return bodys, nil
	}

	//本地数据不存在或已过期，则向临近节点查询
	return p.mustFetchChunk(req)
}

func (p *Protocol) getHeaders(param *types.ReqBlocks) *types.Headers {
	for _, pid := range p.healthyRoutingTable.ListPeers() {
		headers, err := p.getHeadersFromPeer(param, pid)
		if err != nil {
			log.Error("getHeaders", "peer", pid, "error", err)
			continue
		}
		return headers
	}

	log.Error("getHeaders", "error", types2.ErrNotFound)
	return &types.Headers{}
}

func (p *Protocol) getHeadersFromPeer(param *types.ReqBlocks, pid peer.ID) (*types.Headers, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := p.Host.NewStream(childCtx, pid, protocol.GetHeader)
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
	for _, prettyID := range param.Pid {
		pid, err := peer.IDB58Decode(prettyID)
		if err != nil {
			log.Error("getChunkRecords", "decode pid error", err)
		}
		records, err := p.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "param peer", pid, "error", err, "start", param.Start, "end", param.End)
			continue
		}
		return records
	}

	for _, pid := range p.healthyRoutingTable.ListPeers() {
		records, err := p.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err, "start", param.Start, "end", param.End)
			continue
		}
		return records
	}

	log.Error("getChunkRecords", "error", types2.ErrNotFound)
	return nil
}

func (p *Protocol) getChunkRecordsFromPeer(param *types.ReqChunkRecords, pid peer.ID) (*types.ChunkRecords, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := p.Host.NewStream(childCtx, pid, protocol.GetChunkRecord)
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
func (p *Protocol) mustFetchChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	//递归查询时间上限一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	var retryCount int
Retry:
	//保存查询过的节点，防止重复查询
	searchedPeers := make(map[peer.ID]struct{})
	searchedPeers[p.Host.ID()] = struct{}{}
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), AlphaValue)
	for len(peers) != 0 {
		var newPeers []peer.ID
		for _, pid := range peers {
			searchedPeers[pid] = struct{}{}
			bodys, nearerPeers, err := p.fetchChunkOrNearerPeers(ctx, req, pid)
			if err != nil {
				log.Error("mustFetchChunk", "fetchChunkOrNearerPeers error", err, "pid", pid, "chunk hash", hex.EncodeToString(req.ChunkHash))
				continue
			}
			if bodys != nil {
				log.Info("mustFetchChunk found", "pid", pid, "maddrs", p.Host.Peerstore().Addrs(pid))
				return bodys, nil
			}
			newPeers = append(newPeers, nearerPeers...)
		}

		peers = nil
		for _, newPeer := range newPeers {
			//已经查询过的节点就不再查询
			if _, ok := searchedPeers[newPeer]; !ok {
				peers = append(peers, newPeer)
			}
		}
	}
	retryCount++
	if retryCount < 3 { //找不到数据重试3次，防止因为网络问题导致数据找不到
		log.Error("mustFetchChunk", "retry count", retryCount)
		goto Retry
	}
	log.Error("mustFetchChunk", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "error", types2.ErrNotFound)
	for pid := range searchedPeers {
		log.Info("mustFetchChunk debug", "pid", pid, "maddr", p.Host.Peerstore().Addrs(pid))
	}
	return nil, types2.ErrNotFound
}

func (p *Protocol) fetchChunkOrNearerPeers(ctx context.Context, params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, []peer.ID, error) {
	childCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	stream, err := p.Host.NewStream(childCtx, pid, protocol.FetchChunk)
	if err != nil {
		log.Error("getBlocksFromRemote", "error", err)
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
		log.Error("fetchChunkOrNearerPeers", "SignAndWriteStream error", err)
		return nil, nil, err
	}
	var res types.P2PResponse
	//err = protocol.ReadStreamAndAuthenticate(&res, stream)
	//if err != nil {
	//	log.Error("fetchChunkFromPeer", "read response error", err, "chunk hash", hex.EncodeToString(params.ChunkHash))
	//	return nil, nil, err
	//}
	var result []byte
	buf := make([]byte, 1024*1024)
	t := time.Now()
	for {
		n, err := stream.Read(buf)
		result = append(result, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
	}
	log.Info("fetchChunkOrNearerPeers", "time cost", time.Since(t))
	err = types.Decode(result, &res)
	if err != nil {
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
			if len(p.Host.Peerstore().Addrs(addrInfo.ID)) == 0 {
				p.Host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, time.Hour)
			}
			peerList = append(peerList, addrInfo.ID)
		}
		return nil, peerList, nil
	}

	return nil, nil, errors.New(res.Error)
}

// 检查网络中是否能查到前一个chunk，最多往前查10个chunk，返回未保存的chunkInfo
func (p *Protocol) checkHistoryChunk(in *types.ChunkInfoMsg) []*types.ChunkInfoMsg {
	chunkLen := in.End - in.Start + 1
	req := &types.ReqChunkRecords{
		Start: in.Start/chunkLen - 10,
		End:   in.Start/chunkLen - 1,
	}
	if req.End < 0 {
		return nil
	}
	if req.Start < 0 {
		req.Start = 0
	}
	records, err := p.getChunkRecordFromBlockchain(req)
	if err != nil || records == nil {
		log.Error("checkHistoryChunk", "getChunkRecordFromBlockchain error", err, "start", req.Start, "end", req.End, "records", records)
		return nil
	}

	var res []*types.ChunkInfoMsg
	for i := len(records.Infos) - 1; i >= 0; i-- {
		info := &types.ChunkInfoMsg{
			ChunkHash: records.Infos[i].ChunkHash,
			Start:     records.Infos[i].Start,
			End:       records.Infos[i].End,
		}
		bodys, err := p.getChunk(info)
		if err == nil && bodys != nil && len(bodys.Items) == int(chunkLen) {
			break
		}
		//网络中找不到上一个chunk,先把上一个chunk保存到本地p2pstore
		log.Debug("checkHistoryChunk", "chunk num", info.Start, "chunk hash", hex.EncodeToString(info.ChunkHash))
		res = append(res, info)
	}
	return res
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

	//本地存储之后立即到其他节点做一次备份
	p.notifyStoreChunk(req)

	return nil
}

func (p *Protocol) checkAndStoreChunk(req *types.ChunkInfoMsg) error {
	//先检查之前的chunk是否可以在网络中查到
	infos := p.checkHistoryChunk(req)
	for _, info := range infos {
		if err := p.storeChunk(info); err != nil {
			log.Error("checkAndStoreChunk", "store chunk error", err)
		}
	}
	return p.storeChunk(req)
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
