package p2pstore

import (
	"encoding/hex"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

func (p *Protocol) handleStreamRequestPeerInfoForChunk(stream network.Stream) {
	remotePid := stream.Conn().RemotePeer()
	localAddr := stream.Conn().LocalMultiaddr()
	var req types.P2PRequest
	err := protocol.ReadStream(&req, stream)
	protocol.CloseStream(stream)
	if err != nil {
		log.Error("handleStreamRequestPeerInfoForChunk", "ReadStream error", err)
		return
	}

	msg := req.Request.(*types.P2PRequest_ChunkInfoMsg).ChunkInfoMsg

	/*
		1. check local storage
		2. check chunk provider cache
		3. query the closer peers
	*/
	if _, ok := p.getChunkInfoByHash(msg.ChunkHash); ok {
		var addrs [][]byte
		for _, addr := range p.Host.Peerstore().Addrs(p.Host.ID()) {
			addrs = append(addrs, addr.Bytes())
		}
		if len(addrs) == 0 {
			log.Error("handleStreamRequestPeerInfoForChunk", "error", "no self addr", "conn local addr", localAddr)
			addrs = append(addrs, localAddr.Bytes())
		}
		info := types.PeerInfo{
			ID:        []byte(p.Host.ID()),
			MultiAddr: addrs,
		}
		if err := p.responsePeerInfoForChunk(&types.ChunkProvider{
			ChunkHash: msg.ChunkHash,
			PeerInfos: []*types.PeerInfo{&info},
		}, remotePid); err != nil {
			log.Error("handleStreamRequestPeerInfoForChunk", "responsePeerInfoForChunk error", err)
		}
	} else if peers := p.getChunkProviderCache(msg.ChunkHash); len(peers) != 0 {
		var infos []*types.PeerInfo
		for _, pid := range peers {
			var addrs [][]byte
			for _, addr := range p.Host.Peerstore().Addrs(pid) {
				addrs = append(addrs, addr.Bytes())
			}
			infos = append(infos, &types.PeerInfo{ID: []byte(pid), MultiAddr: addrs})
		}
		if err := p.responsePeerInfoForChunk(&types.ChunkProvider{
			ChunkHash: msg.ChunkHash,
			PeerInfos: infos,
		}, remotePid); err != nil {
			log.Error("handleStreamRequestPeerInfoForChunk", "responsePeerInfoForChunk error", err)
		}
	} else if exist := p.addChunkRequestTrace(msg.ChunkHash, remotePid); !exist {
		for _, remotePid := range p.RoutingTable.NearestPeers(genDHTID(msg.ChunkHash), AlphaValue) {
			err := p.requestPeerInfoForChunk(msg, remotePid)
			if err != nil {
				log.Error("handleStreamPeerAddrAsync", "requestPeerAddr error", err)
			}
		}
	}
}

func (p *Protocol) handleStreamResponsePeerInfoForChunk(stream network.Stream) {
	var req types.P2PRequest
	err := protocol.ReadStream(&req, stream)
	protocol.CloseStream(stream)
	if err != nil {
		log.Error("handleStreamResponsePeerInfoForChunk", "ReadStream error", err)
		return
	}

	provider := req.Request.(*types.P2PRequest_Provider).Provider
	chunkHash := hex.EncodeToString(provider.ChunkHash)

	// add provider cache
	savePeers(provider.PeerInfos, p.Host.Peerstore())
	for _, info := range provider.PeerInfos {
		p.addChunkProviderCache(provider.ChunkHash, peer.ID(info.ID))
	}

	p.wakeupMutex.Lock()
	if ch, ok := p.wakeup[chunkHash]; ok {
		ch <- struct{}{}
		delete(p.wakeup, chunkHash)
	}
	p.wakeupMutex.Unlock()

	p.chunkStatusCacheMutex.Lock()
	if cs, ok := p.chunkStatusCache[chunkHash]; ok {
		// 防止重复下载
		if cs.status == Waiting {
			select {
			case p.chunkToDownload <- cs.info:
				cs.status = ToDownload
				p.chunkStatusCache[chunkHash] = cs
			default:
				log.Error("handleStreamResponsePeerInfoForChunk", "error", "chunkToDownload channel is full")
			}
		}
	}
	p.chunkStatusCacheMutex.Unlock()

	// check trace and response
	for _, pid := range p.getChunkRequestTrace(provider.ChunkHash) {
		// delete trace and response
		p.removeChunkRequestTrace(provider.ChunkHash, pid)
		//TODO: retry when error occurs
		if err := p.responsePeerInfoForChunk(provider, pid); err != nil {
			log.Error("handleStreamResponsePeerInfoForChunk", "responsePeerInfoForChunk error", err, "pid", pid)
		}
	}
}

func (p *Protocol) handleStreamRequestPeerAddr(stream network.Stream) {
	remotePid := stream.Conn().RemotePeer()
	var req types.P2PRequest
	err := protocol.ReadStream(&req, stream)
	protocol.CloseStream(stream)
	if err != nil {
		log.Error("handleStreamRequestPeerAddr", "ReadStream error", err)
		return
	}

	pid := req.GetRequest().(*types.P2PRequest_Pid).Pid
	var addrs [][]byte
	for _, addr := range p.Host.Peerstore().Addrs(peer.ID(pid)) {
		addrs = append(addrs, addr.Bytes())
	}
	if len(addrs) != 0 {
		// response
		err := p.responsePeerAddr(&types.PeerInfo{ID: []byte(pid), MultiAddr: addrs}, remotePid)
		if err != nil {
			log.Error("handleStreamRequestPeerAddr", "responsePeerAddr error", err)
		}
	} else if exist := p.addPeerAddrRequestTrace(peer.ID(pid), remotePid); !exist {
		for _, remotePid := range p.RoutingTable.NearestPeers(kb.ConvertPeerID(peer.ID(pid)), AlphaValue) {
			err := p.requestPeerAddr(peer.ID(pid), remotePid)
			if err != nil {
				log.Error("handleStreamRequestPeerAddr", "requestPeerAddr error", err)
			}
		}
	}
}

func (p *Protocol) handleStreamResponsePeerAddr(stream network.Stream) {
	var req types.P2PRequest
	err := protocol.ReadStream(&req, stream)
	protocol.CloseStream(stream)
	if err != nil {
		log.Error("handleStreamResponsePeerAddr", "ReadStream error", err)
		return
	}

	info := req.GetRequest().(*types.P2PRequest_PeerInfo).PeerInfo
	savePeers([]*types.PeerInfo{info}, p.Host.Peerstore())

	// response and delete trace
	for _, pid := range p.getPeerAddrRequestTrace(peer.ID(info.ID)) {
		p.removePeerAddrRequestTrace(peer.ID(info.ID), pid)
		if err := p.responsePeerAddr(info, pid); err != nil {
			log.Error("handleStreamRequestPeerAddr", "responsePeerAddr error", err)
		}
	}
}

func (p *Protocol) handleStreamFetchActivePeer(res *types.P2PResponse) error {
	var peerInfos []*types.PeerInfo
	for _, pid := range p.RoutingTable.ListPeers() {
		var addrs [][]byte
		for _, addr := range p.Host.Peerstore().Addrs(pid) {
			addrs = append(addrs, addr.Bytes())
		}
		peerInfos = append(peerInfos, &types.PeerInfo{
			ID:        []byte(pid),
			MultiAddr: addrs,
		})

	}
	res.Response = &types.P2PResponse_PeerInfos{
		PeerInfos: &types.PeerInfoList{
			PeerInfos: peerInfos,
		},
	}
	return nil
}

func (p *Protocol) handleStreamPeerAddr(req *types.P2PRequest, res *types.P2PResponse) error {
	pid := req.GetRequest().(*types.P2PRequest_Pid).Pid
	var addrs [][]byte
	for _, addr := range p.Host.Peerstore().Addrs(peer.ID(pid)) {
		addrs = append(addrs, addr.Bytes())
	}

	res.Response = &types.P2PResponse_PeerInfo{
		PeerInfo: &types.PeerInfo{
			ID:        []byte(pid),
			MultiAddr: addrs,
		},
	}
	return nil
}

//TODO:
// Deprecated:
func (p *Protocol) handleStreamIsFullNode(resp *types.P2PResponse) error {
	resp.Response = &types.P2PResponse_NodeInfo{
		NodeInfo: &types.NodeInfo{
			Answer: p.SubConfig.IsFullNode,
			Height: p.PeerInfoManager.PeerHeight(p.Host.ID()),
		},
	}
	return nil
}

func (p *Protocol) handleStreamFetchShardPeers(req *types.P2PRequest, res *types.P2PResponse) error {
	reqPeers := req.GetRequest().(*types.P2PRequest_ReqPeers).ReqPeers
	count := int(reqPeers.Count)
	if count <= 0 {
		count = backup
	}
	peers := p.RoutingTable.NearestPeers(genDHTID(reqPeers.ReferKey), p.RoutingTable.Size())
	var activePeers []peer.ID
	for _, pid := range peers {
		if p.PeerInfoManager.PeerHeight(pid)+512 > p.PeerInfoManager.PeerHeight(p.Host.ID()) {
			activePeers = append(activePeers, pid)
		}
		if len(activePeers) >= count {
			break
		}
	}
	for _, pid := range activePeers {
		var addrs [][]byte
		for _, addr := range p.Host.Peerstore().Addrs(pid) {
			addrs = append(addrs, addr.Bytes())
		}
		res.CloserPeers = append(res.CloserPeers, &types.PeerInfo{
			ID:        []byte(pid),
			MultiAddr: addrs,
		})
	}
	return nil
}

func (p *Protocol) handleStreamFetchChunk(stream network.Stream) {
	var req types.P2PRequest
	if err := protocol.ReadStreamAndAuthenticate(&req, stream); err != nil {
		return
	}
	param := req.Request.(*types.P2PRequest_ChunkInfoMsg).ChunkInfoMsg
	var res types.P2PResponse
	defer func() {
		_ = protocol.WriteStream(&res, stream)
		log.Info("handleStreamFetchChunk", "chunk hash", hex.EncodeToString(param.ChunkHash), "start", param.Start, "remote peer", stream.Conn().RemotePeer(), "addrs", stream.Conn().RemoteMultiaddr())
	}()

	peers := p.RoutingTable.NearestPeers(genDHTID(param.ChunkHash), p.RoutingTable.Size())
	var closerPeers []peer.ID
	for _, pid := range peers {
		if p.PeerInfoManager.PeerHeight(pid) > param.End+2048 && kb.Closer(pid, p.Host.ID(), genChunkNameSpaceKey(param.ChunkHash)) {
			closerPeers = append(closerPeers, pid)
		}
		if len(closerPeers) >= AlphaValue {
			break
		}
	}
	if len(closerPeers) == 0 {
		closerPeers = p.RoutingTable.NearestPeers(genDHTID(param.ChunkHash), AlphaValue)
	}
	for _, pid := range closerPeers {
		var addrs [][]byte
		for _, addr := range p.Host.Peerstore().Addrs(pid) {
			addrs = append(addrs, addr.Bytes())
		}
		res.CloserPeers = append(res.CloserPeers, &types.PeerInfo{
			ID:        []byte(pid),
			MultiAddr: addrs,
		})

	}
	if atomic.LoadInt64(&p.concurrency) > maxConcurrency {
		return
	}
	atomic.AddInt64(&p.concurrency, 1)
	defer atomic.AddInt64(&p.concurrency, -1)
	//分片节点模式,检查本地是否存在
	bodys, err := p.getChunkBlock(param)
	if err != nil {
		res.Error = err.Error()
		return
	}
	t := time.Now()
	writeBodys(bodys, stream)
	log.Info("handleStreamFetchChunk", "bodys len", len(bodys.Items), "write bodys cost", time.Since(t))
}

func (p *Protocol) handleStreamGetHeader(req *types.P2PRequest, res *types.P2PResponse) error {
	param := req.Request.(*types.P2PRequest_ReqBlocks)
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetHeaders, param.ReqBlocks)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return err
	}
	resp, err := p.QueueClient.Wait(msg)
	if err != nil {
		return err
	}

	if headers, ok := resp.GetData().(*types.Headers); ok {
		res.Response = &types.P2PResponse_BlockHeaders{BlockHeaders: headers}
		return nil
	}
	return types.ErrNotFound
}

