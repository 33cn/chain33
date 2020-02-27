package download

import (
	"errors"
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
)

var (
	log             = log15.New("module", "p2p.download")
	MaxJobLimit int = 100
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

	err = d.SendProtoMessage(blocksResp, s)
	if err != nil {
		log.Error("SendProtoMessage", "err", err)
		return
	}

	log.Info("OnReq", "blocksResp BlockHeight+++++++", blocksResp.Message.GetItems()[0].GetBlock().GetHeight(), "send block response to", s.Conn().RemotePeer().String())

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
		if maxgoroutin > 50 {
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

}

func (d *DownloadProtol) syncDownloadBlock(blockheight int64, jbs jobs) error {
	var retryCount uint
	var downloadStart = time.Now().UnixNano()
ReDownload:
	retryCount++
	if retryCount > 50 {
		return errors.New("beyound max try count 50")
	}
	freeJob := d.getFreeJob(jbs)
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

	stream, err := d.SendToStream(freeJob.Pid.Pretty(), blockReq, DownloadBlockReq, d.GetHost())
	if err != nil {
		log.Error("NewStream", "err", err, "remotePid", freeJob.Pid)
		d.GetConnsManager().Delete(freeJob.Pid.Pretty())
		d.releaseJob(freeJob)
		goto ReDownload
	}
	defer stream.Close()

	var resp types.MessageGetBlocksResp
	err = d.ReadProtoMessage(&resp, stream)
	if err != nil {
		log.Error("handleEvent", "ReadProtoMessage", err)
		d.releaseJob(freeJob)
		goto ReDownload
	}

	block := resp.GetMessage().GetItems()[0].GetBlock()
	remotePid := freeJob.Pid.Pretty()
	costTime := (time.Now().UnixNano() - downloadStart) / 1e6

	//rate := float64(block.Size()) / float64(costTime)

	log.Info("download+++++", "from", remotePid, "blockheight", block.GetHeight(),
		"blockSize (bytes)", block.Size(), "costTime ms", costTime)

	client := d.GetQueueClient()
	newmsg := client.NewMessage("blockchain", types.EventSyncBlock, &types.BlockPid{Pid: remotePid, Block: block}) //加入到输出通道)
	client.SendTimeout(newmsg, false, 10*time.Second)
	d.releaseJob(freeJob)
	return nil
}

type JobPeerId struct {
	Limit   int
	Pid     peer.ID
	Latency time.Duration
	mtx     sync.Mutex
}

// jobs datastruct
type jobs []*JobPeerId

//Len size of the Invs data
func (i jobs) Len() int {
	return len(i)
}

//Less Sort from low to high
func (i jobs) Less(a, b int) bool {
	return i[a].Latency < i[b].Latency
}

//Swap  the param
func (i jobs) Swap(a, b int) {
	i[a], i[b] = i[b], i[a]
}

func (d *DownloadProtol) initJob() jobs {
	var JobPeerIds jobs
	log.Info("initJob", "peersize", d.GetHost().Peerstore().Peers().Len())
	pids := d.ConnManager.Fetch()
	for _, pid := range pids {
		if pid == d.GetHost().ID().Pretty() {
			continue
		}
		var job JobPeerId
		log.Info("initJob", "pid", pid)
		rID, err := peer.IDB58Decode(pid)
		if err != nil {
			continue
		}
		job.Pid = rID
		job.Limit = 0
		JobPeerIds = append(JobPeerIds, &job)
	}

	return JobPeerIds
}

func (d *DownloadProtol) getFreeJob(js jobs) *JobPeerId {
	//配置各个节点的速率
	var peerIDs []peer.ID
	for _, job := range js {
		peerIDs = append(peerIDs, job.Pid)
	}
	latency := d.GetConnsManager().GetLatencyByPeer(peerIDs)
	for _, jb := range js {
		jb.Latency = latency[jb.Pid.Pretty()]
	}
	sort.Sort(js)
	log.Info("show sort result", "sort of jobs", js)
	for _, job := range js {
		if job.Limit < MaxJobLimit {
			job.mtx.Lock()
			job.Limit++
			job.mtx.Unlock()
			log.Info("getFreeJob", " limit", job.Limit, "latency", job.Latency, "peerid", job.Pid)
			return job
		}
	}

	return nil

}

func (d *DownloadProtol) releaseJob(js *JobPeerId) {
	js.mtx.Lock()
	defer js.mtx.Unlock()
	js.Limit--
	if js.Limit < 0 {
		js.Limit = 0
	}
}
