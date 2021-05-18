package peer

import (
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
)

const (
	// Deprecated: old version, use peerInfo instead
	peerInfoOld = "/chain33/peerinfoReq/1.0.0"
	peerInfo    = "/chain33/peer-info/1.0.0" //老版本
	// Deprecated: old version, use peerVersion instead
	peerVersionOld = "/chain33/peerVersion/1.0.0"
	peerVersion    = "/chain33/peer-version/1.0.0"
)

const (
	blockchain = "blockchain"
	mempool    = "mempool"
)

var log = log15.New("module", "p2p.peer")

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

// Protocol ...
type Protocol struct {
	*protocol.P2PEnv
	refreshing int32

	// "/ip4/{ip}/tcp/{port}"
	externalAddr string
	mutex        sync.Mutex

	topicMutex  sync.RWMutex
	topicModule sync.Map
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
		if isPublicIP(strings.Split(addr, "/")[2]) {
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

	go p.detectNodeAddr()
	go func() {
		ticker := time.NewTicker(time.Second / 2)
		ticker2 := time.NewTicker(time.Minute * 5)
		for {
			select {
			case <-p.Ctx.Done():
				return
			case <-ticker.C:
				p.refreshSelf()
			case <-ticker2.C:
				p.checkOutBound(p.PeerInfoManager.Fetch(p.Host.ID()).GetHeader().GetHeight())
			}
		}
	}()
	go func() {
		ticker1 := time.NewTicker(time.Second * 5)
		if p.ChainCfg.IsTestNet() {
			ticker1 = time.NewTicker(time.Second)
		}
		for {
			select {
			case <-p.Ctx.Done():
				return
			case <-ticker1.C:
				p.refreshPeerInfo()
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
