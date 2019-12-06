package broadcast

import (
	"io"
	"sync"
	"time"

	core "github.com/libp2p/go-libp2p-core"

	logger "github.com/33cn/chain33/common/log/log15"
	common "github.com/33cn/chain33/p2p"
	p2p "github.com/33cn/chain33/p2p/p2p-next"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	host "github.com/libp2p/go-libp2p-core/host"
)

var log = logger.New("module", "p2p.broadcast")

const ID = "/chain33/p2p/broadcast/1.0.0"

//
type Service struct {
	streams         sync.Map
	txFilter        *common.Filterdata
	blockFilter     *common.Filterdata
	txSendFilter    *common.Filterdata
	blockSendFilter *common.Filterdata
	totalBlockCache *common.SpaceLimitCache
	ltBlockCache    *common.SpaceLimitCache

	node   *p2p.Node
	client queue.Client
}

//  NewService
func New(h host.Host) *Service {

	handler := &Service{}
	h.SetStreamHandler(ID, handler.recvStream)

	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	handler.txFilter = common.NewFilter(TxRecvFilterCacheNum)
	handler.blockFilter = common.NewFilter(BlockFilterCacheNum)

	//发送交易和区块时过滤缓存, 解决冗余广播发送
	handler.txSendFilter = common.NewFilter(TxSendFilterCacheNum)
	handler.blockSendFilter = common.NewFilter(BlockFilterCacheNum)

	//在本地暂时缓存一些区块数据, 限制最大大小
	handler.totalBlockCache = common.NewSpaceLimitCache(BlockCacheNum, MaxBlockCacheByteSize)
	//接收到短哈希区块数据,只构建出区块部分交易,需要缓存, 并继续向对端节点请求剩余数据
	handler.ltBlockCache = common.NewSpaceLimitCache(BlockCacheNum/2, MaxBlockCacheByteSize/2)

	return handler
}

//
func (s *Service) sendAllStream(msg *queue.Message) {

	data := msg.GetData()
	s.streams.Range(func(k, v interface{}) bool {
		stream := v.(core.Stream)
		_, err := s.sendStream(k.(core.Stream), data)
		if err != nil {
			stream.Close()
			s.streams.Delete(k)
		}
		return true
	})
}

func (s *Service) queryStream(pid string, data interface{}) {

	if v, ok := s.streams.Load(pid); ok {
		stream := v.(core.Stream)
		_, err := s.sendStream(stream, data)
		if err != nil {
			s.streams.Delete(stream)
		}
	}
}

func (s *Service) sendStream(stream core.Stream, data interface{}) (int, error) {

	pid := stream.Conn().RemotePeer().Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	sendData, _ := s.handleSend(data, pid, peerAddr)
	return stream.Write(types.Encode(sendData))
}

// Service
func (s *Service) recvStream(stream core.Stream) {

	pid := stream.Conn().RemotePeer().Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	s.streams.Store(pid, stream)
	var buf []byte
	for {
		_, err := io.ReadFull(stream, buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			stream.Close()
			s.streams.Delete(stream)
			return

		}
		//解析处理
		recvData := &types.BroadCastData{}
		err = types.Decode(buf, recvData)
		if err != nil {
			continue
		}

		_ = s.handleReceive(recvData, pid, peerAddr)

	}

}

func (s *Service) handleSend(rawData interface{}, pid, peerAddr string) (sendData *types.BroadCastData, doSend bool) {
	//出错处理
	defer func() {
		if r := recover(); r != nil {
			log.Error("processSendP2P_Panic", "sendData", rawData, "peerAddr", peerAddr, "recoverErr", r)
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
	log.Debug("ProcessSendP2PEnd", "peerAddr", peerAddr, "doSend", doSend)
	return
}

func (s *Service) handleReceive(data *types.BroadCastData, pid string, peerAddr string) (handled bool) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("ProcessRecvP2P_Panic", "recvData", data, "peerAddr", peerAddr, "recoverErr", r)
		}
	}()
	log.Debug("ProcessRecvP2P", "peerID", pid, "peerAddr", peerAddr)
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
	log.Debug("ProcessRecvP2P", "peerAddr", peerAddr, "handled", handled)
	return
}

func (s *Service) queryMempool(ty int64, data interface{}) (interface{}, error) {

	msg := s.client.NewMessage("mempool", ty, data)
	err := s.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *Service) postBlockChain(block *types.Block, pid string) error {

	msg := s.client.NewMessage("blockchain", types.EventBroadcastAddBlock, &types.BlockPid{Pid: pid, Block: block})
	err := s.client.Send(msg, false)
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
	if !filter.QueryRecvData(key) {
		info = &sendFilterInfo{ignoreSendPeers: make(map[string]bool)}
		filter.Add(key, info)
	} else {
		info = filter.Get(key).(*sendFilterInfo)
	}
	_, exist = info.ignoreSendPeers[pid]
	info.ignoreSendPeers[pid] = true
	return exist
}
