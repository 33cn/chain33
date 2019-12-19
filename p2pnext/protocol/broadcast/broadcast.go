package broadcast

import (
	"errors"
	"time"

	//uuid "github.com/google/uuid"

	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	core "github.com/libp2p/go-libp2p-core"

	"github.com/33cn/chain33/common/log/log15"
	common "github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	net "github.com/libp2p/go-libp2p-core/network"
)

var log = log15.New("module", "p2p.broadcast")

const (
	protoTypeID = "BroadcastProtocolType"
	ID          = "/chain33/p2p/broadcast/1.0.0"
)

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &broadCastProtocol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, ID, &broadCastHandler{})
}

//
type broadCastProtocol struct {
	*prototypes.BaseProtocol
	txFilter        *common.Filterdata
	blockFilter     *common.Filterdata
	txSendFilter    *common.Filterdata
	blockSendFilter *common.Filterdata
	totalBlockCache *common.SpaceLimitCache
	ltBlockCache    *common.SpaceLimitCache
	p2pCfg          *types.P2P
}

//New
func (p *broadCastProtocol) InitProtocol(data *prototypes.GlobalData) {
	p.BaseProtocol = new(prototypes.BaseProtocol)

	p.GlobalData = data
	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	p.txFilter = common.NewFilter(TxRecvFilterCacheNum)
	p.blockFilter = common.NewFilter(BlockFilterCacheNum)

	//发送交易和区块时过滤缓存, 解决冗余广播发送
	p.txSendFilter = common.NewFilter(TxSendFilterCacheNum)
	p.blockSendFilter = common.NewFilter(BlockFilterCacheNum)

	//在本地暂时缓存一些区块数据, 限制最大大小
	p.totalBlockCache = common.NewSpaceLimitCache(BlockCacheNum, MaxBlockCacheByteSize)
	//接收到短哈希区块数据,只构建出区块部分交易,需要缓存, 并继续向对端节点请求剩余数据
	p.ltBlockCache = common.NewSpaceLimitCache(BlockCacheNum/2, MaxBlockCacheByteSize/2)
	p.p2pCfg = p.GetChainCfg().GetModuleConfig().P2P
	//注册事件处理函数
	prototypes.RegisterEventHandler(types.EventTxBroadcast, p.handleEvent)
	prototypes.RegisterEventHandler(types.EventBlockBroadcast, p.handleEvent)
}

type broadCastHandler struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (h *broadCastHandler) Handle(req []byte, stream core.Stream) {

	protocol := h.GetProtocol().(*broadCastProtocol)
	pid := stream.Conn().RemotePeer().Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	//s.node.Store(pid, stream)

	//解析处理
	var data types.MessageBroadCast
	err := types.Decode(req, &data)
	if err != nil {
		return
	}

	recvData := data.Message

	_ = protocol.handleReceive(recvData, pid, peerAddr)
	return
}
func (b *broadCastHandler) SetProtocol(protocol prototypes.IProtocol) {
	b.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	b.Protocol = protocol
}

func (h *broadCastHandler) VerifyRequest(data []byte) bool {

	return true
}

//
func (s *broadCastProtocol) handleEvent(msg *queue.Message) {

	data := msg.GetData()
	streams := s.GetStreamManager().FetchStreams()
	for _, stream := range streams {
		s.sendStream(stream, data)
	}

}

func (s *broadCastProtocol) queryStream(pid string, data interface{}) error {

	stream := s.GetStreamManager().GetStream(pid)
	if stream != nil {
		return s.sendStream(stream, data)
	}

	return nil

}

func (s *broadCastProtocol) sendStream(stream net.Stream, data interface{}) error {

	pid := stream.Conn().RemotePeer().Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	sendData, doSend := s.handleSend(data, pid, peerAddr)
	if !doSend {
		log.Debug("sendStream", "doSend", doSend)
		return nil
	}
	//TODO,send stream
	//包装一层MessageBroadCast
	broadData := &types.MessageBroadCast{
		Message: sendData}

	// TODO sign the data
	// signature, err := s.node.SignProtoMessage(broadData)
	// if err != nil {
	// 	logger.Error("failed to sign pb data")
	// 	return err
	// }

	// broadData.Common.Sign = signature

	//s.GetStreamManager().GetStream()
	ok := s.GetStreamManager().SendProtoMessage(broadData, stream)
	if !ok {
		stream.Close()
		s.GetStreamManager().DeleteStream(pid)
		return errors.New("SendStreamErr")
	}

	return nil

}

