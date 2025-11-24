package extension

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/33cn/chain33/types"

	"github.com/33cn/chain33/common/log/log15"
	p2ptypes "github.com/33cn/chain33/system/p2p/dht/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var log = log15.New("module", "extension")

var setOnce sync.Once

// TopicMap topic map
type TopicMap map[string]*topicinfo

type topicinfo struct {
	pubtopic *pubsub.Topic
	sub      *pubsub.Subscription
	ctx      context.Context
	cancel   context.CancelFunc
	topic    string
}

// PubSub pub sub
type PubSub struct {
	*pubsub.PubSub
	host       host.Host
	topics     TopicMap
	topicMutex sync.RWMutex
	ctx        context.Context
	config     *p2ptypes.PubSubConfig
	pt         *pubsubTracer
}

// SubMsg sub message
type SubMsg *pubsub.Message

// SubCallBack 订阅消息回调函数
type SubCallBack func(topic string, msg SubMsg)

func setPubSubParameters(psConf *p2ptypes.PubSubConfig) {
	// mb -> byte
	psConf.MaxMsgSize *= 1 << 20
	if psConf.MaxMsgSize <= 0 {
		psConf.MaxMsgSize = types.MaxBlockSize
	}
	if psConf.PeerOutboundQueueSize <= 0 {
		psConf.PeerOutboundQueueSize = 1024
	}

	if psConf.SubBufferSize <= 0 {
		psConf.SubBufferSize = 1024
	}

	if psConf.ValidateQueueSize <= 0 {
		psConf.ValidateQueueSize = 1024
	}

	if psConf.GossipSubDlo <= 0 {
		psConf.GossipSubDlo = 6
	}

	if psConf.GossipSubD <= 0 {
		psConf.GossipSubD = 8
	}

	if psConf.GossipSubDhi <= 0 {
		psConf.GossipSubDhi = 15
	}

	if psConf.GossipSubHeartbeatInterval <= 0 {
		psConf.GossipSubHeartbeatInterval = 700
	}

	if psConf.GossipSubHistoryGossip <= 0 {
		psConf.GossipSubHistoryGossip = 3
	}

	if psConf.GossipSubHistoryLength <= 0 {
		psConf.GossipSubHistoryLength = 10
	}

	if psConf.GossipSubDlo > psConf.GossipSubD || psConf.GossipSubD > psConf.GossipSubDhi {
		panic("setPubSubParameters, must GossipSubDlo <= GossipSubD <= GossipSubDhi")
	}

	// libp2p库内部全局变量暂不支持接口访问，强制只设置一次，主要是避免单元测试datarace，不影响常规执行
	setOnce.Do(func() {

		pubsub.GossipSubDlo = psConf.GossipSubDlo
		pubsub.GossipSubD = psConf.GossipSubD
		pubsub.GossipSubDhi = psConf.GossipSubDhi
		pubsub.GossipSubHeartbeatInterval = time.Duration(psConf.GossipSubHeartbeatInterval) * time.Millisecond
		pubsub.GossipSubHistoryGossip = psConf.GossipSubHistoryGossip
		pubsub.GossipSubHistoryLength = psConf.GossipSubHistoryLength
	})

}

// NewPubSub new pub sub
func NewPubSub(ctx context.Context, host host.Host, psConf *p2ptypes.PubSubConfig) (*PubSub, error) {
	p := &PubSub{
		topics: make(TopicMap),
		pt:     newPubsubTracer(ctx, host),
	}
	setPubSubParameters(psConf)

	psOpts := make([]pubsub.Option, 0)
	// pubsub消息默认会基于节点私钥进行签名和验签，支持关闭
	if psConf.DisablePubSubMsgSign {
		psOpts = append(psOpts, pubsub.WithMessageSigning(false), pubsub.WithStrictSignatureVerification(false))
	}
	psOpts = append(psOpts, pubsub.WithFloodPublish(true),
		pubsub.WithPeerOutboundQueueSize(psConf.PeerOutboundQueueSize),
		pubsub.WithValidateQueueSize(psConf.ValidateQueueSize),
		pubsub.WithPeerExchange(psConf.EnablePeerExchange),
		pubsub.WithMaxMessageSize(psConf.MaxMsgSize),
		pubsub.WithRawTracer(p.pt))

	//选择使用GossipSub
	ps, err := pubsub.NewGossipSub(ctx, host, psOpts...)
	if err != nil {
		return nil, err
	}

	p.PubSub = ps
	p.ctx = ctx
	p.host = host
	p.topics = make(TopicMap)
	p.config = psConf
	return p, nil
}

