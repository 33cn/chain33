package peer

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/common/utils"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
)

const (
	// Deprecated: old version, use peerInfo instead
	peerInfoOld = "/chain33/peerinfoReq/1.0.0" //老版本
	peerInfo    = "/chain33/peer-info/1.0.0"
	// Deprecated: old version, use peerVersion instead
	peerVersionOld = "/chain33/peerVersion/1.0.0"
	peerVersion    = "/chain33/peer-version/1.0.0"
)

const (
	blockchain = "blockchain"
	mempool    = "mempool"
)

// UnitTime key time value:time.Second
var UnitTime = map[string]int64{
	"hour":   3600,
	"min":    60,
	"second": 1,
}
var log = log15.New("module", "p2p.peer")
var processStart = time.Now()

// CaculateLifeTime parase time string to time.Duration
func CaculateLifeTime(timestr string) (time.Duration, error) {
	var lifetime int64
	if timestr == "" {
		return 0, nil
	}
	for substr, time := range UnitTime {
		if strings.HasSuffix(timestr, substr) {

			num := strings.TrimRight(timestr, substr)
			num = strings.TrimSpace(num)
			f, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0, err
			}
			lifetime = int64(f * float64(time))
			break
		}
	}

	return time.Duration(lifetime) * time.Second, nil
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

// Protocol ...
type Protocol struct {
	*protocol.P2PEnv

	// "/ip4/{ip}/tcp/{port}"
	externalAddr string
	mutex        sync.Mutex

	topicMutex  sync.RWMutex
	topicModule sync.Map
	latestBlock sync.Map
	blocked     int32
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	p := Protocol{
		P2PEnv: env,
	}
	//先进行外部地址预设置
	for _, multiAddr := range p.Host.Addrs() {
		addr := multiAddr.String()
		// 如果找到公网ip直接break，否则预设置一个内网ip
		if utils.IsPublicIP(strings.Split(addr, "/")[2]) {
			p.mutex.Lock()
			p.externalAddr = addr
			p.mutex.Unlock()
			break
		}
		if !strings.Contains(addr, "127.0.0.1") && !strings.Contains(addr, "localhost") {
			p.mutex.Lock()
			p.externalAddr = addr
			p.mutex.Unlock()
		}
	}

	// Deprecated: old version, use peerInfo instead
	protocol.RegisterStreamHandler(p.Host, peerInfoOld, p.handleStreamPeerInfoOld)
	protocol.RegisterStreamHandler(p.Host, peerInfo, p.handleStreamPeerInfo)
	// Deprecated: old version, use peerVersion instead
	protocol.RegisterStreamHandler(p.Host, peerVersionOld, p.handleStreamVersionOld)
	protocol.RegisterStreamHandler(p.Host, peerVersion, p.handleStreamVersion)
	//统计信息
	protocol.RegisterStreamHandler(p.Host, statisticalInfo, p.handlerStreamStatistical)
	protocol.RegisterEventHandler(types.EventPeerInfo, p.handleEventPeerInfo)
	protocol.RegisterEventHandler(types.EventGetNetInfo, p.handleEventNetInfo)
	protocol.RegisterEventHandler(types.EventNetProtocols, p.handleEventNetProtocols)

	//绑定订阅事件与相关处理函数
	protocol.RegisterEventHandler(types.EventSubTopic, p.handleEventSubTopic)
	//获取订阅topic列表
	protocol.RegisterEventHandler(types.EventFetchTopics, p.handleEventGetTopics)
	//移除订阅主题
	protocol.RegisterEventHandler(types.EventRemoveTopic, p.handleEventRemoveTopic)
	//发布消息
	protocol.RegisterEventHandler(types.EventPubTopicMsg, p.handleEventPubMsg)
	//添加节点到黑名单
	protocol.RegisterEventHandler(types.EventAddBlacklist, p.handleEventAddBlacklist)
	//删除黑名单的某个节点
	protocol.RegisterEventHandler(types.EventDelBlacklist, p.handleEventDelBlacklist)
	//获取当前的黑名单节点列表
	protocol.RegisterEventHandler(types.EventShowBlacklist, p.handleEventShowBlacklist)
	//连接指定的节点
	protocol.RegisterEventHandler(types.EventDialPeer, p.handleEventDialPeer)
	//关闭指定的节点
	protocol.RegisterEventHandler(types.EventClosePeer, p.handleEventClosePeer)
	go p.detectNodeAddr()
	go p.checkBlocked()
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		defer ticker.Stop()
		ticker2 := time.NewTicker(time.Minute * 5)
		defer ticker2.Stop()
		ticker3 := time.NewTicker(time.Minute * 10)
		defer ticker3.Stop()

		for {
			select {
			case <-p.Ctx.Done():
				return
			case <-ticker.C:
				p.refreshSelf()
			case <-ticker2.C:
				peers := p.RoutingTable.ListPeers()
				if len(peers) <= maxPeers {
					break
				}
				p.refreshPeerInfo(peers[maxPeers:])

			case <-ticker3.C:
				p.checkOutBound(p.PeerInfoManager.Fetch(p.Host.ID()).GetHeader().GetHeight())
			}
		}
	}()
	go func() {
		ticker1 := time.NewTicker(time.Second * 10)
		if p.ChainCfg.IsTestNet() {
			ticker1 = time.NewTicker(time.Second)
		}
		defer ticker1.Stop()

		for {
			select {
			case <-p.Ctx.Done():
				return
			case <-ticker1.C:
				peers := p.RoutingTable.ListPeers()
				if len(peers) > maxPeers {
					peers = peers[:maxPeers]
				}
				p.refreshPeerInfo(peers)
			}
		}
	}()
}

func (p *Protocol) checkDone() bool {
	select {
	case <-p.Ctx.Done():
		return true
	default:
		return false
	}
}
