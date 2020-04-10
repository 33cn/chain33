package p2pstore

import (
	"bufio"
	"bytes"
	"fmt"
	"runtime"
	"sync"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
)

const (
	FetchChunk     = "/chain33/fetch-chunk/1.0.0"
	StoreChunk     = "/chain33/store-chunk/1.0.0"
	GetHeader      = "/chain33/headers/1.0.0"
	GetChunkRecord = "/chain33/chunk-record/1.0.0"
)

var log = log15.New("module", "protocol.p2pstore")

type StoreProtocol struct {
	protocol.BaseProtocol //default协议实现
	*protocol.P2PEnv      //协议共享接口变量

	saving sync.Map
}

func Init(env *protocol.P2PEnv) {
	p := &StoreProtocol{
		P2PEnv: env,
	}

	//注册p2p通信协议，用于处理节点之间请求
	p.Host.SetStreamHandler(StoreChunk, p.Handle)
	p.Host.SetStreamHandler(FetchChunk, p.Handle)
	p.Host.SetStreamHandler(GetHeader, p.Handle)
	p.Host.SetStreamHandler(GetChunkRecord, p.Handle)
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, p.HandleEvent)
	protocol.RegisterEventHandler(types.EventGetChunkBlock, p.HandleEvent)
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, p.HandleEvent)
	protocol.RegisterEventHandler(types.EventGetChunkRecord, p.HandleEvent)

	go p.startRepublish()
}

// Handle 处理节点之间的请求
func (s *StoreProtocol) Handle(stream network.Stream) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("handle stream", "panic error", r)
			fmt.Println(string(panicTrace(4)))
			stream.Reset()
		} else {
			stream.Close()
		}

	}()

	// Create a buffer stream for non blocking read and write
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	var req types.P2PStoreRequest
	err := readMessage(rw.Reader, &req)
	if err != nil {
		log.Error("handle", "read request error", err)
		stream.Reset()
		return
	}
	//distribute message to corresponding handler
	switch req.ProtocolID {
	//不同的协议交给不同的处理逻辑
	case FetchChunk:
		// reply -> *types.BlockBodys
		s.onFetchChunk(rw.Writer, req.Data.(*types.P2PStoreRequest_ChunkInfoMsg).ChunkInfoMsg)
	case StoreChunk:
		// reply -> none
		s.onStoreChunk(stream, req.Data.(*types.P2PStoreRequest_ChunkInfoMsg).ChunkInfoMsg)
	case GetHeader:
		// reply -> *types.Headers
		s.onGetHeader(rw.Writer, req.Data.(*types.P2PStoreRequest_ReqBlocks).ReqBlocks)
	case GetChunkRecord:
		// reply -> *types.ChunkRecords
		s.onGetChunkRecord(rw.Writer, req.Data.(*types.P2PStoreRequest_ReqChunkRecords).ReqChunkRecords)
	default:
		log.Error("Handle", "error", types2.ErrProtocolNotSupport, "protocol", req.ProtocolID)
	}
	//TODO 管理connection
}

// HandleEvent 处理模块之间的事件
func (s *StoreProtocol) HandleEvent(m *queue.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("handle event", "panic error", r)
			fmt.Println(string(panicTrace(4)))
		}
	}()
	switch m.Ty {
	// 检查本节点是否需要进行区块数据归档
	case types.EventNotifyStoreChunk:
		m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk:true}))
		err := s.StoreChunk(m.GetData().(*types.ChunkInfoMsg))
		if err != nil {
			log.Error("HandleEvent", "storeChunk error", err)
		}


	// 获取chunkBlock数据
	case types.EventGetChunkBlock:
		m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk:true}))
		req := m.GetData().(*types.ChunkInfoMsg)
		bodys, err := s.GetChunk(req)
		if err != nil {
			log.Error("HandleEvent", "GetChunk error", err)
			return
		}
		headers := s.getHeaders(&types.ReqBlocks{Start: req.Start, End: req.End})
		if len(headers.Items) != len(bodys.Items) {
			log.Error("GetBlockHeader", "error", types2.ErrLength, "header length", len(headers.Items), "body length", len(bodys.Items))
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

	// 获取chunkBody数据
	case types.EventGetChunkBlockBody:
		blockBodys, err := s.GetChunk(m.GetData().(*types.ChunkInfoMsg))
		if err != nil {
			log.Error("HandleEvent", "GetChunkBlockBody error", err)
			m.ReplyErr("", err)
			return
		}
		m.Reply(&queue.Message{Data: blockBodys})

	// 获取归档索引
	case types.EventGetChunkRecord:
		m.Reply(queue.NewMessage(0, "", 0, &types.Reply{IsOk:true}))
		req := m.GetData().(*types.ReqChunkRecords)
		records := s.getChunkRecords(req)
		if records == nil {
			log.Error("HandleEvent", "GetChunkRecord error", types2.ErrNotFound)
			return
		}
		msg := s.QueueClient.NewMessage("blockchain", types.EventAddChunkRecord, records)
		err := s.QueueClient.Send(msg, true)
		if err != nil {
			log.Error("EventGetChunkBlockBody", "reply message error", err)
		}
		//等待回复
		_, _ = s.QueueClient.Wait(msg)
	}
}

//// VerifyRequest  验证请求数据
//func (s *StoreProtocol) VerifyRequest(message proto.Message, messageComm *types.MessageComm) bool {
//	return true
//}
//
//// SignMessage 对消息签名
//func (s *StoreProtocol) SignProtoMessage(message proto.Message) ([]byte, error) {
//	return nil, nil
//}

// panicTrace trace panic stack info.
func panicTrace(kb int) []byte {
	s := []byte("/src/runtime/panic.go")
	e := []byte("\ngoroutine ")
	line := []byte("\n")
	stack := make([]byte, kb<<10) //4KB
	length := runtime.Stack(stack, true)
	start := bytes.Index(stack, s)
	stack = stack[start:length]
	start = bytes.Index(stack, line) + 1
	stack = stack[start:]
	end := bytes.LastIndex(stack, line)
	if end != -1 {
		stack = stack[:end]
	}
	end = bytes.Index(stack, e)
	if end != -1 {
		stack = stack[:end]
	}
	stack = bytes.TrimRight(stack, "\n")
	return stack
}