package extension

import (
	"context"
	"fmt"
	"sync"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var log = log15.New("module", "pubsub")

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
	ps         *pubsub.PubSub
	topics     TopicMap
	topicMutex sync.RWMutex
	ctx        context.Context
}

// SubMsg sub message
type SubMsg *pubsub.Message

// SubCallBack 订阅消息回调函数
type SubCallBack func(topic string, msg SubMsg)

// NewPubSub new pub sub
func NewPubSub(ctx context.Context, host host.Host, opts ...pubsub.Option) (*PubSub, error) {
	p := &PubSub{
		ps:     nil,
		topics: make(TopicMap),
	}
	//选择使用GossipSub
	ps, err := pubsub.NewGossipSub(ctx, host, opts...)
	if err != nil {
		return nil, err
	}

	p.ps = ps
	p.ctx = ctx
	p.topics = make(TopicMap)
	return p, nil
}

// GetTopics get topics
func (p *PubSub) GetTopics() []string {
	return p.ps.GetTopics()
}

// HasTopic check topic exist
func (p *PubSub) HasTopic(topic string) bool {
	p.topicMutex.RLock()
	defer p.topicMutex.RUnlock()
	_, ok := p.topics[topic]
	return ok
}

// JoinAndSubTopic 加入topic&subTopic
func (p *PubSub) JoinAndSubTopic(topic string, callback SubCallBack, opts ...pubsub.TopicOpt) error {

	Topic, err := p.ps.Join(topic, opts...)
	if err != nil {
		return err
	}

	subscription, err := Topic.Subscribe()
	if err != nil {
		return err
	}
	//p.topics = append(p.topics, Topic)
	ctx, cancel := context.WithCancel(p.ctx)

	p.topicMutex.Lock()
	p.topics[topic] = &topicinfo{
		pubtopic: Topic,
		ctx:      ctx,
		topic:    topic,
		cancel:   cancel,
		sub:      subscription,
	}
	p.topicMutex.Unlock()
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
		callback(topic, got)
	}
}

// RemoveTopic remove topic
func (p *PubSub) RemoveTopic(topic string) {

	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()

	info, ok := p.topics[topic]
	if ok {
		log.Info("RemoveTopic", "topic", topic)
		info.cancel()
		info.sub.Cancel()
		err := info.pubtopic.Close()
		if err != nil {
			log.Error("RemoveTopic", "topic", topic, "close topic err", err)
		}
		delete(p.topics, topic)
	}

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
	return len(p.ps.GetTopics())

}
