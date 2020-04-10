package net

import (
	"context"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/libp2p/go-libp2p-pubsub"
	"sync"
)

type TopicMap map[string]*pubsub.Topic
type SubMap map[string]*pubsub.Subscription

type PubSub struct {
	ps           *pubsub.PubSub
	topics       TopicMap
	topicMutex   sync.Mutex
	subscription SubMap
	subMutex     sync.Mutex
	ctx          context.Context
}

func NewPubSub(ctx context.Context, host host.Host) (*PubSub, error) {
	p := &PubSub{
		ps:     nil,
		topics: make(TopicMap, 0),
	}
	//选择使用GossipSub
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	p.ps = ps
	p.ctx = ctx
	p.subscription = make(SubMap)
	p.topics = make(TopicMap)
	return p, nil
}

//加入topic&subTopic
func (p *PubSub) JoinTopicAndSubTopic(topic string, opts ...pubsub.TopicOpt) error {
	Topic, err := p.ps.Join(topic, opts...)
	if err != nil {
		return err
	}
	subscription, err := Topic.Subscribe()
	if err != nil {
		return err
	}
	//p.topics = append(p.topics, Topic)
	p.topicMutex.Lock()
	p.topics[topic] = Topic
	p.topicMutex.Unlock()

	p.subMutex.Lock()
	p.subscription[topic] = subscription
	p.subMutex.Unlock()
	return nil
}

//Publish 发布消息
func (p *PubSub) Publish(topic string, msg []byte) bool {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	t, ok := p.topics[topic]
	if !ok {
		log.Error("publish", "no this topic", topic)
		return false
	}

	err := t.Publish(p.ctx, msg)
	if err != nil {
		log.Error("publish", "err", err)
		return false
	}
	return true
}

func (p *PubSub) SubMsg() {
	p.subMutex.Lock()
	defer p.subMutex.Unlock()

	for _, sub := range p.subscription {

		go func(subscription *pubsub.Subscription) {
			for {

				got, err := subscription.Next(p.ctx)
				if err != nil {
					log.Error("SubMsg", "topic msg err", err, "topic", subscription.Topic())
					if err == p.ctx.Err() {
						return
					}
				}
				log.Info("SubMsg", "readData", string(got.GetData()), "msgID")
			}
		}(sub)
	}
}

func (p *PubSub) RemoveTopic(topic string) {

	p.subMutex.Lock()
	defer p.subMutex.Unlock()
	sub, ok := p.subscription[topic]
	if ok {
		sub.Cancel() //取消对topic的订阅
	}

	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	v, ok := p.topics[topic]
	if ok {
		v.Close()
	}

}

func (p *PubSub) FetchTopicPeers(topic string) []peer.ID {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	topicobj, ok := p.topics[topic]
	if ok {
		return topicobj.ListPeers()
	}
	return nil
}

func (p *PubSub) FetchTopics() []string {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	var topics []string
	for topic := range p.topics {
		topics = append(topics, topic)
	}
	return topics
}

func (p *PubSub) TopicNum() int {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	return len(p.topics)
}
