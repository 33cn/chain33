package download

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	//protobufCodec "github.com/multiformats/go-multicodec/protobuf"

	"github.com/33cn/chain33/common/log/log15"

	core "github.com/libp2p/go-libp2p-core"

	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	uuid "github.com/google/uuid"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	log = log15.New("module", "p2p.download")
)

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &downloadProtol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, DownloadBlockReq, &downloadHander{})
}

const (
	protoTypeID      = "DownloadProtocolType"
	DownloadBlockReq = "/chain33/downloadBlockReq/1.0.0"
)

//type Istream
type downloadProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler
}

func (d *downloadProtol) InitProtocol(env *prototypes.P2PEnv) {
	d.P2PEnv = env
	//注册事件处理函数
	prototypes.RegisterEventHandler(types.EventFetchBlocks, d.handleEvent)

}

type downloadHander struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (d *downloadHander) Handle(stream core.Stream) {
	protocol := d.GetProtocol().(*downloadProtol)

	//解析处理
	if stream.Protocol() == DownloadBlockReq {
		var data types.MessageGetBlocksReq
		err := d.ReadProtoMessage(&data, stream)
		if err != nil {
			log.Error("Handle", "err", err)
			return
		}
		recvData := data.Message
		protocol.OnReq(data.MessageData.Id, recvData, stream)
	}

}

func (d *downloadProtol) OnReq(id string, message *types.P2PGetBlocks, s core.Stream) {
	defer s.Close()
	//获取headers 信息
	//允许下载的最大高度区间为256
	if message.GetEndHeight()-message.GetStartHeight() > 256 || message.GetEndHeight() < message.GetStartHeight() {
		return

	}
	//开始下载指定高度
	log.Info("OnReq", "start", message.GetStartHeight(), "end", message.GetStartHeight())

	reqblock := &types.ReqBlocks{Start: message.GetStartHeight(), End: message.GetEndHeight()}
	client := d.GetQueueClient()
	msg := client.NewMessage("blockchain", types.EventGetBlocks, reqblock)
	err := client.Send(msg, true)
	if err != nil {
		log.Error("GetBlocks", "Error", err.Error())
		return
	}
	resp, err := client.WaitTimeout(msg, time.Second*20)
	if err != nil {
		log.Error("GetBlocks Err", "blockchain Err", err.Error(), "blockheight", message.GetStartHeight())
		return
	}

	blockDetails := resp.Data.(*types.BlockDetails)
	var p2pInvData = make([]*types.InvData, 0)
	var invdata types.InvData
	for _, item := range blockDetails.Items {
		invdata.Reset()
		invdata.Ty = 2 //2 block,1 tx
		invdata.Value = &types.InvData_Block{Block: item.Block}
		p2pInvData = append(p2pInvData, &invdata)
	}

	peerID := d.GetHost().ID()
	pubkey, _ := d.GetHost().Peerstore().PubKey(peerID).Bytes()
	blocksResp := &types.MessageGetBlocksResp{MessageData: d.NewMessageCommon(id, peerID.Pretty(), pubkey, false),
		Message: &types.InvDatas{Items: p2pInvData}}

	err = d.SendProtoMessage(blocksResp, s)
	if err != nil {
		log.Error("SendProtoMessage", "err", err)
		return
	}

	log.Info("OnReq", "Send Block Height+++++++", blocksResp.Message.GetItems()[0].GetBlock().GetHeight(), "send  to", s.Conn().RemotePeer().String())

}