// handleSend 对数据进行处理，包装成BroadCast结构
func (s *broadCastProtocol) handleSend(rawData interface{}, pid, peerAddr string) (sendData *types.BroadCastData, doSend bool) {
	//出错处理
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleSend_Panic", "sendData", rawData, "peerAddr", peerAddr, "recoverErr", r)
			doSend = false
		}
	}()
	log.Debug("ProcessSendP2PBegin", "peerID", pid, "peerAddr", peerAddr)
	sendData = &types.BroadCastData{}

	doSend = false
	if tx, ok := rawData.(*types.P2PTx); ok {
		doSend = s.sendTx(tx, sendData, pid, peerAddr)
	} else if blc, ok := rawData.(*types.P2PBlock); ok {
		doSend = s.sendBlock(blc, sendData, pid, peerAddr)
	} else if query, ok := rawData.(*types.P2PQueryData); ok {
		doSend = s.sendQueryData(query, sendData, peerAddr)
	} else if rep, ok := rawData.(*types.P2PBlockTxReply); ok {
		doSend = s.sendQueryReply(rep, sendData, peerAddr)
	} else if ping, ok := rawData.(*types.P2PPing); ok {
		doSend = true
		sendData.Value = &types.BroadCastData_Ping{Ping: ping}
	}
	log.Debug("handleSend", "peerAddr", peerAddr, "doSend", doSend)
	return
}

func (s *broadCastProtocol) handleReceive(data *types.BroadCastData, pid string, peerAddr string) (handled bool) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleReceive_Panic", "recvData", data, "peerAddr", peerAddr, "recoverErr", r)
		}
	}()
	log.Debug("handleReceive", "peerID", pid, "peerAddr", peerAddr)
	if pid == "" {
		return false
	}
	handled = true
	if tx := data.GetTx(); tx != nil {
		s.recvTx(tx, pid, peerAddr)
	} else if ltTx := data.GetLtTx(); ltTx != nil {
		s.recvLtTx(ltTx, pid, peerAddr)
	} else if ltBlc := data.GetLtBlock(); ltBlc != nil {
		s.recvLtBlock(ltBlc, pid, peerAddr)
	} else if blc := data.GetBlock(); blc != nil {
		s.recvBlock(blc, pid, peerAddr)
	} else if query := data.GetQuery(); query != nil {
		s.recvQueryData(query, pid, peerAddr)
	} else if rep := data.GetBlockRep(); rep != nil {
		s.recvQueryReply(rep, pid, peerAddr)
	} else {
		handled = false
	}
	log.Debug("handleReceive", "peerAddr", peerAddr, "handled", handled)
	return
}

func (s *broadCastProtocol) sendToMempool(ty int64, data interface{}) (interface{}, error) {

	client := s.GetQueueClient()
	msg := client.NewMessage("mempool", ty, data)
	err := client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *broadCastProtocol) postBlockChain(block *types.Block, pid string) error {

	client := s.GetQueueClient()
	msg := client.NewMessage("blockchain", types.EventBroadcastAddBlock, &types.BlockPid{Pid: pid, Block: block})
	err := client.Send(msg, false)
	if err != nil {
		log.Error("postBlockChain", "send to blockchain Error", err.Error())
		return err
	}
	return nil
}

//同时收到多个节点相同交易, 需要加锁保证原子操作, 检测是否存在以及添加过滤
func checkAndRegFilterAtomic(filter *common.Filterdata, key string) (exist bool) {

	filter.GetLock()
	defer filter.ReleaseLock()
	if filter.QueryRecvData(key) {
		return true
	}
	filter.RegRecvData(key)
	return false
}

type sendFilterInfo struct {
	//记录广播交易或区块时需要忽略的节点, 这些节点可能是交易的来源节点,也可能节点间维护了多条连接, 冗余发送
	ignoreSendPeers map[string]bool
}

//检测是否冗余发送, 或者添加到发送过滤(内部存在直接修改读写保护的数据, 对filter lru的读写需要外层锁保护)
func addIgnoreSendPeerAtomic(filter *common.Filterdata, key, pid string) (exist bool) {

	filter.GetLock()
	defer filter.ReleaseLock()
	var info *sendFilterInfo
	if !filter.QueryRecvData(key) { //之前没有收到过这个key
		info = &sendFilterInfo{ignoreSendPeers: make(map[string]bool)}
		filter.Add(key, info)
	} else {
		info = filter.Get(key).(*sendFilterInfo)
	}
	_, exist = info.ignoreSendPeers[pid]
	info.ignoreSendPeers[pid] = true
	return exist
}