// HasTopic check topic exist
func (p *PubSub) HasTopic(topic string) bool {
	p.topicMutex.RLock()
	defer p.topicMutex.RUnlock()
	_, ok := p.topics[topic]
	return ok
}

func (p *PubSub) setTopic(topic string, tpInfo *topicinfo) {

	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	p.topics[topic] = tpInfo
}

// TryJoinTopic check exist before join topic
func (p *PubSub) TryJoinTopic(topic string, opts ...pubsub.TopicOpt) error {

	if p.HasTopic(topic) {
		return nil
	}
	return p.JoinTopic(topic, opts...)
}

// JoinTopic join topic
func (p *PubSub) JoinTopic(topic string, opts ...pubsub.TopicOpt) error {
	tp, err := p.Join(topic, opts...)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(p.ctx)
	p.setTopic(topic, &topicinfo{
		pubtopic: tp,
		ctx:      ctx,
		topic:    topic,
		cancel:   cancel,
	})
	return nil
}

// JoinAndSubTopic 加入topic&subTopic
func (p *PubSub) JoinAndSubTopic(topic string, callback SubCallBack, opts ...pubsub.TopicOpt) error {

	tp, err := p.Join(topic, opts...)
	if err != nil {
		return err
	}

	subscription, err := tp.Subscribe(pubsub.WithBufferSize(p.config.SubBufferSize))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(p.ctx)
	p.setTopic(topic, &topicinfo{
		pubtopic: tp,
		ctx:      ctx,
		topic:    topic,
		cancel:   cancel,
		sub:      subscription,
	})
	p.pt.addSubscriber(topic, callback)
	go p.subTopic(ctx, subscription, callback)
	return nil
}

// Publish 发布消息
func (p *PubSub) Publish(topic string, msg []byte) error {
	p.topicMutex.RLock()
	defer p.topicMutex.RUnlock()
	t, ok := p.topics[topic]
	if !ok {
		log.Error("pubsub publish", "no this topic", topic)
		return fmt.Errorf("no this topic:%v", topic)
	}
	err := t.pubtopic.Publish(t.ctx, msg)
	if err != nil {
		log.Error("pubsub publish", "err", err)
		return err
	}
	return nil

}

func (p *PubSub) subTopic(ctx context.Context, sub *pubsub.Subscription, callback SubCallBack) {
	topic := sub.Topic()
	for {
		got, err := sub.Next(ctx)
		if err != nil {
			log.Error("SubMsg", "topic", topic, "sub err", err)
			p.RemoveTopic(topic)
			return
		}
		// 不接收自己发布的信息, 即不用于内部模块之间的消息通信
		if p.host.ID() == got.ReceivedFrom {
			continue
		}
		callback(topic, got)
	}
}

// RemoveTopic remove topic
func (p *PubSub) RemoveTopic(topic string) {

	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()

	info, ok := p.topics[topic]
	if !ok {
		return
	}
	log.Info("RemoveTopic", "topic", topic)
	info.cancel()
	if info.sub != nil {
		info.sub.Cancel()
	}
	err := info.pubtopic.Close()
	if err != nil {
		log.Error("RemoveTopic", "topic", topic, "close topic err", err)
	}
	delete(p.topics, topic)

}

// FetchTopicPeers fetch peers with topic
func (p *PubSub) FetchTopicPeers(topic string) []peer.ID {
	p.topicMutex.RLock()
	defer p.topicMutex.RUnlock()
	topicobj, ok := p.topics[topic]
	if ok {
		return topicobj.pubtopic.ListPeers()
	}
	return nil
}

// TopicNum get topic number
func (p *PubSub) TopicNum() int {
	return len(p.GetTopics())

}
