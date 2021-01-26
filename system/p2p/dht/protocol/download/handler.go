package download

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
)

func (p *Protocol) handleStreamDownloadBlock(stream network.Stream) {
	var req types.ReqBlocks
	err := protocol.ReadStream(&req, stream)
	if err != nil {
		log.Error("Handle", "err", err)
		return
	}
	//允许下载的最大高度区间为256
	if req.End-req.Start > 256 || req.End < req.Start {
		log.Error("handleStreamDownloadBlock", "error", "wrong parameter")
		return
	}

	msg := p.QueueClient.NewMessage("blockchain", types.EventGetBlocks, &req)
	err = p.QueueClient.Send(msg, true)
	if err != nil {
		return
	}
	reply, err := p.QueueClient.WaitTimeout(msg, time.Second*3)
	if err != nil {
		return
	}
	blocks := reply.Data.(*types.BlockDetails)
	if len(blocks.Items) == 0 {
		return
	}
	block := blocks.Items[0].Block
	err = protocol.WriteStream(block, stream)
	if err != nil {
		log.Error("WriteStream", "error", err, "remote pid", stream.Conn().RemotePeer().String())
		return
	}
	log.Debug("handleStreamDownloadBlock", "block height", block.GetHeight(), "remote peer", stream.Conn().RemotePeer().String())
}

func (p *Protocol) handleStreamDownloadBlockOld(stream network.Stream) {
	var data types.MessageGetBlocksReq
	err := protocol.ReadStream(&data, stream)
	if err != nil {
		log.Error("Handle", "err", err)
		return
	}
	req := types.ReqBlocks{
		Start: data.Message.StartHeight,
		End:   data.Message.EndHeight,
	}
	//允许下载的最大高度区间为256
	if req.End-req.Start > 256 || req.End < req.Start {
		log.Error("handleStreamDownloadBlock", "error", "wrong parameter")
		return
	}

	msg := p.QueueClient.NewMessage("blockchain", types.EventGetBlocks, &req)
	err = p.QueueClient.Send(msg, true)
	if err != nil {
		return
	}
	reply, err := p.QueueClient.WaitTimeout(msg, time.Second*3)
	if err != nil {
		return
	}
	blocks := reply.Data.(*types.BlockDetails)
	if len(blocks.Items) == 0 {
		log.Error("handleStreamDownloadBlockOld", "error", "block not found")
		return
	}
	var resp types.MessageGetBlocksResp
	var list []*types.InvData
	for _, blockDetail := range blocks.Items {
		list = append(list, &types.InvData{
			Ty: 2,
			Value: &types.InvData_Block{
				Block: blockDetail.Block,
			},
		})
	}
	resp.Message = &types.InvDatas{
		Items: list,
	}
	err = protocol.WriteStream(&resp, stream)
	if err != nil {
		log.Error("WriteStream", "error", err, "remote pid", stream.Conn().RemotePeer().String())
		return
	}
}

func (p *Protocol) handleEventDownloadBlock(msg *queue.Message) {
	req := msg.GetData().(*types.ReqBlocks)
	if req.GetStart() > req.GetEnd() {
		log.Error("handleEventDownloadBlock", "download start", req.GetStart(), "download end", req.GetEnd())
		msg.Reply(p.QueueClient.NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("start>end")}))
		return
	}
	pids := req.GetPid()
	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		log.Debug("GetBlocks:pid is nil")
		msg.Reply(p.QueueClient.NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(p.QueueClient.NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	var taskID = uuid.New().String() + "+" + fmt.Sprintf("%d-%d", req.GetStart(), req.GetEnd())

	log.Debug("handleEventDownloadBlock", "taskID", taskID, "download start", req.GetStart(), "download end", req.GetEnd(), "pids", pids)

	//具体的下载逻辑
	jobS := p.initJob(pids, taskID)
	log.Debug("handleEventDownloadBlock", "jobs", jobS)
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var maxGoroutine int32
	var reDownload = make(map[string]interface{})
	var startTime = time.Now().UnixNano()

	for height := req.GetStart(); height <= req.GetEnd(); height++ {
		wg.Add(1)
	Wait:
		if atomic.LoadInt32(&maxGoroutine) > 50 {
			time.Sleep(time.Millisecond * 200)
			goto Wait
		}
		atomic.AddInt32(&maxGoroutine, 1)
		go func(blockheight int64, tasks tasks) {
			err := p.downloadBlock(blockheight, tasks)
			if err != nil {
				mutex.Lock()
				defer mutex.Unlock()

				if err == p.Ctx.Err() {
					log.Error("syncDownloadBlock", "err", err.Error())
					return
				}

				log.Error("syncDownloadBlock", "downloadBlock err", err.Error())
				v, ok := reDownload[taskID]
				if ok {
					failedJob := v.(map[int64]bool)
					failedJob[blockheight] = false
					reDownload[taskID] = failedJob

				} else {
					var failedJob = make(map[int64]bool)
					failedJob[blockheight] = false
					reDownload[taskID] = failedJob

				}
			}
			wg.Done()
			atomic.AddInt32(&maxGoroutine, -1)

		}(height, jobS)

	}

	wg.Wait()
	p.checkTask(taskID, pids, reDownload)
	log.Debug("Download Job Complete!", "TaskID++++++++++++++", taskID,
		"cost time", fmt.Sprintf("cost time:%d ms", (time.Now().UnixNano()-startTime)/1e6),
		"from", pids)

}
