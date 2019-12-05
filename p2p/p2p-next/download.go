package p2pnext

import (
	"io"
	"io/ioutil"
	"sort"
	"sync"
	"time"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	net "github.com/libp2p/go-libp2p-net"
)

const (
	downloadBlockReq  = "/chain33/downloadBlockReq/1.0.0"
	downloadBlockResp = "/chain33/downloadBlockResp/1.0.0"
)

//type Istream
type DownloadBlockProtol struct {
	client   queue.Client
	done     chan struct{}
	node     *Node                                 // local host
	requests map[string]*types.MessageGetBlocksReq // used to access request data from response handlers
}

func NewDownloadBlockProtol(node *Node, cli queue.Client, done chan struct{}) *DownloadBlockProtol {

	Server := &DownloadBlockProtol{}
	node.host.SetStreamHandler(downloadBlockReq, Server.onReq)
	node.host.SetStreamHandler(downloadBlockResp, Server.onResp)
	Server.requests = make(map[string]*types.MessageGetBlocksReq)
	Server.node = node
	Server.client = cli
	Server.done = done
	return Server
}

//接收Response消息
func (d *DownloadBlockProtol) onResp(s net.Stream) {
	for {

		data := &types.MessageGetBlocksResp{}
		buf, err := ioutil.ReadAll(s)
		if err != nil {
			s.Reset()
			logger.Error(err)
			continue
		}

		// unmarshal it
		proto.Unmarshal(buf, data)
		if err != nil {
			logger.Error(err)
			continue
		}

		valid := d.node.authenticateMessage(data, data.MessageData)

		if !valid {
			logger.Error("Failed to authenticate message")
			continue
		}

		// locate request data and remove it if found
		_, ok := d.requests[data.MessageData.Id]
		if ok {
			// remove request from map as we have processed it here
			delete(d.requests, data.MessageData.Id)
			//TODO release Job
		} else {
			logger.Error("Failed to locate request data boject for response")
			continue
		}
		logger.Debug("%s: Received ping response from %s. Message id:%s. Message: %s.", s.Conn().LocalPeer().String(), s.Conn().RemotePeer(), data.MessageData.Id, data.Message)

		for _, invdata := range data.Message.GetItems() {
			if invdata.Ty != 2 {
				continue
			}

			block := invdata.GetBlock()
			newmsg := d.client.NewMessage("blockchain", types.EventSyncBlock,
				&types.BlockPid{Pid: s.Conn().LocalPeer().String(), Block: block})
			d.client.SendTimeout(newmsg, false, 20*time.Second)

		}

	}
}

func (d *DownloadBlockProtol) onReq(s net.Stream) {
	for {

		var buf []byte

		_, err := io.ReadFull(s, buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			s.Close()
			return

		}

		//解析处理
		var data types.MessageGetBlocksReq
		err = types.Decode(buf, &data)
		if err != nil {
			continue
		}

		valid := d.node.authenticateMessage(&data, data.MessageData)
		if !valid {
			logger.Error("Failed to authenticate message")
			continue
		}

		//获取headers 信息
		data.GetMessage().GetStartHeight()
		message := data.GetMessage()
		//允许下载的最大高度区间为256
		if message.GetEndHeight()-message.GetStartHeight() > 256 || message.GetEndHeight() < message.GetStartHeight() {
			continue

		}
		//开始下载指定高度
		reqblock := &types.ReqBlocks{Start: message.GetStartHeight(), End: message.GetEndHeight()}
		msg := d.client.NewMessage("blockchain", types.EventGetBlocks, reqblock)
		err = d.client.Send(msg, true)
		if err != nil {
			logger.Error("GetBlocks", "Error", err.Error())
			continue
		}
		resp, err := d.client.WaitTimeout(msg, time.Second*20)
		if err != nil {
			logger.Error("GetBlocks Err", "Err", err.Error())
			continue
		}

		blockDetails := resp.Data.(*types.BlockDetails)
		var p2pInvData = make([]*types.InvData, 0)
		var invdata types.InvData
		for _, item := range blockDetails.Items {
			invdata.Ty = 2 //2 block,1 tx
			invdata.Value = &types.InvData_Block{Block: item.Block}
			p2pInvData = append(p2pInvData, &invdata)
		}

		blocksResp := &types.MessageGetBlocksResp{MessageData: d.node.NewMessageData(data.MessageData.Id, false),
			Message: &types.InvDatas{p2pInvData}}

		// sign the data
		signature, err := d.node.signProtoMessage(blocksResp)
		if err != nil {
			logger.Error("failed to sign response")
			continue
		}

		// add the signature to the message
		blocksResp.MessageData.Sign = signature
		ok := d.node.sendProtoMessage(s, headerInfoResp, blocksResp)

		if ok {
			logger.Info("%s: Ping response to %s sent.", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())
		}

	}
}

//GetBlocks 接收来自chain33 blockchain模块发来的请求
func (d *DownloadBlockProtol) GetBlocks(msg *queue.Message) {

	req := msg.GetData().(*types.ReqBlocks)
	pids := req.GetPid()
	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		logger.Debug("GetBlocks:pid is nil")
		msg.Reply(d.client.NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(d.client.NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	peersMap := d.node.peersInfo

	for _, pid := range pids {
		data, ok := peersMap.Load(pid)
		if !ok {
			continue
		}
		peerinfo := data.(*types.P2PPeerInfo)
		//去指定的peer上获取对应的blockHeader
		peerId := peerinfo.GetName()

		pstream := d.node.streamMange.GetStream(peerId)
		if pstream == nil {
			continue
		}
		//TODO具体的下载逻辑
		jobS := d.initStreamJob()

		for height := req.GetStart(); height <= req.GetEnd(); height++ {
			jStream := d.getFreeJobStream(jobS)
			getblocks := &types.P2PGetBlocks{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
				Version: 0}
			blockReq := &types.MessageGetBlocksReq{MessageData: d.node.NewMessageData(uuid.New().String(), false),
				Message: getblocks}

			// sign the data
			signature, err := d.node.signProtoMessage(blockReq)
			if err != nil {
				logger.Error("failed to sign pb data")
				return
			}

			blockReq.MessageData.Sign = signature
			if d.node.sendProtoMessage(jStream.Stream, downloadBlockReq, blockReq) {
				d.requests[blockReq.MessageData.Id] = blockReq
			}

		}
	}

}

type JobStream struct {
	Limit  int
	Stream net.Stream
	mtx    sync.Mutex
}

// jobs datastruct
type jobs []*JobStream

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

func (d *DownloadBlockProtol) initStreamJob() jobs {
	var jobStreams jobs
	streams := d.node.streamMange.fetchStreams()
	for _, stream := range streams {
		var jstream JobStream
		jstream.Stream = stream
		jstream.Limit = 0
		jobStreams = append(jobStreams, &jstream)

	}

	return jobStreams
}

func (d *DownloadBlockProtol) getFreeJobStream(js jobs) *JobStream {
	var MaxJobLimit int = 10
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

func (d *DownloadBlockProtol) releaseJobStream(js *JobStream) {
	js.mtx.Lock()
	defer js.mtx.Unlock()
	js.Limit--
	if js.Limit < 0 {
		js.Limit = 0
	}
}