func (p *Protocol) handleStreamGetHeaderOld(stream network.Stream) {
	var req types.MessageHeaderReq
	err := protocol.ReadStream(&req, stream)
	if err != nil {
		return
	}
	param := &types.ReqBlocks{
		Start: req.Message.StartHeight,
		End:   req.Message.EndHeight,
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetHeaders, param)
	err = p.QueueClient.Send(msg, true)
	if err != nil {
		return
	}
	resp, err := p.QueueClient.Wait(msg)
	if err != nil {
		return
	}

	if headers, ok := resp.GetData().(*types.Headers); ok {
		err = protocol.WriteStream(&types.MessageHeaderResp{
			Message: &types.P2PHeaders{
				Headers: headers.Items,
			},
		}, stream)
		if err != nil {
			return
		}
	}

}

func (p *Protocol) handleStreamGetChunkRecord(req *types.P2PRequest, res *types.P2PResponse) error {
	param := req.Request.(*types.P2PRequest_ReqChunkRecords).ReqChunkRecords
	records, err := p.getChunkRecordFromBlockchain(param)
	if err != nil {
		return err
	}
	res.Response = &types.P2PResponse_ChunkRecords{ChunkRecords: records}
	return nil
}

//handleEventNotifyStoreChunk handles notification of blockchain,
// store chunk if this node is the nearest *count* node in the local routing table.
func (p *Protocol) handleEventNotifyStoreChunk(m *queue.Message) {
	req := m.GetData().(*types.ChunkInfoMsg)
	var err error
	defer func() {
		m.Reply(p.QueueClient.NewMessage("blockchain", 0, &types.Reply{
			IsOk: err == nil,
		}))
	}()
	if p.SubConfig.IsFullNode {
		//全节点保存所有chunk, blockchain模块通知保存chunk时直接保存到本地
		if err = p.storeChunk(req); err != nil {
			log.Error("HandleEventNotifyStoreChunk", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		}
		return
	}

	//如果本节点是扩展路由表中距离该chunk最近的 backup 个节点之一，则保存数据；否则不需要保存数据
	extendRoutingTable := p.getExtendRoutingTable()
	pids := extendRoutingTable.NearestPeers(genDHTID(req.ChunkHash), backup)
	if len(pids) >= backup && kb.Closer(pids[backup-1], p.Host.ID(), genChunkNameSpaceKey(req.ChunkHash)) {
		return
	}
	log.Info("handleEventNotifyStoreChunk", "local nearest peer", p.Host.ID(), "chunk hash", hex.EncodeToString(req.ChunkHash))
	if err = p.storeChunk(req); err != nil {
		log.Error("HandleEventNotifyStoreChunk", "storeChunk error", err, "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End)
	}
}

func (p *Protocol) handleEventGetChunkBlock(m *queue.Message) {
	req := m.GetData().(*types.ChunkInfoMsg)
	bodys, _, err := p.getChunk(req)
	if err != nil {
		log.Error("GetChunkBlock", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		return
	}
	headers, _ := p.getHeaders(&types.ReqBlocks{Start: req.Start, End: req.End})
	if headers == nil {
		log.Error("GetBlockHeader", "error", types2.ErrNotFound)
		return
	}
	if len(headers.Items) != len(bodys.Items) {
		log.Error("GetBlockHeader", "error", types2.ErrLength, "header length", len(headers.Items), "body length", len(bodys.Items), "start", req.Start, "end", req.End)
		return
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
	msg := p.QueueClient.NewMessage("blockchain", types.EventAddChunkBlock, &types.Blocks{Items: blockList})
	err = p.QueueClient.Send(msg, false)
	if err != nil {
		log.Error("EventGetChunkBlock", "reply message error", err)
		return
	}
	log.Info("GetChunkBlock", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End)
}

func (p *Protocol) handleEventGetChunkBlockBody(m *queue.Message) {
	req := m.GetData().(*types.ChunkInfoMsg)
	blockBodys, _, err := p.getChunk(req)
	if err != nil {
		log.Error("GetChunkBlockBody", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		m.ReplyErr("", err)
		return
	}
	m.Reply(&queue.Message{Data: blockBodys})
}

func (p *Protocol) handleEventGetChunkRecord(m *queue.Message) {
	req := m.GetData().(*types.ReqChunkRecords)
	records := p.getChunkRecords(req)
	if records == nil {
		log.Error("handleEventGetChunkRecord", "getChunkRecords error", types2.ErrNotFound)
		return
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventAddChunkRecord, records)
	err := p.QueueClient.Send(msg, false)
	if err != nil {
		log.Error("handleEventGetChunkRecord", "reply message error", err)
	}
}

func (p *Protocol) handleEventGetHeaders(m *queue.Message) {
	req := m.GetData().(*types.ReqBlocks)
	if len(req.GetPid()) == 0 { //根据指定的pidlist 获取对应的block header
		log.Debug("GetHeaders:pid is nil")
		m.Reply(p.QueueClient.NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}
	m.Reply(p.QueueClient.NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	headers, pid := p.getHeadersOld(req)
	if headers == nil || len(headers.Items) == 0 {
		return
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventAddBlockHeaders, &types.HeadersPid{Pid: pid.Pretty(), Headers: headers})
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		log.Error("handleEventGetHeaders", "send message error", err)
		return
	}
	_, _ = p.QueueClient.WaitTimeout(msg, time.Second)
}

func writeBodys(bodys *types.BlockBodys, stream network.Stream) {
	if bodys == nil {
		return
	}
	var data types.P2PResponse
	for _, body := range bodys.Items {
		data.Response = &types.P2PResponse_BlockBody{
			BlockBody: body,
		}
		if err := protocol.WriteStream(&data, stream); err != nil {
			log.Error("writeBodys", "error", err)
			return
		}
	}
}
