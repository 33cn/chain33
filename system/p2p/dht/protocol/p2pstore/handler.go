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
	kb "github.com/libp2p/go-libp2p-kbucket"
)

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
	count := reqPeers.Count
	if count <= 0 {
		count = Backup
	}
	closerPeers := p.ShardHealthyRoutingTable.NearestPeers(genDHTID(reqPeers.ReferKey), int(count))
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
	return nil
}

func (p *Protocol) handleStreamFetchChunk(stream network.Stream) {
	var req types.P2PRequest
	if err := protocol.ReadStreamAndAuthenticate(&req, stream); err != nil {
		return
	}
	param := req.Request.(*types.P2PRequest_ChunkInfoMsg).ChunkInfoMsg
	var res types.P2PResponse
	var bodys *types.BlockBodys
	var err error
	defer func() {
		t := time.Now()
		writeBodys(bodys, stream)
		_ = protocol.WriteStream(&res, stream)
		log.Info("handleStreamFetchChunk", "chunk hash", hex.EncodeToString(param.ChunkHash), "start", param.Start, "remote peer", stream.Conn().RemotePeer(), "addrs", stream.Conn().RemoteMultiaddr(), "time cost", time.Since(t))
	}()

	// 全节点模式，只有网络中出现数据丢失时才提供数据
	if p.SubConfig.IsFullNode {
		hexHash := hex.EncodeToString(param.ChunkHash)
		if _, ok := p.chunkWhiteList.Load(hexHash); !ok { //该chunk不在白名单里
			if pid, ok := p.checkChunkInNetwork(param); ok {
				//网络中可以查到数据，不应该到全节点来要数据
				var addrs [][]byte
				for _, addr := range p.Host.Peerstore().Addrs(pid) {
					addrs = append(addrs, addr.Bytes())
				}
				res.CloserPeers = []*types.PeerInfo{{ID: []byte(pid), MultiAddr: addrs}}
				return
			}

			//该chunk添加到白名单，10分钟内无条件提供数据
			p.chunkWhiteList.Store(hexHash, time.Now())
			//分片网络中出现数据丢失，备份该chunk到分片网络中
			go func() {
				chunkInfo, ok := p.getChunkInfoByHash(param.ChunkHash)
				if !ok {
					log.Error("HandleStreamFetchChunk chunkInfo not found", "chunk hash", hexHash)
					return
				}
				p.notifyStoreChunk(chunkInfo.ChunkInfoMsg)
			}()

		}
		bodys, err = p.getChunkBlock(param)
		if err != nil {
			res.Error = err.Error()
			return
		}
		return
	}

	closerPeers := p.ShardHealthyRoutingTable.NearestPeers(genDHTID(param.ChunkHash), AlphaValue)
	if len(closerPeers) != 0 && kb.Closer(p.Host.ID(), closerPeers[0], genChunkNameSpaceKey(param.ChunkHash)) {
		closerPeers = p.ShardHealthyRoutingTable.NearestPeers(genDHTID(param.ChunkHash), Backup-1)
	}
	for _, pid := range closerPeers {
		if pid == p.Host.ID() {
			continue
		}
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
	bodys, err = p.getChunkBlock(param)
	if err != nil {
		res.Error = err.Error()
		return
	}
}

// 对端节点通知本节点保存数据
/*
检查点p2pStore是否保存了数据，
	1）若已保存则只更新时间即可
	2）若未保存则从网络中请求chunk数据
*/
func (p *Protocol) handleStreamStoreChunks(req *types.P2PRequest) {
	param := req.Request.(*types.P2PRequest_ChunkInfoList).ChunkInfoList.Items
	for _, info := range param {
		chunkHash := hex.EncodeToString(info.ChunkHash)
		//已有其他节点通知该节点保存该chunk，避免接收到多个节点的通知后重复查询数据
		if _, ok := p.notifying.LoadOrStore(chunkHash, nil); ok {
			continue
		}
		//检查本地 p2pStore，如果已存在数据则直接更新
		if err := p.updateChunk(info); err == nil {
			p.notifying.Delete(chunkHash)
			continue
		}
		//send message to notifying queue to process
		select {
		case p.notifyingQueue <- info:
			//drop the notify message if queue is full
		default:
			p.notifying.Delete(chunkHash)
		}
	}
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
			log.Error("HandleEventNotifyStoreChunk", "storeChunk error", err)
		}
		return
	}

	//如果本节点是本地路由表中距离该chunk最近的节点，则保存数据；否则不需要保存数据
	tmpRoutingTable := p.genTempRoutingTable(req.ChunkHash, 100)
	pid := tmpRoutingTable.NearestPeer(genDHTID(req.ChunkHash))
	if pid != "" && kb.Closer(pid, p.Host.ID(), genChunkNameSpaceKey(req.ChunkHash)) {
		return
	}
	log.Info("handleEventNotifyStoreChunk", "local nearest peer", p.Host.ID(), "chunk hash", hex.EncodeToString(req.ChunkHash))
	err = p.checkNetworkAndStoreChunk(req)
	if err != nil {
		log.Error("storeChunk", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		return
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
	_, _ = p.QueueClient.Wait(msg)
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
			return
		}
	}
}