//GetBlocks 接收来自chain33 blockchain模块发来的请求
func (d *downloadProtol) handleEvent(msg *queue.Message) {

	req := msg.GetData().(*types.ReqBlocks)
	if req.GetStart() > req.GetEnd() {
		log.Error("handleEvent", "download start", req.GetStart(), "download end", req.GetEnd())

		msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("start>end")}))
		return
	}
	pids := req.GetPid()
	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		log.Info("GetBlocks:pid is nil")
		msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	var taskID = uuid.New().String() + "+" + fmt.Sprintf("%d-%d", req.GetStart(), req.GetEnd())

	log.Info("handleEvent", "taskID", taskID, "download start", req.GetStart(), "download end", req.GetEnd(), "pids", pids)

	//具体的下载逻辑
	jobS := d.initJob(pids, taskID)
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var maxgoroutin int32
	var reDownload = make(map[string]interface{})
	var startTime = time.Now().UnixNano()

	for height := req.GetStart(); height <= req.GetEnd(); height++ {
		wg.Add(1)
	Wait:
		if maxgoroutin > 50 {
			time.Sleep(time.Millisecond * 200)
			goto Wait
		}
		atomic.AddInt32(&maxgoroutin, 1)
		go func(blockheight int64, tasks Tasks) {
			err := d.downloadBlock(blockheight, tasks)
			if err != nil {
				mutex.Lock()
				defer mutex.Unlock()

				log.Error("syncDownloadBlock", "err", err.Error())
				v, ok := reDownload[taskID]
				if ok {
					faildJob := v.(map[int64]bool)
					faildJob[blockheight] = false
					//faildJob.Store(blockheight, false)
					reDownload[taskID] = faildJob

				} else {
					var faildJob = make(map[int64]bool)
					faildJob[blockheight] = false
					//faildJob.Store(blockheight, false)
					reDownload[taskID] = faildJob

				}
			}
			wg.Done()
			atomic.AddInt32(&maxgoroutin, -1)

		}(height, jobS)

	}

	wg.Wait()
	d.CheckTask(taskID, pids, reDownload)
	log.Info("Download Job Complete!", "TaskID++++++++++++++", taskID,
		"cost time", fmt.Sprintf("cost time:%d ms", (time.Now().UnixNano()-startTime)/1e6),
		"from", pids)

}

func (d *downloadProtol) downloadBlock(blockheight int64, tasks Tasks) error {
	var retryCount uint
	tasks.Sort() //对任务节点时延进行排序，优先选择时延低的节点进行下载
ReDownload:
	if tasks.Size() == 0 {
		return errors.New("no peer for download")
	}

	retryCount++
	if retryCount > 50 {
		return errors.New("beyound max try count 50")
	}

	task := d.availbTask(tasks, blockheight)
	if task == nil {
		time.Sleep(time.Millisecond * 400)
		goto ReDownload
	}

	var downloadStart = time.Now().UnixNano()

	getblocks := &types.P2PGetBlocks{StartHeight: blockheight, EndHeight: blockheight,
		Version: 0}

	peerID := d.GetHost().ID()
	pubkey, _ := d.GetHost().Peerstore().PubKey(peerID).Bytes()
	blockReq := &types.MessageGetBlocksReq{MessageData: d.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
		Message: getblocks}

	req := &prototypes.StreamRequest{
		PeerID:  task.Pid,
		Host:    d.GetHost(),
		Data:    blockReq,
		ProtoID: DownloadBlockReq,
	}
	var resp types.MessageGetBlocksResp
	err := d.StreamSendHandler(req, &resp)
	if err != nil {
		log.Error("handleEvent", "StreamSendHandler", err, "pid", task.Pid)
		d.releaseJob(task)
		tasks = tasks.Remove(task.Index)
		goto ReDownload
	}

	block := resp.GetMessage().GetItems()[0].GetBlock()
	remotePid := task.Pid.Pretty()
	costTime := (time.Now().UnixNano() - downloadStart) / 1e6

	log.Debug("download+++++", "from", remotePid, "blockheight", block.GetHeight(),
		"blockSize (bytes)", block.Size(), "costTime ms", costTime)

	client := d.GetQueueClient()
	newmsg := client.NewMessage("blockchain", types.EventSyncBlock, &types.BlockPid{Pid: remotePid, Block: block}) //加入到输出通道)
	client.SendTimeout(newmsg, false, 10*time.Second)
	d.releaseJob(task)

	return nil
}
