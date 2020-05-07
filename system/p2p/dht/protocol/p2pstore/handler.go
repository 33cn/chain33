package p2pstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

var log = log15.New("module", "protocol.p2pstore")

type Protocol struct {
	*protocol.P2PEnv //协议共享接口变量

	saving              sync.Map
	notifying           sync.Map
	healthyRoutingTable *kb.RoutingTable
	localChunkInfo      map[string]LocalChunkInfo
	localChunkInfoMutex sync.RWMutex
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

func InitProtocol(env *protocol.P2PEnv) {
	p := &Protocol{
		P2PEnv:              env,
		healthyRoutingTable: kb.NewRoutingTable(dht.KValue, kb.ConvertPeerID(env.Host.ID()), time.Minute, env.Host.Peerstore()),
	}
	p.initLocalChunkInfoMap()

	for k, v := range p.localChunkInfo {
		v.Time = time.Now()
		p.localChunkInfo[k] = v
	}

	//注册p2p通信协议，用于处理节点之间请求
	p.Host.SetStreamHandler(protocol.FetchChunk, protocol.HandlerWithSignCheck(p.HandleStreamFetchChunk))
	p.Host.SetStreamHandler(protocol.StoreChunk, protocol.HandlerWithAuth(p.HandleStreamStoreChunk))
	p.Host.SetStreamHandler(protocol.GetHeader, protocol.HandlerWithSignCheck(p.HandleStreamGetHeader))
	p.Host.SetStreamHandler(protocol.GetChunkRecord, protocol.HandlerWithSignCheck(p.HandleStreamGetChunkRecord))
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, protocol.EventHandlerWithRecover(p.HandleEventNotifyStoreChunk))
	protocol.RegisterEventHandler(types.EventGetChunkBlock, protocol.EventHandlerWithRecover(p.HandleEventGetChunkBlock))
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, protocol.EventHandlerWithRecover(p.HandleEventGetChunkBlockBody))
	protocol.RegisterEventHandler(types.EventGetChunkRecord, protocol.EventHandlerWithRecover(p.HandleEventGetChunkRecord))

	go p.startRepublish()
	go p.startUpdateHealthyRoutingTable()
}

func (p *Protocol) HandleStreamFetchChunk(req *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
	param := req.Request.(*types.P2PRequest_ChunkInfoMsg).ChunkInfoMsg
	//优先检查本地是否存在
	bodys, _ := p.getChunkBlock(param.ChunkHash)
	if bodys != nil {
		l := int64(len(bodys.Items))
		start, end := param.Start%l, param.End%l+1
		bodys.Items = bodys.Items[start:end]
		res.Response = &types.P2PResponse_BlockBodys{BlockBodys: bodys}
		return nil
	}

	//本地没有数据
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(param.ChunkHash), Backup)
	var addrInfos []peer.AddrInfo
	for _, pid := range peers {
		if kb.Closer(p.Host.ID(), pid, genChunkPath(param.ChunkHash)) {
			continue
		}
		addrInfos = append(addrInfos, p.Host.Peerstore().PeerInfo(pid))
	}

	addrInfosData, err := json.Marshal(addrInfos)
	if err != nil {
		return err
	}
	res.Response = &types.P2PResponse_AddrInfo{AddrInfo: addrInfosData}
	return nil
}

// 对端节点通知本节点保存数据
/*
检查本节点p2pStore是否保存了数据，
	1）若已保存则只更新时间即可
	2）若未保存则从网络中请求chunk数据
*/
func (p *Protocol) HandleStreamStoreChunk(req *types.P2PRequest, stream network.Stream) {
	param := req.Request.(*types.P2PRequest_ChunkInfoMsg).ChunkInfoMsg
	chunkHashHex := hex.EncodeToString(param.ChunkHash)
	//该chunk正在保存
	if _, ok := p.saving.Load(chunkHashHex); ok {
		return
	}
	//已有其他节点通知该节点保存该chunk，正在网络中查找数据, 避免接收到多个节点的通知后重复查询数据
	if _, ok := p.notifying.LoadOrStore(chunkHashHex, nil); ok {
		return
	}
	defer p.notifying.Delete(chunkHashHex)

	//检查本地 p2pStore，如果已存在数据则直接更新
	if err := p.updateChunk(param); err == nil {
		return
	}

	//对端节点通知本节点保存数据，则对端节点应该有数据
	bodys, _, err := p.fetchChunkOrNearerPeers(context.Background(), param, stream.Conn().RemotePeer())
	if err != nil || bodys == nil {
		//对端节点没有数据，则从网络中搜索数据
		bodys, err = p.mustFetchChunk(param)
		if err != nil {
			log.Error("onStoreChunk", "get bodys from remote peer error", err)
			return
		}
	}

	err = p.addChunkBlock(param, bodys)
	if err != nil {
		log.Error("onStoreChunk", "store block error", err)
		return
	}
}

func (p *Protocol) HandleStreamGetHeader(req *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
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

func (p *Protocol) HandleStreamGetChunkRecord(req *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
	param := req.Request.(*types.P2PRequest_ReqChunkRecords).ReqChunkRecords
	records, err := p.getChunkRecordFromBlockchain(param)
	if err != nil {
		return err
	}
	res.Response = &types.P2PResponse_ChunkRecords{ChunkRecords: records}
	return nil
}

//HandleEventNotifyStoreChunk handles notification of blockchain,
// store chunk if this node is the nearest *BackUp* node in the local routing table.
func (p *Protocol) HandleEventNotifyStoreChunk(m *queue.Message) {
	m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk: true}))
	req := m.GetData().(*types.ChunkInfoMsg)
	//如果本节点是本地路由表中距离该chunk最近的 *count* 个节点之一，则保存数据；否则不需要保存数据
	count := Backup / 2
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), count)
	if len(peers) == count && kb.Closer(peers[count-1], p.Host.ID(), genChunkPath(req.ChunkHash)) {
		return
	}
	err := p.storeChunk(req)
	if err != nil {
		log.Error("StoreChunk", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		return
	}
	log.Info("StoreChunk", "local pid", p.Host.ID(), "chunk hash", hex.EncodeToString(req.ChunkHash))
}

func (p *Protocol) HandleEventGetChunkBlock(m *queue.Message) {
	m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk: true}))
	req := m.GetData().(*types.ChunkInfoMsg)
	bodys, err := p.getChunk(req)
	if err != nil {
		log.Error("GetChunkBlock", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		return
	}
	headers := p.getHeaders(&types.ReqBlocks{Start: req.Start, End: req.End})
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
	err = p.QueueClient.Send(msg, true)
	if err != nil {
		log.Error("EventGetChunkBlock", "reply message error", err)
	}
	//等待回复
	_, _ = p.QueueClient.Wait(msg)
}

func (p *Protocol) HandleEventGetChunkBlockBody(m *queue.Message) {
	req := m.GetData().(*types.ChunkInfoMsg)
	blockBodys, err := p.getChunk(req)
	if err != nil {
		log.Error("GetChunkBlockBody", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		m.ReplyErr("", err)
		return
	}
	m.Reply(&queue.Message{Data: blockBodys})
}

func (p *Protocol) HandleEventGetChunkRecord(m *queue.Message) {
	m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk: true}))
	req := m.GetData().(*types.ReqChunkRecords)
	records := p.getChunkRecords(req)
	if records == nil {
		log.Error("GetChunkRecord", "error", types2.ErrNotFound)
		return
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventAddChunkRecord, records)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		log.Error("EventGetChunkBlockBody", "reply message error", err)
	}
	//等待回复
	_, _ = p.QueueClient.Wait(msg)
}
