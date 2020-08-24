package net

import (
	"context"
	"fmt"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

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
	topicMutex sync.Mutex
	ctx        context.Context
}

// SubMsg sub message
type SubMsg struct {
	Data  []byte
	Topic string
	From  string
}

type subCallBack func(msg *SubMsg)

// NewPubSub new pub sub
func NewPubSub(ctx context.Context, host host.Host) (*PubSub, error) {
	p := &PubSub{
		ps:     nil,
		topics: make(TopicMap),
	}
	//选择使用GossipSub
	ps, err := pubsub.NewGossipSub(ctx, host)
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
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	_, ok := p.topics[topic]
	return ok
}

// JoinTopicAndSubTopic 加入topic&subTopic
func (p *PubSub) JoinTopicAndSubTopic(topic string, callback subCallBack, opts ...pubsub.TopicOpt) error {

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
	p.subTopic(ctx, subscription, callback)
	return nil
}

// Publish 发布消息
func (p *PubSub) Publish(topic string, msg []byte) error {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	t, ok := p.topics[topic]
	if !ok {
		log.Error("publish", "no this topic", topic)
		return fmt.Errorf("no this topic:%v", topic)
	}

	err := t.pubtopic.Publish(t.ctx, msg)
	if err != nil {
		log.Error("publish", "err", err)
		return err
	}
	return nil
}

func (p *PubSub) subTopic(ctx context.Context, sub *pubsub.Subscription, callback subCallBack) {
	topic := sub.Topic()
	go func() {

		for {
			got, err := sub.Next(ctx)
			if err != nil {
				log.Error("SubMsg", "topic msg err", err, "topic", topic)
				p.RemoveTopic(topic)
				return
			}
			log.Debug("SubMsg", "readData size", len(got.GetData()), "from", got.GetFrom().String(), "recieveFrom", got.ReceivedFrom.Pretty(), "topic", topic)
			var data SubMsg
			data.Data = got.GetData()
			data.Topic = topic
			data.From = got.GetFrom().String()
			callback(&data)
		}
	}()

}

// RemoveTopic remove topic
func (p *PubSub) RemoveTopic(topic string) {

	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()

	info, ok := p.topics[topic]
	if ok {
		log.Info("RemoveTopic", topic, "")
		info.cancel()
		info.sub.Cancel()
		err := info.pubtopic.Close()
		if err != nil {
			log.Error("RemoveTopic", "topic", err)
		}
		delete(p.topics, topic)
	}

}

// FetchTopicPeers fetch peers with topic
func (p *PubSub) FetchTopicPeers(topic string) []peer.ID {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
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
