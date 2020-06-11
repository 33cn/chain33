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
func (p *Protocol) mustFetchChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	//递归查询时间上限一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	//保存查询过的节点，防止重复查询
	searchedPeers := make(map[peer.ID]struct{})
	searchedPeers[p.Host.ID()] = struct{}{}
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), Backup)
	for len(peers) != 0 {
		newPeers := make([]peer.ID, 0, Backup)
		for _, pid := range peers {
			searchedPeers[pid] = struct{}{}
			bodys, nearerPeers, err := p.fetchChunkOrNearerPeers(ctx, req, pid)
			if err != nil {
				log.Error("mustFetchChunk", "fetchChunkOrNearerPeers error", err, "pid", pid, "chunk hash", hex.EncodeToString(req.ChunkHash))
				continue
			}
			if bodys != nil {
				return bodys, nil
			}
			for _, nearerPeer := range nearerPeers {
				if _, ok := searchedPeers[nearerPeer]; !ok {
					newPeers = append(newPeers, nearerPeer)
				}
			}
		}
		peers = newPeers
	}
	return nil, types2.ErrNotFound
}

//func (s *Protocol) fetchChunkOrNearerPeersAsync(ctx context.Context, param *types.ChunkInfoMsg, peers []peer.ID) (*types.BlockBodys, error) {
//	//用于遍历查询
//	peerCh := make(chan peer.ID, 1024)
//	//保存查询过的节点，防止二次查询
//	peersMap := make(map[peer.ID]struct{})
//	peersMap[s.Host.ID()] = struct{}{}
//	//用于接收并发查询的回复
//	responseCh := make(chan *types.BlockBodys, 1)
//	for _, pid := range peers {
//		peersMap[pid] = struct{}{}
//		peerCh <- pid
//	}
//	cancelCtx, cancelFunc := context.WithCancel(ctx)
//	defer cancelFunc()
//	// 多个节点并发请求
//	for peerID := range peerCh {
//		bodys, pids, err := s.fetchChunkOrNearerPeers(cancelCtx, param, peerID)
//		if err != nil {
//			log.Error("fetchChunkOrNearerPeersAsync", "fetch error", err, "peer id", peerID)
//			continue
//		}
//		responseCh <- bodys
//		//没查到区块数据，返回了更近的节点信息
//		for _, newPeer := range pids {
//			//过滤掉已经查询过的节点
//			if _, ok := peersMap[newPeer]; !ok {
//				peersMap[newPeer] = struct{}{}
//				peerCh <- newPeer
//			}
//		}
//
//		bodys := <-responseCh
//		if bodys != nil {
//			return bodys, nil
//		}
//	}
//
//	return nil, newPeerMap
//}

func (p *Protocol) fetchChunkOrNearerPeers(ctx context.Context, params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, []peer.ID, error) {
	childCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	stream, err := p.Host.NewStream(childCtx, pid, protocol.FetchChunk)
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
	err = protocol.ReadStreamAndAuthenticate(&res, stream)
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
			if len(p.Host.Peerstore().Addrs(addrInfo.ID)) == 0 {
				p.Host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, time.Hour)
			}
			peerList = append(peerList, addrInfo.ID)
		}
		return nil, peerList, nil
	}

	return nil, nil, errors.New(res.Error)
}

func (p *Protocol) checkLastChunk(in *types.ChunkInfoMsg) {
	l := in.End - in.Start + 1
	req := &types.ReqChunkRecords{
		Start: in.Start/l - 1,
		End:   in.End/l - 1,
	}
	if req.Start < 0 {
		return
	}
	records, err := p.getChunkRecordFromBlockchain(req)
	if err != nil || len(records.Infos) != 1 {
		log.Error("checkLastChunk", "getChunkRecordFromBlockchain error", err, "start", req.Start, "end", req.End, "records", records)
		return
	}
	info := &types.ChunkInfoMsg{
		ChunkHash: records.Infos[0].ChunkHash,
		Start:     records.Infos[0].Start,
		End:       records.Infos[0].End,
	}
	bodys, err := p.getChunk(info)
	if err == nil && bodys != nil && len(bodys.Items) == int(l) {
		return
	}
	//网络中找不到上一个chunk,先把上一个chunk保存到本地p2pstore
	log.Debug("checkLastChunk", "chunk num", info.Start, "chunk hash", hex.EncodeToString(info.ChunkHash))
	err = p.storeChunk(info)
	if err != nil {
		log.Error("checkLastChunk", "chunk hash", hex.EncodeToString(info.ChunkHash), "start", info.Start, "end", info.End, "error", err)
	}
}

func (p *Protocol) storeChunk(req *types.ChunkInfoMsg) error {
	//先检查上个chunk是否可以在网络中查到
	p.checkLastChunk(req)
	//如果p2pStore已保存数据，只更新时间即可
	if err := p.updateChunk(req); err == nil {
		return nil
	}
	//blockchain通知p2pStore保存数据，则blockchain应该有数据
	bodys, err := p.getChunkFromBlockchain(req)
	if err != nil {
		log.Error("StoreChunk", "getChunkFromBlockchain error", err)
		return err
	}
	err = p.addChunkBlock(req, bodys)
	if err != nil {
		log.Error("StoreChunk", "addChunkBlock error", err)
		return err
	}

	//本地存储之后立即到其他节点做一次备份
	p.notifyStoreChunk(req)
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
	resp, err := p.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if records, ok := resp.GetData().(*types.ChunkRecords); ok {
		return records, nil
	}

	return nil, types2.ErrNotFound
}

//func peersToMap(peers []peer.ID) map[peer.ID]struct{} {
//	m := make(map[peer.ID]struct{})
//	for _, pid := range peers {
//		m[pid] = struct{}{}
//	}
//	return m
//}
