package p2pstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

const (
	FetchChunk     = "/chain33/fetch-chunk/" + types2.Version
	StoreChunk     = "/chain33/store-chunk/" + types2.Version
	GetHeader      = "/chain33/headers/" + types2.Version
	GetChunkRecord = "/chain33/chunk-record/" + types2.Version
)

var log = log15.New("module", "protocol.p2pstore")

type StoreProtocol struct {
	*protocol.P2PEnv //协议共享接口变量

	saving    sync.Map
	notifying sync.Map
}

func Init(env *protocol.P2PEnv) {
	p := &StoreProtocol{
		P2PEnv: env,
	}

	//注册p2p通信协议，用于处理节点之间请求
	p.Host.SetStreamHandler(FetchChunk, protocol.HandlerWithRW(p.HandleStreamFetchChunk))
	p.Host.SetStreamHandler(StoreChunk, protocol.HandlerWithRead(p.HandleStreamStoreChunk))
	p.Host.SetStreamHandler(GetHeader, protocol.HandlerWithRW(p.HandleStreamGetHeader))
	p.Host.SetStreamHandler(GetChunkRecord, protocol.HandlerWithRW(p.HandleStreamGetChunkRecord))
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, protocol.EventHandlerWithRecover(p.HandleEventNotifyStoreChunk))
	protocol.RegisterEventHandler(types.EventGetChunkBlock, protocol.EventHandlerWithRecover(p.HandleEventGetChunkBlock))
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, protocol.EventHandlerWithRecover(p.HandleEventGetChunkBlockBody))
	protocol.RegisterEventHandler(types.EventGetChunkRecord, protocol.EventHandlerWithRecover(p.HandleEventGetChunkRecord))

	go p.startRepublish()
}

func (s *StoreProtocol) HandleStreamFetchChunk(req *types.P2PRequest, res *types.P2PResponse) error {
	param := req.Request.(*types.P2PRequest_ChunkInfoMsg).ChunkInfoMsg
	//优先检查本地是否存在
	bodys, _ := s.getChunkBlock(param.ChunkHash)
	if bodys != nil {
		l := int64(len(bodys.Items))
		start, end := param.Start%l, param.End%l+1
		bodys.Items = bodys.Items[start:end]
		res.Response = &types.P2PResponse_BlockBodys{BlockBodys: bodys}
		return nil
	}

	//本地没有数据
	peers := s.Discovery.FindNearestPeers(peer.ID(genChunkPath(param.ChunkHash)), Backup)
	var addrInfos []peer.AddrInfo
	for _, pid := range peers {
		if kb.Closer(s.Host.ID(), pid, genChunkPath(param.ChunkHash)) {
			continue
		}
		addrInfos = append(addrInfos, s.Discovery.FindLocalPeer(pid))
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
func (s *StoreProtocol) HandleStreamStoreChunk(stream network.Stream, req *types.P2PRequest) {
	param := req.Request.(*types.P2PRequest_ChunkInfoMsg).ChunkInfoMsg
	chunkHashHex := hex.EncodeToString(param.ChunkHash)
	//该chunk正在保存
	if _, ok := s.saving.Load(chunkHashHex); ok {
		return
	}
	//已有其他节点通知该节点保存该chunk，正在网络中查找数据, 避免接收到多个节点的通知后重复查询数据
	if _, ok := s.notifying.LoadOrStore(chunkHashHex, nil); ok {
		return
	}
	defer s.notifying.Delete(chunkHashHex)

	//检查本地 p2pStore，如果已存在数据则直接更新
	if err := s.updateChunk(param); err == nil {
		return
	}

	//对端节点通知本节点保存数据，则对端节点应该有数据
	bodys, _, err := s.fetchChunkOrNearerPeers(context.Background(), param, stream.Conn().RemotePeer())
	if err != nil || bodys == nil {
		//对端节点没有数据，则从网络中搜索数据
		bodys, err = s.mustFetchChunk(param)
		if err != nil {
			log.Error("onStoreChunk", "get bodys from remote peer error", err)
			return
		}
	}

	err = s.addChunkBlock(param, bodys)
	if err != nil {
		log.Error("onStoreChunk", "store block error", err)
		return
	}
}

func (s *StoreProtocol) HandleStreamGetHeader(req *types.P2PRequest, res *types.P2PResponse) error {
	param := req.Request.(*types.P2PRequest_ReqBlocks)
	msg := s.QueueClient.NewMessage("blockchain", types.EventGetHeaders, param.ReqBlocks)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		return err
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		return err
	}

	if headers, ok := resp.GetData().(*types.Headers); ok {
		res.Response = &types.P2PResponse_BlockHeaders{BlockHeaders: headers}
		return nil
	}
	return types.ErrNotFound
}

func (s *StoreProtocol) HandleStreamGetChunkRecord(req *types.P2PRequest, res *types.P2PResponse) error {
	param := req.Request.(*types.P2PRequest_ReqChunkRecords).ReqChunkRecords
	records, err := s.getChunkRecordFromBlockchain(param)
	if err != nil {
		return err
	}
	res.Response = &types.P2PResponse_ChunkRecords{ChunkRecords: records}
	return nil
}

//HandleEventNotifyStoreChunk handles notification of blockchain,
// store chunk if this node is the nearest *BackUp* node in the local routing table.
func (s *StoreProtocol) HandleEventNotifyStoreChunk(m *queue.Message) {
	m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk: true}))
	req := m.GetData().(*types.ChunkInfoMsg)
	//如果本节点是本地路由表中距离该chunk最近的 *count* 个节点之一，则保存数据；否则不需要保存数据
	count := Backup
	peers := s.Discovery.FindNearestPeers(peer.ID(genChunkPath(req.ChunkHash)), count)
	if len(peers) == 0 {
		log.Error("HandleEventNotifyStoreChunk", "error", types2.ErrEmptyRoutingTable)
		return
	}
	if len(peers) == count && kb.Closer(peers[count-1], s.Host.ID(), genChunkPath(req.ChunkHash)) {
		return
	}
	err := s.storeChunk(req)
	if err != nil {
		log.Error("StoreChunk", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		return
	}
	log.Info("StoreChunk", "local pid", s.Host.ID(), "chunk hash", hex.EncodeToString(req.ChunkHash))
}

