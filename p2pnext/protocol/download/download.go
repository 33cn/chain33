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
	prototypes.RegisterProtocolType(protoTypeID, &DownloadProtol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, DownloadBlockReq, &DownloadHander{})
}

const (
	protoTypeID      = "DownloadProtocolType"
	DownloadBlockReq = "/chain33/downloadBlockReq/1.0.0"
)

//type Istream
type DownloadProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler
}

func (d *DownloadProtol) InitProtocol(data *prototypes.GlobalData) {
	d.GlobalData = data
	//注册事件处理函数
	prototypes.RegisterEventHandler(types.EventFetchBlocks, d.handleEvent)

}

type DownloadHander struct {
	*prototypes.BaseStreamHandler
}

func (h *DownloadHander) VerifyRequest(data []byte, messageComm *types.MessageComm) bool {

	return true
}

//Handle 处理请求
func (d *DownloadHander) Handle(stream core.Stream) {
	protocol := d.GetProtocol().(*DownloadProtol)

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
		return
	}

	return

}

func (d *DownloadProtol) OnReq(id string, message *types.P2PGetBlocks, s core.Stream) {
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
		log.Error("GetBlocks Err", "Err", err.Error())
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
func (d *DownloadProtol) handleEvent(msg *queue.Message) {

	req := msg.GetData().(*types.ReqBlocks)
	pids := req.GetPid()
	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		log.Info("GetBlocks:pid is nil")
		msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	var jobID = uuid.New().String() + "+" + fmt.Sprintf("%d-%d", req.GetStart(), req.GetEnd())

	log.Info("handleEvent", "jobID", jobID, "download start", req.GetStart(), "download end", req.GetEnd(), "pids", pids)

	//具体的下载逻辑
	jobS := d.initJob(pids, jobID)
	var wg sync.WaitGroup
	var maxgoroutin int32
	var reDownload sync.Map
	var startTime = time.Now().UnixNano()

	for height := req.GetStart(); height <= req.GetEnd(); height++ {
		wg.Add(1)
	Wait:
		if maxgoroutin > 50 {
			time.Sleep(time.Millisecond * 200)
			goto Wait
		}
		atomic.AddInt32(&maxgoroutin, 1)
		go func(blockheight int64, jbs jobs) {
			err := d.syncDownloadBlock(blockheight, jbs)
			if err != nil {
				log.Error("syncDownloadBlock", "err", err.Error())
				v, ok := reDownload.Load(jobID)
				if ok {
					faildJob := v.(map[int64]bool)
					faildJob[blockheight] = false
					reDownload.Store(jobID, faildJob)

				} else {
					faildJob := make(map[int64]bool)
					faildJob[blockheight] = false
					reDownload.Store(jobID, faildJob)

				}
			}
			wg.Done()
			atomic.AddInt32(&maxgoroutin, -1)

		}(height, jobS)

	}

	wg.Wait()
	d.CheckJob(jobID, pids, reDownload)
	log.Info("handleEvent", "jobID", jobID, "download process done", fmt.Sprintf("cost time:%d ms", (time.Now().UnixNano()-startTime)/1e6))

}

func (d *DownloadProtol) syncDownloadBlock(blockheight int64, jbs jobs) error {
	var retryCount uint
	jbs.Sort() //对任务节点时延进行排序，优先选择时延低的节点进行下载
ReDownload:
	retryCount++
	if retryCount > 50 {
		return errors.New("beyound max try count 50")
	}

	freeJob := d.availbJob(jbs)
	if freeJob == nil {
		time.Sleep(time.Millisecond * 800)
		goto ReDownload
	}
	defer d.releaseJob(freeJob)

	var downloadStart = time.Now().UnixNano()

	getblocks := &types.P2PGetBlocks{StartHeight: blockheight, EndHeight: blockheight,
		Version: 0}

	peerID := d.GetHost().ID()
	pubkey, _ := d.GetHost().Peerstore().PubKey(peerID).Bytes()
	blockReq := &types.MessageGetBlocksReq{MessageData: d.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
		Message: getblocks}

	req := &prototypes.StreamRequst{
		PeerID:  freeJob.Pid,
		Host:    d.GetHost(),
		Data:    blockReq,
		ProtoID: DownloadBlockReq,
	}
	var resp types.MessageGetBlocksResp
	err := d.StreamSendHandler(req, &resp)
	if err != nil {
		log.Error("handleEvent", "StreamSendHandler", err)
		jbs.Remove(freeJob.Index)
		goto ReDownload
	}

	block := resp.GetMessage().GetItems()[0].GetBlock()
	remotePid := freeJob.Pid.Pretty()
	costTime := (time.Now().UnixNano() - downloadStart) / 1e6

	log.Debug("download+++++", "from", remotePid, "blockheight", block.GetHeight(),
		"blockSize (bytes)", block.Size(), "costTime ms", costTime)

	client := d.GetQueueClient()
	newmsg := client.NewMessage("blockchain", types.EventSyncBlock, &types.BlockPid{Pid: remotePid, Block: block}) //加入到输出通道)
	client.SendTimeout(newmsg, false, 10*time.Second)
	return nil
}
