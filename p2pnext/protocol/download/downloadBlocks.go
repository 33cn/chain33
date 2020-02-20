package download

import (
	"errors"
	//"bufio"
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	//protobufCodec "github.com/multiformats/go-multicodec/protobuf"

	"github.com/33cn/chain33/common/log/log15"

	core "github.com/libp2p/go-libp2p-core"

	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	uuid "github.com/google/uuid"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	//net "github.com/libp2p/go-libp2p-core/network"
)

var (
	log = log15.New("module", "p2p.download")
)

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &DownloadProtol{})
	var downloadHandler = new(DownloadHander)
	prototypes.RegisterStreamHandlerType(protoTypeID, DownloadBlockReq, downloadHandler)
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
	//d.BaseProtocol = new(prototypes.BaseProtocol)
	d.GlobalData = data
	//注册事件处理函数
	prototypes.RegisterEventHandler(types.EventFetchBlocks, d.handleEvent)
}

type DownloadHander struct {
	*prototypes.BaseStreamHandler
}

func (h *DownloadHander) VerifyRequest(data []byte) bool {

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

	log.Info("OnReq", "blocksResp BlockHeight+++++++", blocksResp.Message.GetItems()[0].GetBlock().GetHeight())
	err = d.SendProtoMessage(blocksResp, s)
	if err != nil {
		log.Error("SendProtoMessage", "err", err)
		d.GetConnsManager().Delete(s.Conn().RemotePeer().Pretty())
		return
	}

	log.Info("%s:  send block response to %s sent.", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())

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
	log.Info("handleEvent", "download start", req.GetStart(), "download end", req.GetEnd(), "pids", pids)

	//具体的下载逻辑
	jobS := d.initJob()
	var wg sync.WaitGroup
	var maxgoroutin int32
	var startTime = time.Now().UnixNano()
	for height := req.GetStart(); height <= req.GetEnd(); height++ {
		wg.Add(1)
	Wait:
		if maxgoroutin > 30 {
			time.Sleep(time.Millisecond * 200)
			goto Wait
		}
		atomic.AddInt32(&maxgoroutin, 1)
		go func(blockheight int64, jbs jobs) {
			err := d.syncDownloadBlock(blockheight, jbs)
			if err != nil {
				log.Error("syncDownloadBlock", "err", err.Error())
				//TODO 50次下载尝试，失败之后异常处理
			}
			wg.Done()
			atomic.AddInt32(&maxgoroutin, -11)

		}(height, jobS)

	}
	wg.Wait()
	log.Info("handleEvent", "download process done", "cost time(ms)", (time.Now().UnixNano()-startTime)/1e6)
	//}

}

func (d *DownloadProtol) syncDownloadBlock(blockheight int64, jbs jobs) error {
	var retryCount uint
	var downloadStart = time.Now().UnixNano()
ReDownload:
	retryCount++
	if retryCount > 50 {
		return errors.New("beyound max try count 50")
	}
	freeJob := d.getFreeJobStream(jbs)
	if freeJob == nil {
		time.Sleep(time.Millisecond * 400)
		goto ReDownload
	}

	getblocks := &types.P2PGetBlocks{StartHeight: blockheight, EndHeight: blockheight,
		Version: 0}

	peerID := d.GetHost().ID()
	pubkey, _ := d.GetHost().Peerstore().PubKey(peerID).Bytes()
	blockReq := &types.MessageGetBlocksReq{MessageData: d.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
		Message: getblocks}

	stream, err := d.Host.NewStream(context.Background(), freeJob.Pid, DownloadBlockReq)
	if err != nil {
		log.Error("NewStream", "err", err, "remotePid", freeJob.Pid)
		goto ReDownload
	}
	defer stream.Close()
	if err := d.SendProtoMessage(blockReq, stream); err != nil {
		log.Error("handleEvent", "SendProtoMessageErr", err)
		goto ReDownload
	}
	log.Info("handleEvent", "sendOk", "beforRead")
	var resp types.MessageGetBlocksResp
	err = d.ReadProtoMessage(&resp, stream)
	if err != nil {
		log.Error("handleEvent", "ReadProtoMessage", err)
		goto ReDownload
	}

	block := resp.GetMessage().GetItems()[0].GetBlock()
	remotePid := freeJob.Pid.Pretty()
	costTime := (time.Now().UnixNano() - downloadStart) / 1e6
	log.Info("download+++++", "to", remotePid, "blockheight", block.GetHeight(),
		"blockSize (bytes)", block.Size(), "costTime ms", costTime)

	client := d.GetQueueClient()
	newmsg := client.NewMessage("blockchain", types.EventSyncBlock, &types.BlockPid{Pid: remotePid, Block: block}) //加入到输出通道)
	client.SendTimeout(newmsg, false, 10*time.Second)
	d.releaseJobStream(freeJob)
	return nil
}

type JobPeerId struct {
	Limit int
	Pid   peer.ID
	mtx   sync.Mutex
}

// jobs datastruct
type jobs []*JobPeerId

//Len size of the Invs data
func (i jobs) Len() int {
	return len(i)
}

//Less Sort from low to high
func (i jobs) Less(a, b int) bool {
	return i[a].Limit < i[b].Limit
}

//Swap  the param
func (i jobs) Swap(a, b int) {
	i[a], i[b] = i[b], i[a]
}

func (d *DownloadProtol) initJob() jobs {
	var JobPeerIds jobs
	conns := d.ConnManager.Fetch()
	for _, conn := range conns {
		var job JobPeerId

		// stream, err := d.Host.NewStream(context.Background(), conn.RemotePeer(), DownloadBlockReq)
		// if err != nil {
		// 	log.Error("NewStream", "err", err)
		// 	continue
		// }
		log.Info("initJob", "pid", conn.RemotePeer().Pretty())
		job.Pid = conn.RemotePeer()
		job.Limit = 0
		JobPeerIds = append(JobPeerIds, &job)

	}

	return JobPeerIds
}

func (d *DownloadProtol) getFreeJobStream(js jobs) *JobPeerId {
	var MaxJobLimit int = 100
	sort.Sort(js)
	for _, job := range js {
		if job.Limit < MaxJobLimit {
			job.mtx.Lock()
			job.Limit++
			job.mtx.Unlock()
			return job
		}
	}

	return nil

}

func (d *DownloadProtol) releaseJobStream(js *JobPeerId) {
	js.mtx.Lock()
	defer js.mtx.Unlock()
	js.Limit--
	if js.Limit < 0 {
		js.Limit = 0
	}
}