func (s *StoreProtocol) HandleEventGetChunkBlock(m *queue.Message) {
	m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk: true}))
	req := m.GetData().(*types.ChunkInfoMsg)
	bodys, err := s.getChunk(req)
	if err != nil {
		log.Error("GetChunkBlock", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		return
	}
	headers := s.getHeaders(&types.ReqBlocks{Start: req.Start, End: req.End})
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
	msg := s.QueueClient.NewMessage("blockchain", types.EventAddChunkBlock, &types.Blocks{Items: blockList})
	err = s.QueueClient.Send(msg, true)
	if err != nil {
		log.Error("EventGetChunkBlock", "reply message error", err)
	}
	//等待回复
	_, _ = s.QueueClient.Wait(msg)
}

func (s *StoreProtocol) HandleEventGetChunkBlockBody(m *queue.Message) {
	req := m.GetData().(*types.ChunkInfoMsg)
	blockBodys, err := s.getChunk(req)
	if err != nil {
		log.Error("GetChunkBlockBody", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "end", req.End, "error", err)
		m.ReplyErr("", err)
		return
	}
	m.Reply(&queue.Message{Data: blockBodys})
}

func (s *StoreProtocol) HandleEventGetChunkRecord(m *queue.Message) {
	fmt.Println(" >>>>>>>>>>>>>>>  into HandleEventGetChunkRecord")
	m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk: true}))
	req := m.GetData().(*types.ReqChunkRecords)
	records := s.getChunkRecords(req)
	if records == nil {
		log.Error("GetChunkRecord", "error", types2.ErrNotFound)
		return
	}
	msg := s.QueueClient.NewMessage("blockchain", types.EventAddChunkRecord, records)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		log.Error("EventGetChunkBlockBody", "reply message error", err)
	}
	fmt.Println(" >>>>>>>>>>>>>>>  HandleEventGetChunkRecord reply success")
	//等待回复
	_, _ = s.QueueClient.Wait(msg)
	fmt.Println(" >>>>>>>>>>>>>>>  HandleEventGetChunkRecord over")
}
